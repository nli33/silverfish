package engine

import "fmt"

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

func UciOk() error {
	if uci_state != UciInitialState {
		return NewUciError("This can only be called in the initial state.")
	}

	uci_state = UciIdleState
	fmt.Print("uciok")
	return nil
}

func UciBestMove(move Move) error {
	if uci_state == UciActiveState || uci_state == UciHaltState || uci_state == UciPingState {
		return NewUciError("Cannot call this in states other than Active, Halt, or Ping")
	}

	fmt.Printf("bestmove %s", move.ToString())
	return nil
}

func UciInfo(message string) {
	fmt.Printf("info string %s", message)
}

func UciError(message string) {
	fmt.Printf("info error %s", message)
}

func UciSetAuthor(name string) {
	fmt.Printf("id author %s", name)
}

func UciSetEngineName(name string) {
	fmt.Printf("id name %s", name)
}

// Normally, one should just use protocol 2, as that is the protocol that I am
// implementing.
func UciSetProtocol(protocol uint8) {
	fmt.Printf("protocol %d", protocol)
}
