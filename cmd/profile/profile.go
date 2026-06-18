package profile

import (
	"github.com/spf13/cobra"

	profileSetCmd "github.com/anyproto/anytype-cli/cmd/profile/set"
)

func NewProfileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile <command>",
		Short: "Manage your account profile",
		Long:  "View and update the account profile (display name and icon).",
	}

	cmd.AddCommand(profileSetCmd.NewSetCmd())

	return cmd
}
