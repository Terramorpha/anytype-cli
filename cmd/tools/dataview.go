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

// firstDataview returns the first dataview block id and content in an object.
func firstDataview(show *pb.RpcObjectShowResponse) (string, *model.BlockContentDataview) {
	for _, b := range show.ObjectView.GetBlocks() {
		if d := b.GetDataview(); d != nil {
			return b.Id, d
		}
	}
	return "", nil
}

// dvinspect prints the relation links and views of an object's dataview.
func newDvinspectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dvinspect <objectId>",
		Short: "Inspect an object's dataview: relation links and views",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return core.GRPCCall(func(ctx context.Context, c service.ClientCommandsClient) error {
				show, err := c.ObjectShow(ctx, &pb.RpcObjectShowRequest{ObjectId: args[0]})
				if err != nil {
					return err
				}
				for _, b := range show.ObjectView.GetBlocks() {
					d := b.GetDataview()
					if d == nil {
						continue
					}
					fmt.Println("RelationLinks on dataview:")
					for _, rl := range d.GetRelationLinks() {
						fmt.Printf("  key=%q format=%v\n", rl.Key, rl.Format)
					}
					for _, v := range d.GetViews() {
						fmt.Printf("view %s name=%q type=%v GROUP=%q\n", v.Id, v.Name, v.Type, v.GroupRelationKey)
					}
				}
				return nil
			})
		},
	}
}

// viewprops sets the visible properties (and order) of the first view.
func newViewpropsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "viewprops <objectId> <relKey>...",
		Short: "Set the visible properties (and order) of the default view",
		Long: "Sets the visible properties of the FIRST (default) view of an object's\n" +
			"dataview. \"name\" is always shown first; the given relation keys follow,\n" +
			"all visible, in order. Any relation already known but not listed is hidden.\n\n" +
			"Relation keys are the dataview's internal keys (see `tools dvinspect`).",
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			objectId := args[0]
			want := args[1:]
			return core.GRPCCall(func(ctx context.Context, client service.ClientCommandsClient) error {
				show, err := client.ObjectShow(ctx, &pb.RpcObjectShowRequest{ObjectId: objectId})
				if err != nil {
					return err
				}
				blockId, dv := firstDataview(show)
				if dv == nil || len(dv.GetViews()) == 0 {
					return fmt.Errorf("no dataview/views in %s", objectId)
				}
				// Register the requested relations on the dataview first, so the
				// columns resolve even on a fresh set that only carries the system
				// relations in its relation links.
				if _, err := client.BlockDataviewRelationAdd(ctx, &pb.RpcBlockDataviewRelationAddRequest{
					ContextId: objectId, BlockId: blockId, RelationKeys: want,
				}); err != nil {
					return err
				}
				visible := append([]string{"name"}, want...)
				shown := map[string]bool{}
				var rels []*model.BlockContentDataviewRelation
				for _, k := range visible {
					rels = append(rels, &model.BlockContentDataviewRelation{Key: k, IsVisible: true})
					shown[k] = true
				}
				// keep remaining known relations present but hidden, so nothing is lost
				for _, rl := range dv.GetRelationLinks() {
					if !shown[rl.Key] {
						rels = append(rels, &model.BlockContentDataviewRelation{Key: rl.Key, IsVisible: false})
					}
				}
				v := dv.GetViews()[0]
				v.Relations = rels
				if _, err := client.BlockDataviewViewUpdate(ctx, &pb.RpcBlockDataviewViewUpdateRequest{
					ContextId: objectId, BlockId: blockId, ViewId: v.Id, View: v,
				}); err != nil {
					return err
				}
				fmt.Printf("OK: view %s (%s) now shows: %v\n", v.Id, v.Name, visible)
				return nil
			})
		},
	}
}

// kanban adds a new Kanban (Board) view to a set/collection.
func newKanbanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "kanban <setObjectId> <groupRelationKey>",
		Short: "Add a Kanban (Board) view to a set/collection, grouped by a relation",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			setId, groupKey := args[0], args[1]
			return core.GRPCCall(func(ctx context.Context, client service.ClientCommandsClient) error {
				show, err := client.ObjectShow(ctx, &pb.RpcObjectShowRequest{ObjectId: setId})
				if err != nil {
					return err
				}
				if show.Error != nil && show.Error.Code != pb.RpcObjectShowResponseError_NULL {
					return fmt.Errorf("ObjectShow: %s", show.Error.Description)
				}
				blockId, _ := firstDataview(show)
				if blockId == "" {
					return fmt.Errorf("no dataview block found in %s", setId)
				}
				resp, err := client.BlockDataviewViewCreate(ctx, &pb.RpcBlockDataviewViewCreateRequest{
					ContextId: setId,
					BlockId:   blockId,
					View: &model.BlockContentDataviewView{
						Type:             model.BlockContentDataviewView_Kanban,
						Name:             "Board",
						GroupRelationKey: groupKey,
						Relations: []*model.BlockContentDataviewRelation{
							{Key: groupKey, IsVisible: true},
						},
					},
				})
				if err != nil {
					return err
				}
				if resp.Error != nil && resp.Error.Code != pb.RpcBlockDataviewViewCreateResponseError_NULL {
					return fmt.Errorf("ViewCreate: %s", resp.Error.Description)
				}
				fmt.Printf("OK: kanban view %s on block %s\n", resp.ViewId, blockId)
				return nil
			})
		},
	}
}

// defaultkanban converts the first view into a Kanban grouped by a relation.
func newDefaultkanbanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "defaultkanban <objectId> <groupRelationKey>",
		Short: "Convert the default view into a Kanban grouped by a relation",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			objectId, groupKey := args[0], args[1]
			return core.GRPCCall(func(ctx context.Context, client service.ClientCommandsClient) error {
				show, err := client.ObjectShow(ctx, &pb.RpcObjectShowRequest{ObjectId: objectId})
				if err != nil {
					return err
				}
				blockId, dv := firstDataview(show)
				if dv == nil || len(dv.GetViews()) == 0 {
					return fmt.Errorf("no dataview/views in %s", objectId)
				}
				if _, err := client.BlockDataviewRelationAdd(ctx, &pb.RpcBlockDataviewRelationAddRequest{
					ContextId: objectId, BlockId: blockId, RelationKeys: []string{groupKey},
				}); err != nil {
					return err
				}
				for _, v := range dv.GetViews() {
					fmt.Printf("view %s (%s) was type=%v\n", v.Id, v.Name, v.Type)
					v.Type = model.BlockContentDataviewView_Kanban
					v.GroupRelationKey = groupKey
					if _, err := client.BlockDataviewViewUpdate(ctx, &pb.RpcBlockDataviewViewUpdateRequest{
						ContextId: objectId, BlockId: blockId, ViewId: v.Id, View: v,
					}); err != nil {
						return err
					}
					fmt.Printf("OK: view %s is now Kanban grouped by %s\n", v.Id, groupKey)
				}
				return nil
			})
		},
	}
}

// setgroup sets the group-by relation on every Kanban view.
func newSetgroupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setgroup <objectId> <groupRelationKey>",
		Short: "Set the group-by relation on a set/collection's Kanban view(s)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			objectId, groupKey := args[0], args[1]
			return core.GRPCCall(func(ctx context.Context, client service.ClientCommandsClient) error {
				show, err := client.ObjectShow(ctx, &pb.RpcObjectShowRequest{ObjectId: objectId})
				if err != nil {
					return err
				}
				blockId, dv := firstDataview(show)
				if dv == nil {
					return fmt.Errorf("no dataview block in %s", objectId)
				}
				if _, err := client.BlockDataviewRelationAdd(ctx, &pb.RpcBlockDataviewRelationAddRequest{
					ContextId: objectId, BlockId: blockId, RelationKeys: []string{groupKey},
				}); err != nil {
					return err
				}
				for _, v := range dv.GetViews() {
					fmt.Printf("view %s type=%v group=%q\n", v.Id, v.Type, v.GroupRelationKey)
					if v.Type != model.BlockContentDataviewView_Kanban {
						continue
					}
					v.GroupRelationKey = groupKey
					if _, err := client.BlockDataviewViewUpdate(ctx, &pb.RpcBlockDataviewViewUpdateRequest{
						ContextId: objectId, BlockId: blockId, ViewId: v.Id, View: v,
					}); err != nil {
						return err
					}
					fmt.Printf("OK: set group-by=%s on view %s\n", groupKey, v.Id)
				}
				return nil
			})
		},
	}
}

// hidegroup sets the Kanban group order and hides the empty (no-value) group.
func newHidegroupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "hidegroup <contextId> <blockId> <viewId> <visibleGroupId>...",
		Short: "Set Kanban group order; list given groups visible and hide the empty group",
		Args:  cobra.MinimumNArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctxId, block, view := args[0], args[1], args[2]
			vis := args[3:]
			groups := []*model.BlockContentDataviewViewGroup{}
			for i, g := range vis {
				groups = append(groups, &model.BlockContentDataviewViewGroup{GroupId: g, Index: int32(i), Hidden: false})
			}
			groups = append(groups, &model.BlockContentDataviewViewGroup{GroupId: "empty", Index: int32(len(vis)), Hidden: true})
			return core.GRPCCall(func(c0 context.Context, c service.ClientCommandsClient) error {
				if _, err := c.BlockDataviewGroupOrderUpdate(c0, &pb.RpcBlockDataviewGroupOrderUpdateRequest{
					ContextId: ctxId, BlockId: block,
					GroupOrder: &model.BlockContentDataviewGroupOrder{ViewId: view, ViewGroups: groups},
				}); err != nil {
					return err
				}
				fmt.Printf("group order set: %d visible + empty hidden\n", len(vis))
				return nil
			})
		},
	}
}

// setq points a Set object at a type by setting its setOf relation.
func newSetqCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setq <setObjectId> <typeId>",
		Short: "Point a Set at a type (setOf) to turn it into a live query",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			setId, typeId := args[0], args[1]
			return core.GRPCCall(func(ctx context.Context, client service.ClientCommandsClient) error {
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
				fmt.Println("OK: setOf set")
				return nil
			})
		},
	}
}

// embedset embeds an existing Set/Collection inline as a live dataview block.
func newEmbedsetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "embedset <pageId> <setId>",
		Short: "Embed an existing Set/Collection inline as a live dataview block",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			page, set := args[0], args[1]
			return core.GRPCCall(func(ctx context.Context, c service.ClientCommandsClient) error {
				r, err := c.BlockCreate(ctx, &pb.RpcBlockCreateRequest{ContextId: page, Position: model.Block_Bottom,
					Block: &model.Block{Content: &model.BlockContentOfDataview{Dataview: &model.BlockContentDataview{}}}})
				if err != nil {
					return fmt.Errorf("create block: %w", err)
				}
				bid := r.BlockId
				resp, err := c.BlockDataviewCreateFromExistingObject(ctx, &pb.RpcBlockDataviewCreateFromExistingObjectRequest{
					ContextId: page, BlockId: bid, TargetObjectId: set})
				if err != nil {
					return fmt.Errorf("populate: %w", err)
				}
				if resp.Error != nil && resp.Error.Code != pb.RpcBlockDataviewCreateFromExistingObjectResponseError_NULL {
					return fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Description)
				}
				fmt.Printf("inline set embedded: block %s -> %s (views copied)\n", bid, set)
				return nil
			})
		},
	}
}
