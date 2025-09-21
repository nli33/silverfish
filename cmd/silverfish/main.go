package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"runtime/pprof"
	"silverfish/engine"
	"time"
)

var shouldProfile *bool = flag.Bool("profile", false, "Enable profiling. Outputs results to cpu.prof")

func HandleMessages(channel chan engine.UciClientMessage) {
	stdinScanner := bufio.NewScanner(os.Stdin)

	for {
		message := engine.UciProcessClientMessage(stdinScanner)
		channel <- message

		quit_message := message.MessageType == engine.UciQuitClientMessage
		// Basically, stop listening for more messages after a valid quit message
		// is received.
		if quit_message {
			return
		}
	}
}

func executeGoCommand(channel chan bool, position *engine.Position, command *engine.UciGoMessage) {
	if command.Perft && command.Depth != 0 {
		engine.UciLog("Perft started.")
		result := engine.Perft(position, int(command.Depth), true)
		engine.UciLog(fmt.Sprintf("Perft result: %d", result))

		// Tell the main thread that we're done.
		channel <- true
		return
	}

	var bestMove engine.Move

	if command.Infinite {
		// We set a limited depth to avoid stack overflow (as we are using recursion
		// to implement the search right now)
		_, bestMove = engine.Search(position, engine.InfiniteDepth, engine.InfiniteMovetime)
	} else if command.Movetime != 0 {
		_, bestMove = engine.Search(position, engine.InfiniteDepth, time.Duration(command.Movetime)*time.Millisecond)
	} else if command.Depth != 0 {
		_, bestMove = engine.Search(position, int(command.Depth), engine.InfiniteMovetime)
	} else {
		_, bestMove = engine.Search(position, engine.InfiniteDepth, engine.TimeLimit(position, command)*time.Millisecond)
	}

	engine.UciBestMove(bestMove)

	channel <- true
}

func main() {
	flag.Parse()

	if *shouldProfile {
		engine.UciLog("Started profiling")
		profFile, err := os.Create("cpu.prof")
		if err != nil {
			fmt.Printf("error: failed to create profiling file: %v\n", err)
			os.Exit(1)
		}
		defer profFile.Close()

		err = pprof.StartCPUProfile(profFile)
		if err != nil {
			fmt.Printf("error: failed to start profiling, %v\n", err)
			os.Exit(1)
		}
	}

	defer pprof.StopCPUProfile()

	engine.Init()

	messageChannel := make(chan engine.UciClientMessage, 5)
	// Used for reporting if an action is done.
	actionAlertChannel := make(chan bool)

	position := engine.StartingPosition()

	go HandleMessages(messageChannel)

mainloop:
	for {
		message := engine.UciClientMessage{}
		select {
		case message = <-messageChannel:
		default:
			continue
		}

		switch message.MessageType {
		case engine.UciUciClientMessage:
			engine.UciSetEngineName("Silverfish 0.0.0a")
			engine.UciSetAuthor("李能和赵梁越")
			engine.UciSetProtocol(2)

			engine.UciOk()
		case engine.UciIsReadyClientMessage:
			engine.UciReadyOk()
		case engine.UciPositionClientMessage:
			position = *message.Position
		case engine.UciQuitClientMessage:
			break mainloop
		case engine.UciGoClientMessage:
			go executeGoCommand(actionAlertChannel, &position, message.GoMessage)
		}
	}
}
