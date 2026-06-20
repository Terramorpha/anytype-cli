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

// textStyles maps friendly --style names to the gRPC text style enum.
var textStyles = map[string]model.BlockContentTextStyle{
	"paragraph": model.BlockContentText_Paragraph,
	"h1":        model.BlockContentText_Header1,
	"header1":   model.BlockContentText_Header1,
	"h2":        model.BlockContentText_Header2,
	"header2":   model.BlockContentText_Header2,
	"header":    model.BlockContentText_Header2, // backward-compat alias
	"h3":        model.BlockContentText_Header3,
	"header3":   model.BlockContentText_Header3,
	"h4":        model.BlockContentText_Header4,
	"header4":   model.BlockContentText_Header4,
	"callout":   model.BlockContentText_Callout,
	"quote":     model.BlockContentText_Quote,
	"code":      model.BlockContentText_Code,
	"checkbox":  model.BlockContentText_Checkbox,
	"todo":      model.BlockContentText_Checkbox,
	"toggle":    model.BlockContentText_Toggle,
	"bulleted":  model.BlockContentText_Marked,
	"bullet":    model.BlockContentText_Marked,
	"numbered":  model.BlockContentText_Numbered,
}

// addblock appends a text block of any style to an object via BlockCreate.
func newAddblockCmd() *cobra.Command {
	var style, parent, after, left, right, lang string
	cmd := &cobra.Command{
		Use:   "addblock <objectId> <text> [style]",
		Short: "Append a text block of any style to an object (BlockCreate)",
		Long: "Append a text block. Style via positional arg or --style:\n" +
			"  paragraph, h1..h4, callout, quote, code, checkbox, toggle, bulleted, numbered\n\n" +
			"Placement (default: bottom of the object):\n" +
			"  --parent <blockId>   nest INSIDE a parent block (e.g. fill a toggle/callout)\n" +
			"  --after  <blockId>   insert directly after a sibling block",
		Args: cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			objectId, text := args[0], args[1]
			styleName := style
			if styleName == "" && len(args) > 2 {
				styleName = args[2]
			}
			if styleName == "" {
				styleName = "paragraph"
			}
			st, ok2 := textStyles[styleName]
			if !ok2 {
				return fmt.Errorf("unknown style %q", styleName)
			}
			placements := 0
			for _, s := range []string{parent, after, left, right} {
				if s != "" {
					placements++
				}
			}
			if placements > 1 {
				return fmt.Errorf("--parent/--after/--left/--right are mutually exclusive")
			}
			// Default: append to the bottom of the object. --parent nests inside
			// that block; --after is the next sibling; --left/--right place the
			// block beside the target, which wraps both into a column row.
			targetId, position := "", model.Block_Bottom
			switch {
			case parent != "":
				targetId, position = parent, model.Block_Inner
			case after != "":
				targetId, position = after, model.Block_Bottom
			case left != "":
				targetId, position = left, model.Block_Left
			case right != "":
				targetId, position = right, model.Block_Right
			}
			// A code block's language lives in the block's Fields["lang"].
			block := &model.Block{
				Content: &model.BlockContentOfText{
					Text: &model.BlockContentText{Text: text, Style: st},
				},
			}
			if lang != "" {
				block.Fields = &types.Struct{Fields: map[string]*types.Value{
					"lang": {Kind: &types.Value_StringValue{StringValue: lang}},
				}}
			}
			return core.GRPCCall(func(ctx context.Context, client service.ClientCommandsClient) error {
				resp, err := client.BlockCreate(ctx, &pb.RpcBlockCreateRequest{
					ContextId: objectId,
					TargetId:  targetId,
					Position:  position,
					Block:     block,
				})
				if err != nil {
					return err
				}
				if resp.Error != nil && resp.Error.Code != pb.RpcBlockCreateResponseError_NULL {
					return fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Description)
				}
				return emit(ok(map[string]any{"blockId": resp.BlockId, "style": styleName, "lang": lang}))
			})
		},
	}
	cmd.Flags().StringVar(&style, "style", "", "block style (overrides positional): paragraph|h1..h4|callout|quote|code|checkbox|toggle|bulleted|numbered")
	cmd.Flags().StringVar(&parent, "parent", "", "nest the block inside this parent block id (e.g. a toggle)")
	cmd.Flags().StringVar(&after, "after", "", "insert the block directly after this sibling block id")
	cmd.Flags().StringVar(&left, "left", "", "place to the LEFT of this block id (forms a column row)")
	cmd.Flags().StringVar(&right, "right", "", "place to the RIGHT of this block id (forms a column row)")
	cmd.Flags().StringVar(&lang, "lang", "", "code block language (e.g. go, python, js) — only meaningful with --style code")
	return cmd
}

// divider appends a horizontal divider block (line or dots).
func newDividerCmd() *cobra.Command {
	var dots bool
	cmd := &cobra.Command{
		Use:   "divider <objectId>",
		Short: "Append a divider block (--dots for a dotted divider)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			style := model.BlockContentDiv_Line
			if dots {
				style = model.BlockContentDiv_Dots
			}
			return core.GRPCCall(func(ctx context.Context, c service.ClientCommandsClient) error {
				resp, err := c.BlockCreate(ctx, &pb.RpcBlockCreateRequest{
					ContextId: args[0],
					Position:  model.Block_Bottom,
					Block:     &model.Block{Content: &model.BlockContentOfDiv{Div: &model.BlockContentDiv{Style: style}}},
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
	cmd.Flags().BoolVar(&dots, "dots", false, "use a dotted divider instead of a line")
	return cmd
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
				type mk struct {
					Type  string `json:"type"`
					From  int32  `json:"from"`
					To    int32  `json:"to"`
					Param string `json:"param,omitempty"`
				}
				type blk struct {
					Id       string   `json:"id"`
					Kind     string   `json:"kind"`
					Text     string   `json:"text,omitempty"`
					Lang     string   `json:"lang,omitempty"`
					Marks    []mk     `json:"marks,omitempty"`
					Children []string `json:"children,omitempty"`
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
					var kind, text string
					switch {
					case b.GetText() != nil:
						kind = "text/" + b.GetText().Style.String()
						text = b.GetText().Text
					case b.GetLink() != nil:
						kind = "link/" + b.GetLink().CardStyle.String() + "->" + b.GetLink().TargetBlockId
					case b.GetDataview() != nil:
						kind = "dataview"
					case b.GetLayout() != nil:
						kind = "layout/" + b.GetLayout().Style.String()
					case b.GetFile() != nil:
						kind = "file"
					case b.GetDiv() != nil:
						kind = "divider/" + b.GetDiv().Style.String()
					case b.GetBookmark() != nil:
						kind = "bookmark"
					case b.GetRelation() != nil:
						kind = "relation"
					case b.GetTableOfContents() != nil:
						kind = "toc"
					case b.GetSmartblock() != nil:
						kind = "smartblock(root)"
					case b.GetFeaturedRelations() != nil:
						kind = "featuredRelations"
					default:
						kind = "other"
					}
					if len(text) > 80 {
						text = text[:77] + "..."
					}
					lang := ""
					if f := b.GetFields(); f != nil {
						if v, okv := f.GetFields()["lang"]; okv {
							lang = v.GetStringValue()
						}
					}
					var marks []mk
					if t := b.GetText(); t != nil && t.GetMarks() != nil {
						for _, m := range t.GetMarks().GetMarks() {
							marks = append(marks, mk{
								Type:  m.Type.String(),
								From:  m.GetRange().GetFrom(),
								To:    m.GetRange().GetTo(),
								Param: m.Param,
							})
						}
					}
					blocks = append(blocks, blk{Id: b.Id, Kind: kind, Text: text, Lang: lang, Marks: marks, Children: b.ChildrenIds})
				}
				return emit(map[string]any{
					"id":      args[0],
					"rootId":  s.ObjectView.RootId,
					"details": detailKeys,
					"blocks":  blocks,
				})
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
