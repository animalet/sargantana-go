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

	if *configFile == "" {
		log.Fatalf("Error: -config is required")
	}

	server.AddController("auth", controller.NewAuthController)
	server.AddController("static", controller.NewStaticController)
	server.AddController("load_balancer", controller.NewLoadBalancerController)

	sargantana, err := server.NewServer(*configFile)
	if err != nil {
		log.Fatalf("%v", err)
	}

	err = sargantana.StartAndWaitForSignal()
	if err != nil {
		log.Fatalf("%v", err)
	}
}
