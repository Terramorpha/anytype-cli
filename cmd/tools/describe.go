package tools

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

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
				if det != nil {
					fmt.Printf("name:        %s\n", pbtypes.GetString(det, bundle.RelationKeyName.String()))
					lay := model.ObjectTypeLayout(pbtypes.GetInt64(det, bundle.RelationKeyResolvedLayout.String()))
					fmt.Printf("layout:      %s\n", layoutName(lay))
					fmt.Printf("type:        %s\n", pbtypes.GetString(det, bundle.RelationKeyType.String()))
					for _, kk := range []struct{ label, key string }{
						{"uniqueKey", bundle.RelationKeyUniqueKey.String()},
						{"apiKey", bundle.RelationKeyApiObjectKey.String()},
						{"relationKey", bundle.RelationKeyRelationKey.String()},
					} {
						if v := pbtypes.GetString(det, kk.key); v != "" {
							fmt.Printf("%-12s %s\n", kk.label+":", v)
						}
					}
				}
				fmt.Println("\nrelations on this object:")
				w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
				fmt.Fprintln(w, "  KEY\tFORMAT")
				for _, rl := range show.ObjectView.GetRelationLinks() {
					fmt.Fprintf(w, "  %s\t%s\n", rl.Key, model.RelationFormat(rl.Format))
				}
				return w.Flush()
			})
		},
	}
	return cmd
}
