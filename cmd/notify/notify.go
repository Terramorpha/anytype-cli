package notify

import (
	"github.com/spf13/cobra"

	notifyWaitCmd "github.com/anyproto/anytype-cli/cmd/notify/wait"
)

func NewNotifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "notify <command>",
		Short: "Wait for notifications (new messages, mentions, events)",
		Long: "Push-based notifications backed by the server event stream.\n\n" +
			"`notify wait` blocks until something happens, then drains every\n" +
			"unseen message and notification (level-triggered), deduplicated by id.",
	}
	cmd.AddCommand(notifyWaitCmd.NewWaitCmd())
	return cmd
}
