package engine

import (
	"errors"
	"math/bits"
)

type Bitboard uint64

// mask: possibly relevant blockers
type MagicEntry struct {
	// ((occupied & mask) * Magic) >> (64 - Index_bits) = index
	Mask      Bitboard
	Magic     uint64
	IndexBits uint8
}

// takes around 7 mb memory
var RookMoves [64][]Bitboard
var BishopMoves [64][]Bitboard
var KnightMoves [64]Bitboard
var KingMoves [64]Bitboard

func MagicIndex(entry MagicEntry, blockers Bitboard) uint64 {
	return (uint64(blockers&entry.Mask) * entry.Magic) >> (64 - entry.IndexBits)
}

func FindMagic(piece uint8, square Square) (MagicEntry, []Bitboard) {
	relevantMask := SliderBlockerMask(piece, square)
	indexBits := uint8(bits.OnesCount64(uint64(relevantMask)))
	for {
		magic := Rng.Uint64() & Rng.Uint64() & Rng.Uint64()
		entry := MagicEntry{relevantMask, magic, indexBits}
		table, err := makeMoveTable(piece, square, entry)
		if err == nil { // OK
			return entry, table
		}
	}
}

func makeMoveTable(piece uint8, square Square, entry MagicEntry) ([]Bitboard, error) {
	table := make([]Bitboard, 1<<entry.IndexBits)
	for _, blockers := range Subsets(entry.Mask) {
		moves := SliderAttacks(piece, square, blockers)
		tableEntry := &table[MagicIndex(entry, blockers)]
		if *tableEntry == BB_Empty {
			*tableEntry = moves
		} else if *tableEntry != moves {
			return nil, errors.New("hash collision")
		}
	}
	return table, nil
}

// List all subsets of set bits of a bitboard
func Subsets(mask Bitboard) []Bitboard {
	bb := BB_Empty
	var maskSubsets []Bitboard
	for {
		maskSubsets = append(maskSubsets, bb)
		bb = (bb - mask) & mask
		if bb == 0 {
			break
		}
	}
	return maskSubsets
}

// Create mask of relevant blockers
func SliderBlockerMask(piece uint8, square Square) Bitboard {
	var attacks []int8
	if piece == Bishop {
		attacks = BishopDirections
	} else if piece == Rook {
		attacks = RookDirections
	} else {
		return BB_Empty
	}

	bb := BB_Empty
	var edgeMask Bitboard
	for _, d := range attacks {
		// Block only the edge relevant to current direction
		// Avoid blocking all squares along an edge for a rook on the edge
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
			// Check for out of bounds
			if !IsValid(next) || Distance(next, sq) > 1 {
				break
			}
			// Remove current direction's edge from mask
			if 1<<next&edgeMask != 0 {
				break
			}
			sq = next
			bb |= 1 << sq
		}
	}
	bb &^= Bitboard(1 << square)
	return bb
}

// Bitboard of valid slider attacks given blocker arrangement
// Include the first blocker (if any) in each direction since it may be a vaid capture
// Check the move for validity later on
func SliderAttacks(piece uint8, square Square, blockers Bitboard) Bitboard {
	var attacks []int8
	if piece == Bishop {
		attacks = BishopDirections
	} else if piece == Rook {
		attacks = RookDirections
	} else {
		return BB_Empty
	}

	bb := BB_Empty
	for _, d := range attacks {
		sq := square
		for {
			next := Square(int8(sq) + d)
			// Check for out of bounds
			if !IsValid(next) || Distance(next, sq) > 1 {
				break
			}
			// Include the first blocker
			bb |= 1 << next
			// Check for blocker
			if 1<<next&blockers != 0 {
				break
			}
			sq = next
		}
	}
	bb &^= Bitboard(1 << square)
	return bb
}

func initKingMoves(square Square) Bitboard {
	bb := BB_Empty
	for dest := SquareA1; dest <= SquareH8; dest++ {
		if Distance(square, dest) == 1 {
			bb |= 1 << dest
		}
	}
	return bb
}

func initKnightMoves(square Square) Bitboard {
	bb := BB_Empty
	for dest := SquareA1; dest <= SquareH8; dest++ {
		hDist := Abs(int(FileOf(dest)) - int(FileOf(square)))
		vDist := Abs(int(RankOf(dest)) - int(RankOf(square)))
		if Distance(square, dest) == 2 && (hDist == 2 && vDist == 1 || hDist == 1 && vDist == 2) {
			bb |= 1 << dest
		}
	}
	return bb
}

func Lsb(bb Bitboard) Square {
	return Square(bits.TrailingZeros(uint(bb)))
}

func PopLsb(bb *Bitboard) Square {
	lsb := Lsb(*bb)
	*bb &= *bb - 1
	return lsb
}

func Merge(bbs []Bitboard) Bitboard {
	result := Bitboard(0)
	for _, bb := range bbs {
		result |= bb
	}
	return result
}

func InitBitboard() {
	// pre-generate magic bitboards and move sets
	for sq := SquareA1; sq <= SquareH8; sq++ {
		table, _ := makeMoveTable(Rook, sq, RookMagics[sq])
		RookMoves[sq] = table

		table, _ = makeMoveTable(Bishop, sq, BishopMagics[sq])
		BishopMoves[sq] = table

		KnightMoves[sq] = initKnightMoves(sq)

		KingMoves[sq] = initKingMoves(sq)
	}
}
