// Command addblock appends a text block to an object via gRPC BlockCreate —
// a block-level edit below the markdown pipeline. Throwaway helper.
//
//	addblock <objectId> <text> [header|paragraph|callout]
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
		fmt.Fprintln(os.Stderr, "usage: addblock <objectId> <text> [header|paragraph|callout]")
		os.Exit(1)
	}
	objectId, text := os.Args[1], os.Args[2]
	style := model.BlockContentText_Paragraph
	if len(os.Args) > 3 {
		switch os.Args[3] {
		case "header":
			style = model.BlockContentText_Header2
		case "callout":
			style = model.BlockContentText_Callout
		}
	}
	err := core.GRPCCall(func(ctx context.Context, client service.ClientCommandsClient) error {
		resp, err := client.BlockCreate(ctx, &pb.RpcBlockCreateRequest{
			ContextId: objectId,
			Position:  model.Block_Bottom,
			Block: &model.Block{
				Content: &model.BlockContentOfText{
					Text: &model.BlockContentText{Text: text, Style: style},
				},
			},
		})
		if err != nil {
			return err
		}
		if resp.Error != nil && resp.Error.Code != pb.RpcBlockCreateResponseError_NULL {
			return fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Description)
		}
		fmt.Println("OK: block", resp.BlockId)
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
}
