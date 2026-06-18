// Command setq points a Set object at a type by setting its setOf relation,
// turning an unconfigured set into a live query. Throwaway helper.
//
//	setq <setObjectId> <typeId>
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/anyproto/anytype-heart/pb"
	"github.com/anyproto/anytype-heart/pb/service"
	"github.com/anyproto/anytype-heart/pkg/lib/pb/model"
	"github.com/anyproto/anytype-heart/util/pbtypes"

	"github.com/anyproto/anytype-cli/core"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: setq <setObjectId> <typeId>")
		os.Exit(1)
	}
	setId, typeId := os.Args[1], os.Args[2]
	err := core.GRPCCall(func(ctx context.Context, client service.ClientCommandsClient) error {
		resp, err := client.ObjectSetDetails(ctx, &pb.RpcObjectSetDetailsRequest{
			ContextId: setId,
			Details: []*model.Detail{
				{Key: "setOf", Value: pbtypes.StringList([]string{typeId})},
			},
		})
		if err != nil {
			return err
		}
		if resp.Error != nil && resp.Error.Code != pb.RpcObjectSetDetailsResponseError_NULL {
			return fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Description)
		}
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
	fmt.Println("OK: setOf set")
}
