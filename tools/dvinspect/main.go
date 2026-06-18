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
	objectId := os.Args[1]
	_ = core.GRPCCall(func(ctx context.Context, c service.ClientCommandsClient) error {
		show, err := c.ObjectShow(ctx, &pb.RpcObjectShowRequest{ObjectId: objectId})
		if err != nil { return err }
		for _, b := range show.ObjectView.GetBlocks() {
			d := b.GetDataview()
			if d == nil { continue }
			fmt.Println("RelationLinks on dataview:")
			for _, rl := range d.GetRelationLinks() {
				fmt.Printf("  key=%q format=%v\n", rl.Key, rl.Format)
			}
			for _, v := range d.GetViews() {
				fmt.Printf("view %s name=%q type=%v GROUP=%q\n", v.Id, v.Name, v.Type, v.GroupRelationKey)
			}
		}
		fmt.Println("--- relations in ObjectView.RelationLinks ---")
		for _, rl := range show.ObjectView.GetRelationLinks() {
			if rl.Key == "status" || rl.Key == "tag" {
				fmt.Printf("  key=%q format=%v\n", rl.Key, model.RelationFormat(rl.Format))
			}
		}
		return nil
	})
}
