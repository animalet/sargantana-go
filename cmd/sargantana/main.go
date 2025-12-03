package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/animalet/sargantana-go/pkg/config"
	"github.com/animalet/sargantana-go/pkg/controller"
	"github.com/animalet/sargantana-go/pkg/server"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	programName = "sargantana"
	exitSuccess = 0
	exitError   = 1
)

func main() {
	os.Exit(run())
}

func run() int {
	return runWithArgs(os.Args[1:])
}

// runWithArgs allows tests to pass custom arguments
func runWithArgs(args []string) int {
	// Parse command-line flags
	opts, err := parseFlags(args)
	if err != nil {
		// parseFlags handles printing errors and usage
		return exitError
	}

	// Handle help flag
	if opts.showHelp {
		printUsage(os.Stdout)
		return exitSuccess
	}

	// Handle version flag
	if opts.showVersion {
		fmt.Printf("%s version %s (commit: %s, built: %s)\n", programName, version, commit, date)
		return exitSuccess
	}

	// Setup logging
	setupLogging(opts.debug)

	// Validate required flags
	if opts.configPath == "" {
		fmt.Fprintf(os.Stderr, "Error: --config flag is required\n\n")
		printUsage(os.Stderr)
		return exitError
	}

	// Run the server
	if err := runServer(opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return exitError
	}

	return exitSuccess
}

// options holds all command-line options
type options struct {
	configPath  string
	debug       bool
	showVersion bool
	showHelp    bool
}

// parseFlags parses command-line flags and returns options or an error
func parseFlags(args []string) (*options, error) {
	opts := &options{}

	fs := flag.NewFlagSet(programName, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	// Define flags
	fs.StringVar(&opts.configPath, "config", "", "Path to configuration file (required)")
	fs.BoolVar(&opts.debug, "debug", false, "Enable debug mode")
	fs.BoolVar(&opts.showVersion, "version", false, "Show version information and exit")
	fs.BoolVar(&opts.showHelp, "help", false, "Show this help message and exit")

	// Custom usage function
	fs.Usage = func() {
		printUsage(os.Stderr)
	}

	// Parse flags
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			// Help was requested via -h or -help
			printUsage(os.Stdout)
			opts.showHelp = true
			return opts, nil
		}
		return nil, err
	}

	return opts, nil
}

// printUsage prints the usage message to the specified writer
func printUsage(w *os.File) {
	usage := `Usage: %s [OPTIONS]

Sargantana is a flexible web authentication gateway and reverse proxy.

OPTIONS:
  --config PATH    Path to configuration file (required)
  --debug          Enable debug mode with verbose logging
  --version        Display version information and exit
  --help           Display this help message and exit

EXAMPLES:
  %s --config /etc/sargantana/config.yaml
  %s --config ./config.yaml --debug

For more information, visit: https://github.com/animalet/sargantana-go
`
	_, err := fmt.Fprintf(w, usage, programName, programName, programName)
	if err != nil {
		panic(err)
	}
}

// setupLogging configures the global logger
func setupLogging(debug bool) {
	// Set log level
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// Configure pretty console output
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		NoColor:    false,
		TimeFormat: "2006-01-02 15:04:05",
	})
}

// initServer initializes and returns the Sargantana server (for tests)
func initServer(opts *options) (*server.Server, func() error, error) {
	// Load configuration
	cfg, err := loadConfig(opts.configPath)
	if err != nil {
		return nil, nil, err
	}

	// Get server configuration
	serverCfg, err := config.Get[server.SargantanaConfig](cfg, "sargantana")
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to load server configuration")
	}
	if serverCfg == nil {
		return nil, nil, errors.New("server configuration is required")
	}

	// Set debug mode
	server.SetDebug(opts.debug)

	// Register all controllers
	server.RegisterController("auth", controller.NewAuthController)
	server.RegisterController("load_balancer", controller.NewLoadBalancerController)
	server.RegisterController("static", controller.NewStaticController)
	server.RegisterController("template", controller.NewTemplateController)

	// Create server
	srv := server.NewServer(*serverCfg)

	// Configure authentication
	srv.SetAuthenticator(controller.NewGothAuthenticator())

	// Configure session store and get cleanup function
	closeSessionStore, err := configureSessionStore(
		cfg,
		srv,
		[]byte(serverCfg.WebServerConfig.SessionSecret),
		opts.debug,
	)
	if err != nil {
		return nil, nil, err
	}

	return srv, closeSessionStore, nil
}

// runServer initializes and runs the Sargantana server
func runServer(opts *options) error {
	srv, closeSessionStore, err := initServer(opts)
	if err != nil {
		return err
	}
	defer func() {
		if err := closeSessionStore(); err != nil {
			log.Error().Err(err).Msg("Failed to close session store")
		}
	}()

	// Start server and wait for termination signal
	log.Info().
		Str("config", opts.configPath).
		Bool("debug", opts.debug).
		Msg("Starting Sargantana server")

	if err := srv.StartAndWaitForSignal(); err != nil {
		return errors.Wrap(err, "server error")
	}

	return nil
}
