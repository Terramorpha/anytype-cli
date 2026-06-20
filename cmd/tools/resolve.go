package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/anyproto/anytype-heart/pb"
	"github.com/anyproto/anytype-heart/pb/service"
	"github.com/anyproto/anytype-heart/pkg/lib/bundle"
	"github.com/anyproto/anytype-heart/pkg/lib/pb/model"
	"github.com/anyproto/anytype-heart/util/pbtypes"
)

// resolveTypeId accepts a type object id (returned as-is if it looks like one)
// or a type api key / name, and returns the type's object id. General-purpose:
// lets commands take human-friendly type references instead of raw ids.
func resolveTypeId(ctx context.Context, c service.ClientCommandsClient, spaceId, typeKeyOrId string) (string, error) {
	if strings.HasPrefix(typeKeyOrId, "bafy") {
		return typeKeyOrId, nil
	}
	resp, err := c.ObjectSearch(ctx, &pb.RpcObjectSearchRequest{
		SpaceId: spaceId,
		Filters: []*model.BlockContentDataviewFilter{{
			RelationKey: bundle.RelationKeyResolvedLayout.String(),
			Condition:   model.BlockContentDataviewFilter_Equal,
			Value:       pbtypes.Int64(int64(model.ObjectType_objectType)),
		}},
		Keys: []string{bundle.RelationKeyId.String(), bundle.RelationKeyApiObjectKey.String(), bundle.RelationKeyUniqueKey.String(), bundle.RelationKeyName.String()},
	})
	if err != nil {
		return "", err
	}
	for _, rec := range resp.Records {
		api := pbtypes.GetString(rec, bundle.RelationKeyApiObjectKey.String())
		uniq := pbtypes.GetString(rec, bundle.RelationKeyUniqueKey.String())
		name := pbtypes.GetString(rec, bundle.RelationKeyName.String())
		if api == typeKeyOrId || uniq == "ot-"+typeKeyOrId || strings.EqualFold(name, typeKeyOrId) {
			return pbtypes.GetString(rec, bundle.RelationKeyId.String()), nil
		}
	}
	return "", fmt.Errorf("type %q not found in space", typeKeyOrId)
}

// resolveRelationKey turns an api key / internal key / display name into the
// internal relation key used in object details and dataview relations. This is
// the bridge between the friendly REST-style keys and the raw gRPC keys.
func resolveRelationKey(ctx context.Context, c service.ClientCommandsClient, spaceId, rel string) (string, error) {
	resp, err := c.ObjectSearch(ctx, &pb.RpcObjectSearchRequest{
		SpaceId: spaceId,
		Filters: []*model.BlockContentDataviewFilter{{
			RelationKey: bundle.RelationKeyResolvedLayout.String(),
			Condition:   model.BlockContentDataviewFilter_Equal,
			Value:       pbtypes.Int64(int64(model.ObjectType_relation)),
		}},
		Keys: []string{bundle.RelationKeyRelationKey.String(), bundle.RelationKeyApiObjectKey.String(), bundle.RelationKeyName.String()},
	})
	if err != nil {
		return "", err
	}
	for _, rec := range resp.Records {
		key := pbtypes.GetString(rec, bundle.RelationKeyRelationKey.String())
		api := pbtypes.GetString(rec, bundle.RelationKeyApiObjectKey.String())
		name := pbtypes.GetString(rec, bundle.RelationKeyName.String())
		if key == rel || api == rel || strings.EqualFold(name, rel) {
			return key, nil
		}
	}
	// fall back to using the argument directly (might already be the key)
	return rel, nil
}
