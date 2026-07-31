package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"runtime/pprof"
	"silverfish/engine"
	"strings"
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
		// Signal completion before logging: the main loop drops any
		// message that arrives while `active` is still true, so a fast
		// client sending its next command right after seeing output on
		// stdout must never be able to race ahead of this.
		channel <- true
		engine.UciLog(fmt.Sprintf("Perft result: %d", result))
		return
	}

	depth := engine.InfiniteDepth
	moveTime := engine.InfiniteMovetime
	switch {
	case command.Infinite:
		// keep defaults

	case command.Movetime != 0:
		moveTime = time.Duration(command.Movetime) * time.Millisecond

	case command.Depth != 0:
		depth = int(command.Depth)

	default:
		moveTime = engine.TimeLimit(position, command) * time.Millisecond
	}
	search := engine.Search{
		MaxDepth:  depth,
		TimeLimit: moveTime,
	}
	search.Init(position)

	var bestMove engine.Move
	_, bestMove = search.Search()

	// Signal completion (clearing `active` in the main loop) before
	// printing bestmove -- see the comment above in the perft branch. A
	// client that reacts to "bestmove" on stdout by immediately sending
	// its next command must never be able to race ahead of `active`
	// being reset, or that next command (including the next "go") gets
	// silently dropped and the engine hangs waiting for a command that
	// will never come.
	channel <- true
	engine.UciBestMove(bestMove)
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
	active := false

	position := engine.StartingPosition()

	go HandleMessages(messageChannel)

mainloop:
	for {
		select {
		case message := <-messageChannel:
			if active {
				// A search goroutine is running. Only quit is handled here;
				// everything else (including stop) is dropped, since Search
				// has no cancellation mechanism yet. Wait for the goroutine
				// to actually finish (and print its result) before exiting,
				// rather than tearing down the process out from under it.
				if message.MessageType == engine.UciQuitClientMessage {
					<-actionAlertChannel
					break mainloop
				}
				continue
			}

			switch message.MessageType {
			case engine.UciUciClientMessage:
				engine.UciSetEngineName("Silverfish 0.0.0a")
				engine.UciSetAuthor("李能和赵梁越")
				engine.UciOptions()
				engine.UciOk()
			case engine.UciIsReadyClientMessage:
				engine.UciReadyOk()
			case engine.UciPositionClientMessage:
				position = message.Position.Clone()
			case engine.UciNewGameClientMessage:
				engine.ClearTT()
			case engine.UciQuitClientMessage:
				break mainloop
			case engine.UciGoClientMessage:
				active = true
				go executeGoCommand(actionAlertChannel, &position, message.GoMessage)
			case engine.UciSetOptionClientMessage:
				handleSetOption(message.SetOption, &position)
			}

		case <-actionAlertChannel:
			active = false
		}
	}
}

func handleSetOption(opt *engine.UciSetOptionMessage, position *engine.Position) {
	if opt == nil || !strings.EqualFold(opt.Name, "EvalFile") {
		return
	}

	path := opt.Value
	if path == "<empty>" {
		path = ""
	}

	if err := engine.LoadDefaultNetwork(path); err != nil {
		engine.UciError(fmt.Sprintf("failed to load EvalFile %q: %v", opt.Value, err))
		return
	}

	// The current position's accumulator was built from the old network,
	// so it can't just be re-pointed at the new one -- reset to a position
	// built with the newly-loaded default network. GUIs always send a
	// `position` command before the next `go`.
	*position = engine.StartingPosition()
}
