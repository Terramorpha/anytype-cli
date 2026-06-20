// Package tools bundles the low-level gRPC helper commands that used to live as
// standalone programs under tools/. They operate directly on blocks, dataviews,
// views, invites, and file objects — below the higher-level command groups.
//
// All of them act on whatever server the gRPC client is configured for
// (DATA_PATH / ANYTYPE_GRPC_PORT), and most take object ids you can discover
// with `anytype tools dvinspect` / `anytype tools blockdump`.
package tools

import (
	"github.com/spf13/cobra"
)

func NewToolsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tools <command>",
		Short: "Low-level gRPC helpers for blocks, dataviews, invites and files",
		Long: "Low-level helpers that operate directly on the gRPC API: block edits,\n" +
			"dataview/view configuration, space invites, and file-object fixups.\n\n" +
			"These are power-user tools. Object and relation ids can be discovered\n" +
			"with `anytype tools dvinspect <id>` and `anytype tools blockdump <id>`.",
	}

	// blocks
	cmd.AddCommand(newAddblockCmd())
	cmd.AddCommand(newDividerCmd())
	cmd.AddCommand(newMarkCmd())
	cmd.AddCommand(newDelblockCmd())
	cmd.AddCommand(newAppendmdCmd())
	cmd.AddCommand(newBlockdumpCmd())
	cmd.AddCommand(newPagedeckCmd())
	cmd.AddCommand(newLinkCmd())
	cmd.AddCommand(newImageCmd())
	// lookup / inspection (label -> id, id -> details)
	cmd.AddCommand(newFindCmd())
	cmd.AddCommand(newDescribeCmd())
	cmd.AddCommand(newQueryCmd())
	cmd.AddCommand(newSetpropCmd())
	// dataviews / views
	cmd.AddCommand(newDvinspectCmd())
	cmd.AddCommand(newViewpropsCmd())
	cmd.AddCommand(newKanbanCmd())
	cmd.AddCommand(newDefaultkanbanCmd())
	cmd.AddCommand(newSetgroupCmd())
	cmd.AddCommand(newHidegroupCmd())
	cmd.AddCommand(newSetqCmd())
	cmd.AddCommand(newEmbedsetCmd())
	// space invites
	cmd.AddCommand(newGeninviteCmd())
	cmd.AddCommand(newGetinviteCmd())
	// files
	cmd.AddCommand(newImgctxCmd())
	// icons / covers
	cmd.AddCommand(newSeticonCmd())
	cmd.AddCommand(newSetcoverCmd())
	// sets / queries / collections
	cmd.AddCommand(newMakesetCmd())
	cmd.AddCommand(newMakecollectionCmd())
	cmd.AddCommand(newCollectaddCmd())

	return cmd
}
