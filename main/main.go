package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/animalet/sargantana-go/controller"
	"github.com/animalet/sargantana-go/server"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Version information set during build
var (
	version = "dev"
)

func main() {
	showVersion := flag.Bool("version", false, "Show version information")
	debugMode := flag.Bool("debug", false, "Enable debug mode")
	configFile := flag.String("config", "", "Path to configuration file")

	flag.Parse()

	if *debugMode {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	if *showVersion {
		fmt.Printf("%s %s\n", "sargantana-go", version)
		os.Exit(0)
	}

	if *configFile == "" {
		n, err := fmt.Fprintln(os.Stderr, "Error: -config flag is required")
		if err != nil || n <= 0 {
			panic("Failed to print error message")
		}
		os.Exit(1)
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		NoColor:    false,
		TimeFormat: "2006-01-02 15:04:05",
	})
	server.SetDebug(*debugMode)
	server.AddControllerType("auth", controller.NewAuthController)
	server.AddControllerType("load_balancer", controller.NewLoadBalancerController)
	server.AddControllerType("static", controller.NewStaticController)
	server.AddControllerType("template", controller.NewTemplateController)

	sargantana, err := server.NewServerFromConfigFile(*configFile)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create server")
		os.Exit(1)
	}

	err = sargantana.StartAndWaitForSignal()
	if err != nil {
		log.Fatal().Err(err).Msg("Server error")
	}
}
