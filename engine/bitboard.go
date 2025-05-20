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
var KnightMoves [64]Bitboard

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

func GenKnightMoves(square Square) Bitboard {
	return KnightMoves[square]
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

func findKnightMoves(square Square) Bitboard {
	mask_1 := Bitboard(0b00001010)
	mask_2 := Bitboard(0b00010001)
	mask_3 := Bitboard(0b00010001)
	mask_4 := Bitboard(0b00001010)
	range_mask := Bitboard(0b11111111)

	file := FileOf(square)
	rank := RankOf(square)

	if file <= FileB {
		mask_1 >>= (2 - file)
		mask_2 >>= (2 - file)
		mask_3 >>= (2 - file)
		mask_4 >>= (2 - file)
	} else if file >= FileD {
		mask_1 = (mask_1 << (file - 2)) & range_mask
		mask_2 = (mask_2 << (file - 2)) & range_mask
		mask_3 = (mask_3 << (file - 2)) & range_mask
		mask_4 = (mask_4 << (file - 2)) & range_mask
	}

	bb := (mask_1 << 32) | (mask_2 << 24) | (mask_3 << 8) | mask_4

	if rank <= Rank2 {
		bb >>= (2 - rank) * 8
	} else if rank >= Rank4 {
		bb = (bb << ((rank - 2) * 8)) & BB_Full
	}

	return bb
}

func InitBitboard() {
	// Rooks
	for sq := SquareA1; sq <= SquareH8; sq++ {
		entry, table := findMagic(Rook, sq)
		RookMagics[sq] = entry
		RookMoves[sq] = table

		entry, table = findMagic(Bishop, sq)
		BishopMagics[sq] = entry
		BishopMoves[sq] = table

		moves := findKnightMoves(sq)
		KnightMoves[sq] = moves
	}
}
