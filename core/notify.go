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

// NotifItem is a single drained notification (a chat message or a local notification).
type NotifItem struct {
	Kind        string // "chat" | "notification"
	Id          string
	SpaceId     string
	ChatId      string
	ChatName    string
	CreatorName string
	Text        string
	HasMention  bool
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

// FormatNotifItem renders a pending item one-line, leading with its id so it can
// be passed to `notify done <id>`.
func FormatNotifItem(it NotifItem) string {
	switch it.Kind {
	case "notification":
		return fmt.Sprintf("id=%s [notification] %s", it.Id, it.Text)
	case "page-mention":
		return fmt.Sprintf("id=%s [page-mention space=%s (%s)] you were mentioned",
			it.Id, it.SpaceId, it.ChatName)
	default: // chat
		mention := ""
		if it.HasMention {
			mention = " @mention"
		}
		return fmt.Sprintf("id=%s [space=%s chat=%s (%s)%s] %s: %s",
			it.Id, it.SpaceId, it.ChatId, it.ChatName, mention, it.CreatorName, it.Text)
	}
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
						RelationKey: bundle.RelationKeyResolvedLayout.String(),
						Condition:   model.BlockContentDataviewFilter_Equal,
						Value:       pbtypes.Int64(int64(model.ObjectType_chatDerived)),
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
		msgs, err := GetChatMessages(ch.SpaceId, ch.ChatId, 30)
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

	// Page mentions: objects that link/mention my member object show up in its
	// backlinks set. New source objects are surfaced (Phase 1: set-keyed by
	// source id, so a repeat mention from the same object is not re-surfaced).
	for spaceId, pid := range myParticipants() {
		for _, srcId := range readBacklinks(spaceId, pid) {
			if srcId == "" || seen[srcId] {
				continue
			}
			items = append(items, NotifItem{
				Kind:     "page-mention",
				Id:       srcId,
				SpaceId:  spaceId,
				ChatName: resolveObjectName(spaceId, srcId),
			})
		}
	}

	return items, nil
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
			if err != nil || resp.Error != nil {
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

// readBacklinks returns the source object ids that link/mention the given object.
func readBacklinks(spaceId, objectId string) []string {
	var links []string
	_ = GRPCCall(func(ctx context.Context, client service.ClientCommandsClient) error {
		resp, err := client.ObjectSearch(ctx, &pb.RpcObjectSearchRequest{
			SpaceId: spaceId,
			Filters: []*model.BlockContentDataviewFilter{{
				RelationKey: bundle.RelationKeyId.String(),
				Condition:   model.BlockContentDataviewFilter_Equal,
				Value:       pbtypes.String(objectId),
			}},
			Keys: []string{bundle.RelationKeyBacklinks.String()},
		})
		if err != nil || resp.Error != nil || len(resp.Records) == 0 {
			return nil
		}
		links = pbtypes.GetStringList(resp.Records[0], bundle.RelationKeyBacklinks.String())
		return nil
	})
	return links
}

// resolveObjectName returns an object's display name (best-effort).
func resolveObjectName(spaceId, objectId string) string {
	name := objectId
	_ = GRPCCall(func(ctx context.Context, client service.ClientCommandsClient) error {
		resp, err := client.ObjectSearch(ctx, &pb.RpcObjectSearchRequest{
			SpaceId: spaceId,
			Filters: []*model.BlockContentDataviewFilter{{
				RelationKey: bundle.RelationKeyId.String(),
				Condition:   model.BlockContentDataviewFilter_Equal,
				Value:       pbtypes.String(objectId),
			}},
			Keys: []string{bundle.RelationKeyName.String()},
		})
		if err != nil || resp.Error != nil || len(resp.Records) == 0 {
			return nil
		}
		if n := pbtypes.GetString(resp.Records[0], bundle.RelationKeyName.String()); n != "" {
			name = n
		}
		return nil
	})
	return name
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
