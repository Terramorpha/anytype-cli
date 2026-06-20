package tools

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/anyproto/anytype-heart/pb"
	"github.com/anyproto/anytype-heart/pb/service"
	"github.com/anyproto/anytype-heart/pkg/lib/bundle"
	"github.com/anyproto/anytype-heart/util/pbtypes"
	"github.com/gogo/protobuf/types"

	"github.com/anyproto/anytype-cli/core"
)

// makecollection creates a Collection object — a manual list of objects (unlike
// a Set, which is a live query over a type). Add objects with `collectadd`.
func newMakecollectionCmd() *cobra.Command {
	var spaceId, name string
	cmd := &cobra.Command{
		Use:   "makecollection",
		Short: "Create a Collection object (manual object list; cf. makeset for type queries)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if spaceId == "" {
				spaceId = os.Getenv("ANYTYPE_SPACE")
			}
			if spaceId == "" {
				return fmt.Errorf("space id required (--space or ANYTYPE_SPACE)")
			}
			if name == "" {
				name = "Collection"
			}
			var newId string
			err := core.GRPCCall(func(ctx context.Context, c service.ClientCommandsClient) error {
				resp, err := c.ObjectCreate(ctx, &pb.RpcObjectCreateRequest{
					SpaceId:             spaceId,
					ObjectTypeUniqueKey: "ot-collection",
					Details: &types.Struct{Fields: map[string]*types.Value{
						bundle.RelationKeyName.String(): pbtypes.String(name),
					}},
				})
				if err != nil {
					return err
				}
				if resp.Error != nil && resp.Error.Code != pb.RpcObjectCreateResponseError_NULL {
					return fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Description)
				}
				newId = resp.ObjectId
				return nil
			})
			if err != nil {
				return err
			}
			return emit(ok(map[string]any{"objectId": newId, "name": name}))
		},
	}
	cmd.Flags().StringVar(&spaceId, "space", "", "space id (or ANYTYPE_SPACE)")
	cmd.Flags().StringVar(&name, "name", "", "collection name (default: \"Collection\")")
	return cmd
}

// collectadd adds objects to a collection (manual membership).
func newCollectaddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "collectadd <collectionId> <objectId>...",
		Short: "Add objects to a collection",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			collectionId, objs := args[0], args[1:]
			return core.GRPCCall(func(ctx context.Context, c service.ClientCommandsClient) error {
				resp, err := c.ObjectCollectionAdd(ctx, &pb.RpcObjectCollectionAddRequest{
					ContextId: collectionId,
					ObjectIds: objs,
				})
				if err != nil {
					return err
				}
				if resp.Error != nil && resp.Error.Code != pb.RpcObjectCollectionAddResponseError_NULL {
					return fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Description)
				}
				return emit(ok(map[string]any{"collectionId": collectionId, "added": objs}))
			})
		},
	}
	return cmd
}
