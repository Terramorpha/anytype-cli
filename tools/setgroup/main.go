// Command setgroup sets the "group by" relation on a set/collection's Kanban
// view: registers the relation on the dataview and updates the view's
// GroupRelationKey. Throwaway helper.
//
//	setgroup <objectId> <groupRelationKey>
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
		fmt.Fprintln(os.Stderr, "usage: setgroup <objectId> <groupRelationKey>")
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
		if dv == nil {
			return fmt.Errorf("no dataview block in %s", objectId)
		}
		// register the relation on the dataview (so grouping can use it)
		if _, err := client.BlockDataviewRelationAdd(ctx, &pb.RpcBlockDataviewRelationAddRequest{
			ContextId: objectId, BlockId: blockId, RelationKeys: []string{groupKey},
		}); err != nil {
			return err
		}
		// update every Kanban view's group-by
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
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
}
