package engine

import (
	"fmt"
)

type Square uint8

const (
	SquareA1 Square = iota
	SquareB1
	SquareC1
	SquareD1
	SquareE1
	SquareF1
	SquareG1
	SquareH1

	SquareA2
	SquareB2
	SquareC2
	SquareD2
	SquareE2
	SquareF2
	SquareG2
	SquareH2

	SquareA3
	SquareB3
	SquareC3
	SquareD3
	SquareE3
	SquareF3
	SquareG3
	SquareH3

	SquareA4
	SquareB4
	SquareC4
	SquareD4
	SquareE4
	SquareF4
	SquareG4
	SquareH4

	SquareA5
	SquareB5
	SquareC5
	SquareD5
	SquareE5
	SquareF5
	SquareG5
	SquareH5

	SquareA6
	SquareB6
	SquareC6
	SquareD6
	SquareE6
	SquareF6
	SquareG6
	SquareH6

	SquareA7
	SquareB7
	SquareC7
	SquareD7
	SquareE7
	SquareF7
	SquareG7
	SquareH7

	SquareA8
	SquareB8
	SquareC8
	SquareD8
	SquareE8
	SquareF8
	SquareG8
	SquareH8

	NoSquare // 64
)

const (
	FileA uint8 = iota
	FileB
	FileC
	FileD
	FileE
	FileF
	FileG
	FileH
)

const (
	Rank1 uint8 = iota
	Rank2
	Rank3
	Rank4
	Rank5
	Rank6
	Rank7
	Rank8
)

const (
	White uint8 = iota
	Black
	NoColor
)

// for Position.Board, +10 for black pieces
const (
	Pawn uint8 = iota
	Knight
	Bishop
	Rook
	Queen
	King
	NoPiece
)

var CharToPiece = map[byte]uint8{
	'k': King,
	'q': Queen,
	'r': Rook,
	'b': Bishop,
	'n': Knight,
	'p': Pawn,
	'_': NoPiece,
}

var PieceToChar = map[uint8]byte{
	King:    'k',
	Queen:   'q',
	Rook:    'r',
	Bishop:  'b',
	Knight:  'n',
	Pawn:    'p',
	NoPiece: '_',
}

const (
	North     = 8
	South     = -8
	East      = 1
	West      = -1
	NorthEast = North + East
	SouthEast = South + East
	NorthWest = North + West
	SouthWest = South + West
)

var BishopDirections = []int8{NorthEast, SouthEast, NorthWest, SouthWest}
var RookDirections = []int8{North, South, East, West}

var Default = [2][6]Bitboard{
	{
		Bitboard(0xff00),
		Bitboard(0x42),
		Bitboard(0x24),
		Bitboard(0x81),
		Bitboard(0x8),
		Bitboard(0x10),
	},
	{
		Bitboard(0xff000000000000),
		Bitboard(0x4200000000000000),
		Bitboard(0x2400000000000000),
		Bitboard(0x8100000000000000),
		Bitboard(0x800000000000000),
		Bitboard(0x1000000000000000),
	},
}

const BB_Edges = Bitboard(0xff818181818181ff)
const BB_Rank1 = Bitboard(0xff)
const BB_Rank8 = Bitboard(0xff00000000000000)
const BB_FileA = Bitboard(0x101010101010101)
const BB_FileH = Bitboard(0x8080808080808080)
const BB_Empty = Bitboard(0)
const BB_Full = Bitboard(0xffffffffffffffff)

// rows
func RankOf(square Square) uint8 {
	return uint8(square >> 3)
}

// columns
func FileOf(square Square) uint8 {
	return uint8(square & 7)
}

func ColorOf(piece uint8) uint8 {
	switch {
	case piece == NoPiece:
		return NoColor
	case piece >= 10:
		return Black
	}
	return White
}

func NewSquare(rank uint8, file uint8) Square {
	return Square(rank<<3 + file)
}

func NewSquareFromStr(squareStr string) Square {
	file := uint8(squareStr[0] - 'a')
	rank := uint8(squareStr[1] - '1')
	return NewSquare(rank, file)
}

func IsValid(square Square) bool {
	return SquareA1 <= square && square <= SquareH8
}

func Abs(x int) int {
	if x < 0 {
		return -x
	} else {
		return x
	}
}

func PawnDisplacement(color uint8) int {
	if color == White {
		return 8
	} else {
		return -8
	}
}

func PawnStartingRank(color uint8) uint8 {
	if color == White {
		return Rank2
	} else {
		return Rank7
	}
}

func PawnPromotionRank(color uint8) uint8 {
	if color == White {
		return Rank7
	} else {
		return Rank2
	}
}

func Distance(x Square, y Square) int {
	return max(
		Abs(int(RankOf(x))-int(RankOf(y))),
		Abs(int(FileOf(x))-int(FileOf(y))),
	)
}

func (bb Bitboard) ToString() string {
	var bbString string
	for i := 7; i >= 0; i-- {
		for j := 0; j <= 7; j++ {
			if bb&(1<<(i*8+j)) == 0 {
				bbString += "0 "
			} else {
				bbString += "1 "
			}
		}
		bbString += "\n"
	}
	return bbString
}

func (bb Bitboard) ToStringSmall() string {
	return fmt.Sprintf("%064b", uint64(bb))
}

func (sq Square) ToString() string {
	fileStr := FileOf(sq) + 'a'
	rankStr := RankOf(sq) + '1'
	slice := append([]byte{fileStr}, rankStr)
	return string(slice)
}
