package tools

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/anyproto/anytype-heart/pb"
	"github.com/anyproto/anytype-heart/pb/service"
	"github.com/anyproto/anytype-heart/pkg/lib/pb/model"

	"github.com/anyproto/anytype-cli/core"
)

var markTypes = map[string]model.BlockContentTextMarkType{
	"bold":      model.BlockContentTextMark_Bold,
	"italic":    model.BlockContentTextMark_Italic,
	"code":      model.BlockContentTextMark_Keyboard,
	"strike":    model.BlockContentTextMark_Strikethrough,
	"underline": model.BlockContentTextMark_Underscored,
	"link":      model.BlockContentTextMark_Link,            // param = URL
	"color":     model.BlockContentTextMark_TextColor,       // param = color name
	"bgcolor":   model.BlockContentTextMark_BackgroundColor, // param = color name
	"mention":   model.BlockContentTextMark_Mention,         // param = object id
	"object":    model.BlockContentTextMark_Object,          // param = object id
}

// mark sets the inline marks of a text block in ONE call. Each --mark is
// "type:from:to[:param]". All marks are applied atomically (BlockTextSetText),
// which both respects ranges and avoids the read-modify-write race you'd hit
// applying marks one invocation at a time. By default this replaces the block's
// marks; --keep merges with the marks already present.
func newMarkCmd() *cobra.Command {
	var specs []string
	var keep bool
	cmd := &cobra.Command{
		Use:   "mark <objectId> <blockId> --mark type:from:to[:param] [--mark …]",
		Short: "Set inline text marks on a block (ranged, applied atomically)",
		Long: "Apply inline marks to a text block. Repeatable --mark \"type:from:to[:param]\":\n" +
			"  types: bold|italic|code|strike|underline|link|color|bgcolor|mention|object\n" +
			"  param: URL (link), color name (color/bgcolor), object id (mention/object)\n\n" +
			"Replaces the block's marks unless --keep (then merges with existing).\n" +
			"Example:\n  anytype tools mark <pg> <blk> --mark bold:0:5 --mark color:11:14:red \\\n" +
			"      --mark mention:21:28:bafy…",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			objectId, blockId := args[0], args[1]
			if len(specs) == 0 {
				return fmt.Errorf("provide at least one --mark type:from:to[:param]")
			}
			var newMarks []*model.BlockContentTextMark
			for _, s := range specs {
				m, err := parseMarkSpec(s)
				if err != nil {
					return err
				}
				newMarks = append(newMarks, m)
			}
			return core.GRPCCall(func(ctx context.Context, c service.ClientCommandsClient) error {
				show, err := c.ObjectShow(ctx, &pb.RpcObjectShowRequest{ObjectId: objectId})
				if err != nil {
					return err
				}
				var text string
				found := false
				marks := newMarks
				for _, b := range show.ObjectView.GetBlocks() {
					if b.Id == blockId && b.GetText() != nil {
						text = b.GetText().Text
						found = true
						if keep {
							marks = append(b.GetText().GetMarks().GetMarks(), newMarks...)
						}
					}
				}
				if !found {
					return fmt.Errorf("text block %s not found in %s", blockId, objectId)
				}
				resp, err := c.BlockTextSetText(ctx, &pb.RpcBlockTextSetTextRequest{
					ContextId: objectId,
					BlockId:   blockId,
					Text:      text,
					Marks:     &model.BlockContentTextMarks{Marks: marks},
				})
				if err != nil {
					return err
				}
				if resp.Error != nil && resp.Error.Code != pb.RpcBlockTextSetTextResponseError_NULL {
					return fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Description)
				}
				return emit(ok(map[string]any{"blockId": blockId, "marks": len(marks)}))
			})
		},
	}
	cmd.Flags().StringArrayVar(&specs, "mark", nil, "mark spec type:from:to[:param] (repeatable)")
	cmd.Flags().BoolVar(&keep, "keep", false, "merge with the block's existing marks instead of replacing")
	return cmd
}

func parseMarkSpec(s string) (*model.BlockContentTextMark, error) {
	parts := strings.SplitN(s, ":", 4)
	if len(parts) < 3 {
		return nil, fmt.Errorf("bad --mark %q (want type:from:to[:param])", s)
	}
	mt, ok := markTypes[parts[0]]
	if !ok {
		return nil, fmt.Errorf("unknown mark type %q", parts[0])
	}
	from, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("bad from in %q: %w", s, err)
	}
	to, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, fmt.Errorf("bad to in %q: %w", s, err)
	}
	param := ""
	if len(parts) == 4 {
		param = parts[3]
	}
	return &model.BlockContentTextMark{
		Range: &model.Range{From: int32(from), To: int32(to)},
		Type:  mt,
		Param: param,
	}, nil
}
