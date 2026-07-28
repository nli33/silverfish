package engine

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

type UciGoMessage struct {
	// When true, the engine should search infinitely
	Infinite bool

	// When true, the engine should perform perft
	Perft bool

	// For traditional α/β engines, the maximum length in ply
	// of the principal variation (before extensions and reductions have been
	// applied, and not including plies examined in a quiescing search) that
	// the engine should explore
	Depth    int16
	Movetime int32

	WTime int32
	BTime int32
	WInc  int32
	BInc  int32
}

// I'm too lazy to copy and paste documentation for every single
// field, so just read the information on Page 9 and you should be up to speed
type UciInfoMessage struct {
	depth             int
	hasDepth          bool
	nodes             int
	hasNodes          bool
	currmove          Move
	hasCurrmove       bool
	currmovenumber    int
	hasCurrMoveNumber bool
	score             int32
	hasScore          bool
	isMate            bool
}

const (
	UciEmptyClientMessage uint8 = iota
	UciPositionClientMessage
	UciUciClientMessage
	UciGoClientMessage
	UciIsReadyClientMessage
	UciQuitClientMessage
	UciStopClientMessage
	UciSetOptionClientMessage
	UciNewGameClientMessage
)

// EvalFileDefaultLabel is the sentinel value UCI GUIs are expected to send
// back (or that engines print as the `default`) to mean "use the network
// built into the binary", matching the convention used by Stockfish's
// EvalFile option.
const EvalFileDefaultLabel = "<empty>"

type UciSetOptionMessage struct {
	Name  string
	Value string
}

// uciProcessSetOptionMessage parses the remainder of a "setoption name <name>
// [value <value>]" command (with "setoption " already trimmed). Splits on
// the literal " value " rather than tokenizing on whitespace, since both the
// option name and its value (e.g. a file path) may contain spaces.
func uciProcessSetOptionMessage(message string) UciSetOptionMessage {
	message = strings.TrimPrefix(message, "name ")

	if idx := strings.Index(message, " value "); idx != -1 {
		return UciSetOptionMessage{
			Name:  strings.TrimSpace(message[:idx]),
			Value: message[idx+len(" value "):],
		}
	}
	return UciSetOptionMessage{Name: strings.TrimSpace(message)}
}

type UciClientMessage struct {
	Position    *Position
	GoMessage   *UciGoMessage
	SetOption   *UciSetOptionMessage
	MessageType uint8
}

func uciProcessGoMessage(message string) UciGoMessage {
	result := UciGoMessage{}

	tokens := strings.Split(message, " ")

	// intArg returns the integer following tokens[i], or ok=false if it's
	// missing (i is the last token) or not a valid integer.
	intArg := func(i int) (int, bool) {
		if i+1 >= len(tokens) {
			UciError(fmt.Sprintf("missing argument for %q", tokens[i]))
			return 0, false
		}
		value, err := strconv.Atoi(tokens[i+1])
		if err != nil {
			UciError(fmt.Sprintf("invalid argument for %q: %q", tokens[i], tokens[i+1]))
			return 0, false
		}
		return value, true
	}

	for i, token := range tokens {
		switch token {
		case "infinite":
			result.Infinite = true
		case "perft":
			result.Perft = true
		case "depth":
			if depth, ok := intArg(i); ok {
				result.Depth = int16(depth)
			}
		case "movetime":
			if movetime, ok := intArg(i); ok {
				result.Movetime = int32(movetime)
			}
		case "wtime":
			if wtime, ok := intArg(i); ok {
				result.WTime = int32(wtime)
			}
		case "btime":
			if btime, ok := intArg(i); ok {
				result.BTime = int32(btime)
			}
		case "winc":
			if winc, ok := intArg(i); ok {
				result.WInc = int32(winc)
			}
		case "binc":
			if binc, ok := intArg(i); ok {
				result.BInc = int32(binc)
			}
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
		var position Position

		if strings.HasPrefix(initial, "fen ") {
			position = FromFEN(strings.TrimPrefix(initial, "fen "))
		} else if initial == "startpos" {
			position = StartingPosition()
		}

		if len(parts) > 1 {
			moves := strings.Split(strings.TrimSpace(parts[1]), " ")

			for _, move := range moves {
				givenMove := NewMoveFromStr(move)
				isLegal := false

				// choose the Move from the list of legal moves, to ensure any required flags are set
				for _, legalMove := range position.LegalMoves() {
					if legalMove.To() == givenMove.To() && legalMove.From() == givenMove.From() && legalMove.Promotion() == givenMove.Promotion() {
						position.DoMove(legalMove)
						isLegal = true
					}
				}

				if !isLegal {
					break
				}
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
	} else if strings.HasPrefix(textMessage, "setoption") {
		opt := uciProcessSetOptionMessage(strings.TrimPrefix(textMessage, "setoption "))
		message.SetOption = &opt
		message.MessageType = UciSetOptionClientMessage
		return message
	} else if textMessage == "ucinewgame" {
		message.MessageType = UciNewGameClientMessage
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

	if info.hasDepth {
		message += fmt.Sprintf(" depth %d", info.depth)
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

// UciOptions prints the engine's supported `option` lines. Should be sent
// after `id`/before `uciok`, per the UCI spec.
func UciOptions() {
	fmt.Printf("option name EvalFile type string default %s\n", EvalFileDefaultLabel)
}
