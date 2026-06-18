// Command kanban adds a Kanban (Board) view to a set/collection, grouped by a
// relation. Finds the dataview block via ObjectShow, then BlockDataviewViewCreate.
//
//	kanban <setObjectId> <groupRelationKey>
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
		fmt.Fprintln(os.Stderr, "usage: kanban <setObjectId> <groupRelationKey>")
		os.Exit(1)
	}
	setId, groupKey := os.Args[1], os.Args[2]
	err := core.GRPCCall(func(ctx context.Context, client service.ClientCommandsClient) error {
		show, err := client.ObjectShow(ctx, &pb.RpcObjectShowRequest{ObjectId: setId})
		if err != nil {
			return err
		}
		if show.Error != nil && show.Error.Code != pb.RpcObjectShowResponseError_NULL {
			return fmt.Errorf("ObjectShow: %s", show.Error.Description)
		}
		blockId := ""
		for _, b := range show.ObjectView.GetBlocks() {
			if b.GetDataview() != nil {
				blockId = b.Id
				break
			}
		}
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
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
}
