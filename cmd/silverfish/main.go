package main

import (
	"bufio"
	"os"
	"silverfish/engine"
)

func HandleMessages(channel chan engine.UciClientMessage) {
	stdinScanner := bufio.NewScanner(os.Stdin)

	for {
		message := engine.UciProcessClientMessage(stdinScanner)
		channel <- message

		quit_message := message.MessageType == engine.UciQuitClientMessage
		engine_idle := engine.UciState == engine.UciIdleState
		// Basically, stop listening for more messages after a valid quit message
		// is received.
		if quit_message && engine_idle {
			return
		}
	}
}

func main() {
	engine.Init()
	engine.InitBitboard()

	messageChannel := make(chan engine.UciClientMessage, 5)

	go HandleMessages(messageChannel)

	mainloop: for {
		message := engine.UciClientMessage{}
		select {
		case message = <-messageChannel:
			break
		default:
			continue
		}

		switch message.MessageType {
		case engine.UciUciClientMessage:
			engine.UciSetEngineName("Silverfish 0.0.69a")
			engine.UciSetAuthor("李能和赵梁越")
			engine.UciSetProtocol(2)

			engine.UciOk()
		case engine.UciIsReadyClientMessage:
			engine.UciReadyOk()
		case engine.UciQuitClientMessage:
			break mainloop
		}
	}
}
