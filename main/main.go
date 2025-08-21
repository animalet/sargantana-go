package main

import (
	"github.com/animalet/sargantana-go/controller"
	"github.com/animalet/sargantana-go/server"
)

func main() {

	controllers := []controller.IController{
		controller.NewStaticFromFlags(),
		controller.NewAuthFromFlags(),
		controller.NewLoadBalancerFromFlags(),
	}

	err := server.NewServerFromFlags().StartAndWaitForSignal(controllers...)
	if err != nil {
		panic(err)
	}
}
