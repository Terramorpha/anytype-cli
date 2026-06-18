package core

import (
	"context"
	"fmt"
	"os"

	"github.com/anyproto/anytype-heart/pb"
	"github.com/anyproto/anytype-heart/pb/service"
	"github.com/anyproto/anytype-heart/pkg/lib/bundle"
	"github.com/anyproto/anytype-heart/pkg/lib/pb/model"
	"github.com/anyproto/anytype-heart/util/pbtypes"
)

// ChatMessageItem is a flattened, display-friendly chat message.
type ChatMessageItem struct {
	Id          string
	OrderId     string
	Creator     string
	CreatorName string
	Text        string
	CreatedAt   int64
	HasMention  bool
}

// ResolveChatId finds the chat object in a space (the space's "General" chat).
// Returns the chat object id and its name.
func ResolveChatId(spaceId string) (string, string, error) {
	var chatId, name string
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
			// Deterministic pick: oldest chat first, so a multi-chat space
			// resolves to the original/primary chat rather than an arbitrary one.
			Sorts: []*model.BlockContentDataviewSort{
				{
					RelationKey: bundle.RelationKeyCreatedDate.String(),
					Type:        model.BlockContentDataviewSort_Asc,
				},
			},
			Keys: []string{bundle.RelationKeyId.String(), bundle.RelationKeyName.String()},
		})
		if err != nil {
			return fmt.Errorf("failed to search for chat: %w", err)
		}
		if resp.Error != nil && resp.Error.Code != pb.RpcObjectSearchResponseError_NULL {
			return fmt.Errorf("object search error: %s", resp.Error.Description)
		}
		if len(resp.Records) == 0 {
			return fmt.Errorf("no chat found in space %s", spaceId)
		}
		chatId = pbtypes.GetString(resp.Records[0], bundle.RelationKeyId.String())
		name = pbtypes.GetString(resp.Records[0], bundle.RelationKeyName.String())
		return nil
	})
	return chatId, name, err
}

// ResolveChatTarget resolves the space and chat to operate on. Empty arguments
// fall back to the ANYTYPE_SPACE / ANYTYPE_CHAT environment variables, and the
// chat id is resolved from the space's default chat when still empty.
func ResolveChatTarget(spaceId, chatId string) (string, string, error) {
	if spaceId == "" {
		spaceId = os.Getenv("ANYTYPE_SPACE")
	}
	if chatId == "" {
		chatId = os.Getenv("ANYTYPE_CHAT")
	}
	if spaceId == "" {
		return "", "", fmt.Errorf("space id required (use --space or set ANYTYPE_SPACE)")
	}
	if chatId == "" {
		var err error
		chatId, _, err = ResolveChatId(spaceId)
		if err != nil {
			return "", "", err
		}
	}
	return spaceId, chatId, nil
}

// participantNames returns a map of participant identity -> display name for a
// space. Chat messages reference their creator by identity, not participant id.
func participantNames(ctx context.Context, client service.ClientCommandsClient, spaceId string) map[string]string {
	names := map[string]string{}
	resp, err := client.ObjectSearch(ctx, &pb.RpcObjectSearchRequest{
		SpaceId: spaceId,
		Filters: []*model.BlockContentDataviewFilter{
			{
				RelationKey: bundle.RelationKeyResolvedLayout.String(),
				Condition:   model.BlockContentDataviewFilter_Equal,
				Value:       pbtypes.Int64(int64(model.ObjectType_participant)),
			},
		},
		Keys: []string{bundle.RelationKeyIdentity.String(), bundle.RelationKeyName.String()},
	})
	if err != nil || (resp.Error != nil && resp.Error.Code != pb.RpcObjectSearchResponseError_NULL) {
		return names
	}
	for _, r := range resp.Records {
		identity := pbtypes.GetString(r, bundle.RelationKeyIdentity.String())
		names[identity] = pbtypes.GetString(r, bundle.RelationKeyName.String())
	}
	return names
}

// SendChatMessage posts a plain-text message to a chat.
func SendChatMessage(chatId, text string) (string, error) {
	var messageId string
	err := GRPCCall(func(ctx context.Context, client service.ClientCommandsClient) error {
		resp, err := client.ChatAddMessage(ctx, &pb.RpcChatAddMessageRequest{
			ChatObjectId: chatId,
			Message: &model.ChatMessage{
				Message: &model.ChatMessageMessageContent{
					Text:  text,
					Style: model.BlockContentText_Paragraph,
				},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}
		if resp.Error != nil && resp.Error.Code != pb.RpcChatAddMessageResponseError_NULL {
			return fmt.Errorf("chat add message error: %s", resp.Error.Description)
		}
		messageId = resp.MessageId
		return nil
	})
	return messageId, err
}

// GetChatMessages returns up to limit most recent messages for a chat, with
// creator display names resolved from the given space's participants.
func GetChatMessages(spaceId, chatId string, limit int) ([]ChatMessageItem, error) {
	var items []ChatMessageItem
	err := GRPCCall(func(ctx context.Context, client service.ClientCommandsClient) error {
		resp, err := client.ChatGetMessages(ctx, &pb.RpcChatGetMessagesRequest{
			ChatObjectId: chatId,
			Limit:        int32(limit),
		})
		if err != nil {
			return fmt.Errorf("failed to get messages: %w", err)
		}
		if resp.Error != nil && resp.Error.Code != pb.RpcChatGetMessagesResponseError_NULL {
			return fmt.Errorf("chat get messages error: %s", resp.Error.Description)
		}
		names := participantNames(ctx, client, spaceId)
		for _, m := range resp.Messages {
			items = append(items, FlattenChatMessage(m, names))
		}
		return nil
	})
	return items, err
}

// FlattenChatMessage converts a raw chat message into a display-friendly item,
// resolving the creator name from the given identity->name map.
func FlattenChatMessage(m *model.ChatMessage, names map[string]string) ChatMessageItem {
	item := ChatMessageItem{
		Id:         m.Id,
		OrderId:    m.OrderId,
		Creator:    m.Creator,
		CreatedAt:  m.CreatedAt,
		HasMention: m.HasMention,
	}
	if m.Message != nil {
		item.Text = m.Message.Text
	}
	if n, ok := names[m.Creator]; ok && n != "" {
		item.CreatorName = n
	} else {
		item.CreatorName = m.Creator
	}
	return item
}

// ChatParticipantNames returns a map of participant identity -> display name.
func ChatParticipantNames(spaceId string) (map[string]string, error) {
	var names map[string]string
	err := GRPCCall(func(ctx context.Context, client service.ClientCommandsClient) error {
		names = participantNames(ctx, client, spaceId)
		return nil
	})
	return names, err
}

// SubscribeChat registers a last-messages subscription for a chat. New messages
// then arrive as EventChatAdd events on the session event stream, tagged subId.
func SubscribeChat(chatId, subId string, limit int) error {
	return GRPCCall(func(ctx context.Context, client service.ClientCommandsClient) error {
		resp, err := client.ChatSubscribeLastMessages(ctx, &pb.RpcChatSubscribeLastMessagesRequest{
			ChatObjectId: chatId,
			Limit:        int32(limit),
			SubId:        subId,
		})
		if err != nil {
			return fmt.Errorf("failed to subscribe to chat: %w", err)
		}
		if resp.Error != nil && resp.Error.Code != pb.RpcChatSubscribeLastMessagesResponseError_NULL {
			return fmt.Errorf("chat subscribe error: %s", resp.Error.Description)
		}
		return nil
	})
}

// UnsubscribeChat removes a chat subscription created with SubscribeChat.
func UnsubscribeChat(chatId, subId string) error {
	return GRPCCall(func(ctx context.Context, client service.ClientCommandsClient) error {
		_, err := client.ChatUnsubscribe(ctx, &pb.RpcChatUnsubscribeRequest{
			ChatObjectId: chatId,
			SubId:        subId,
		})
		return err
	})
}
