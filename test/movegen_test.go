package test

import (
	"fmt"
	"silverfish/engine"
	"testing"
)

func TestSliderAttacks(t *testing.T) {
	var piece uint8
	var square engine.Square
	var blockers engine.Bitboard
	var gotBB, wantBB engine.Bitboard

	piece = engine.Rook
	square = engine.SquareD1
	blockers = engine.Bitboard(0x800000000200444)
	gotBB = engine.SliderAttacks(piece, square, blockers)
	wantBB = engine.Bitboard(0x808080808080874)
	if gotBB != wantBB {
		t.Errorf(`SliderAttacks(%d, %d, %d)`, piece, square, blockers)
		fmt.Printf("Got:\n%s\n", gotBB.ToString())
		fmt.Printf("Want:\n%s\n", wantBB.ToString())
		fmt.Printf("Blockers:\n%s\n", blockers.ToString())
	}

	piece = engine.Bishop
	square = engine.SquareD1
	gotBB = engine.SliderAttacks(piece, square, blockers)
	wantBB = engine.Bitboard(0x201400)
	if gotBB != wantBB {
		t.Errorf(`SliderAttacks(%d, %d, %d)`, piece, square, blockers)
		fmt.Printf("Got:\n%s\n", gotBB.ToString())
		fmt.Printf("Want:\n%s\n", wantBB.ToString())
		fmt.Printf("Blockers:\n%s\n", blockers.ToString())
	}

	piece = engine.Rook
	square = engine.SquareD3
	blockers = engine.Bitboard(0x800000000210046)
	gotBB = engine.SliderAttacks(piece, square, blockers)
	wantBB = engine.Bitboard(0x808080808370808)
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
	gotBB = engine.GetRookMoves(square, blockers)
	wantBB = engine.Bitboard(0x8080808f7080808)
	if gotBB != wantBB {
		t.Errorf(`MagicBitboard(%d, %d)`, square, blockers)
		fmt.Printf("Got:\n%s\n", gotBB.ToString())
		fmt.Printf("Want:\n%s\n", wantBB.ToString())
		fmt.Printf("Blockers:\n%s\n", blockers.ToString())
	}

	// Rook, board is all blockers
	square = engine.SquareD4
	blockers = engine.BB_Full &^ (1 << square)
	gotBB = engine.GetRookMoves(square, blockers)
	wantBB = engine.Bitboard(0x814080000)
	if gotBB != wantBB {
		t.Errorf(`MagicBitboard(%d, %d)`, square, blockers)
		fmt.Printf("Got:\n%s\n", gotBB.ToString())
		fmt.Printf("Want:\n%s\n", wantBB.ToString())
		fmt.Printf("Blockers:\n%s\n", blockers.ToString())
	}

	// Rook, all edge blockers
	square = engine.SquareD4
	blockers = engine.BB_Edges
	gotBB = engine.GetRookMoves(square, blockers)
	wantBB = engine.Bitboard(0x8080808f7080808)
	if gotBB != wantBB {
		t.Errorf(`MagicBitboard(%d, %d)`, square, blockers)
		fmt.Printf("Got:\n%s\n", gotBB.ToString())
		fmt.Printf("Want:\n%s\n", wantBB.ToString())
		fmt.Printf("Blockers:\n%s\n", blockers.ToString())
	}

	// Rook, mix
	square = engine.SquareD3
	blockers = engine.Bitboard(0x800000000210046)
	gotBB = engine.GetRookMoves(square, blockers)
	wantBB = engine.Bitboard(0x808080808370808)
	if gotBB != wantBB {
		t.Errorf(`MagicBitboard(%d, %d)`, square, blockers)
		fmt.Printf("Got:\n%s\n", gotBB.ToString())
		fmt.Printf("Want:\n%s\n", wantBB.ToString())
		fmt.Printf("Blockers:\n%s\n", blockers.ToString())
	}

	// Bishop on G6. Adjacent blocker, edge blocker, faraway blocker
	square = engine.SquareG6
	blockers = engine.Bitboard(0x1080000000080000)
	gotBB = engine.GetBishopMoves(square, blockers)
	wantBB = engine.Bitboard(0x10a000a010080000)
	if gotBB != wantBB {
		t.Errorf(`MagicBitboard(%d, %d)`, square, blockers)
		fmt.Printf("Got:\n%s\n", gotBB.ToString())
		fmt.Printf("Want:\n%s\n", wantBB.ToString())
		fmt.Printf("Blockers:\n%s\n", blockers.ToString())
	}

	// Bishop on A1, long diagonal without blockers
	square = engine.SquareA1
	blockers = engine.BB_Empty
	gotBB = engine.GetBishopMoves(square, blockers)
	wantBB = engine.Bitboard(0x8040201008040200)
	if gotBB != wantBB {
		t.Errorf(`MagicBitboard(%d, %d)`, square, blockers)
		fmt.Printf("Got:\n%s\n", gotBB.ToString())
		fmt.Printf("Want:\n%s\n", wantBB.ToString())
		fmt.Printf("Blockers:\n%s\n", blockers.ToString())
	}

	// Bishop, board is all blockers
	square = engine.SquareE5
	blockers = engine.BB_Full &^ (1 << square)
	gotBB = engine.GetBishopMoves(square, blockers)
	wantBB = engine.Bitboard(0x280028000000)
	if gotBB != wantBB {
		t.Errorf(`MagicBitboard(%d, %d)`, square, blockers)
		fmt.Printf("Got:\n%s\n", gotBB.ToString())
		fmt.Printf("Want:\n%s\n", wantBB.ToString())
		fmt.Printf("Blockers:\n%s\n", blockers.ToString())
	}

	// Bishop, all edge blockers
	square = engine.SquareE5
	blockers = engine.BB_Edges
	gotBB = engine.GetBishopMoves(square, blockers)
	wantBB = engine.Bitboard(0x8244280028448201)
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
	gotBB = engine.GetKnightMoves(square)
	wantBB = engine.Bitboard(0x50800080500)
	if gotBB != wantBB {
		t.Errorf(`TestKnightMoves(%d)`, square)
		fmt.Printf("Got:\n%s\n", gotBB.ToString())
		fmt.Printf("Want:\n%s\n", wantBB.ToString())
	}

	// 3 destination squares
	square = engine.SquareG1
	gotBB = engine.GetKnightMoves(square)
	wantBB = engine.Bitboard(0xa01000)
	if gotBB != wantBB {
		t.Errorf(`TestKnightMoves(%d)`, square)
		fmt.Printf("Got:\n%s\n", gotBB.ToString())
		fmt.Printf("Want:\n%s\n", wantBB.ToString())
	}

	// 8 destination squares
	square = engine.SquareD6
	gotBB = engine.GetKnightMoves(square)
	wantBB = engine.Bitboard(0x1422002214000000)
	if gotBB != wantBB {
		t.Errorf(`TestKnightMoves(%d)`, square)
		fmt.Printf("Got:\n%s\n", gotBB.ToString())
		fmt.Printf("Want:\n%s\n", wantBB.ToString())
	}
}

func TestKingMoves(t *testing.T) {
	var square engine.Square
	var gotBB, wantBB engine.Bitboard

	square = engine.SquareG7
	gotBB = engine.GetKingMoves(square)
	wantBB = engine.Bitboard(0xe0a0e00000000000)
	if gotBB != wantBB {
		t.Errorf(`TestKingMoves(%d)`, square)
		fmt.Printf("Got:\n%s\n", gotBB.ToString())
		fmt.Printf("Want:\n%s\n", wantBB.ToString())
	}

	square = engine.SquareE1
	gotBB = engine.GetKingMoves(square)
	wantBB = engine.Bitboard(0x3828)
	if gotBB != wantBB {
		t.Errorf(`TestKingMoves(%d)`, square)
		fmt.Printf("Got:\n%s\n", gotBB.ToString())
		fmt.Printf("Want:\n%s\n", wantBB.ToString())
	}

	square = engine.SquareA8
	gotBB = engine.GetKingMoves(square)
	wantBB = engine.Bitboard(0x203000000000000)
	if gotBB != wantBB {
		t.Errorf(`TestKingMoves(%d)`, square)
		fmt.Printf("Got:\n%s\n", gotBB.ToString())
		fmt.Printf("Want:\n%s\n", wantBB.ToString())
	}
}

func TestPawnMoves(t *testing.T) {
	var square engine.Square
	var blockers engine.Bitboard
	var pos engine.Position
	var gotMoves, wantMoves []engine.Move

	var pieces = [2][6]engine.Bitboard{
		{
			engine.Bitboard(0x0014008000086200),
			engine.Bitboard(0x0000000020400000),
			engine.Bitboard(0),
			engine.Bitboard(0),
			engine.Bitboard(0),
			engine.Bitboard(0),
		},
		{
			engine.Bitboard(0x0001400014008000),
			engine.Bitboard(0),
			engine.Bitboard(0x0800000000000000),
			engine.Bitboard(0),
			engine.Bitboard(0),
			engine.Bitboard(0x1000000000000000),
		},
	}
	pos = engine.Position{
		Pieces:          pieces,
		EnPassantSquare: engine.NoSquare,
	}
	blockers = pos.Blockers()

	pos.Turn = engine.White
	gotMoves = engine.GetPawnMoves(pos, blockers)
	wantMoves = []engine.Move{
		engine.NewMoveFromStr("b2b3"),
		engine.NewMoveFromStr("b2b4"),
		engine.NewMoveFromStr("d3c4"),
		engine.NewMoveFromStr("d3d4"),
		engine.NewMoveFromStr("d3e4"),
		engine.NewMoveFromStr("f2f3"),
		engine.NewMoveFromStr("f2g3"),
		engine.NewMoveFromStr("h5g6"),
		engine.NewMoveFromStr("h5h6"),

		engine.NewMoveFromStr("c7c8n"),
		engine.NewMoveFromStr("c7c8b"),
		engine.NewMoveFromStr("c7c8r"),
		engine.NewMoveFromStr("c7c8q"),
		engine.NewMoveFromStr("c7d8n"),
		engine.NewMoveFromStr("c7d8b"),
		engine.NewMoveFromStr("c7d8r"),
		engine.NewMoveFromStr("c7d8q"),

		engine.NewMoveFromStr("e7d8n"),
		engine.NewMoveFromStr("e7d8b"),
		engine.NewMoveFromStr("e7d8r"),
		engine.NewMoveFromStr("e7d8q"),
	}
	if !equalSets(gotMoves, wantMoves) {
		t.Errorf(`TestPawnMoves(%d)`, square)
		fmt.Println("Got:")
		for _, move := range gotMoves {
			fmt.Printf("%s\n", move.ToString())
		}
		fmt.Println("\nWant:")
		for _, move := range wantMoves {
			fmt.Printf("%s\n", move.ToString())
		}
	}

	pos.Turn = engine.Black
	gotMoves = engine.GetPawnMoves(pos, blockers)
	wantMoves = []engine.Move{
		engine.NewMoveFromStr("a7a6"),
		engine.NewMoveFromStr("a7a5"),
		engine.NewMoveFromStr("c4c3"),
		engine.NewMoveFromStr("c4d3"),
		engine.NewMoveFromStr("e4d3"),
		engine.NewMoveFromStr("e4e3"),
		engine.NewMoveFromStr("g6g5"),
		engine.NewMoveFromStr("g6h5"),

		engine.NewMoveFromStr("h2h1n"),
		engine.NewMoveFromStr("h2h1b"),
		engine.NewMoveFromStr("h2h1r"),
		engine.NewMoveFromStr("h2h1q"),
	}
	if !equalSets(gotMoves, wantMoves) {
		t.Errorf(`TestPawnMoves(%d)`, pos.Turn)
		fmt.Println("Got:")
		for _, move := range gotMoves {
			fmt.Printf("%s\n", move.ToString())
		}
		fmt.Println("\nWant:")
		for _, move := range wantMoves {
			fmt.Printf("%s\n", move.ToString())
		}
	}
}

func init() {
	engine.InitBitboard()
}
