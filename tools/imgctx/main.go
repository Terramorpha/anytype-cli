// Command imgctx sets a file object's createdInContext / createdInContextRef to
// the chat it was posted in (REST /files uploads omit these). Throwaway helper.
//
//	imgctx <fileObjectId> <contextId(chatObjectId)>
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
		fmt.Fprintln(os.Stderr, "usage: imgctx <fileObjectId> <contextId>")
		os.Exit(1)
	}
	fileId, ctxId := os.Args[1], os.Args[2]
	err := core.GRPCCall(func(ctx context.Context, client service.ClientCommandsClient) error {
		resp, err := client.ObjectSetDetails(ctx, &pb.RpcObjectSetDetailsRequest{
			ContextId: fileId,
			Details: []*model.Detail{
				{Key: "createdInContext", Value: pbtypes.String(ctxId)},
				{Key: "createdInContextRef", Value: pbtypes.String(ctxId)},
			},
		})
		if err != nil {
			return err
		}
		if resp.Error != nil && resp.Error.Code != pb.RpcObjectSetDetailsResponseError_NULL {
			return fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Description)
		}
		fmt.Println("OK: createdInContext set on", fileId)
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
}
