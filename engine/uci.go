package engine

import (
	"bufio"
	"fmt"
	"strconv"
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

type UciGoMessage struct {
	// When true, the engine should search infinitely
	Infinite bool

	// When true, the engine should perform perft
	Perft bool

	// A collection of moves to which the engine should restrict its
	// consideration (in other words, the move reported with the bestmove
	// message should be one of the moves in this collection),
	SearchMoves []Move

	// Remember - 0 indicates that it was not specified.
	// Hopefully this doesn't bite us in the ass

	// An indication that the engine should attempt to prove
	// a mate in this many full moves (or twice this many plies) and may
	// assume that it does not need to examine lines beyond this many full
	// moves (or twice this many plies)
	Mate int16

	// Time limits (read spec for information. The one we are referencing
	// has information about this on Page 14.)
	WhiteTime          int16
	BlackTime          int16
	WhiteClockIncrease int16
	BlackClockIncrease int16
	MovesToGo          int16

	// For traditional α/β engines, the maximum length in ply
	// of the principal variation (before extensions and reductions have been
	// applied, and not including plies examined in a quiescing search) that
	// the engine should explore
	Depth int16

	// For traditional engines, the maximum number of positions (counted with
	// multiplicity) that the engine should examine,
	Nodes int16
}

// I'm too lazy to copy and paste documentation for every single 
// field, so just read the information on Page 9 and you should be up to speed
type UciInfoMessage struct {
	nodes int
	hasNodes bool
	currmove Move
	hasCurrmove bool
	currmovenumber int
	hasCurrMoveNumber bool
	score int32
	hasScore bool
	isMate bool
}

const (
	UciEmptyClientMessage uint8 = iota
	UciPositionClientMessage
	UciUciClientMessage
	UciGoClientMessage
	UciIsReadyClientMessage
	UciQuitClientMessage
	UciStopClientMessage
)

type UciClientMessage struct {
	Position    *Position
	GoMessage   *UciGoMessage
	MessageType uint8
}

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

func uciProcessGoMessage(message string) UciGoMessage {
	result := UciGoMessage{}

	tokens := strings.Split(message, " ")

	for i, token := range tokens {
		switch token {
		case "infinite":
			result.Infinite = true
		case "perft":
			result.Perft = true
		case "depth":
			depth, err := strconv.Atoi(tokens[i+1])
			if err != nil {
				UciError("something unknown")
			}

			result.Depth = int16(depth)
		}
	}

	return result
}

func UciProcessClientMessage(stdin *bufio.Scanner) UciClientMessage {
	message := UciClientMessage{}

	result := stdin.Scan()
	if !result {
		UciError("I/O error or something.")
		return message
	}

	textMessage := stdin.Text()

	if strings.HasPrefix(textMessage, "position") {
		parts := strings.Split(strings.TrimPrefix(textMessage, "position "), "moves")
		initial := strings.TrimSpace(parts[0])
		position := NewPosition()

		if strings.HasPrefix(initial, "fen ") {
			position = FromFEN(strings.TrimPrefix(initial, "fen "))
		} else if initial == "startpos" {
			position = StartingPosition()
		}

		if len(parts) > 1 {
			moves := strings.Split(strings.TrimSpace(parts[1]), " ")

			for _, move := range moves {
				position.DoMove(NewMoveFromStr(move))
			}
		}

		message.Position = &position
		message.MessageType = UciPositionClientMessage
		return message
	} else if strings.HasPrefix(textMessage, "go") {
		message.MessageType = UciGoClientMessage
		goMessage := uciProcessGoMessage(strings.TrimPrefix(textMessage, "go "))
		message.GoMessage = &goMessage
		return message
	} else if textMessage == "isready" {
		message.MessageType = UciIsReadyClientMessage
		return message
	} else if textMessage == "quit" {
		message.MessageType = UciQuitClientMessage
		return message
	} else if textMessage == "uci" {
		message.MessageType = UciUciClientMessage
		return message
	} else if textMessage == "stop" {
		message.MessageType = UciStopClientMessage
		return message
	}

	// Just return the empty message at this point
	return message
}

func UciOk() {
	fmt.Println("uciok")
}

func UciReadyOk() {
	fmt.Println("readyok")
}

func UciBestMove(move Move) {
	fmt.Printf("bestmove %s\n", move.ToString())
}

func UciInfo(info UciInfoMessage) {
	message := "info"

	if info.hasNodes {
		message += fmt.Sprintf(" nodes %d", info.nodes)
	}

	if info.hasCurrmove {
		message += fmt.Sprintf(" currmove %s", info.currmove.ToString())
	}

	if info.hasCurrMoveNumber {
		message += fmt.Sprintf(" currmovenumber %d", info.currmovenumber)
	}

	if info.hasScore && !info.isMate {
		message += fmt.Sprintf(" score cp %d", info.score)
	}

	if info.hasScore && info.isMate {
		message += fmt.Sprintf(" score mate %d", info.score)
	}

	fmt.Println(message)
}

func UciLog(message string) {
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
