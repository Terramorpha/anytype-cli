package send

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/anyproto/anytype-cli/core"
	"github.com/anyproto/anytype-cli/core/output"
)

func NewSendCmd() *cobra.Command {
	var spaceId, chatId, replyTo, attachType string
	var attach, files []string

	cmd := &cobra.Command{
		Use:   "send [message]",
		Short: "Send a message to a space chat (optionally with attachments)",
		Long: "Send a message to a space chat.\n\n" +
			"Attach existing objects with --attach <id> (repeatable), or upload and\n" +
			"attach local files with --file <path> (repeatable). Uploaded files are tied\n" +
			"to the chat (createdInContext) and typed automatically (image for image\n" +
			"files, otherwise file). The message text is optional when attachments are given.",
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			spaceId, chatId, err := core.ResolveChatTarget(spaceId, chatId)
			if err != nil {
				return output.Error("%w", err)
			}
			text := strings.Join(args, " ")

			var attachments []core.Attachment
			// Existing objects: default to a "link" card unless overridden.
			for _, id := range attach {
				t := attachType
				if t == "" {
					t = "link"
				}
				attachments = append(attachments, core.Attachment{Target: id, Type: t})
			}
			// Local files: upload (tied to the chat), then attach.
			for _, p := range files {
				att, err := core.UploadFileToChat(spaceId, chatId, p)
				if err != nil {
					return output.Error("Failed to upload %s: %w", p, err)
				}
				if attachType != "" {
					att.Type = attachType
				}
				attachments = append(attachments, att)
				output.Info("Uploaded %s (%s)", p, att.Target)
			}

			if text == "" && len(attachments) == 0 {
				return output.Error("nothing to send: provide a message, --attach, or --file")
			}

			id, err := core.SendChatMessage(chatId, text, replyTo, attachments)
			if err != nil {
				return output.Error("Failed to send message: %w", err)
			}
			output.Success("Message sent (%s)", id)
			return nil
		},
	}

	cmd.Flags().StringVar(&spaceId, "space", "", "space id (or set ANYTYPE_SPACE)")
	cmd.Flags().StringVar(&chatId, "chat", "", "chat object id (defaults to the space's chat, or ANYTYPE_CHAT)")
	cmd.Flags().StringVar(&replyTo, "reply-to", "", "message id to reply to (threaded answer)")
	cmd.Flags().StringArrayVar(&attach, "attach", nil, "attach an existing object by id (repeatable)")
	cmd.Flags().StringArrayVar(&files, "file", nil, "upload a local file and attach it (repeatable)")
	cmd.Flags().StringVar(&attachType, "attach-type", "", "attachment render type: image|file|link (default: link for --attach, auto for --file)")
	return cmd
}
