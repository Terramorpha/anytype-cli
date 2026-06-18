// Command geninvite generates a join invite for a space (WithoutApprove, Writer).
// Targets whatever server the ANYTYPE_GRPC_PORT env points at. Throwaway helper.
//
//	geninvite <spaceId>
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/anyproto/anytype-heart/pb"
	"github.com/anyproto/anytype-heart/pb/service"
	"github.com/anyproto/anytype-heart/pkg/lib/pb/model"

	"github.com/anyproto/anytype-cli/core"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: geninvite <spaceId>")
		os.Exit(1)
	}
	spaceId := os.Args[1]
	err := core.GRPCCall(func(ctx context.Context, client service.ClientCommandsClient) error {
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
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
}
