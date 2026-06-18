package core

import (
	"context"
	"fmt"

	"github.com/anyproto/anytype-heart/pb"
	"github.com/anyproto/anytype-heart/pb/service"
	"github.com/anyproto/anytype-heart/pkg/lib/bundle"
	"github.com/anyproto/anytype-heart/pkg/lib/pb/model"
	"github.com/anyproto/anytype-heart/util/pbtypes"

	"github.com/anyproto/anytype-cli/core/config"
)

// resolveProfile returns the account's profile object id and the space its
// objects (e.g. the icon image) should live in (the tech space).
func resolveProfile(ctx context.Context, client service.ClientCommandsClient) (profileId, spaceId string, err error) {
	accountId, err := config.GetAccountIdFromConfig()
	if err != nil {
		return "", "", fmt.Errorf("account id not found - please login first: %w", err)
	}
	sel, err := client.AccountSelect(ctx, &pb.RpcAccountSelectRequest{
		Id:       accountId,
		RootPath: config.GetDataDir(),
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to select account: %w", err)
	}
	if sel.Error != nil && sel.Error.Code != pb.RpcAccountSelectResponseError_NULL {
		return "", "", fmt.Errorf("account select error: %s", sel.Error.Description)
	}
	if sel.Account == nil || sel.Account.Info == nil {
		return "", "", fmt.Errorf("no account info returned")
	}
	profileId = sel.Account.Info.ProfileObjectId
	spaceId = sel.Account.Info.TechSpaceId
	if spaceId == "" {
		spaceId = sel.Account.Info.AccountSpaceId
	}
	return profileId, spaceId, nil
}

// SetProfile sets the account profile's display name and/or icon image. Empty
// arguments are left unchanged. The icon is set from a local image file.
func SetProfile(name, imagePath string) error {
	if name == "" && imagePath == "" {
		return fmt.Errorf("nothing to set: provide a name and/or an image")
	}
	return GRPCCall(func(ctx context.Context, client service.ClientCommandsClient) error {
		profileId, spaceId, err := resolveProfile(ctx, client)
		if err != nil {
			return err
		}

		var details []*model.Detail
		if name != "" {
			details = append(details, &model.Detail{
				Key:   string(bundle.RelationKeyName),
				Value: pbtypes.String(name),
			})
		}
		if imagePath != "" {
			up, err := client.FileUpload(ctx, &pb.RpcFileUploadRequest{
				SpaceId:   spaceId,
				LocalPath: imagePath,
				Type:      model.BlockContentFile_Image,
			})
			if err != nil {
				return fmt.Errorf("failed to upload image: %w", err)
			}
			if up.Error != nil && up.Error.Code != pb.RpcFileUploadResponseError_NULL {
				return fmt.Errorf("file upload error: %s", up.Error.Description)
			}
			details = append(details, &model.Detail{
				Key:   string(bundle.RelationKeyIconImage),
				Value: pbtypes.String(up.ObjectId),
			})
		}

		set, err := client.ObjectSetDetails(ctx, &pb.RpcObjectSetDetailsRequest{
			ContextId: profileId,
			Details:   details,
		})
		if err != nil {
			return fmt.Errorf("failed to set profile details: %w", err)
		}
		if set.Error != nil && set.Error.Code != pb.RpcObjectSetDetailsResponseError_NULL {
			return fmt.Errorf("set details error: %s", set.Error.Description)
		}
		return nil
	})
}
