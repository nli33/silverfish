package engine_test

import (
	"fmt"
	"silverfish/engine"
	"testing"
)

func TestSliderAttacks(t *testing.T) {
	piece := engine.Rook
	square := engine.SquareD1
	blockers := engine.Bitboard(0x800000000200444)
	gotBB := engine.SliderAttacks(piece, square, blockers)
	wantBB := engine.Bitboard(0x808080808080874)
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
	square := engine.SquareD4
	blockers := engine.BB_Empty
	gotBB := engine.GetRookMoves(square, blockers)
	wantBB := engine.Bitboard(0x8080808f7080808)
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
	// 6 destination squares
	square := engine.SquareB4
	gotBB := engine.GetKnightMoves(square)
	wantBB := engine.Bitboard(0x50800080500)
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
	square := engine.SquareG7
	gotBB := engine.GetKingMoves(square)
	wantBB := engine.Bitboard(0xe0a0e00000000000)
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
	pos := engine.FromFEN("3bk3/p1P1P3/6p1/7P/2p1pN2/3P2N1/1P3PPp/8 w - - 0 1")

	moveList := engine.MoveList{}
	pos.Turn = engine.White
	engine.GenPawnMoves(&pos, &moveList, engine.BB_Full)
	gotMoves := moveList.Moves[:moveList.Count]
	wantMoves := []engine.Move{
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
		t.Errorf(`TestPawnMoves(%d)`, pos.Turn)
		fmt.Println("Got:")
		for _, move := range gotMoves {
			fmt.Printf("%s\n", move.ToString())
		}
		fmt.Println("Want:")
		for _, move := range wantMoves {
			fmt.Printf("%s\n", move.ToString())
		}
		fmt.Println()
	}

	moveList = engine.MoveList{}
	pos.Turn = engine.Black
	engine.GenPawnMoves(&pos, &moveList, engine.BB_Full)
	gotMoves = moveList.Moves[:moveList.Count]
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
		fmt.Println("Want:")
		for _, move := range wantMoves {
			fmt.Printf("%s\n", move.ToString())
		}
		fmt.Println()
	}
}

func TestEnPassant(t *testing.T) {
	pos := engine.FromFEN("8/8/8/6pP/1Pp5/8/8/8 w - - 0 1")

	moveList := engine.MoveList{}
	pos.Turn = engine.Black
	pos.EnPassantSquare = engine.SquareB3
	engine.GenPawnMoves(&pos, &moveList, engine.BB_Full)
	gotMoves := moveList.Moves[:moveList.Count]
	wantMoves := []engine.Move{
		engine.NewMoveFromStr("c4b3") | engine.EnPassantFlag,
		engine.NewMoveFromStr("c4c3"),
		engine.NewMoveFromStr("g5g4"),
	}
	if !equalSets(gotMoves, wantMoves) {
		t.Errorf(`TestPawnMoves(%d)`, pos.Turn)
		fmt.Println("Got:")
		for _, move := range gotMoves {
			fmt.Printf("%s\n", move.ToString())
		}
		fmt.Println("Want:")
		for _, move := range wantMoves {
			fmt.Printf("%s\n", move.ToString())
		}
		fmt.Println()
	}

	moveList = engine.MoveList{}
	pos.Turn = engine.White
	pos.EnPassantSquare = engine.SquareG6
	engine.GenPawnMoves(&pos, &moveList, engine.BB_Full)
	gotMoves = moveList.Moves[:moveList.Count]
	wantMoves = []engine.Move{
		engine.NewMoveFromStr("b4b5"),
		engine.NewMoveFromStr("h5h6"),
		engine.NewMoveFromStr("h5g6") | engine.EnPassantFlag,
	}
	if !equalSets(gotMoves, wantMoves) {
		t.Errorf(`TestPawnMoves(%d)`, pos.Turn)
		fmt.Println("Got:")
		for _, move := range gotMoves {
			fmt.Printf("%s\n", move.ToString())
		}
		fmt.Println("Want:")
		for _, move := range wantMoves {
			fmt.Printf("%s\n", move.ToString())
		}
		fmt.Println()
	}
}

func TestCastlingMoves(t *testing.T) {
	pos := engine.FromFEN("rn1qk2r/8/8/8/8/8/8/R3KB1R w KQkq - 0 1")

	moveList := engine.MoveList{}
	pos.Turn = engine.White
	engine.GenCastlingMoves(&pos, &moveList)
	gotMoves := moveList.Moves[:moveList.Count]
	wantMoves := []engine.Move{
		engine.NewMoveCastle(engine.WhiteQueenside),
	}
	if !equalSets(gotMoves, wantMoves) {
		t.Errorf(`TestCastlingMoves(%d)`, pos.Turn)
		fmt.Println("Got:")
		for _, move := range gotMoves {
			fmt.Printf("%s\n", move.ToString())
		}
		fmt.Println("Want:")
		for _, move := range wantMoves {
			fmt.Printf("%s\n", move.ToString())
		}
		fmt.Println()
	}

	// No castling rights on white queenside
	moveList = engine.MoveList{}
	pos.CastlingRights = 0b1101
	engine.GenCastlingMoves(&pos, &moveList)
	gotMoves = moveList.Moves[:moveList.Count]
	wantMoves = []engine.Move{}
	if !equalSets(gotMoves, wantMoves) {
		t.Errorf(`TestCastlingMoves(%d)`, pos.Turn)
		fmt.Println("Got:")
		for _, move := range gotMoves {
			fmt.Printf("%s\n", move.ToString())
		}
		fmt.Println("Want:")
		for _, move := range wantMoves {
			fmt.Printf("%s\n", move.ToString())
		}
		fmt.Println()
	}

	moveList = engine.MoveList{}
	pos.Turn = engine.Black
	engine.GenCastlingMoves(&pos, &moveList)
	gotMoves = moveList.Moves[:moveList.Count]
	wantMoves = []engine.Move{
		engine.NewMoveCastle(engine.BlackKingside),
	}
	if !equalSets(gotMoves, wantMoves) {
		t.Errorf(`TestCastlingMoves(%d)`, pos.Turn)
		fmt.Println("Got:")
		for _, move := range gotMoves {
			fmt.Printf("%s\n", move.ToString())
		}
		fmt.Println("Want:")
		for _, move := range wantMoves {
			fmt.Printf("%s\n", move.ToString())
		}
		fmt.Println()
	}
}

func init() {
	engine.InitBitboard()
}
