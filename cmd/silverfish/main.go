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
	// Currently, only Perft is implemented, as that was all that was demanded.
	if command.Perft && command.Depth != 0 {
		engine.UciInfo("Perft started.")
		result := engine.Perft(position, int(command.Depth))
		engine.UciInfo(fmt.Sprintf("Perft result: %d", result))
	}

	// Tell the main thread that we're done.
	channel <- true
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
			break
		default:
			continue
		}

		if active {
			select {
			case _ = <-actionAlertChannel:
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
