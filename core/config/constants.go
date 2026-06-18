package config

import (
	"os"
	"path/filepath"
	"runtime"
)

const (
	// Default addresses
	LocalhostIP = "127.0.0.1"

	// Port configuration
	GRPCPort    = "31010"
	GRPCWebPort = "31011"
	APIPort     = "31012"

	// Full addresses
	DefaultGRPCAddress    = LocalhostIP + ":" + GRPCPort
	DefaultGRPCWebAddress = LocalhostIP + ":" + GRPCWebPort
	DefaultAPIAddress     = LocalhostIP + ":" + APIPort

	// URLs
	GRPCDNSAddress = "dns:///" + DefaultGRPCAddress

	// External URLs
	GitHubBaseURL    = "https://github.com/anyproto/anytype-cli"
	GitHubCommitURL  = GitHubBaseURL + "/commit/"
	GitHubReleaseURL = GitHubBaseURL + "/releases/tag/"

	// Anytype network address
	AnytypeNetworkAddress = "N83gJpVd9MuNRZAuJLZ7LiMntTThhPc6DtzWWVjb1M3PouVU"

	// Directory and file names
	AnytypeDirName = ".anytype"
	ConfigFileName = "config.json"
	DataDirName    = "data"
	LogsDirName    = "logs"
	AnytypeName    = "anytype"
)

func GetWorkDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "."
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support", AnytypeName)
	case "windows":
		return filepath.Join(homeDir, "AppData", "Roaming", AnytypeName)
	default:
		return filepath.Join(homeDir, ".config", AnytypeName)
	}
}

// envOr returns the environment variable value or a default.
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// GRPCAddr / GRPCWebAddr / APIAddr / GRPCDNSAddr return the listen/dial
// addresses, overridable per-instance via ANYTYPE_GRPC_PORT / ANYTYPE_GRPCWEB_PORT
// / ANYTYPE_API_PORT so multiple servers can run side by side.
func GRPCAddr() string    { return LocalhostIP + ":" + envOr("ANYTYPE_GRPC_PORT", GRPCPort) }
func GRPCWebAddr() string { return LocalhostIP + ":" + envOr("ANYTYPE_GRPCWEB_PORT", GRPCWebPort) }
func APIAddr() string     { return LocalhostIP + ":" + envOr("ANYTYPE_API_PORT", APIPort) }
func GRPCDNSAddr() string { return "dns:///" + GRPCAddr() }

func GetConfigDir() string {
	// A per-instance DATA_PATH also relocates config (accountId, techSpaceId),
	// so a second instance doesn't share the default account's config.
	if dataPath := os.Getenv("DATA_PATH"); dataPath != "" {
		return dataPath
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, AnytypeDirName)
}

func GetConfigFilePath() string {
	return filepath.Join(GetConfigDir(), ConfigFileName)
}

func GetDataDir() string {
	if dataPath := os.Getenv("DATA_PATH"); dataPath != "" {
		return dataPath
	}
	return filepath.Join(GetWorkDir(), DataDirName)
}

func GetLogsDir() string {
	return filepath.Join(GetConfigDir(), LogsDirName)
}
