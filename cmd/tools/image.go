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

// image appends an image/file block to a page in one call
// (BlockFileCreateAndUpload), from a local path or a URL.
func newImageCmd() *cobra.Command {
	var file, url, kind, parent, after string
	cmd := &cobra.Command{
		Use:   "image <pageId>",
		Short: "Append an image/file block to a page (upload from --file or --url)",
		Long: "Append a media block and upload its content in one call.\n\n" +
			"  --file ./pic.jpg    upload a local file\n" +
			"  --url  https://…    fetch and upload from a URL\n" +
			"  --kind image|file|video|audio|pdf   block type (default image)\n" +
			"  --parent/--after    placement (see addblock)",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			page := args[0]
			if (file == "") == (url == "") {
				return fmt.Errorf("provide exactly one of --file or --url")
			}
			ft, ok2 := fileKinds[kind]
			if !ok2 {
				return fmt.Errorf("unknown --kind %q (image|file|video|audio|pdf|none)", kind)
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
				resp, err := c.BlockFileCreateAndUpload(ctx, &pb.RpcBlockFileCreateAndUploadRequest{
					ContextId: page,
					TargetId:  targetId,
					Position:  position,
					Url:       url,
					LocalPath: file,
					FileType:  ft,
				})
				if err != nil {
					return err
				}
				if resp.Error != nil && resp.Error.Code != pb.RpcBlockFileCreateAndUploadResponseError_NULL {
					return fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Description)
				}
				return emit(ok(map[string]any{"blockId": resp.BlockId, "kind": kind}))
			})
		},
	}
	cmd.Flags().StringVar(&file, "file", "", "local file path to upload")
	cmd.Flags().StringVar(&url, "url", "", "URL to fetch and upload")
	cmd.Flags().StringVar(&kind, "kind", "image", "block type: image|file|video|audio|pdf|none")
	cmd.Flags().StringVar(&parent, "parent", "", "nest inside this parent block id")
	cmd.Flags().StringVar(&after, "after", "", "insert after this sibling block id")
	return cmd
}

var fileKinds = map[string]model.BlockContentFileType{
	"none":  model.BlockContentFile_None,
	"file":  model.BlockContentFile_File,
	"image": model.BlockContentFile_Image,
	"video": model.BlockContentFile_Video,
	"audio": model.BlockContentFile_Audio,
	"pdf":   model.BlockContentFile_PDF,
}
