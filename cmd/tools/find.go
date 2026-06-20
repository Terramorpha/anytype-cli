package tools

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/anyproto/anytype-heart/pb"
	"github.com/anyproto/anytype-heart/pb/service"
	"github.com/anyproto/anytype-heart/pkg/lib/bundle"
	"github.com/anyproto/anytype-heart/pkg/lib/pb/model"
	"github.com/anyproto/anytype-heart/util/pbtypes"

	"github.com/anyproto/anytype-cli/core"
)

// layoutFilter maps a friendly --kind value to a resolvedLayout filter.
var layoutFilter = map[string]model.ObjectTypeLayout{
	"type":       model.ObjectType_objectType,
	"relation":   model.ObjectType_relation,
	"set":        model.ObjectType_set,
	"collection": model.ObjectType_collection,
	"page":       model.ObjectType_basic,
	"note":       model.ObjectType_note,
	"bookmark":   model.ObjectType_bookmark,
}

// find looks up objects by name and prints each match with its id, layout and
// key — the explicit label→id step, so duplicate labels are visible rather
// than silently resolved. Pass the printed id to the action tools.
func newFindCmd() *cobra.Command {
	var spaceId, kind string
	var limit int
	cmd := &cobra.Command{
		Use:   "find <name>",
		Short: "Look up objects by name; print id, layout and key for each match",
		Long: "Search a space by name and list every match with its bafy id, layout and\n" +
			"internal key. Use this to get the id to pass to action tools, and to spot\n" +
			"duplicate labels.\n\n" +
			"  anytype tools find \"BibTeX\" --kind relation\n" +
			"  anytype tools find Source --kind type",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if spaceId == "" {
				spaceId = os.Getenv("ANYTYPE_SPACE")
			}
			if spaceId == "" {
				return fmt.Errorf("space id required (--space or ANYTYPE_SPACE)")
			}
			query := args[0]
			for _, a := range args[1:] {
				query += " " + a
			}
			var filters []*model.BlockContentDataviewFilter
			if kind != "" {
				lay, ok := layoutFilter[kind]
				if !ok {
					return fmt.Errorf("unknown --kind %q (type|relation|set|collection|page|note|bookmark)", kind)
				}
				filters = append(filters, &model.BlockContentDataviewFilter{
					RelationKey: bundle.RelationKeyResolvedLayout.String(),
					Condition:   model.BlockContentDataviewFilter_Equal,
					Value:       pbtypes.Int64(int64(lay)),
				})
			}
			return core.GRPCCall(func(ctx context.Context, c service.ClientCommandsClient) error {
				resp, err := c.ObjectSearch(ctx, &pb.RpcObjectSearchRequest{
					SpaceId:  spaceId,
					FullText: query,
					Filters:  filters,
					Limit:    int32(limit),
					Keys: []string{
						bundle.RelationKeyId.String(), bundle.RelationKeyName.String(),
						bundle.RelationKeyResolvedLayout.String(), bundle.RelationKeyRelationKey.String(),
						bundle.RelationKeyUniqueKey.String(), bundle.RelationKeyApiObjectKey.String(),
					},
				})
				if err != nil {
					return err
				}
				if resp.Error != nil && resp.Error.Code != pb.RpcObjectSearchResponseError_NULL {
					return fmt.Errorf("search: %s", resp.Error.Description)
				}
				if len(resp.Records) == 0 {
					fmt.Println("no matches")
					return nil
				}
				w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
				fmt.Fprintln(w, "ID\tLAYOUT\tKEY\tNAME")
				for _, rec := range resp.Records {
					id := pbtypes.GetString(rec, bundle.RelationKeyId.String())
					name := pbtypes.GetString(rec, bundle.RelationKeyName.String())
					lay := model.ObjectTypeLayout(pbtypes.GetInt64(rec, bundle.RelationKeyResolvedLayout.String()))
					key := pbtypes.GetString(rec, bundle.RelationKeyRelationKey.String())
					if key == "" {
						key = pbtypes.GetString(rec, bundle.RelationKeyApiObjectKey.String())
					}
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", id, layoutName(lay), key, name)
				}
				return w.Flush()
			})
		},
	}
	cmd.Flags().StringVar(&spaceId, "space", "", "space id (or ANYTYPE_SPACE)")
	cmd.Flags().StringVar(&kind, "kind", "", "filter by layout: type|relation|set|collection|page|note|bookmark")
	cmd.Flags().IntVar(&limit, "limit", 50, "max matches")
	return cmd
}

func layoutName(l model.ObjectTypeLayout) string {
	if n, ok := model.ObjectTypeLayout_name[int32(l)]; ok {
		return n
	}
	return fmt.Sprintf("layout%d", int32(l))
}
