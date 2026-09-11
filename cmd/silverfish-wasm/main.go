//go:build js && wasm

// Command silverfish-wasm runs the engine's UCI loop in a browser/WebAssembly
// host instead of over stdin/stdout. It mirrors cmd/silverfish/main.go's
// mainloop, but the transport is two JS-facing functions instead of a
// terminal:
//
//   - global `silverfishSend(line)` feeds a UCI command line to the engine,
//     as if it had been written to stdin.
//   - global `silverfishOnOutput(callback)` registers a JS function that
//     receives each UCI response line the engine would otherwise print to
//     stdout.
package main

import (
	"bufio"
	"fmt"
	"io"
	"silverfish/engine"
	"strconv"
	"strings"
	"syscall/js"
	"time"
)

type jsWriter struct {
	callback js.Value
}

func (w *jsWriter) Write(p []byte) (int, error) {
	if w.callback.Truthy() {
		w.callback.Invoke(string(p))
	}
	return len(p), nil
}

func executeGoCommand(channel chan bool, position *engine.Position, command *engine.UciGoMessage) {
	if command.Perft && command.Depth != 0 {
		result := engine.Perft(position, int(command.Depth), true)
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
	_, bestMove = engine.SearchLazySMP(&search)

	channel <- true
	engine.UciBestMove(bestMove)
}

func handleSetOption(opt *engine.UciSetOptionMessage, position *engine.Position) {
	if opt == nil {
		return
	}

	if strings.EqualFold(opt.Name, "Threads") {
		// The wasm build is single-threaded (no OS threads under
		// GOOS=js), so Threads is accepted but pinned to 1.
		n, err := strconv.Atoi(opt.Value)
		if err != nil || n < 1 {
			engine.UciError(fmt.Sprintf("invalid Threads value %q", opt.Value))
		}
		engine.Threads = 1
		return
	}

	if !strings.EqualFold(opt.Name, "EvalFile") {
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

	*position = engine.StartingPosition()
}

func main() {
	engine.Threads = 1
	engine.Init()

	writer := &jsWriter{}
	engine.Output = writer

	pr, pw := io.Pipe()
	scanner := bufio.NewScanner(pr)

	js.Global().Set("silverfishSend", js.FuncOf(func(this js.Value, args []js.Value) any {
		line := args[0].String()
		go func() {
			io.WriteString(pw, line+"\n")
		}()
		return nil
	}))

	js.Global().Set("silverfishOnOutput", js.FuncOf(func(this js.Value, args []js.Value) any {
		writer.callback = args[0]
		return nil
	}))

	if onReady := js.Global().Get("silverfishReady"); onReady.Truthy() {
		onReady.Invoke()
	}

	messageChannel := make(chan engine.UciClientMessage, 5)
	actionAlertChannel := make(chan bool)
	active := false
	position := engine.StartingPosition()

	go func() {
		for {
			message := engine.UciProcessClientMessage(scanner)
			messageChannel <- message
			if message.MessageType == engine.UciQuitClientMessage {
				return
			}
		}
	}()

	for {
		select {
		case message := <-messageChannel:
			if active {
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
