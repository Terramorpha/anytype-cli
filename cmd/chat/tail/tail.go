package tail

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/anyproto/anytype-cli/core"
	"github.com/anyproto/anytype-cli/core/output"
)

func NewTailCmd() *cobra.Command {
	var spaceId, chatId string
	var limit int

	cmd := &cobra.Command{
		Use:   "tail",
		Short: "Show the most recent messages in a space chat",
		RunE: func(cmd *cobra.Command, args []string) error {
			if spaceId == "" {
				spaceId = os.Getenv("ANYTYPE_SPACE")
			}
			if chatId == "" {
				chatId = os.Getenv("ANYTYPE_CHAT")
			}
			spaceId, chatId, err := core.ResolveChatTarget(spaceId, chatId)
			if err != nil {
				return output.Error("%w", err)
			}

			msgs, err := core.GetChatMessages(spaceId, chatId, limit)
			if err != nil {
				return output.Error("Failed to get messages: %w", err)
			}
			if len(msgs) == 0 {
				output.Info("No messages")
				return nil
			}
			for _, m := range msgs {
				output.Print("%s: %s", m.CreatorName, m.Text)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&spaceId, "space", "", "space id (or set ANYTYPE_SPACE)")
	cmd.Flags().StringVar(&chatId, "chat", "", "chat object id (defaults to the space's chat, or ANYTYPE_CHAT)")
	cmd.Flags().IntVarP(&limit, "limit", "n", 20, "number of recent messages to show")
	return cmd
}
