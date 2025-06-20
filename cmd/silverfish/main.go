package main

import (
	"bufio"
	"fmt"
	"os"
	"silverfish/engine"
)

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

	bestMove := engine.NewMove(0, 0)

	if command.Infinite {
		// We set a limited depth to avoid stack overflow (as we are using recursion
		// to implement the search right now)
		_, bestMove = engine.AlphaBeta(*position, 100)
	} else {
		_, bestMove = engine.AlphaBeta(*position, int(command.Depth))
	}

	engine.UciBestMove(bestMove)

	channel <- true
	return
}

func main() {
	engine.Init()
	engine.InitBitboard()

	messageChannel := make(chan engine.UciClientMessage, 5)
	// Used for reporting if an action is done.
	actionAlertChannel := make(chan bool)
	active := false

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

		if active {
			select {
			case <-actionAlertChannel:
				active = false
			default:
				// Do nothing :ye:
			}
		} else {
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
}
