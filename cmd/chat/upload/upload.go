package upload

import (
	"github.com/spf13/cobra"

	"github.com/anyproto/anytype-cli/core"
	"github.com/anyproto/anytype-cli/core/output"
)

func NewUploadCmd() *cobra.Command {
	var spaceId, chatId string

	cmd := &cobra.Command{
		Use:   "upload <path>",
		Short: "Upload a local file into the chat's space and print its object id",
		Long: "Upload a local file into the space and tie it to the chat (createdInContext).\n" +
			"Prints the resulting object id, which you can pass to `chat send --attach`.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			spaceId, chatId, err := core.ResolveChatTarget(spaceId, chatId)
			if err != nil {
				return output.Error("%w", err)
			}
			att, err := core.UploadFileToChat(spaceId, chatId, args[0])
			if err != nil {
				return output.Error("Failed to upload: %w", err)
			}
			output.Success("Uploaded %s — object id: %s (type %s)", args[0], att.Target, att.Type)
			return nil
		},
	}

	cmd.Flags().StringVar(&spaceId, "space", "", "space id (or set ANYTYPE_SPACE)")
	cmd.Flags().StringVar(&chatId, "chat", "", "chat object id (defaults to the space's chat, or ANYTYPE_CHAT)")
	return cmd
}
