package chat

import (
	"github.com/spf13/cobra"

	chatSendCmd "github.com/anyproto/anytype-cli/cmd/chat/send"
	chatTailCmd "github.com/anyproto/anytype-cli/cmd/chat/tail"
	chatWatchCmd "github.com/anyproto/anytype-cli/cmd/chat/watch"
)

func NewChatCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chat <command>",
		Short: "Send and read space chat messages",
		Long: "Send, read, and watch messages in a space's chat.\n\n" +
			"The space and chat can be set via --space/--chat flags or the\n" +
			"ANYTYPE_SPACE and ANYTYPE_CHAT environment variables. When the chat\n" +
			"is omitted, the space's default chat is used.",
	}

	cmd.AddCommand(chatSendCmd.NewSendCmd())
	cmd.AddCommand(chatTailCmd.NewTailCmd())
	cmd.AddCommand(chatWatchCmd.NewWatchCmd())

	return cmd
}
