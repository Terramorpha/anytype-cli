package list

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/anyproto/anytype-cli/core"
	"github.com/anyproto/anytype-cli/core/output"
)

func NewListCmd() *cobra.Command {
	var statePath string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Show pending (unacknowledged) notifications as JSON, without blocking",
		RunE: func(cmd *cobra.Command, args []string) error {
			if statePath == "" {
				statePath = core.DefaultAckPath()
			}
			acked, err := core.LoadAckSet(statePath)
			if err != nil {
				return output.Error("failed to read ack-set: %w", err)
			}
			items, err := core.DrainNotifications(acked)
			if err != nil {
				return output.Error("failed to read notifications: %w", err)
			}
			if items == nil {
				items = []core.NotifItem{}
			}
			out, err := json.Marshal(items)
			if err != nil {
				return output.Error("failed to encode notifications: %w", err)
			}
			output.Print("%s", out)
			return nil
		},
	}

	cmd.Flags().StringVar(&statePath, "state", "", "ack-set file (default ~/.anytype/notify-acked)")
	return cmd
}
