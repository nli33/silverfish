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

const (
	Pawn uint8 = iota
	Knight
	Bishop
	Rook
	Queen
	King
	NoPiece
)

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

var BishopAttacks = []int8{NorthEast, SouthWest, NorthWest, SouthWest}
var RookAttacks = []int8{North, South, East, West}

var Default = [2][6]Bitboard{
	{
		Bitboard(0b00000000_00000000_00000000_00000000_00000000_00000000_11111111_00000000),
		Bitboard(0b00000000_00000000_00000000_00000000_00000000_00000000_00000000_01000010),
		Bitboard(0b00000000_00000000_00000000_00000000_00000000_00000000_00000000_00100100),
		Bitboard(0b00000000_00000000_00000000_00000000_00000000_00000000_00000000_10000001),
		Bitboard(0b00000000_00000000_00000000_00000000_00000000_00000000_00000000_00001000),
		Bitboard(0b00000000_00000000_00000000_00000000_00000000_00000000_00000000_00010000),
	},
	{
		Bitboard(0b00000000_11111111_00000000_00000000_00000000_00000000_00000000_00000000),
		Bitboard(0b01000010_00000000_00000000_00000000_00000000_00000000_00000000_00000000),
		Bitboard(0b00100100_00000000_00000000_00000000_00000000_00000000_00000000_00000000),
		Bitboard(0b10000001_00000000_00000000_00000000_00000000_00000000_00000000_00000000),
		Bitboard(0b00001000_00000000_00000000_00000000_00000000_00000000_00000000_00000000),
		Bitboard(0b00010000_00000000_00000000_00000000_00000000_00000000_00000000_00000000),
	},
}

const BB_Edges = Bitboard(0b11111111_10000001_10000001_10000001_10000001_10000001_10000001_11111111)
const BB_Rank1 = Bitboard(0b00000000_00000000_00000000_00000000_00000000_00000000_00000000_11111111)
const BB_Rank8 = Bitboard(0b11111111_00000000_00000000_00000000_00000000_00000000_00000000_00000000)
const BB_FileA = Bitboard(0b00000001_00000001_00000001_00000001_00000001_00000001_00000001_00000001)
const BB_FileH = Bitboard(0b10000000_10000000_10000000_10000000_10000000_10000000_10000000_10000000)

// rows
func RankOf(square Square) uint8 {
	return uint8(square >> 3)
}

// columns
func FileOf(square Square) uint8 {
	return uint8(square & 7)
}

func NewSquare(rank uint8, file uint8) Square {
	return Square((rank << 3) + file)
}

func IsValid(square Square) bool {
	return SquareA1 <= square && square <= SquareH8
}

func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func Distance(x Square, y Square) int {
	return max(
		Abs(int(RankOf(x))-int(RankOf(y))),
		Abs(int(FileOf(x))-int(FileOf(y))),
	)
}

func PrintBB(bb Bitboard) string {
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
	fmt.Println(bbString)
	return bbString
}
