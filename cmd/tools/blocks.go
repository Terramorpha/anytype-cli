package tools

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/anyproto/anytype-heart/pb"
	"github.com/anyproto/anytype-heart/pb/service"
	"github.com/anyproto/anytype-heart/pkg/lib/pb/model"
	"github.com/gogo/protobuf/types"

	"github.com/anyproto/anytype-cli/core"
)

// addblock appends a text block to an object via BlockCreate — a block-level
// edit below the markdown pipeline.
func newAddblockCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "addblock <objectId> <text> [header|paragraph|callout]",
		Short: "Append a text block to an object (BlockCreate)",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			objectId, text := args[0], args[1]
			style := model.BlockContentText_Paragraph
			if len(args) > 2 {
				switch args[2] {
				case "header":
					style = model.BlockContentText_Header2
				case "callout":
					style = model.BlockContentText_Callout
				}
			}
			return core.GRPCCall(func(ctx context.Context, client service.ClientCommandsClient) error {
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
				return emit(ok(map[string]any{"blockId": resp.BlockId}))
			})
		},
	}
}

// delblock deletes one or more blocks from an object.
func newDelblockCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delblock <objectId> <blockId>...",
		Short: "Delete blocks from an object (BlockListDelete)",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return core.GRPCCall(func(ctx context.Context, c service.ClientCommandsClient) error {
				if _, err := c.BlockListDelete(ctx, &pb.RpcBlockListDeleteRequest{
					ContextId: args[0], BlockIds: args[1:],
				}); err != nil {
					return err
				}
				return emit(ok(map[string]any{"deleted": args[1:]}))
			})
		},
	}
}

// appendmd appends markdown (parsed into blocks) to a page via BlockPaste's
// text slot.
func newAppendmdCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "appendmd <pageId> <mdFile>",
		Short: "Append markdown (parsed into blocks) to a page (BlockPaste)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			md, err := os.ReadFile(args[1])
			if err != nil {
				return err
			}
			return core.GRPCCall(func(ctx context.Context, c service.ClientCommandsClient) error {
				if _, err := c.BlockPaste(ctx, &pb.RpcBlockPasteRequest{
					ContextId: args[0], TextSlot: string(md),
				}); err != nil {
					return err
				}
				return emit(ok(map[string]any{"pageId": args[0]}))
			})
		},
	}
}

// blockdump prints a terse summary of an object's blocks and notable details.
func newBlockdumpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "blockdump <objectId>",
		Short: "Print a terse summary of an object's blocks (ObjectShow)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return core.GRPCCall(func(ctx context.Context, c service.ClientCommandsClient) error {
				s, err := c.ObjectShow(ctx, &pb.RpcObjectShowRequest{ObjectId: args[0]})
				if err != nil {
					return err
				}
				type blk struct {
					Id   string `json:"id"`
					Kind string `json:"kind"`
				}
				var detailKeys []string
				for _, det := range s.ObjectView.GetDetails() {
					for k := range det.GetDetails().GetFields() {
						switch k {
						case "coverId", "coverType", "iconImage", "iconEmoji":
							detailKeys = append(detailKeys, k)
						}
					}
				}
				blocks := []blk{}
				for _, b := range s.ObjectView.GetBlocks() {
					kind := ""
					switch {
					case b.GetText() != nil:
						kind = "text/" + b.GetText().Style.String()
					case b.GetLink() != nil:
						kind = "link"
					case b.GetDataview() != nil:
						kind = "dataview"
					case b.GetLayout() != nil:
						kind = "layout/" + b.GetLayout().Style.String()
					case b.GetFile() != nil:
						kind = "file"
					case b.GetDiv() != nil:
						kind = "divider"
					case b.GetBookmark() != nil:
						kind = "bookmark"
					case b.GetRelation() != nil:
						kind = "relation"
					case b.GetTableOfContents() != nil:
						kind = "toc"
					default:
						continue
					}
					blocks = append(blocks, blk{Id: b.Id, Kind: kind})
				}
				return emit(map[string]any{"id": args[0], "details": detailKeys, "blocks": blocks})
			})
		},
	}
}

// pagedeck sets a gradient cover and appends clickable link-card blocks.
func newPagedeckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pagedeck <pageId> <coverId> <objId>...",
		Short: "Set a gradient cover and append object link-cards to a page",
		Args:  cobra.MinimumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			page, cover := args[0], args[1]
			objs := args[2:]
			return core.GRPCCall(func(ctx context.Context, c service.ClientCommandsClient) error {
				if _, err := c.ObjectSetDetails(ctx, &pb.RpcObjectSetDetailsRequest{ContextId: page, Details: []*model.Detail{
					{Key: "coverType", Value: &types.Value{Kind: &types.Value_NumberValue{NumberValue: 3}}},
					{Key: "coverId", Value: &types.Value{Kind: &types.Value_StringValue{StringValue: cover}}},
				}}); err != nil {
					return err
				}
				prev := ""
				cards := []string{}
				for _, o := range objs {
					r, err := c.BlockCreate(ctx, &pb.RpcBlockCreateRequest{
						ContextId: page, TargetId: prev, Position: model.Block_Bottom,
						Block: &model.Block{Content: &model.BlockContentOfLink{Link: &model.BlockContentLink{
							TargetBlockId: o, Style: model.BlockContentLink_Page, CardStyle: model.BlockContentLink_Card,
						}}},
					})
					if err != nil {
						return err
					}
					prev = r.BlockId
					cards = append(cards, r.BlockId)
				}
				return emit(ok(map[string]any{"pageId": page, "coverSet": true, "cards": cards}))
			})
		},
	}
}
