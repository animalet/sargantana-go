package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/animalet/sargantana-go/controller"
	"github.com/animalet/sargantana-go/server"
)

// Version information set during build
var (
	version = "dev"
)

func main() {
	showVersion := flag.Bool("version", false, "Show version information")
	configFile := flag.String("config", "", "Path to configuration file")

	flag.Parse()

	if *showVersion {
		fmt.Printf("%s %s\n", "sargantana-go", version)
		os.Exit(0)
	}

	// Create a set of controllers that will be initialized from command line arguments (flags)
	controllerConfigurators := []controller.IControllerConfigurator{
		controller.NewStaticConfigurator(),
		controller.NewAuthConfigurator(),
		controller.NewLoadBalancerConfigurator(),
	}

	// Parse command line flags and create the server and controllers based on those flags
	sargantana, err := server.NewServer(*configFile, controllerConfigurators...)
	if err != nil {
		log.Fatalf("%v", err)
	}

	// Start the server with the controllers and wait for a termination signal
	err = sargantana.StartAndWaitForSignal()
	if err != nil {
		log.Fatalf("%v", err)
	}
}
