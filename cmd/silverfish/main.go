package main

import (
	"bufio"
	"fmt"
	"os"
	"silverfish/engine"
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

	position := engine.NewPosition()
	should_continue := true

	stdinScanner := bufio.NewScanner(os.Stdin)

	for should_continue {
		engine.UciHandleMessages(*stdinScanner, &position, &should_continue)
	}
}
