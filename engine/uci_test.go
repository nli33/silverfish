package engine_test

import (
	"bufio"
	"strings"
	"testing"

	"silverfish/engine"
)

func TestUciProcessClientMessageParsesNewGame(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader("ucinewgame\n"))
	message := engine.UciProcessClientMessage(scanner)
	if message.MessageType != engine.UciNewGameClientMessage {
		t.Errorf("got MessageType %d, want UciNewGameClientMessage", message.MessageType)
	}
}
