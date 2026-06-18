// Command getinvite fetches a space's CURRENT invite (if one exists), without
// owner rights. Throwaway helper.   getinvite <spaceId>
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/anyproto/anytype-heart/pb"
	"github.com/anyproto/anytype-heart/pb/service"

	"github.com/anyproto/anytype-cli/core"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: getinvite <spaceId>")
		os.Exit(1)
	}
	spaceId := os.Args[1]
	err := core.GRPCCall(func(ctx context.Context, client service.ClientCommandsClient) error {
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
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
}
