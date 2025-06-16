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

var UciState = UciInitialState

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

	switch UciState {
	case UciIdleState:
		if strings.HasPrefix(message, "position") {
			position := strings.SplitN(message, " ", 2)
			if position[1] == "fen" && len(position) > 2 {
				*pos = FromFEN(position[2])
			} else if position[1] == "startpos" {
				*pos = StartingPosition()
			}
		} else if strings.HasPrefix(message, "go") {
			UciState = UciActiveState
		} else if message == "isready" {
			UciState = UciSyncState
		} else if message == "quit" {
			*should_continue = false
		} else {
			UciInfo("Not implemented")
		}

		break
	case UciActiveState:
		switch message {
		case "isready":
			UciState = UciPingState
			break
		case "stop":
			UciState = UciHaltState
			break
		default:
			UciError(fmt.Sprintf("Sir, it is currently the %d state. You can't do %s", UciState, message))
		}

		break
	default:
		UciError(fmt.Sprintf("Sir, it is currently the %d state. You can't do %s", UciState, message))
	}
}

func UciOk() error {
	if UciState != UciInitialState {
		return NewUciError("This can only be called in the initial state.")
	}

	UciState = UciIdleState
	fmt.Print("uciok\n")
	return nil
}

func UciReadyOk() {
	switch UciState {
	case UciSyncState:
		fmt.Println("readyok")
		UciState = UciIdleState
		break
	case UciPingState:
		fmt.Println("readyok")
		UciState = UciActiveState
		break
	default:
		_ = fmt.Errorf("i think this is a bug...")
	}
}

func UciBestMove(move Move) error {
	if UciState != UciActiveState && UciState != UciHaltState && UciState != UciPingState {
		return NewUciError("Cannot call this in states other than Active, Halt, or Ping")
	}

	fmt.Printf("bestmove %s\n", move.ToString())
	UciState = UciIdleState
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
