package main

import "github.com/animalet/sargantana-go/server"

func main() {
	err := server.NewServerFromFlags().StartAndBlock(
	// Put your controllers here
	)
	if err != nil {
		panic(err)
	}
}
