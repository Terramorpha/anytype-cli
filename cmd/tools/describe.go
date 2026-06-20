package tools

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/types"
	"github.com/spf13/cobra"

	"github.com/anyproto/anytype-heart/pb"
	"github.com/anyproto/anytype-heart/pb/service"
	"github.com/anyproto/anytype-heart/pkg/lib/bundle"
	"github.com/anyproto/anytype-heart/pkg/lib/pb/model"
	"github.com/anyproto/anytype-heart/util/pbtypes"

	"github.com/anyproto/anytype-cli/core"
)

// describe dumps an object's identity (name, layout, type, keys) and the
// relations it carries (key + format) — so you can read off the exact ids/keys
// to feed into the action tools.
func newDescribeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe <objectId>",
		Short: "Dump an object's identity and its relations (keys + formats)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return core.GRPCCall(func(ctx context.Context, c service.ClientCommandsClient) error {
				show, err := c.ObjectShow(ctx, &pb.RpcObjectShowRequest{ObjectId: args[0]})
				if err != nil {
					return err
				}
				if show.Error != nil && show.Error.Code != pb.RpcObjectShowResponseError_NULL {
					return fmt.Errorf("ObjectShow: %s", show.Error.Description)
				}
				var det *types.Struct
				for _, d := range show.ObjectView.GetDetails() {
					if d.Id == args[0] {
						det = d.GetDetails()
						break
					}
				}
				type relInfo struct {
					Key    string `json:"key"`
					Format string `json:"format"`
				}
				out := map[string]any{"id": args[0]}
				if det != nil {
					out["name"] = pbtypes.GetString(det, bundle.RelationKeyName.String())
					out["layout"] = layoutName(model.ObjectTypeLayout(pbtypes.GetInt64(det, bundle.RelationKeyResolvedLayout.String())))
					out["type"] = pbtypes.GetString(det, bundle.RelationKeyType.String())
					for label, key := range map[string]string{
						"uniqueKey":   bundle.RelationKeyUniqueKey.String(),
						"apiKey":      bundle.RelationKeyApiObjectKey.String(),
						"relationKey": bundle.RelationKeyRelationKey.String(),
					} {
						if v := pbtypes.GetString(det, key); v != "" {
							out[label] = v
						}
					}
				}
				rels := []relInfo{}
				for _, rl := range show.ObjectView.GetRelationLinks() {
					rels = append(rels, relInfo{Key: rl.Key, Format: model.RelationFormat(rl.Format).String()})
				}
				out["relations"] = rels
				return emit(out)
			})
		},
	}
	return cmd
}
