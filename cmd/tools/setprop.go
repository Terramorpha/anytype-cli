package tools

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/spf13/cobra"

	"github.com/anyproto/anytype-heart/pb/service"
	"github.com/anyproto/anytype-heart/pkg/lib/pb/model"
	"github.com/anyproto/anytype-heart/util/pbtypes"

	"github.com/anyproto/anytype-cli/core"
)

// setprop sets any relation value on an object, typed by --type. This is the
// general write primitive (object links / tags / selects take ids — get them
// with `tools find`). The relation key is the internal key (see `tools find
// --kind relation` / `tools describe`).
func newSetpropCmd() *cobra.Command {
	var vtype string
	cmd := &cobra.Command{
		Use:   "setprop <objectId> <relationKey> <value>",
		Short: "Set a relation value on an object (any format)",
		Long: "Set a relation value, typed via --type:\n" +
			"  text    (default)            a string\n" +
			"  number                       a number\n" +
			"  bool                         true/false\n" +
			"  date                         YYYY-MM-DD (stored as unix seconds)\n" +
			"  list                         comma-separated ids (objects/tags/multi-select)\n\n" +
			"For object/tag/select relations the value is the target/option id — get it\n" +
			"with `tools find`. The relation key is the internal key (`tools describe`).",
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			objectId, key, raw := args[0], args[1], args[2]
			var val *types.Value
			switch vtype {
			case "", "text":
				val = pbtypes.String(raw)
			case "number":
				n, err := strconv.ParseFloat(raw, 64)
				if err != nil {
					return fmt.Errorf("--type number: %w", err)
				}
				val = pbtypes.Float64(n)
			case "bool":
				val = pbtypes.Bool(raw == "true" || raw == "1")
			case "date":
				t, err := time.Parse("2006-01-02", raw)
				if err != nil {
					return fmt.Errorf("--type date: want YYYY-MM-DD: %w", err)
				}
				val = pbtypes.Float64(float64(t.Unix()))
			case "list":
				parts := strings.Split(raw, ",")
				for i := range parts {
					parts[i] = strings.TrimSpace(parts[i])
				}
				val = pbtypes.StringList(parts)
			default:
				return fmt.Errorf("unknown --type %q (text|number|bool|date|list)", vtype)
			}
			return core.GRPCCall(func(ctx context.Context, c service.ClientCommandsClient) error {
				return setDetails(ctx, c, objectId, []*model.Detail{{Key: key, Value: val}})
			})
		},
	}
	cmd.Flags().StringVar(&vtype, "type", "text", "value type: text|number|bool|date|list")
	return cmd
}
