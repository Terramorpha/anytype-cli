package tools

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gogo/protobuf/types"
	"github.com/spf13/cobra"

	"github.com/anyproto/anytype-heart/pb"
	"github.com/anyproto/anytype-heart/pb/service"
	"github.com/anyproto/anytype-heart/pkg/lib/bundle"
	"github.com/anyproto/anytype-heart/pkg/lib/pb/model"
	"github.com/anyproto/anytype-heart/util/pbtypes"

	"github.com/anyproto/anytype-cli/core"
)

// filterOps maps a textual operator to a filter condition. Order matters: the
// two-char ops are checked before "=" / ">" / "<".
var filterOps = []struct {
	tok  string
	cond model.BlockContentDataviewFilterCondition
}{
	{"!=", model.BlockContentDataviewFilter_NotEqual},
	{">=", model.BlockContentDataviewFilter_GreaterOrEqual},
	{"<=", model.BlockContentDataviewFilter_LessOrEqual},
	{"~", model.BlockContentDataviewFilter_Like},
	{"=", model.BlockContentDataviewFilter_Equal},
	{">", model.BlockContentDataviewFilter_Greater},
	{"<", model.BlockContentDataviewFilter_Less},
}

// query is the structured read primitive: search objects of a type and/or
// matching filters, returning the requested relation values as JSON rows.
func newQueryCmd() *cobra.Command {
	var spaceId, typeId, keysCsv string
	var filters []string
	var limit int
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Structured object search → JSON rows (--type, --filter, --keys)",
		Long: "Search objects and return relation values as JSON.\n\n" +
			"  --type <typeId>        restrict to a type (filters on the type relation)\n" +
			"  --filter \"key=value\"   repeatable; ops: = != ~ (contains) > < >= <=\n" +
			"  --keys k1,k2           relation keys to return (id and name always included)\n\n" +
			"Filter/return keys are internal relation keys (see `tools find --kind relation`).",
		RunE: func(cmd *cobra.Command, args []string) error {
			if spaceId == "" {
				spaceId = os.Getenv("ANYTYPE_SPACE")
			}
			if spaceId == "" {
				return fmt.Errorf("space id required (--space or ANYTYPE_SPACE)")
			}
			var fs []*model.BlockContentDataviewFilter
			if typeId != "" {
				fs = append(fs, &model.BlockContentDataviewFilter{
					RelationKey: bundle.RelationKeyType.String(),
					Condition:   model.BlockContentDataviewFilter_Equal,
					Value:       pbtypes.String(typeId),
				})
			}
			for _, f := range filters {
				key, cond, val, err := parseFilter(f)
				if err != nil {
					return err
				}
				// Type the value: numbers as numbers (so >/< compare numerically),
				// true/false as bools, else string.
				var pv *types.Value
				switch {
				case val == "true" || val == "false":
					pv = pbtypes.Bool(val == "true")
				default:
					if n, perr := strconv.ParseFloat(val, 64); perr == nil {
						pv = pbtypes.Float64(n)
					} else {
						pv = pbtypes.String(val)
					}
				}
				fs = append(fs, &model.BlockContentDataviewFilter{
					RelationKey: key, Condition: cond, Value: pv,
				})
			}
			keys := []string{bundle.RelationKeyId.String(), bundle.RelationKeyName.String()}
			if keysCsv != "" {
				for _, k := range strings.Split(keysCsv, ",") {
					if k = strings.TrimSpace(k); k != "" {
						keys = append(keys, k)
					}
				}
			}
			var rows []map[string]any
			err := core.GRPCCall(func(ctx context.Context, c service.ClientCommandsClient) error {
				resp, err := c.ObjectSearch(ctx, &pb.RpcObjectSearchRequest{
					SpaceId: spaceId, Filters: fs, Keys: keys, Limit: int32(limit),
				})
				if err != nil {
					return err
				}
				if resp.Error != nil && resp.Error.Code != pb.RpcObjectSearchResponseError_NULL {
					return fmt.Errorf("search: %s", resp.Error.Description)
				}
				for _, rec := range resp.Records {
					row := map[string]any{}
					for _, k := range keys {
						if v, okv := rec.GetFields()[k]; okv {
							row[k] = valueToAny(v)
						}
					}
					rows = append(rows, row)
				}
				return nil
			})
			if err != nil {
				return err
			}
			if rows == nil {
				rows = []map[string]any{}
			}
			return emit(rows)
		},
	}
	cmd.Flags().StringVar(&spaceId, "space", "", "space id (or ANYTYPE_SPACE)")
	cmd.Flags().StringVar(&typeId, "type", "", "restrict to this type object id")
	cmd.Flags().StringArrayVar(&filters, "filter", nil, "filter \"key<op>value\" (op: = != ~ > < >= <=); repeatable")
	cmd.Flags().StringVar(&keysCsv, "keys", "", "comma-separated relation keys to return")
	cmd.Flags().IntVar(&limit, "limit", 100, "max rows")
	return cmd
}

func parseFilter(f string) (key string, cond model.BlockContentDataviewFilterCondition, val string, err error) {
	for _, op := range filterOps {
		if i := strings.Index(f, op.tok); i > 0 {
			return f[:i], op.cond, f[i+len(op.tok):], nil
		}
	}
	return "", 0, "", fmt.Errorf("bad filter %q (want key<op>value, op one of = != ~ > < >= <=)", f)
}

// valueToAny converts a protobuf Value into a plain Go value for JSON.
func valueToAny(v *types.Value) any {
	switch k := v.GetKind().(type) {
	case *types.Value_StringValue:
		return k.StringValue
	case *types.Value_NumberValue:
		return k.NumberValue
	case *types.Value_BoolValue:
		return k.BoolValue
	case *types.Value_ListValue:
		out := []any{}
		for _, e := range k.ListValue.GetValues() {
			out = append(out, valueToAny(e))
		}
		return out
	default:
		return nil
	}
}
