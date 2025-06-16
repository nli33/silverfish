package engine

import (
	"bufio"
	"fmt"
	"strings"
)

const (
	UciInitialState = iota
	UciIdleState
	UciActiveState
	UciPingState
	UciHaltState
	UciSyncState
)

var uci_state = UciInitialState

type UciErrorType struct {
	err string
}

func NewUciError(err string) *UciErrorType {
	return &UciErrorType{
		err: err,
	}
}

func (err *UciErrorType) Error() string {
	return err.err
}

func UciHandleMessages(stdin bufio.Scanner, pos *Position, should_continue *bool) {
	result := stdin.Scan()
	if !result {
		UciError("I/O error or some shit.")
		return
	}

	message := stdin.Text()

	switch uci_state {
	case UciIdleState:
		if strings.HasPrefix(message, "position") {
			position := strings.SplitN(message, " ", 2)
			if position[1] == "fen" && len(position) > 2 {
				*pos = FromFEN(position[2])
			} else if position[1] == "startpos" {
				*pos = NewPosition()
			}
		} else if strings.HasPrefix(message, "go") {
			uci_state = UciActiveState
		} else if message == "isready" {
			uci_state = UciSyncState
		} else if message == "quit" {
			*should_continue = false
		} else {
			UciInfo("Not implemented")
		}

		break
	case UciActiveState:
		switch message {
		case "isready":
			uci_state = UciPingState
			break
		case "stop":
			uci_state = UciHaltState
			break
		default:
			UciError(fmt.Sprintf("Sir, it is currently the %d state. You can't do %s", uci_state, message))
		}

		break
	default:
		UciError(fmt.Sprintf("Sir, it is currently the %d state. You can't do %s", uci_state, message))
	}
}

func UciOk() error {
	if uci_state != UciInitialState {
		return NewUciError("This can only be called in the initial state.")
	}

	uci_state = UciIdleState
	fmt.Print("uciok\n")
	return nil
}

func UciReadyOk() {
	switch uci_state {
	case UciSyncState:
		fmt.Println("readyok")
		uci_state = UciIdleState
		break
	case UciPingState:
		fmt.Println("readyok")
		uci_state = UciActiveState
		break
	default:
		_ = fmt.Errorf("i think this is a bug...")
	}
}

func UciBestMove(move Move) error {
	if uci_state != UciActiveState && uci_state != UciHaltState && uci_state != UciPingState {
		return NewUciError("Cannot call this in states other than Active, Halt, or Ping")
	}

	fmt.Printf("bestmove %s\n", move.ToString())
	uci_state = UciIdleState
	return nil
}

func UciInfo(message string) {
	fmt.Printf("info string %s\n", message)
}

func UciError(message string) {
	fmt.Printf("info error %s\n", message)
}

func UciSetAuthor(name string) {
	fmt.Printf("id author %s\n", name)
}

func UciSetEngineName(name string) {
	fmt.Printf("id name %s\n", name)
}

// Normally, one should just use protocol 2, as that is the protocol that I am
// implementing.
func UciSetProtocol(protocol uint8) {
	fmt.Printf("protocol %d\n", protocol)
}
