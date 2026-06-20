package tools

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/anyproto/anytype-heart/pb"
	"github.com/anyproto/anytype-heart/pb/service"
	"github.com/anyproto/anytype-heart/pkg/lib/pb/model"
	"github.com/anyproto/anytype-heart/util/pbtypes"

	"github.com/anyproto/anytype-cli/core"
)

// imgctx sets a file object's createdInContext to the chat it was posted in
// (REST /files uploads omit this, leaving images orphaned from the chat).
func newImgctxCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "imgctx <fileObjectId> <contextId>",
		Short: "Set a file object's createdInContext to its chat (fix orphaned REST uploads)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			fileId, ctxId := args[0], args[1]
			return core.GRPCCall(func(ctx context.Context, client service.ClientCommandsClient) error {
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
				return emit(ok(map[string]any{"id": fileId, "createdInContext": ctxId}))
			})
		},
	}
}
