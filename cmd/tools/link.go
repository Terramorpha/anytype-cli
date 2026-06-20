package tools

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/anyproto/anytype-heart/pb"
	"github.com/anyproto/anytype-heart/pb/service"
	"github.com/anyproto/anytype-heart/pkg/lib/pb/model"

	"github.com/anyproto/anytype-cli/core"
)

var linkCardStyles = map[string]model.BlockContentLinkCardStyle{
	"text":   model.BlockContentLink_Text,
	"card":   model.BlockContentLink_Card,
	"inline": model.BlockContentLink_Inline,
}

var linkIconSizes = map[string]model.BlockContentLinkIconSize{
	"none":   model.BlockContentLink_SizeNone,
	"small":  model.BlockContentLink_SizeSmall,
	"medium": model.BlockContentLink_SizeMedium,
}

var linkDescriptions = map[string]model.BlockContentLinkDescription{
	"none":    model.BlockContentLink_None,
	"added":   model.BlockContentLink_Added,
	"content": model.BlockContentLink_Content,
}

// link appends a "link to object" block, with a selectable render style:
// text (inline mention-style link), card (preview card), or inline.
func newLinkCmd() *cobra.Command {
	var style, icon, desc, parent, after string
	cmd := &cobra.Command{
		Use:   "link <pageId> <targetObjectId>",
		Short: "Append a link-to-object block (text | card | inline render)",
		Long: "Append a block that links to another object.\n\n" +
			"  --style text|card|inline   render style (default card)\n" +
			"  --icon  none|small|medium  icon size (card/inline)\n" +
			"  --desc  none|added|content description shown on the card\n" +
			"  --parent <blockId>         nest inside a parent block\n" +
			"  --after  <blockId>         insert after a sibling block",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			page, target := args[0], args[1]
			cs, ok2 := linkCardStyles[style]
			if !ok2 {
				return fmt.Errorf("unknown --style %q (text|card|inline)", style)
			}
			is, ok2 := linkIconSizes[icon]
			if !ok2 {
				return fmt.Errorf("unknown --icon %q (none|small|medium)", icon)
			}
			ds, ok2 := linkDescriptions[desc]
			if !ok2 {
				return fmt.Errorf("unknown --desc %q (none|added|content)", desc)
			}
			if parent != "" && after != "" {
				return fmt.Errorf("--parent and --after are mutually exclusive")
			}
			targetId, position := "", model.Block_Bottom
			if parent != "" {
				targetId, position = parent, model.Block_Inner
			} else if after != "" {
				targetId, position = after, model.Block_Bottom
			}
			return core.GRPCCall(func(ctx context.Context, c service.ClientCommandsClient) error {
				resp, err := c.BlockCreate(ctx, &pb.RpcBlockCreateRequest{
					ContextId: page,
					TargetId:  targetId,
					Position:  position,
					Block: &model.Block{Content: &model.BlockContentOfLink{Link: &model.BlockContentLink{
						TargetBlockId: target,
						Style:         model.BlockContentLink_Page,
						CardStyle:     cs,
						IconSize:      is,
						Description:   ds,
					}}},
				})
				if err != nil {
					return err
				}
				if resp.Error != nil && resp.Error.Code != pb.RpcBlockCreateResponseError_NULL {
					return fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Description)
				}
				return emit(ok(map[string]any{"blockId": resp.BlockId, "target": target, "style": style}))
			})
		},
	}
	cmd.Flags().StringVar(&style, "style", "card", "render style: text|card|inline")
	cmd.Flags().StringVar(&icon, "icon", "small", "icon size: none|small|medium")
	cmd.Flags().StringVar(&desc, "desc", "none", "card description: none|added|content")
	cmd.Flags().StringVar(&parent, "parent", "", "nest inside this parent block id")
	cmd.Flags().StringVar(&after, "after", "", "insert after this sibling block id")
	return cmd
}
