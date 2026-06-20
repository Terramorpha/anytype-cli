package tools

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/anyproto/anytype-heart/pb"
	"github.com/anyproto/anytype-heart/pb/service"
	"github.com/anyproto/anytype-heart/pkg/lib/bundle"
	"github.com/anyproto/anytype-heart/pkg/lib/pb/model"
	"github.com/anyproto/anytype-heart/util/pbtypes"

	"github.com/anyproto/anytype-cli/core"
)

// uploadImage uploads a local image and returns its file object id (no chat
// context — for icons/covers).
func uploadImage(ctx context.Context, c service.ClientCommandsClient, spaceId, path string) (string, error) {
	resp, err := c.FileUpload(ctx, &pb.RpcFileUploadRequest{
		SpaceId: spaceId, LocalPath: path, Type: model.BlockContentFile_Image,
	})
	if err != nil {
		return "", err
	}
	if resp.Error != nil && resp.Error.Code != pb.RpcFileUploadResponseError_NULL {
		return "", fmt.Errorf("file upload: %s", resp.Error.Description)
	}
	return resp.ObjectId, nil
}

// seticon sets an object's icon to an emoji or an uploaded image.
func newSeticonCmd() *cobra.Command {
	var emoji, image string
	cmd := &cobra.Command{
		Use:   "seticon <objectId>",
		Short: "Set an object's icon to an emoji or an uploaded image",
		Long:  "Set the icon of any object.\n\n  --emoji 📚            set an emoji icon\n  --image ./logo.png    upload an image and use it as the icon",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			objectId := args[0]
			if (emoji == "") == (image == "") {
				return fmt.Errorf("provide exactly one of --emoji or --image")
			}
			spaceId := spaceOf(cmd)
			return core.GRPCCall(func(ctx context.Context, c service.ClientCommandsClient) error {
				var details []*model.Detail
				if emoji != "" {
					// emoji and image are mutually exclusive; clear the other.
					details = []*model.Detail{
						{Key: bundle.RelationKeyIconEmoji.String(), Value: pbtypes.String(emoji)},
						{Key: bundle.RelationKeyIconImage.String(), Value: pbtypes.String("")},
					}
				} else {
					if spaceId == "" {
						return fmt.Errorf("space id required for --image (use --space or ANYTYPE_SPACE)")
					}
					fileId, err := uploadImage(ctx, c, spaceId, image)
					if err != nil {
						return err
					}
					details = []*model.Detail{
						{Key: bundle.RelationKeyIconImage.String(), Value: pbtypes.String(fileId)},
						{Key: bundle.RelationKeyIconEmoji.String(), Value: pbtypes.String("")},
					}
				}
				return setDetails(ctx, c, objectId, details)
			})
		},
	}
	cmd.Flags().StringVar(&emoji, "emoji", "", "emoji to use as the icon")
	cmd.Flags().StringVar(&image, "image", "", "local image path to upload and use as the icon")
	cmd.Flags().String("space", "", "space id (or ANYTYPE_SPACE) — needed for --image")
	return cmd
}

// setcover sets an object's cover: a gradient, a solid color, or an image.
// Cover types (learned empirically): 1 = uploaded image (coverId = file id),
// 3 = gradient (coverId = gradient name), 4 = solid color (coverId = color name).
func newSetcoverCmd() *cobra.Command {
	var gradient, color, image string
	var unset bool
	cmd := &cobra.Command{
		Use:   "setcover <objectId>",
		Short: "Set an object's cover (gradient, color, or image)",
		Long: "Set the cover of any object.\n\n" +
			"  --gradient blue     gradient cover (yellow, red, blue, teal, pink, …)\n" +
			"  --color red         solid color cover\n" +
			"  --image ./bg.jpg    upload an image as the cover\n" +
			"  --unset             remove the cover",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			objectId := args[0]
			n := 0
			for _, s := range []string{gradient, color, image} {
				if s != "" {
					n++
				}
			}
			if unset {
				n++
			}
			if n != 1 {
				return fmt.Errorf("provide exactly one of --gradient, --color, --image, or --unset")
			}
			spaceId := spaceOf(cmd)
			return core.GRPCCall(func(ctx context.Context, c service.ClientCommandsClient) error {
				var coverType int64
				var coverId string
				switch {
				case unset:
					coverType, coverId = 0, ""
				case gradient != "":
					coverType, coverId = 3, gradient
				case color != "":
					coverType, coverId = 4, color
				case image != "":
					if spaceId == "" {
						return fmt.Errorf("space id required for --image (use --space or ANYTYPE_SPACE)")
					}
					fileId, err := uploadImage(ctx, c, spaceId, image)
					if err != nil {
						return err
					}
					coverType, coverId = 1, fileId
				}
				return setDetails(ctx, c, objectId, []*model.Detail{
					{Key: bundle.RelationKeyCoverType.String(), Value: pbtypes.Int64(coverType)},
					{Key: bundle.RelationKeyCoverId.String(), Value: pbtypes.String(coverId)},
				})
			})
		},
	}
	cmd.Flags().StringVar(&gradient, "gradient", "", "gradient cover name (e.g. blue, teal, pink)")
	cmd.Flags().StringVar(&color, "color", "", "solid color cover name (e.g. red, yellow)")
	cmd.Flags().StringVar(&image, "image", "", "local image path to upload as the cover")
	cmd.Flags().BoolVar(&unset, "unset", false, "remove the cover")
	cmd.Flags().String("space", "", "space id (or ANYTYPE_SPACE) — needed for --image")
	return cmd
}

// spaceOf resolves the space id from the --space flag or ANYTYPE_SPACE.
func spaceOf(cmd *cobra.Command) string {
	s, _ := cmd.Flags().GetString("space")
	if s == "" {
		s = os.Getenv("ANYTYPE_SPACE")
	}
	return s
}

// setDetails applies object detail updates and checks the response error.
func setDetails(ctx context.Context, c service.ClientCommandsClient, objectId string, details []*model.Detail) error {
	resp, err := c.ObjectSetDetails(ctx, &pb.RpcObjectSetDetailsRequest{ContextId: objectId, Details: details})
	if err != nil {
		return err
	}
	if resp.Error != nil && resp.Error.Code != pb.RpcObjectSetDetailsResponseError_NULL {
		return fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Description)
	}
	keys := make([]string, 0, len(details))
	for _, d := range details {
		keys = append(keys, d.Key)
	}
	return emit(ok(map[string]any{"id": objectId, "set": keys}))
}
