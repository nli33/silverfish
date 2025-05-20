package engine

import (
	"errors"
	"math/bits"
	"math/rand/v2"
)

type Bitboard uint64

// occupied squares
// mask: possibly relevant blockers

type MagicEntry struct {
	// ((occupied & mask) * Magic) >> (64 - Index_bits) = index
	Mask      Bitboard
	Magic     uint64
	IndexBits uint8
}

var RookMagics [64]MagicEntry
var BishopMagics [64]MagicEntry
var RookMoves [64][]Bitboard
var BishopMoves [64][]Bitboard

// RookMoves: {64 : [magicindex : attackset]}

func MagicIndex(entry MagicEntry, blockers Bitboard) uint64 {
	return (uint64(blockers&entry.Mask) * entry.Magic) >> (64 - entry.IndexBits)
}

func GenRookMoves(square Square, blockers Bitboard) Bitboard {
	magic := RookMagics[square]
	moves := RookMoves[square]
	return moves[MagicIndex(magic, blockers)]
}

func GenBishopMoves(square Square, blockers Bitboard) Bitboard {
	magic := BishopMagics[square]
	moves := BishopMoves[square]
	return moves[MagicIndex(magic, blockers)]
}

func findMagic(piece uint8, square Square) (MagicEntry, []Bitboard) {
	relevantMask := SliderBlockerMask(piece, square)
	indexBits := uint8(bits.OnesCount64(uint64(relevantMask)))
	for {
		magic := rand.Uint64() & rand.Uint64() & rand.Uint64()
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
		attacks = BishopAttacks
	} else if piece == Rook {
		attacks = RookAttacks
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
			if (1<<next)&edgeMask != 0 {
				break
			}
			sq = next
			bb |= (1 << sq)
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
		attacks = BishopAttacks
	} else if piece == Rook {
		attacks = RookAttacks
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
			bb |= (1 << next)
			// Check for blocker
			if blockers&(1<<next) != 0 {
				break
			}
			sq = next
		}
	}
	bb &^= Bitboard(1 << square)
	return bb
}

func InitBitboard() {
	// Rooks
	for sq := SquareA1; sq <= SquareH8; sq++ {
		entry, table := findMagic(Rook, sq)
		RookMagics[sq] = entry
		RookMoves[sq] = table
	}
	// Bishops
	for sq := SquareA1; sq <= SquareH8; sq++ {
		entry, table := findMagic(Bishop, sq)
		BishopMagics[sq] = entry
		BishopMoves[sq] = table
	}
}
