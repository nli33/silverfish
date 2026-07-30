package engine

import (
	"fmt"
	"strconv"
	"strings"
)

func (pos *Position) ToFEN() string {
	fen := ""
	spaceCounter := 0
	for rank := Rank8; Rank8 >= rank && rank >= Rank1; rank-- {
		for file := FileA; file <= FileH; file++ {
			sq := NewSquare(rank, file)
			color, piece := pos.GetSquare(sq)

			if piece == NoPiece {
				spaceCounter++
				continue
			}
			if spaceCounter > 0 {
				fen += fmt.Sprint(spaceCounter)
				spaceCounter = 0
			}

			char := PieceToChar[piece]
			if color == White { // capitalize
				char -= 32
			}
			fen += string(char)
		}
		if spaceCounter > 0 {
			fen += fmt.Sprint(spaceCounter)
			spaceCounter = 0
		}
		if rank != Rank1 {
			fen += "/"
		}
	}

	if pos.Turn == White {
		fen += " w "
	} else {
		fen += " b "
	}

	// only checking castling rights, not blockers
	if pos.CastlingRights&WhiteKingside > 0 {
		fen += "K"
	}
	if pos.CastlingRights&WhiteQueenside > 0 {
		fen += "Q"
	}
	if pos.CastlingRights&BlackKingside > 0 {
		fen += "k"
	}
	if pos.CastlingRights&BlackQueenside > 0 {
		fen += "q"
	}
	if pos.CastlingRights == 0 {
		fen += "-"
	}

	fen += " "
	if pos.EnPassantSquare != NoSquare &&
		(RankOf(pos.EnPassantSquare) == Rank3 ||
			RankOf(pos.EnPassantSquare) == Rank6) {
		fen += pos.EnPassantSquare.ToString()
	} else {
		fen += "-"
	}

	fen += " " + fmt.Sprint(pos.Rule50)
	fen += " " + fmt.Sprint(pos.FullMoves())

	return fen
}

// note: full refresh automatically on creation
func FromFEN(fen string) Position {
	var pos Position

	if defaultNet == nil {
		panic("engine.Init() must be called before engine.FromFEN()")
	}
	pos.Net = defaultNet
	pos.Acc = NewAccumulator(pos.Net)
	// install input bias before any Add/Remove
	pos.Acc.Reset(pos.Net)

	parts := strings.Split(fen, " ")
	if len(parts) < 6 {
		panic("invalid FEN: not enough parts")
	}

	boardPart := parts[0]
	turnPart := parts[1]
	castlingPart := parts[2]
	enPassantPart := parts[3]
	rule50Part := parts[4]
	fullmovePart := parts[5]

	rank := Rank8
	file := FileA

	for _, char := range boardPart {
		if char == '/' {
			rank--
			file = FileA
		} else if char >= '1' && char <= '8' {
			spaces := uint8(char - '0')
			sq := NewSquare(rank, file)
			for i := uint8(0); i < spaces; i++ {
				//pos.RemovePiece(sq)
				pos.Board[sq] = NoPiece
				sq++
			}
			file += spaces
		} else {
			var color uint8
			if char >= 'A' && char <= 'Z' {
				color = White
				char += 32 // make lowercase
			} else {
				color = Black
			}

			piece, exists := CharToPiece[byte(char)]

			if !exists {
				panic("invalid piece character: " + string(char))
			}

			sq := NewSquare(rank, file)
			pos.PutPiece(sq, piece, color)
			file++
		}
	}

	switch turnPart {
	case "w":
		pos.Turn = White
	case "b":
		pos.Turn = Black
	default:
		panic("invalid turn field")
	}

	pos.CastlingRights = 0
	if castlingPart != "-" {
		for _, c := range castlingPart {
			switch c {
			case 'K':
				pos.CastlingRights |= WhiteKingside
			case 'Q':
				pos.CastlingRights |= WhiteQueenside
			case 'k':
				pos.CastlingRights |= BlackKingside
			case 'q':
				pos.CastlingRights |= BlackQueenside
			default:
				panic("invalid castling character")
			}
		}
	}

	if enPassantPart == "-" {
		pos.EnPassantSquare = NoSquare
	} else {
		pos.EnPassantSquare = NewSquareFromStr(enPassantPart)
	}

	rule50, err := strconv.Atoi(rule50Part)
	if err != nil {
		panic("invalid rule50 field")
	}
	pos.Rule50 = uint8(rule50)

	fullmove, err := strconv.Atoi(fullmovePart)
	if err != nil {
		panic("invalid fullmove field")
	}
	pos.Ply = uint16((fullmove-1)*2 + int(pos.Turn))
	pos.Hash = Hash(&pos)

	return pos
}
