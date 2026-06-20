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

// makeset creates a live Set object whose source is a type, so it shows every
// object of that type (a query). The type is resolvable by api key / name / id.
func newMakesetCmd() *cobra.Command {
	var spaceId, typeKey, name string
	cmd := &cobra.Command{
		Use:   "makeset",
		Short: "Create a live Set/Query object over a type",
		Long: "Create a Set whose source is a type, so it lists every object of that\n" +
			"type as a live query.\n\nExample:\n  anytype tools makeset --type source --name Bibliography",
		RunE: func(cmd *cobra.Command, args []string) error {
			if spaceId == "" {
				spaceId = os.Getenv("ANYTYPE_SPACE")
			}
			if spaceId == "" {
				return fmt.Errorf("space id required (--space or ANYTYPE_SPACE)")
			}
			if name == "" {
				name = typeKey + " (all)"
			}
			var newId string
			err := core.GRPCCall(func(ctx context.Context, c service.ClientCommandsClient) error {
				typeId, err := resolveTypeId(ctx, c, spaceId, typeKey)
				if err != nil {
					return err
				}
				details := &types.Struct{Fields: map[string]*types.Value{
					bundle.RelationKeyName.String(): pbtypes.String(name),
				}}
				resp, err := c.ObjectCreateSet(ctx, &pb.RpcObjectCreateSetRequest{
					SpaceId: spaceId,
					Source:  []string{typeId},
					Details: details,
				})
				if err != nil {
					return err
				}
				if resp.Error != nil && resp.Error.Code != pb.RpcObjectCreateSetResponseError_NULL {
					return fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Description)
				}
				newId = resp.ObjectId
				return nil
			})
			if err != nil {
				return err
			}
			fmt.Printf("OK: created set %q -> %s\n", name, newId)
			return nil
		},
	}
	cmd.Flags().StringVar(&spaceId, "space", "", "space id (or ANYTYPE_SPACE)")
	cmd.Flags().StringVar(&typeKey, "type", "", "type api key / name / id to query (required)")
	cmd.Flags().StringVar(&name, "name", "", "name for the set (default: \"<type> (all)\")")
	_ = cmd.MarkFlagRequired("type")
	return cmd
}
