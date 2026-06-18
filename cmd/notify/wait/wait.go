package wait

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

const previewsSubId = "anytype-cli-notify"

func NewWaitCmd() *cobra.Command {
	var statePath string

	cmd := &cobra.Command{
		Use:   "wait",
		Short: "Block until there are pending notifications, then show them",
		Long: "Blocks on the server event stream (no polling). When something is\n" +
			"pending, prints the full pending list (chat messages, mentions, page\n" +
			"mentions, notifications), each leading with its id, and exits WITHOUT\n" +
			"consuming them. Acknowledge with `notify done <id...>` once handled, so a\n" +
			"crash before acking re-surfaces rather than drops (at-least-once).",
		RunE: func(cmd *cobra.Command, args []string) error {
			if statePath == "" {
				statePath = core.DefaultAckPath()
			}
			token, _, err := core.GetStoredSessionToken()
			if err != nil {
				return output.Error("not authenticated: %w", err)
			}

			er, err := core.ListenForEvents(token)
			if err != nil {
				return output.Error("failed to start event stream: %w", err)
			}
			if err := core.SubscribeMessagePreviews(previewsSubId); err != nil {
				return output.Error("%w", err)
			}
			if err := core.SubscribeMemberObjects(previewsSubId + "-member"); err != nil {
				output.Warning("page-mention subscription unavailable: %v", err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			sig := make(chan os.Signal, 1)
			signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
			go func() { <-sig; cancel() }()

			isWake := func(m *pb.EventMessage) bool {
				return m.GetChatAdd() != nil ||
					m.GetNotificationSend() != nil ||
					m.GetObjectDetailsAmend() != nil ||
					m.GetObjectDetailsSet() != nil
			}

			for {
				acked, err := core.LoadAckSet(statePath)
				if err != nil {
					return output.Error("failed to read ack-set: %w", err)
				}
				items, err := core.DrainNotifications(acked)
				if err != nil {
					output.Warning("drain error: %v", err)
				}
				if len(items) > 0 {
					for _, it := range items {
						output.Print("%s", core.FormatNotifItem(it))
					}
					return nil // peek only; consume via `notify done`
				}
				if _, err := er.WaitForEvent(ctx, isWake); err != nil {
					return nil // cancelled / stream closed
				}
			}
		},
	}

	cmd.Flags().StringVar(&statePath, "state", "", "ack-set file (default ~/.anytype/notify-acked)")
	return cmd
}
