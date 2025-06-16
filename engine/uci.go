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

func UciOk() {
	fmt.Print("uciok")
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


