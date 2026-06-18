package send

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/anyproto/anytype-cli/core"
	"github.com/anyproto/anytype-cli/core/output"
)

func NewSendCmd() *cobra.Command {
	var spaceId, chatId string

	cmd := &cobra.Command{
		Use:   "send <message>",
		Short: "Send a message to a space chat",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, chatId, err := core.ResolveChatTarget(spaceId, chatId)
			if err != nil {
				return output.Error("%w", err)
			}

			text := strings.Join(args, " ")
			id, err := core.SendChatMessage(chatId, text)
			if err != nil {
				return output.Error("Failed to send message: %w", err)
			}
			output.Success("Message sent (%s)", id)
			return nil
		},
	}

	cmd.Flags().StringVar(&spaceId, "space", "", "space id (or set ANYTYPE_SPACE)")
	cmd.Flags().StringVar(&chatId, "chat", "", "chat object id (defaults to the space's chat, or ANYTYPE_CHAT)")
	return cmd
}
