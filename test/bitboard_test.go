package board

import (
	"fmt"
	"silverfish/engine"
	"sort"
	"testing"
)

func TestSliderBlockerMask(t *testing.T) {
	var piece uint8
	var square engine.Square
	var gotBB, wantBB engine.Bitboard

	piece = engine.Bishop
	square = engine.SquareD3
	gotBB = engine.SliderBlockerMask(piece, square)
	wantBB = engine.Bitboard(0b00000000_00000000_01000000_00100010_00010100_00000000_00010100_00000000)
	if gotBB != wantBB {
		t.Errorf(`SliderBlockerMask(%d, %d)`, piece, square)
		fmt.Printf("Got:\n%s\n", gotBB.ToString())
		fmt.Printf("Want:\n%s\n", wantBB.ToString())
	}

	piece = engine.Rook
	square = engine.SquareH1
	gotBB = engine.SliderBlockerMask(piece, square)
	wantBB = engine.Bitboard(0b00000000_10000000_10000000_10000000_10000000_10000000_10000000_01111110)
	if gotBB != wantBB {
		t.Errorf(`SliderBlockerMask(%d, %d)`, piece, square)
		fmt.Printf("Got:\n%s\n", gotBB.ToString())
		fmt.Printf("Want:\n%s\n", wantBB.ToString())
	}
}

func equalSets(a, b []engine.Bitboard) bool {
	if len(a) != len(b) {
		return false
	}

	aCopy := append([]engine.Bitboard{}, a...)
	bCopy := append([]engine.Bitboard{}, b...)

	sort.Slice(aCopy, func(i, j int) bool {
		return aCopy[i] < aCopy[j]
	})
	sort.Slice(bCopy, func(i, j int) bool {
		return bCopy[i] < bCopy[j]
	})

	for i := range aCopy {
		if aCopy[i] != bCopy[i] {
			return false
		}
	}
	return true
}

func TestSubsets(t *testing.T) {
	var mask engine.Bitboard
	var wantSubsets, gotSubsets []engine.Bitboard

	mask = engine.Bitboard(0b01010001)
	wantSubsets = []engine.Bitboard{
		engine.Bitboard(0b00000000),
		engine.Bitboard(0b00010001),
		engine.Bitboard(0b01000001),
		engine.Bitboard(0b01010000),
		engine.Bitboard(0b01000000),
		engine.Bitboard(0b00010000),
		engine.Bitboard(0b00000001),
		engine.Bitboard(0b01010001),
	}
	gotSubsets = engine.Subsets(mask)

	if !equalSets(wantSubsets, gotSubsets) {
		t.Errorf(`TestSubsets(%s)`, mask.ToStringSmall())
		fmt.Println("Got:")
		for _, bb := range gotSubsets {
			fmt.Printf("%s\n", bb.ToStringSmall())
		}
		fmt.Println("Want:")
		for _, bb := range wantSubsets {
			fmt.Printf("%s\n", bb.ToStringSmall())
		}
	}
}

func TestSliderAttacks(t *testing.T) {
	var piece uint8
	var square engine.Square
	var blockers engine.Bitboard
	var gotBB, wantBB engine.Bitboard

	piece = engine.Rook
	square = engine.SquareD1
	blockers = engine.Bitboard(0b00001000_00000000_00000000_00000000_00000000_00100000_00000100_01000100)
	gotBB = engine.SliderAttacks(piece, square, blockers)
	wantBB = engine.Bitboard(0b00001000_00001000_00001000_00001000_00001000_00001000_00001000_01110100)
	if gotBB != wantBB {
		t.Errorf(`SliderAttacks(%d, %d, %d)`, piece, square, blockers)
		fmt.Printf("Got:\n%s\n", gotBB.ToString())
		fmt.Printf("Want:\n%s\n", wantBB.ToString())
		fmt.Printf("Blockers:\n%s\n", blockers.ToString())
	}

	piece = engine.Bishop
	square = engine.SquareD1
	gotBB = engine.SliderAttacks(piece, square, blockers)
	wantBB = engine.Bitboard(0b00000000_00000000_00000000_00000000_00000000_00100000_00010100_00000000)
	if gotBB != wantBB {
		t.Errorf(`SliderAttacks(%d, %d, %d)`, piece, square, blockers)
		fmt.Printf("Got:\n%s\n", gotBB.ToString())
		fmt.Printf("Want:\n%s\n", wantBB.ToString())
		fmt.Printf("Blockers:\n%s\n", blockers.ToString())
	}

	piece = engine.Rook
	square = engine.SquareD3
	blockers = engine.Bitboard(0b00001000_00000000_00000000_00000000_00000000_00100001_00000000_01000110)
	gotBB = engine.SliderAttacks(piece, square, blockers)
	wantBB = engine.Bitboard(0b00001000_00001000_00001000_00001000_00001000_00110111_00001000_00001000)
	if gotBB != wantBB {
		t.Errorf(`SliderAttacks(%d, %d, %d)`, piece, square, blockers)
		fmt.Printf("Got:\n%s\n", gotBB.ToString())
		fmt.Printf("Want:\n%s\n", wantBB.ToString())
		fmt.Printf("Blockers:\n%s\n", blockers.ToString())
	}
}

func TestMagicBitboard(t *testing.T) {
	var square engine.Square
	var blockers, gotBB, wantBB engine.Bitboard

	square = engine.SquareD4
	blockers = engine.BB_Empty
	gotBB = engine.GenRookMoves(square, blockers)
	wantBB = engine.Bitboard(0b00001000_00001000_00001000_00001000_11110111_00001000_00001000_00001000)
	if gotBB != wantBB {
		t.Errorf(`MagicBitboard(%d, %d)`, square, blockers)
		fmt.Printf("Got:\n%s\n", gotBB.ToString())
		fmt.Printf("Want:\n%s\n", wantBB.ToString())
		fmt.Printf("Blockers:\n%s\n", blockers.ToString())
	}

	// Rook, board is all blockers
	square = engine.SquareD4
	blockers = engine.BB_Full &^ (1 << square)
	gotBB = engine.GenRookMoves(square, blockers)
	wantBB = engine.Bitboard(0b00000000_00000000_00000000_00001000_00010100_00001000_00000000_00000000)
	if gotBB != wantBB {
		t.Errorf(`MagicBitboard(%d, %d)`, square, blockers)
		fmt.Printf("Got:\n%s\n", gotBB.ToString())
		fmt.Printf("Want:\n%s\n", wantBB.ToString())
		fmt.Printf("Blockers:\n%s\n", blockers.ToString())
	}

	// Rook, all edge blockers
	square = engine.SquareD4
	blockers = engine.BB_Edges
	gotBB = engine.GenRookMoves(square, blockers)
	wantBB = engine.Bitboard(0b00001000_00001000_00001000_00001000_11110111_00001000_00001000_00001000)
	if gotBB != wantBB {
		t.Errorf(`MagicBitboard(%d, %d)`, square, blockers)
		fmt.Printf("Got:\n%s\n", gotBB.ToString())
		fmt.Printf("Want:\n%s\n", wantBB.ToString())
		fmt.Printf("Blockers:\n%s\n", blockers.ToString())
	}

	// Rook, mix
	square = engine.SquareD3
	blockers = engine.Bitboard(0b00001000_00000000_00000000_00000000_00000000_00100001_00000000_01000110)
	gotBB = engine.GenRookMoves(square, blockers)
	wantBB = engine.Bitboard(0b00001000_00001000_00001000_00001000_00001000_00110111_00001000_00001000)
	if gotBB != wantBB {
		t.Errorf(`MagicBitboard(%d, %d)`, square, blockers)
		fmt.Printf("Got:\n%s\n", gotBB.ToString())
		fmt.Printf("Want:\n%s\n", wantBB.ToString())
		fmt.Printf("Blockers:\n%s\n", blockers.ToString())
	}

	// Bishop on G6. Adjacent blocker, edge blocker, faraway blocker
	square = engine.SquareG6
	blockers = engine.Bitboard(0b00010000_10000000_00000000_00000000_00000000_00001000_00000000_00000000)
	gotBB = engine.GenBishopMoves(square, blockers)
	wantBB = engine.Bitboard(0b00010000_10100000_00000000_10100000_00010000_00001000_00000000_00000000)
	if gotBB != wantBB {
		t.Errorf(`MagicBitboard(%d, %d)`, square, blockers)
		fmt.Printf("Got:\n%s\n", gotBB.ToString())
		fmt.Printf("Want:\n%s\n", wantBB.ToString())
		fmt.Printf("Blockers:\n%s\n", blockers.ToString())
	}

	// Bishop on A1, long diagonal without blockers
	square = engine.SquareA1
	blockers = engine.BB_Empty
	gotBB = engine.GenBishopMoves(square, blockers)
	wantBB = engine.Bitboard(0b10000000_01000000_00100000_00010000_00001000_00000100_00000010_00000000)
	if gotBB != wantBB {
		t.Errorf(`MagicBitboard(%d, %d)`, square, blockers)
		fmt.Printf("Got:\n%s\n", gotBB.ToString())
		fmt.Printf("Want:\n%s\n", wantBB.ToString())
		fmt.Printf("Blockers:\n%s\n", blockers.ToString())
	}

	// Bishop, board is all blockers
	square = engine.SquareE5
	blockers = engine.BB_Full &^ (1 << square)
	gotBB = engine.GenBishopMoves(square, blockers)
	wantBB = engine.Bitboard(0b00000000_00000000_00101000_00000000_00101000_00000000_00000000_00000000)
	if gotBB != wantBB {
		t.Errorf(`MagicBitboard(%d, %d)`, square, blockers)
		fmt.Printf("Got:\n%s\n", gotBB.ToString())
		fmt.Printf("Want:\n%s\n", wantBB.ToString())
		fmt.Printf("Blockers:\n%s\n", blockers.ToString())
	}

	// Bishop, all edge blockers
	square = engine.SquareE5
	blockers = engine.BB_Edges
	gotBB = engine.GenBishopMoves(square, blockers)
	wantBB = engine.Bitboard(0b10000010_01000100_00101000_00000000_00101000_01000100_10000010_00000001)
	if gotBB != wantBB {
		t.Errorf(`MagicBitboard(%d, %d)`, square, blockers)
		fmt.Printf("Got:\n%s\n", gotBB.ToString())
		fmt.Printf("Want:\n%s\n", wantBB.ToString())
		fmt.Printf("Blockers:\n%s\n", blockers.ToString())
	}
}

func TestKnightMoves(t *testing.T) {
	var square engine.Square
	var gotBB, wantBB engine.Bitboard

	// 6 destination squares
	square = engine.SquareB4
	gotBB = engine.GenKnightMoves(square)
	wantBB = engine.Bitboard(0b00000000_00000000_00000101_00001000_00000000_00001000_00000101_00000000)
	if gotBB != wantBB {
		t.Errorf(`TestKnightMoves(%d)`, square)
		fmt.Printf("Got:\n%s\n", gotBB.ToString())
		fmt.Printf("Want:\n%s\n", wantBB.ToString())
	}

	// 3 destination squares
	square = engine.SquareG1
	gotBB = engine.GenKnightMoves(square)
	wantBB = engine.Bitboard(0b00000000_00000000_00000000_00000000_00000000_10100000_00010000_00000000)
	if gotBB != wantBB {
		t.Errorf(`TestKnightMoves(%d)`, square)
		fmt.Printf("Got:\n%s\n", gotBB.ToString())
		fmt.Printf("Want:\n%s\n", wantBB.ToString())
	}

	// 8 destination squares
	square = engine.SquareD6
	gotBB = engine.GenKnightMoves(square)
	wantBB = engine.Bitboard(0b00010100_00100010_00000000_00100010_00010100_00000000_00000000_00000000)
	if gotBB != wantBB {
		t.Errorf(`TestKnightMoves(%d)`, square)
		fmt.Printf("Got:\n%s\n", gotBB.ToString())
		fmt.Printf("Want:\n%s\n", wantBB.ToString())
	}
}

func init() {
	engine.InitBitboard()
}
