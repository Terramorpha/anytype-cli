package core

import (
	"context"
	"fmt"

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

	return items, nil
}
