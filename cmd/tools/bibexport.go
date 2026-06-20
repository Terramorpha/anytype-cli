package tools

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/anyproto/anytype-heart/pb"
	"github.com/anyproto/anytype-heart/pb/service"
	"github.com/anyproto/anytype-heart/pkg/lib/bundle"
	"github.com/anyproto/anytype-heart/pkg/lib/pb/model"
	"github.com/anyproto/anytype-heart/util/pbtypes"

	"github.com/anyproto/anytype-cli/core"
)

// bibexport collects a text property (default the BibTeX field) from every
// object of a given type and writes the concatenation to a .bib file or stdout.
//
// The relation is resolved by api/relation key OR by display name, so you can
// pass either the internal key or e.g. "BibTeX".
func newBibexportCmd() *cobra.Command {
	var spaceId, typeKey, relation, out string
	cmd := &cobra.Command{
		Use:   "bibexport",
		Short: "Dump a text property (default BibTeX) from all objects of a type into a .bib file",
		Long: "Query every object of --type in a space and concatenate one of its text\n" +
			"properties (default the BibTeX field) into a .bib file (or stdout).\n\n" +
			"Example:\n  anytype tools bibexport --type source --relation BibTeX --out refs.bib",
		RunE: func(cmd *cobra.Command, args []string) error {
			if spaceId == "" {
				spaceId = os.Getenv("ANYTYPE_SPACE")
			}
			if spaceId == "" {
				return fmt.Errorf("space id required (--space or ANYTYPE_SPACE)")
			}
			var entries []string
			err := core.GRPCCall(func(ctx context.Context, c service.ClientCommandsClient) error {
				typeId, err := resolveTypeId(ctx, c, spaceId, typeKey)
				if err != nil {
					return err
				}
				relKey, err := resolveRelationKey(ctx, c, spaceId, relation)
				if err != nil {
					return err
				}
				resp, err := c.ObjectSearch(ctx, &pb.RpcObjectSearchRequest{
					SpaceId: spaceId,
					Filters: []*model.BlockContentDataviewFilter{{
						RelationKey: bundle.RelationKeyType.String(),
						Condition:   model.BlockContentDataviewFilter_Equal,
						Value:       pbtypes.String(typeId),
					}},
					Sorts: []*model.BlockContentDataviewSort{{
						RelationKey: bundle.RelationKeyName.String(),
						Type:        model.BlockContentDataviewSort_Asc,
					}},
					Keys: []string{bundle.RelationKeyName.String(), relKey},
				})
				if err != nil {
					return err
				}
				if resp.Error != nil && resp.Error.Code != pb.RpcObjectSearchResponseError_NULL {
					return fmt.Errorf("search: %s", resp.Error.Description)
				}
				for _, rec := range resp.Records {
					if v := strings.TrimSpace(pbtypes.GetString(rec, relKey)); v != "" {
						entries = append(entries, v)
					}
				}
				return nil
			})
			if err != nil {
				return err
			}
			body := strings.Join(entries, "\n\n") + "\n"
			if out == "" {
				fmt.Print(body)
				fmt.Fprintf(os.Stderr, "%d entries\n", len(entries))
				return nil
			}
			if err := os.WriteFile(out, []byte(body), 0o644); err != nil {
				return err
			}
			fmt.Printf("wrote %d entries to %s\n", len(entries), out)
			return nil
		},
	}
	cmd.Flags().StringVar(&spaceId, "space", "", "space id (or ANYTYPE_SPACE)")
	cmd.Flags().StringVar(&typeKey, "type", "source", "type api key (e.g. source) or type object id")
	cmd.Flags().StringVar(&relation, "relation", "BibTeX", "property to export: api key, internal key, or display name")
	cmd.Flags().StringVar(&out, "out", "", "output .bib file (default: stdout)")
	return cmd
}

// resolveTypeId accepts a type object id (returned as-is if it looks like one)
// or a type api key / name, and returns the type's object id.
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
// internal relation key used in object details.
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
