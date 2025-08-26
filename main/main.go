package main

import (
	"github.com/animalet/sargantana-go/controller"
	"github.com/animalet/sargantana-go/server"
)

// Version information set during build
var (
	version = "dev"
)

func main() {
	// Create a set of controllers that will be initialized from command line arguments (flags)
	controllerInitializers := []server.ControllerFlagInitializer{
		controller.NewStaticFromFlags,
		controller.NewAuthFromFlags,
		controller.NewLoadBalancerFromFlags,
	}

	// Parse command line flags and create the server and controllers based on those flags
	sargantana, controllers := server.NewServerFromFlagsWithVersion(version, controllerInitializers...)

	// Start the server with the controllers and wait for a termination signal
	err := sargantana.StartAndWaitForSignal(controllers...)
	if err != nil {
		panic(err)
	}
}
