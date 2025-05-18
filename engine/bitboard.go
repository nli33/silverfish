package engine

type Bitboard uint64

// occupied squares
// mask: possibly relevant blockers

type MagicEntry struct {
	// ((occupied & mask) * Magic) >> (64 - Index_bits) = index
	Mask       Bitboard
	Magic      uint64
	Index_bits uint8
}

var RookMagics [64]MagicEntry
var BishopMagics [64]MagicEntry
var RookMoves [64][]Bitboard
var BishopMoves [64][]Bitboard

// RookMoves: {64 : [magicindex : attackset]}

func MagicIndex(entry MagicEntry, occupied Bitboard) uint64 {
	return (uint64(occupied&entry.Mask) * entry.Magic) >> (64 - entry.Index_bits)
}

func GenRookMoves(square Square, occupied Bitboard) Bitboard {
	magic := RookMagics[square]
	moves := RookMoves[square]
	return moves[MagicIndex(magic, occupied)]
}

func FindMagic(piece uint8, square Square, indexBits uint8) {

}

func SlidingAttack(piece uint8, square Square) Bitboard {
	var attacks []int8
	if piece == Bishop {
		attacks = []int8{NorthEast, SouthEast, NorthWest, SouthWest}
	} else if piece == Rook {
		attacks = []int8{North, South, East, West}
	} else {
		return Bitboard(0)
	}
	bitboard := Bitboard(0)
	var edgeMask Bitboard

	for _, d := range attacks {
		switch d {
		case North:
			edgeMask = BB_Rank8
		case South:
			edgeMask = BB_Rank1
		case East:
			edgeMask = BB_FileH
		case West:
			edgeMask = BB_FileA
		default:
			edgeMask = BB_Edges
		}

		sq := square
		for {
			next := Square(int8(sq) + d)
			if !IsValid(next) || Distance(next, sq) > 1 {
				break
			}
			// if next is the “true” edge in this direction, stop *before* adding:
			if (Bitboard(1)<<next)&edgeMask != 0 {
				break
			}
			sq = next
			bitboard |= (1 << sq)
		}
	}
	bitboard &^= Bitboard(1 << square)
	return bitboard
}
