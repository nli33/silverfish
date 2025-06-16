package main

import (
	"silverfish/engine"
	"fmt"
	"os"
)

func main() {
	engine.Init()
	engine.InitBitboard()

	err := engine.UciOk()
	// This should not happen, ideally, but if it does, something is deeply wrong
	// with the program.
	if err != nil {
		_ = fmt.Errorf("WHAT THE FUCK??")
		os.Exit(69)
	}
}
