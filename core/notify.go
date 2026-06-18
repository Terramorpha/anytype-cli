package core

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/anyproto/anytype-heart/pb"
	"github.com/anyproto/anytype-heart/pb/service"
	"github.com/anyproto/anytype-heart/pkg/lib/bundle"
	"github.com/anyproto/anytype-heart/pkg/lib/pb/model"
	"github.com/anyproto/anytype-heart/util/pbtypes"

	"github.com/anyproto/anytype-cli/core/config"
)

// ChatRef identifies a chat within a space.
type ChatRef struct {
	SpaceId string
	ChatId  string
	Name    string
}

// NotifItem is a single drained notification (a chat message, page mention, or
// local notification).
type NotifItem struct {
	Kind        string       `json:"kind"` // "chat" | "page-mention" | "notification"
	Id          string       `json:"id"`
	SpaceId     string       `json:"space_id,omitempty"`
	ChatId      string       `json:"chat_id,omitempty"`
	ChatName    string       `json:"chat_name,omitempty"`
	CreatorName string       `json:"creator_name,omitempty"`
	Text        string       `json:"text,omitempty"`
	HasMention  bool         `json:"has_mention,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// DefaultAckPath is where consumed (acked) notification ids are recorded.
func DefaultAckPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".anytype", "notify-acked")
}

// LoadAckSet reads the set of acknowledged notification ids.
func LoadAckSet(path string) (map[string]bool, error) {
	acked := map[string]bool{}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return acked, nil
		}
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if line := sc.Text(); line != "" {
			acked[line] = true
		}
	}
	return acked, sc.Err()
}

// MarkAcked records ids as consumed (idempotent append).
func MarkAcked(path string, ids []string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, id := range ids {
		if _, err := fmt.Fprintln(f, id); err != nil {
			return err
		}
	}
	return nil
}

// ListAllChats enumerates every chat in every space the account belongs to.
func ListAllChats() ([]ChatRef, error) {
	spaces, err := ListSpaces()
	if err != nil {
		return nil, err
	}
	var chats []ChatRef
	for _, sp := range spaces {
		spaceId := sp.SpaceId
		err := GRPCCall(func(ctx context.Context, client service.ClientCommandsClient) error {
			resp, err := client.ObjectSearch(ctx, &pb.RpcObjectSearchRequest{
				SpaceId: spaceId,
				Filters: []*model.BlockContentDataviewFilter{
					{
						// Space chats (chatDerived) AND page "discussion" sections,
						// which are chat objects with the distinct discussion layout.
						RelationKey: bundle.RelationKeyResolvedLayout.String(),
						Condition:   model.BlockContentDataviewFilter_In,
						Value:       pbtypes.IntList(int(model.ObjectType_chatDerived), int(model.ObjectType_discussion)),
					},
					{
						RelationKey: bundle.RelationKeyIsArchived.String(),
						Condition:   model.BlockContentDataviewFilter_NotEqual,
						Value:       pbtypes.Bool(true),
					},
				},
				Keys: []string{bundle.RelationKeyId.String(), bundle.RelationKeyName.String()},
			})
			if err != nil {
				return err
			}
			if resp.Error != nil && resp.Error.Code != pb.RpcObjectSearchResponseError_NULL {
				return fmt.Errorf("object search error: %s", resp.Error.Description)
			}
			for _, r := range resp.Records {
				chats = append(chats, ChatRef{
					SpaceId: spaceId,
					ChatId:  pbtypes.GetString(r, bundle.RelationKeyId.String()),
					Name:    pbtypes.GetString(r, bundle.RelationKeyName.String()),
				})
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return chats, nil
}

// SubscribeMessagePreviews registers the global message-previews subscription so
// that new messages in ANY chat are pushed onto the session event stream. This
// is the wake source; message content is read authoritatively during the drain.
func SubscribeMessagePreviews(subId string) error {
	return GRPCCall(func(ctx context.Context, client service.ClientCommandsClient) error {
		resp, err := client.ChatSubscribeToMessagePreviews(ctx, &pb.RpcChatSubscribeToMessagePreviewsRequest{
			SubId: subId,
		})
		if err != nil {
			return fmt.Errorf("failed to subscribe to message previews: %w", err)
		}
		if resp.Error != nil && resp.Error.Code != pb.RpcChatSubscribeToMessagePreviewsResponseError_NULL {
			return fmt.Errorf("message previews subscribe error: %s", resp.Error.Description)
		}
		return nil
	})
}

// ListNotifications returns local notifications (imports, membership, etc.).
func ListNotifications(includeRead bool) ([]*model.Notification, error) {
	var out []*model.Notification
	err := GRPCCall(func(ctx context.Context, client service.ClientCommandsClient) error {
		resp, err := client.NotificationList(ctx, &pb.RpcNotificationListRequest{
			IncludeRead: includeRead,
			Limit:       100,
		})
		if err != nil {
			return fmt.Errorf("failed to list notifications: %w", err)
		}
		if resp.Error != nil && resp.Error.Code != pb.RpcNotificationListResponseError_NULL {
			return fmt.Errorf("notification list error: %s", resp.Error.Description)
		}
		out = resp.Notifications
		return nil
	})
	return out, err
}

// DrainNotifications reads authoritative state — every chat's recent messages
// plus local notifications — and returns items whose id is not in seen. It does
// not mutate seen; the caller records ids after emitting (so a crash mid-emit
// re-surfaces rather than drops). Messages authored by this account are skipped.
func DrainNotifications(seen map[string]bool) ([]NotifItem, error) {
	myIdentity, _ := config.GetAccountIdFromConfig()

	chats, err := ListAllChats()
	if err != nil {
		return nil, err
	}

	var items []NotifItem
	for _, ch := range chats {
		msgs, err := GetChatMessages(ch.SpaceId, ch.ChatId, 100)
		if err != nil {
			continue // transient; next drain reconciles
		}
		for _, m := range msgs {
			if m.Id == "" || seen[m.Id] || m.Creator == myIdentity {
				continue
			}
			items = append(items, NotifItem{
				Kind:        "chat",
				Id:          m.Id,
				SpaceId:     ch.SpaceId,
				ChatId:      ch.ChatId,
				ChatName:    ch.Name,
				CreatorName: m.CreatorName,
				Text:        m.Text,
				HasMention:  m.HasMention,
				Attachments: m.Attachments,
			})
		}
	}

	notifs, err := ListNotifications(false)
	if err == nil {
		for _, n := range notifs {
			if n.Id == "" || seen[n.Id] {
				continue
			}
			items = append(items, NotifItem{
				Kind: "notification",
				Id:   n.Id,
				Text: fmt.Sprintf("%T", n.Payload),
			})
		}
	}

	// Page mentions: find objects whose outgoing `links` include my member
	// object — i.e. pages that mention me. (Reading the member's own backlinks
	// is unreliable for system objects; the forward query over normal objects is
	// well-indexed.) Phase 1: keyed by source id, so a repeat mention from the
	// same object is not re-surfaced.
	for spaceId, pid := range myParticipants() {
		for _, src := range findLinkingObjects(spaceId, pid) {
			if src.Id == "" || seen[src.Id] {
				continue
			}
			items = append(items, NotifItem{
				Kind:     "page-mention",
				Id:       src.Id,
				SpaceId:  spaceId,
				ChatName: src.Name,
			})
		}
	}

	return items, nil
}

type objectRef struct {
	Id   string
	Name string
}

// findLinkingObjects returns the objects that link/mention targetId, read from
// targetId's `backlinks` relation. ObjectShow is used (not ObjectSearch) because
// the computed backlinks relation is only populated by ObjectShow; the same call
// also returns the linked objects as dependencies, giving their names.
func findLinkingObjects(spaceId, targetId string) []objectRef {
	var refs []objectRef
	_ = GRPCCall(func(ctx context.Context, client service.ClientCommandsClient) error {
		resp, err := client.ObjectShow(ctx, &pb.RpcObjectShowRequest{
			SpaceId:                            spaceId,
			ObjectId:                           targetId,
			IncludeRelationsAsDependentObjects: true,
		})
		if err != nil || resp.Error != nil && resp.Error.Code != pb.RpcObjectShowResponseError_NULL {
			return nil
		}
		if resp.ObjectView == nil {
			return nil
		}
		names := map[string]string{}
		var backlinks []string
		for _, d := range resp.ObjectView.Details {
			names[d.Id] = pbtypes.GetString(d.Details, bundle.RelationKeyName.String())
			if d.Id == targetId {
				backlinks = pbtypes.GetStringList(d.Details, bundle.RelationKeyBacklinks.String())
			}
		}
		for _, id := range backlinks {
			refs = append(refs, objectRef{Id: id, Name: names[id]})
		}
		return nil
	})
	return refs
}

// myParticipants returns this account's participant object id in each space.
func myParticipants() map[string]string {
	identity, _ := config.GetAccountIdFromConfig()
	out := map[string]string{}
	spaces, err := ListSpaces()
	if err != nil {
		return out
	}
	for _, sp := range spaces {
		spaceId := sp.SpaceId
		_ = GRPCCall(func(ctx context.Context, client service.ClientCommandsClient) error {
			resp, err := client.ObjectSearch(ctx, &pb.RpcObjectSearchRequest{
				SpaceId: spaceId,
				Filters: []*model.BlockContentDataviewFilter{{
					RelationKey: bundle.RelationKeyResolvedLayout.String(),
					Condition:   model.BlockContentDataviewFilter_Equal,
					Value:       pbtypes.Int64(int64(model.ObjectType_participant)),
				}},
				Keys: []string{bundle.RelationKeyId.String(), bundle.RelationKeyIdentity.String()},
			})
			if err != nil || (resp.Error != nil && resp.Error.Code != pb.RpcObjectSearchResponseError_NULL) {
				return nil
			}
			for _, r := range resp.Records {
				if pbtypes.GetString(r, bundle.RelationKeyIdentity.String()) == identity {
					out[spaceId] = pbtypes.GetString(r, bundle.RelationKeyId.String())
				}
			}
			return nil
		})
	}
	return out
}

// SubscribeMemberObjects registers a push subscription on this account's member
// object in each space, so a backlinks change (a page mentioning me) emits an
// ObjectDetailsAmend on the session event stream — the wake for page mentions.
func SubscribeMemberObjects(subId string) error {
	for spaceId, pid := range myParticipants() {
		sp, id := spaceId, pid
		if err := GRPCCall(func(ctx context.Context, client service.ClientCommandsClient) error {
			_, err := client.ObjectSubscribeIds(ctx, &pb.RpcObjectSubscribeIdsRequest{
				SpaceId: sp,
				SubId:   subId,
				Ids:     []string{id},
				Keys:    []string{bundle.RelationKeyBacklinks.String(), bundle.RelationKeyName.String()},
			})
			return err
		}); err != nil {
			return fmt.Errorf("failed to subscribe to member object: %w", err)
		}
	}
	return nil
}
