// Command defaultkanban converts a set/collection's FIRST (default) view into a
// Kanban grouped by the given relation, so the object leads with the board.
// Throwaway helper.
//
//	defaultkanban <objectId> <groupRelationKey>
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/anyproto/anytype-heart/pb"
	"github.com/anyproto/anytype-heart/pb/service"
	"github.com/anyproto/anytype-heart/pkg/lib/pb/model"

	"github.com/anyproto/anytype-cli/core"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: defaultkanban <objectId> <groupRelationKey>")
		os.Exit(1)
	}
	objectId, groupKey := os.Args[1], os.Args[2]
	err := core.GRPCCall(func(ctx context.Context, client service.ClientCommandsClient) error {
		show, err := client.ObjectShow(ctx, &pb.RpcObjectShowRequest{ObjectId: objectId})
		if err != nil {
			return err
		}
		var blockId string
		var dv *model.BlockContentDataview
		for _, b := range show.ObjectView.GetBlocks() {
			if d := b.GetDataview(); d != nil {
				blockId, dv = b.Id, d
				break
			}
		}
		if dv == nil || len(dv.GetViews()) == 0 {
			return fmt.Errorf("no dataview/views in %s", objectId)
		}
		// register the relation so grouping can use it
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
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
}
