package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/animalet/sargantana-go/controller"
	"github.com/animalet/sargantana-go/logger"
	"github.com/animalet/sargantana-go/server"
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

	if *showVersion {
		fmt.Printf("%s %s\n", "sargantana-go", version)
		os.Exit(0)
	}

	if *configFile == "" {
		logger.Fatal("Error: -config is required")
	}

	server.SetDebug(*debugMode)
	server.AddControllerType("auth", controller.NewAuthController)
	server.AddControllerType("static", controller.NewStaticController)
	server.AddControllerType("load_balancer", controller.NewLoadBalancerController)

	sargantana, err := server.NewServer(*configFile)
	if err != nil {
		logger.Fatalf("%v", err)
		os.Exit(1)
	}

	err = sargantana.StartAndWaitForSignal()
	if err != nil {
		logger.Fatalf("%v", err)
	}
}
