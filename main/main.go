package main

import (
	"flag"

	"github.com/animalet/sargantana-go/controller"
	"github.com/animalet/sargantana-go/server"
)

func main() {

	flagControllerInitializers := []func(*flag.FlagSet) func() controller.IController{
		controller.NewStaticFromFlags,
		controller.NewAuthFromFlags,
		controller.NewLoadBalancerFromFlags,
	}

	sargantana, controllers := server.NewServerFromFlags(flagControllerInitializers...)
	err := sargantana.StartAndWaitForSignal(controllers...)
	if err != nil {
		panic(err)
	}
}
