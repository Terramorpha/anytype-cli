package set

import (
	"github.com/spf13/cobra"

	"github.com/anyproto/anytype-cli/core"
	"github.com/anyproto/anytype-cli/core/output"
)

func NewSetCmd() *cobra.Command {
	var name, icon string

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set the profile display name and/or icon",
		Long:  "Set the account profile's display name and/or icon image.\n\nExample:\n  anytype profile set --name Claude --icon ./star.png",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := core.SetProfile(name, icon); err != nil {
				return output.Error("Failed to set profile: %w", err)
			}
			output.Success("Profile updated")
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "display name to set")
	cmd.Flags().StringVar(&icon, "icon", "", "local path to an image to use as the icon")
	return cmd
}
