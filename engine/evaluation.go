package engine

import (
	"math"
	"math/bits"
)

const Infinity int32 = 1000000

var MaterialValues = map[uint8]int32{
	Pawn:   100,
	Knight: 320,
	Bishop: 330,
	Rook:   500,
	Queen:  900,
	King:   Infinity,
}

var PieceSqMidgame = [6][64]int32{
	{ // pawn
		0, 0, 0, 0, 0, 0, 0, 0,
		50, 50, 50, 50, 50, 50, 50, 50,
		10, 10, 20, 30, 30, 20, 10, 10,
		5, 5, 10, 25, 25, 10, 5, 5,
		0, 0, 0, 20, 20, 0, 0, 0,
		5, -5, -10, 0, 0, -10, -5, 5,
		5, 10, 10, -20, -20, 10, 10, 5,
		0, 0, 0, 0, 0, 0, 0, 0,
	},
	{ // knight
		-50, -40, -30, -30, -30, -30, -40, -50,
		-40, -20, 0, 0, 0, 0, -20, -40,
		-30, 0, 10, 15, 15, 10, 0, -30,
		-30, 5, 15, 20, 20, 15, 5, -30,
		-30, 0, 15, 20, 20, 15, 0, -30,
		-30, 5, 10, 15, 15, 10, 5, -30,
		-40, -20, 0, 5, 5, 0, -20, -40,
		-50, -40, -30, -30, -30, -30, -40, -50,
	},
	{ // bishop
		-20, -10, -10, -10, -10, -10, -10, -20,
		-10, 0, 0, 0, 0, 0, 0, -10,
		-10, 0, 5, 10, 10, 5, 0, -10,
		-10, 5, 5, 10, 10, 5, 5, -10,
		-10, 0, 10, 10, 10, 10, 0, -10,
		-10, 10, 10, 10, 10, 10, 10, -10,
		-10, 5, 0, 0, 0, 0, 5, -10,
		-20, -10, -10, -10, -10, -10, -10, -20,
	},
	{ // rook
		0, 0, 0, 0, 0, 0, 0, 0,
		5, 10, 10, 10, 10, 10, 10, 5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		0, 0, 0, 5, 5, 0, 0, 0,
	},
	{ // queen
		-20, -10, -10, -5, -5, -10, -10, -20,
		-10, 0, 0, 0, 0, 0, 0, -10,
		-10, 0, 5, 5, 5, 5, 0, -10,
		-5, 0, 5, 5, 5, 5, 0, -5,
		0, 0, 5, 5, 5, 5, 0, -5,
		-10, 5, 5, 5, 5, 5, 0, -10,
		-10, 0, 5, 0, 0, 0, 0, -10,
		-20, -10, -10, -5, -5, -10, -10, -20,
	},
	{ // king
		-30, -40, -40, -50, -50, -40, -40, -30,
		-30, -40, -40, -50, -50, -40, -40, -30,
		-30, -40, -40, -50, -50, -40, -40, -30,
		-30, -40, -40, -50, -50, -40, -40, -30,
		-20, -30, -30, -40, -40, -30, -30, -20,
		-10, -20, -20, -20, -20, -20, -20, -10,
		20, 20, 0, 0, 0, 0, 20, 20,
		20, 30, 10, 0, 0, 10, 30, 20,
	},
}

var PieceSqEndgame = [6][64]int32{
	{ // pawn
		0, 0, 0, 0, 0, 0, 0, 0,
		50, 50, 50, 50, 50, 50, 50, 50,
		10, 10, 20, 30, 30, 20, 10, 10,
		5, 5, 10, 25, 25, 10, 5, 5,
		0, 0, 0, 20, 20, 0, 0, 0,
		5, -5, -10, 0, 0, -10, -5, 5,
		5, 10, 10, -20, -20, 10, 10, 5,
		0, 0, 0, 0, 0, 0, 0, 0,
	},
	{ // knight
		-50, -40, -30, -30, -30, -30, -40, -50,
		-40, -20, 0, 0, 0, 0, -20, -40,
		-30, 0, 10, 15, 15, 10, 0, -30,
		-30, 5, 15, 20, 20, 15, 5, -30,
		-30, 0, 15, 20, 20, 15, 0, -30,
		-30, 5, 10, 15, 15, 10, 5, -30,
		-40, -20, 0, 5, 5, 0, -20, -40,
		-50, -40, -30, -30, -30, -30, -40, -50,
	},
	{ // bishop
		-20, -10, -10, -10, -10, -10, -10, -20,
		-10, 0, 0, 0, 0, 0, 0, -10,
		-10, 0, 5, 10, 10, 5, 0, -10,
		-10, 5, 5, 10, 10, 5, 5, -10,
		-10, 0, 10, 10, 10, 10, 0, -10,
		-10, 10, 10, 10, 10, 10, 10, -10,
		-10, 5, 0, 0, 0, 0, 5, -10,
		-20, -10, -10, -10, -10, -10, -10, -20,
	},
	{ // rook
		0, 0, 0, 0, 0, 0, 0, 0,
		5, 10, 10, 10, 10, 10, 10, 5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		0, 0, 0, 5, 5, 0, 0, 0,
	},
	{ // queen
		-20, -10, -10, -5, -5, -10, -10, -20,
		-10, 0, 0, 0, 0, 0, 0, -10,
		-10, 0, 5, 5, 5, 5, 0, -10,
		-5, 0, 5, 5, 5, 5, 0, -5,
		0, 0, 5, 5, 5, 5, 0, -5,
		-10, 5, 5, 5, 5, 5, 0, -10,
		-10, 0, 5, 0, 0, 0, 0, -10,
		-20, -10, -10, -5, -5, -10, -10, -20,
	},
	{ // king
		-50, -40, -30, -20, -20, -30, -40, -50,
		-30, -20, -10, 0, 0, -10, -20, -30,
		-30, -10, 20, 30, 30, 20, -10, -30,
		-30, -10, 30, 40, 40, 30, -10, -30,
		-30, -10, 30, 40, 40, 30, -10, -30,
		-30, -10, 20, 30, 30, 20, -10, -30,
		-30, -30, 0, 0, 0, 0, -30, -30,
		-50, -30, -30, -30, -30, -30, -30, -50,
	},
}

func (pos *Position) Material(color uint8) int32 {
	pawnMaterial := int32(bits.OnesCount64(uint64(pos.Pieces[color][Pawn]))) * MaterialValues[Pawn]
	return pos.EndgameMaterial(color) + pawnMaterial
}

func (pos *Position) EndgameMaterial(color uint8) int32 {
	var ans int32
	for piece := Knight; piece <= Queen; piece++ {
		ans += int32(bits.OnesCount64(uint64(pos.Pieces[color][piece]))) * MaterialValues[piece]
	}
	return ans
}

func (pos *Position) KingSafety(color uint8) int32 {
	safetyScore := int32(1000)

	// Obviously, if the King is in check, it is not going to be safe.
	checkers := pos.Checkers(color)
	checkerCount := int32(bits.OnesCount64(uint64(checkers)))
	safetyScore -= checkerCount * 100

	kingSquare := pos.GetKingSquare(color)
	kingRank := RankOf(kingSquare)
	kingFile := FileOf(kingSquare)

	// Enemy pieces being close to the King might be a bit of an issue.
	// TODO: Have a more optimal way of doing this later. Probably by using the bitboards and stuff
	for square := range NoSquare {
		colorOfPiece, piece := pos.GetSquare(square)

		pieceRank := RankOf(square)
		pieceFile := FileOf(square)

		rankDiff := Abs(int(pieceRank - kingRank))
		fileDiff := Abs(int(pieceFile - kingFile))

		distance := math.Sqrt(float64(rankDiff*rankDiff + fileDiff*fileDiff))
		distanceWeight := 10.0 - distance // the closer it is the higher the weight

		if colorOfPiece != color {
			switch piece {
			case Queen:
				safetyScore -= int32(20 * distanceWeight)
			case Rook:
				safetyScore -= int32(10 * distanceWeight)
			case Bishop:
				safetyScore -= int32(5 * distanceWeight)
			case Knight:
				safetyScore -= int32(5 * distanceWeight)
			case Pawn:
				// Not as small as a pawn storm could be dangerous,
				safetyScore -= int32(3 * distanceWeight)
			}
		}
	}

	return safetyScore
}

func Evaluate(pos *Position) int32 {
	us := pos.Turn
	them := pos.Turn ^ 1

	ourMaterial := pos.Material(us)
	theirMaterial := pos.Material(them)
	ourEGMaterial := pos.EndgameMaterial(us)
	theirEGMaterial := pos.EndgameMaterial(them)

	ourKingSafety := pos.KingSafety(us)
	theirKingSafety := pos.KingSafety(them)

	isEndgame := (ourEGMaterial + theirEGMaterial) <= 1400

	eval := ourMaterial + ourKingSafety - theirMaterial - theirKingSafety

	for piece := Pawn; piece <= King; piece++ {
		bb := pos.Pieces[us][piece]
		for bb != 0 {
			sq := PopLsb(&bb)

			if us == White {
				sq = FlipSq[sq]
			}

			if isEndgame {
				eval += PieceSqEndgame[piece][sq]
			} else {
				eval += PieceSqMidgame[piece][sq]
			}
		}
	}

	for piece := Pawn; piece <= King; piece++ {
		bb := pos.Pieces[them][piece]
		for bb != 0 {
			sq := PopLsb(&bb)

			if them == White {
				sq = FlipSq[sq]
			}

			if isEndgame {
				eval -= PieceSqEndgame[piece][sq]
			} else {
				eval -= PieceSqMidgame[piece][sq]
			}
		}
	}

	return eval
}
