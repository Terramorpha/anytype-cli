package done

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/anyproto/anytype-cli/core"
	"github.com/anyproto/anytype-cli/core/output"
)

func NewDoneCmd() *cobra.Command {
	var statePath string

	cmd := &cobra.Command{
		Use:   "done <id> [id...]",
		Short: "Acknowledge (consume) pending notifications by id",
		Long:  "Marks the given notification ids as consumed so they no longer appear in `notify wait`/`notify list`.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if statePath == "" {
				statePath = core.DefaultAckPath()
			}
			if err := core.MarkAcked(statePath, args); err != nil {
				return output.Error("failed to record ack: %w", err)
			}
			out, _ := json.Marshal(map[string]any{"acknowledged": len(args), "ids": args})
			output.Print("%s", out)
			return nil
		},
	}

	cmd.Flags().StringVar(&statePath, "state", "", "ack-set file (default ~/.anytype/notify-acked)")
	return cmd
}
