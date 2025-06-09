package test

import (
	"fmt"
	"silverfish/engine"
	"testing"
)

func TestSliderBlockerMask(t *testing.T) {
	piece := engine.Bishop
	square := engine.SquareD3
	gotBB := engine.SliderBlockerMask(piece, square)
	wantBB := engine.Bitboard(0b00000000_00000000_01000000_00100010_00010100_00000000_00010100_00000000)
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

func equalSets[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}

	counts := make(map[T]int, len(a))
	for _, v := range a {
		counts[v]++
	}

	for _, v := range b {
		if counts[v] == 0 {
			return false
		}
		counts[v]--
	}

	return true
}

func TestSubsets(t *testing.T) {
	mask := engine.Bitboard(0b01010001)
	wantSubsets := []engine.Bitboard{
		engine.Bitboard(0b00000000),
		engine.Bitboard(0b00010001),
		engine.Bitboard(0b01000001),
		engine.Bitboard(0b01010000),
		engine.Bitboard(0b01000000),
		engine.Bitboard(0b00010000),
		engine.Bitboard(0b00000001),
		engine.Bitboard(0b01010001),
	}
	gotSubsets := engine.Subsets(mask)

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
