package main

import (
	"flag"

	"github.com/animalet/sargantana-go/controller"
	"github.com/animalet/sargantana-go/server"
)

// Version information set during build
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	// Create version info struct
	versionInfo := &server.VersionInfo{
		Version: version,
		Commit:  commit,
		Date:    date,
	}

	// Create a set of controllers that will be initialized from command line arguments (flags)
	controllerInitializers := []func(*flag.FlagSet) func() controller.IController{
		controller.NewStaticFromFlags,
		controller.NewAuthFromFlags,
		controller.NewLoadBalancerFromFlags,
	}

	// Parse command line flags and create the server and controllers based on those flags
	sargantana, controllers := server.NewServerFromFlagsWithVersion(versionInfo, controllerInitializers...)

	// Start the server with the controllers and wait for a termination signal
	err := sargantana.StartAndWaitForSignal(controllers...)
	if err != nil {
		panic(err)
	}
}
