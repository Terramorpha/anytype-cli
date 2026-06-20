package tools

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/anyproto/anytype-heart/pb"
	"github.com/anyproto/anytype-heart/pb/service"
	"github.com/anyproto/anytype-heart/pkg/lib/pb/model"

	"github.com/anyproto/anytype-cli/core"
)

// geninvite makes a space shareable and generates a Writer invite (no approval).
func newGeninviteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "geninvite <spaceId>",
		Short: "Generate a join invite for a space (WithoutApprove, Writer)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			spaceId := args[0]
			return core.GRPCCall(func(ctx context.Context, client service.ClientCommandsClient) error {
				if ms, err := client.SpaceMakeShareable(ctx, &pb.RpcSpaceMakeShareableRequest{SpaceId: spaceId}); err != nil {
					return err
				} else if ms.Error != nil && ms.Error.Code != pb.RpcSpaceMakeShareableResponseError_NULL {
					return fmt.Errorf("make shareable: %s", ms.Error.Description)
				}
				resp, err := client.SpaceInviteGenerate(ctx, &pb.RpcSpaceInviteGenerateRequest{
					SpaceId:     spaceId,
					InviteType:  model.InviteType_WithoutApprove,
					Permissions: model.ParticipantPermissions_Writer,
				})
				if err != nil {
					return err
				}
				if resp.Error != nil && resp.Error.Code != pb.RpcSpaceInviteGenerateResponseError_NULL {
					return fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Description)
				}
				fmt.Printf("cid=%s\nkey=%s\nlink=https://invite.any.coop/%s#%s\n",
					resp.InviteCid, resp.InviteFileKey, resp.InviteCid, resp.InviteFileKey)
				return nil
			})
		},
	}
}

// getinvite fetches a space's current invite, if one exists.
func newGetinviteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "getinvite <spaceId>",
		Short: "Fetch a space's current invite, if one exists",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			spaceId := args[0]
			return core.GRPCCall(func(ctx context.Context, client service.ClientCommandsClient) error {
				resp, err := client.SpaceInviteGetCurrent(ctx, &pb.RpcSpaceInviteGetCurrentRequest{SpaceId: spaceId})
				if err != nil {
					return err
				}
				if resp.Error != nil && resp.Error.Code != pb.RpcSpaceInviteGetCurrentResponseError_NULL {
					return fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Description)
				}
				if resp.InviteCid == "" {
					fmt.Println("no current invite")
					return nil
				}
				fmt.Printf("cid=%s\nkey=%s\ntype=%v perms=%v\nlink=https://invite.any.coop/%s#%s\n",
					resp.InviteCid, resp.InviteFileKey, resp.InviteType, resp.Permissions, resp.InviteCid, resp.InviteFileKey)
				return nil
			})
		},
	}
}
