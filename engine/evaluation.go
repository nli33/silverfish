package engine

import "math/bits"

const Infinity int32 = 1000000

var MaterialValues = map[uint8]int32{
	Pawn:   100,
	Knight: 300,
	Bishop: 300,
	Rook:   500,
	Queen:  900,
	King:   Infinity,
}

func (pos *Position) Material(color uint8) int32 {
	var ans int32
	for piece := Pawn; piece <= Queen; piece++ {
		ans += int32(bits.OnesCount64(uint64(pos.Pieces[color][piece]))) * MaterialValues[piece]
	}
	return ans
}

func Evaluate(pos *Position) int32 {
	return pos.Material(White) - pos.Material(Black)
}
