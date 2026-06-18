package watch

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/anyproto/anytype-heart/pb"

	"github.com/anyproto/anytype-cli/core"
	"github.com/anyproto/anytype-cli/core/output"
)

const watchSubId = "anytype-cli-watch"

func NewWatchCmd() *cobra.Command {
	var spaceId, chatId string

	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch a space chat and print new messages as they arrive",
		Long:  "Streams new messages from the server (no polling) until interrupted (Ctrl-C).",
		RunE: func(cmd *cobra.Command, args []string) error {
			spaceId, chatId, err := core.ResolveChatTarget(spaceId, chatId)
			if err != nil {
				return output.Error("%w", err)
			}

			token, _, err := core.GetStoredSessionToken()
			if err != nil {
				return output.Error("not authenticated: %w", err)
			}

			// Start listening to the session event stream, then subscribe to the
			// chat so new messages are delivered as EventChatAdd events.
			er, err := core.ListenForEvents(token)
			if err != nil {
				return output.Error("failed to start event stream: %w", err)
			}
			if err := core.SubscribeChat(chatId, watchSubId, 1); err != nil {
				return output.Error("%w", err)
			}
			defer func() { _ = core.UnsubscribeChat(chatId, watchSubId) }()

			names, _ := core.ChatParticipantNames(spaceId)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			sig := make(chan os.Signal, 1)
			signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
			go func() {
				<-sig
				cancel()
			}()

			output.Info("Watching chat (streaming). Press Ctrl-C to stop.")

			isOurAdd := func(m *pb.EventMessage) bool {
				ca := m.GetChatAdd()
				if ca == nil {
					return false
				}
				for _, s := range ca.SubIds {
					if s == watchSubId {
						return true
					}
				}
				return false
			}

			for {
				ev, err := er.WaitForEvent(ctx, isOurAdd)
				if err != nil {
					// context cancelled (Ctrl-C) or stream closed
					output.Info("Stopped.")
					return nil
				}
				// Validate at the trust boundary: events come from the server,
				// so a body-less ChatAdd is possible and must not panic the watcher.
				ca := ev.GetChatAdd()
				if ca == nil || ca.Message == nil {
					continue
				}
				msg := core.FlattenChatMessage(ca.Message, names)
				output.Print("%s: %s", msg.CreatorName, msg.Text)
			}
		},
	}

	cmd.Flags().StringVar(&spaceId, "space", "", "space id (or set ANYTYPE_SPACE)")
	cmd.Flags().StringVar(&chatId, "chat", "", "chat object id (defaults to the space's chat, or ANYTYPE_CHAT)")
	return cmd
}
