package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charignon/umcp/internal/config"
	"github.com/charignon/umcp/internal/logger"
	"github.com/charignon/umcp/internal/mcp"
	"github.com/rs/zerolog/log"
)

var version = "1.0.0"

func main() {
	var (
		configPaths     stringSlice
		workingDir      string
		timeout         int
		logLevel        string
		generateClaude  bool
		validateOnly    bool
		testMode        bool
		showVersion     bool
		debugMode       bool
		debugTrace      string
		replayTrace     string
	)

	flag.Var(&configPaths, "config", "Path to YAML configuration file (can be specified multiple times)")
	flag.StringVar(&workingDir, "working-dir", "", "Working directory for command execution")
	flag.IntVar(&timeout, "timeout", 60, "Default timeout in seconds")
	flag.StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	flag.BoolVar(&generateClaude, "generate-claude-config", false, "Generate Claude Desktop configuration")
	flag.BoolVar(&validateOnly, "validate", false, "Validate configuration only")
	flag.BoolVar(&testMode, "test", false, "Run in test mode")
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.BoolVar(&debugMode, "debug", false, "Enable debug mode with message tracing")
	flag.StringVar(&debugTrace, "debug-trace", "", "File to save debug trace (enables debug mode)")
	flag.StringVar(&replayTrace, "replay-trace", "", "File to replay debug trace from")
	flag.Parse()

	if showVersion {
		fmt.Printf("umcp version %s\n", version)
		os.Exit(0)
	}

	// Setup logging to stderr
	logger.SetupLogger(logLevel)

	if len(configPaths) == 0 {
		log.Fatal().Msg("At least one config file must be specified with --config")
	}

	// Load configurations
	configs := make([]*config.Config, 0, len(configPaths))
	for _, path := range configPaths {
		cfg, err := config.LoadConfig(path)
		if err != nil {
			log.Fatal().Err(err).Str("config", path).Msg("Failed to load configuration")
		}
		configs = append(configs, cfg)
		log.Info().Str("config", path).Msg("Loaded configuration")
	}

	if validateOnly {
		fmt.Println("All configurations are valid")
		os.Exit(0)
	}

	if generateClaude {
		generateClaudeConfig(configs, configPaths)
		os.Exit(0)
	}

	// Enable debug mode if specified
	if debugTrace != "" {
		debugMode = true
	}

	// Create and run MCP server
	server := mcp.NewServer(configs, mcp.ServerOptions{
		DebugMode:   debugMode,
		DebugTrace:  debugTrace,
		ReplayTrace: replayTrace,
	})

	if testMode {
		log.Info().Msg("Running in test mode")
		// In test mode, just validate that everything initializes correctly
		os.Exit(0)
	}

	if err := server.Run(); err != nil {
		log.Fatal().Err(err).Msg("Server failed")
	}
}

func generateClaudeConfig(configs []*config.Config, paths []string) {
	fmt.Println("{")
	fmt.Println(`  "mcpServers": {`)

	for i, cfg := range configs {
		fmt.Printf(`    "%s": {`+"\n", cfg.Metadata.Name)
		fmt.Printf(`      "command": "umcp",`+"\n")
		fmt.Printf(`      "args": ["--config", "%s"]`+"\n", paths[i])
		fmt.Print(`    }`)
		if i < len(configs)-1 {
			fmt.Print(",")
		}
		fmt.Println()
	}

	fmt.Println("  }")
	fmt.Println("}")
}

type stringSlice []string

func (s *stringSlice) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSlice) Set(value string) error {
	absPath, err := filepath.Abs(value)
	if err != nil {
		return err
	}
	*s = append(*s, absPath)
	return nil
}