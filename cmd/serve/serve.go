package serve

import (
	"os"

	"github.com/kardianos/service"
	"github.com/spf13/cobra"

	"github.com/anyproto/anytype-cli/core/config"
	"github.com/anyproto/anytype-cli/core/output"
	"github.com/anyproto/anytype-cli/core/serviceprogram"
)

var (
	listenAddress string
	quietMode     bool
	verboseMode   bool
)

func NewServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "serve",
		Aliases: []string{"start"},
		Short:   "Run anytype in foreground",
		Long:    "Run anytype in the foreground. Use Ctrl+C to stop. For background operation, use the service commands instead.",
		RunE:    runServer,
	}

	cmd.Flags().StringVar(&listenAddress, "listen-address", config.APIAddr(), "API listen address in `host:port` format")
	cmd.Flags().BoolVarP(&quietMode, "quiet", "q", false, "Suppress most output (only errors)")
	cmd.Flags().BoolVarP(&verboseMode, "verbose", "v", false, "Show detailed output (debug level)")
	cmd.MarkFlagsMutuallyExclusive("quiet", "verbose")

	return cmd
}

func runServer(cmd *cobra.Command, args []string) error {
	// Configure anytype-heart log level via environment variables (must be set before server starts)
	if quietMode {
		os.Setenv("ANYTYPE_LOG_LEVEL", "*=FATAL")
		os.Setenv("ANYTYPE_LOG_NOGELF", "1")
	} else if verboseMode {
		os.Setenv("ANYTYPE_LOG_LEVEL", "*=DEBUG")
	}
	// Default log level (ERROR) is set in grpcserver/server.go if not specified

	svcConfig := &service.Config{
		Name:        "anytype",
		DisplayName: "Anytype",
		Description: "Anytype",
	}

	prg := serviceprogram.New(listenAddress)

	s, err := service.New(prg, svcConfig)
	if err != nil {
		return output.Error("Failed to create service: %w", err)
	}

	err = s.Run()
	if err != nil {
		return output.Error("service failed: %w", err)
	}

	return nil
}
