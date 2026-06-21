package tools

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gogo/protobuf/types"
	"github.com/spf13/cobra"

	"github.com/anyproto/anytype-heart/pb"
	"github.com/anyproto/anytype-heart/pb/service"
	"github.com/anyproto/anytype-heart/pkg/lib/pb/model"
	"github.com/anyproto/anytype-heart/util/pbtypes"

	"github.com/anyproto/anytype-cli/core"
)

// typedFilterValue types a filter value like query does: number, bool, or string.
func typedFilterValue(val string) *types.Value {
	if val == "true" || val == "false" {
		return pbtypes.Bool(val == "true")
	}
	if n, err := strconv.ParseFloat(val, 64); err == nil {
		return pbtypes.Float64(n)
	}
	return pbtypes.String(val)
}

// pickView returns the view to edit: the one matching viewId, or the first.
func pickView(dv *model.BlockContentDataview, viewId string) *model.BlockContentDataviewView {
	for _, v := range dv.GetViews() {
		if viewId != "" && v.Id == viewId {
			return v
		}
	}
	if viewId == "" && len(dv.GetViews()) > 0 {
		return dv.GetViews()[0]
	}
	return nil
}

// viewfilter sets the filters of a dataview view (replacing existing filters).
func newViewfilterCmd() *cobra.Command {
	var viewId string
	var filters []string
	cmd := &cobra.Command{
		Use:   "viewfilter <objectId> --filter key<op>value [--filter …]",
		Short: "Set a dataview view's filters (e.g. show only kind=paper)",
		Long: "Replace the filters of a view (default first view). Repeatable --filter\n" +
			"\"key<op>value\" (ops = != ~ > < >= <=). Keys are internal relation keys\n" +
			"(`tools find --kind relation`). Pass no --filter to clear.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			objectId := args[0]
			var fs []*model.BlockContentDataviewFilter
			for _, f := range filters {
				key, cond, val, err := parseFilter(f)
				if err != nil {
					return err
				}
				fs = append(fs, &model.BlockContentDataviewFilter{
					RelationKey: key, Condition: cond, Value: typedFilterValue(val),
				})
			}
			return core.GRPCCall(func(ctx context.Context, c service.ClientCommandsClient) error {
				show, err := c.ObjectShow(ctx, &pb.RpcObjectShowRequest{ObjectId: objectId})
				if err != nil {
					return err
				}
				blockId, dv := firstDataview(show)
				if dv == nil {
					return fmt.Errorf("no dataview in %s", objectId)
				}
				v := pickView(dv, viewId)
				if v == nil {
					return fmt.Errorf("view not found in %s", objectId)
				}
				// Filters/sorts are NOT applied via ViewUpdate — they have
				// dedicated RPCs. Clear the existing filters, then add the new ones.
				var oldIds []string
				for _, f := range v.GetFilters() {
					oldIds = append(oldIds, f.Id)
				}
				if len(oldIds) > 0 {
					if _, err := c.BlockDataviewFilterRemove(ctx, &pb.RpcBlockDataviewFilterRemoveRequest{
						ContextId: objectId, BlockId: blockId, ViewId: v.Id, Ids: oldIds,
					}); err != nil {
						return err
					}
				}
				for i, f := range fs {
					f.Id = fmt.Sprintf("flt%d", i)
					if _, err := c.BlockDataviewFilterAdd(ctx, &pb.RpcBlockDataviewFilterAddRequest{
						ContextId: objectId, BlockId: blockId, ViewId: v.Id, Filter: f,
					}); err != nil {
						return err
					}
				}
				return emit(ok(map[string]any{"viewId": v.Id, "filters": len(fs)}))
			})
		},
	}
	cmd.Flags().StringVar(&viewId, "view", "", "view id (default: first view)")
	cmd.Flags().StringArrayVar(&filters, "filter", nil, "filter \"key<op>value\" (repeatable)")
	return cmd
}

// viewsort sets the sort of a dataview view.
func newViewsortCmd() *cobra.Command {
	var viewId, by string
	var desc bool
	cmd := &cobra.Command{
		Use:   "viewsort <objectId> --by <relationKey> [--desc]",
		Short: "Set a dataview view's sort order",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			objectId := args[0]
			if by == "" {
				return fmt.Errorf("--by <relationKey> is required")
			}
			sortType := model.BlockContentDataviewSort_Asc
			if desc {
				sortType = model.BlockContentDataviewSort_Desc
			}
			return core.GRPCCall(func(ctx context.Context, c service.ClientCommandsClient) error {
				show, err := c.ObjectShow(ctx, &pb.RpcObjectShowRequest{ObjectId: objectId})
				if err != nil {
					return err
				}
				blockId, dv := firstDataview(show)
				if dv == nil {
					return fmt.Errorf("no dataview in %s", objectId)
				}
				v := pickView(dv, viewId)
				if v == nil {
					return fmt.Errorf("view not found in %s", objectId)
				}
				// Replace existing sorts with the requested one (dedicated RPCs).
				var oldIds []string
				for _, s := range v.GetSorts() {
					oldIds = append(oldIds, s.Id)
				}
				if len(oldIds) > 0 {
					if _, err := c.BlockDataviewSortRemove(ctx, &pb.RpcBlockDataviewSortRemoveRequest{
						ContextId: objectId, BlockId: blockId, ViewId: v.Id, Ids: oldIds,
					}); err != nil {
						return err
					}
				}
				if _, err := c.BlockDataviewSortAdd(ctx, &pb.RpcBlockDataviewSortAddRequest{
					ContextId: objectId, BlockId: blockId, ViewId: v.Id,
					Sort: &model.BlockContentDataviewSort{Id: "srt0", RelationKey: by, Type: sortType},
				}); err != nil {
					return err
				}
				return emit(ok(map[string]any{"viewId": v.Id, "sortBy": by, "desc": desc}))
			})
		},
	}
	cmd.Flags().StringVar(&viewId, "view", "", "view id (default: first view)")
	cmd.Flags().StringVar(&by, "by", "", "relation key to sort by (required)")
	cmd.Flags().BoolVar(&desc, "desc", false, "sort descending")
	return cmd
}
