package notify

import (
	"github.com/spf13/cobra"

	notifyDoneCmd "github.com/anyproto/anytype-cli/cmd/notify/done"
	notifyListCmd "github.com/anyproto/anytype-cli/cmd/notify/list"
	notifyWaitCmd "github.com/anyproto/anytype-cli/cmd/notify/wait"
)

func NewNotifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "notify <command>",
		Short: "Wait for notifications (new messages, mentions, events)",
		Long: "Push-based notifications backed by the server event stream.\n\n" +
			"`notify wait` blocks until something is pending, then shows it (peek).\n" +
			"`notify list` peeks without blocking. `notify done <id...>` consumes.\n" +
			"Explicit acknowledgment gives at-least-once: nothing is dropped on a crash.",
	}
	cmd.AddCommand(notifyWaitCmd.NewWaitCmd())
	cmd.AddCommand(notifyListCmd.NewListCmd())
	cmd.AddCommand(notifyDoneCmd.NewDoneCmd())
	return cmd
}
