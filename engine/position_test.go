package engine_test

import (
	"fmt"
	"silverfish/engine"
	"testing"
)

func TestRankFile(t *testing.T) {
	startingSquare := engine.SquareG8
	file := engine.FileOf(startingSquare)
	rank := engine.RankOf(startingSquare)
	wantSquare := engine.SquareD6
	gotSquare := engine.NewSquare(rank-2, file-3)
	if wantSquare != gotSquare {
		t.Errorf(`(r,f) = %d, (r-2, f-1) = %d; want %d`, startingSquare, wantSquare, gotSquare)
	}
}

func TestGetSquare(t *testing.T) {
	p := engine.StartingPosition()

	square := engine.SquareD1
	gotColor, gotPiece := p.GetSquare(square)
	wantColor := engine.White
	wantPiece := engine.Queen
	if gotColor != wantColor || gotPiece != wantPiece {
		t.Errorf(`p.GetSquare(%d) = (%d, %d); want (%d, %d)`, square, gotColor, gotPiece, wantColor, wantPiece)
	}

	square = engine.SquareG8
	gotColor, gotPiece = p.GetSquare(square)
	wantColor = engine.Black
	wantPiece = engine.Knight
	if gotColor != wantColor || gotPiece != wantPiece {
		t.Errorf(`p.GetSquare(%d) = (%d, %d); want (%d, %d)`, square, gotColor, gotPiece, wantColor, wantPiece)
	}

	square = engine.SquareH6
	gotColor, gotPiece = p.GetSquare(square)
	wantColor = engine.NoColor
	wantPiece = engine.NoPiece
	if gotColor != wantColor || gotPiece != wantPiece {
		t.Errorf(`p.GetSquare(%d) = (%d, %d); want (%d, %d)`, square, gotColor, gotPiece, wantColor, wantPiece)
	}
}

func TestFEN(t *testing.T) {
	pos := engine.StartingPosition()
	wantFEN := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	gotFEN := pos.ToFEN()
	if wantFEN != gotFEN {
		t.Errorf("ToFEN()")
		fmt.Println("Got:")
		fmt.Println(gotFEN)
		fmt.Println("Want:")
		fmt.Println(wantFEN)
	}

	pieces := [2][6]engine.Bitboard{
		{
			engine.Bitboard(0x0080000000004000),
			engine.Bitboard(0x0000000000000000),
			engine.Bitboard(0x0000000000000000),
			engine.Bitboard(0x0000000000000001),
			engine.Bitboard(0x0000000000000000),
			engine.Bitboard(0x0000000000000010),
		},
		{
			engine.Bitboard(0x0000000020000000),
			engine.Bitboard(0x0000000001000000),
			engine.Bitboard(0x0000000000000000),
			engine.Bitboard(0x0000000000000000),
			engine.Bitboard(0x0000000000000000),
			engine.Bitboard(0x0200000000000000),
		},
	}
	pos = engine.Position{
		Pieces:          pieces,
		Turn:            engine.White,
		CastlingRights:  0b0010,
		Rule50:          4,
		Ply:             56,
		EnPassantSquare: engine.NoSquare,
	}

	wantFEN = "1k6/7P/8/8/n4p2/8/6P1/R3K3 w Q - 4 29"
	gotFEN = pos.ToFEN()
	if wantFEN != gotFEN {
		t.Errorf("ToFEN()")
		fmt.Println("Got:")
		fmt.Println(gotFEN)
		fmt.Println("Want:")
		fmt.Println(wantFEN)
	}

	wantPos := pos
	gotPos := engine.FromFEN(wantFEN)
	if !wantPos.Equals(gotPos) {
		t.Errorf("FromFEN()")
		fmt.Println("Parsed Position:")
		fmt.Println(gotFEN)
		fmt.Println("Actual Position:")
		fmt.Println(wantFEN)
	}
}

func TestAttackers(t *testing.T) {
	pos := engine.FromFEN("8/1Q5b/6B1/2np4/Q6R/3K1P2/3Nr3/7B w - - 0 1")
	square := engine.SquareE4

	color := engine.White
	gotAttackers := pos.AttackersFrom(square, color)
	wantAttackers := engine.Bitboard(0x0000400081280800)
	if gotAttackers != wantAttackers {
		t.Errorf(`AttackersFrom(%d, %d)`, square, color)
		fmt.Println("Got:")
		fmt.Println(gotAttackers.ToString())
		fmt.Println("Want:")
		fmt.Println(wantAttackers.ToString())
	}

	color = engine.Black
	gotAttackers = pos.AttackersFrom(square, color)
	wantAttackers = engine.Bitboard(0x0000000c00001000)
	if gotAttackers != wantAttackers {
		t.Errorf(`AttackersFrom(%d, %d)`, square, color)
		fmt.Println("Got:")
		fmt.Println(gotAttackers.ToString())
		fmt.Println("Want:")
		fmt.Println(wantAttackers.ToString())
	}

	gotAttackers = pos.Attackers(square)
	wantAttackers = engine.Bitboard(0x0000400c81281800)
	if gotAttackers != wantAttackers {
		t.Errorf(`Attackers(%d)`, square)
		fmt.Println("Got:")
		fmt.Println(gotAttackers.ToString())
		fmt.Println("Want:")
		fmt.Println(wantAttackers.ToString())
	}
}

func TestLegalMoves(t *testing.T) {
	pos := engine.FromFEN("1k6/7P/8/8/5b2/8/8/R3K2R w KQ - 0 1")

	pos.Turn = engine.White
	gotMoves := pos.LegalMoves()
	wantMoves := []engine.Move{
		// A rook north
		engine.NewMoveFromStr("a1a2"),
		engine.NewMoveFromStr("a1a3"),
		engine.NewMoveFromStr("a1a4"),
		engine.NewMoveFromStr("a1a5"),
		engine.NewMoveFromStr("a1a6"),
		engine.NewMoveFromStr("a1a7"),
		engine.NewMoveFromStr("a1a8"),
		// A rook east
		engine.NewMoveFromStr("a1b1"),
		engine.NewMoveFromStr("a1c1"),
		engine.NewMoveFromStr("a1d1"),
		// king
		engine.NewMoveFromStr("e1d1"),
		engine.NewMoveFromStr("e1e2"),
		engine.NewMoveFromStr("e1f2"),
		engine.NewMoveFromStr("e1f1"),
		// kingside castling (queenside is illegal due to check)
		engine.NewMoveCastle(engine.WhiteKingside),
		// H rook west
		engine.NewMoveFromStr("h1f1"),
		engine.NewMoveFromStr("h1g1"),
		// H rook north
		engine.NewMoveFromStr("h1h2"),
		engine.NewMoveFromStr("h1h3"),
		engine.NewMoveFromStr("h1h4"),
		engine.NewMoveFromStr("h1h5"),
		engine.NewMoveFromStr("h1h6"),
		// promotion
		engine.NewMoveFromStr("h7h8n"),
		engine.NewMoveFromStr("h7h8b"),
		engine.NewMoveFromStr("h7h8r"),
		engine.NewMoveFromStr("h7h8q"),
	}
	if !equalSets(gotMoves, wantMoves) {
		t.Errorf(`TestLegalMoves(%d)`, pos.Turn)
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

	pos.Turn = engine.Black
	gotMoves = pos.LegalMoves()
	wantMoves = []engine.Move{
		// king
		engine.NewMoveFromStr("b8b7"),
		engine.NewMoveFromStr("b8c7"),
		engine.NewMoveFromStr("b8c8"),
		// bishop
		engine.NewMoveFromStr("f4e5"),
		engine.NewMoveFromStr("f4d6"),
		engine.NewMoveFromStr("f4c7"),
		engine.NewMoveFromStr("f4g5"),
		engine.NewMoveFromStr("f4h6"),
		engine.NewMoveFromStr("f4g3"),
		engine.NewMoveFromStr("f4h2"),
		engine.NewMoveFromStr("f4e3"),
		engine.NewMoveFromStr("f4d2"),
		engine.NewMoveFromStr("f4c1"),
	}
	if !equalSets(gotMoves, wantMoves) {
		t.Errorf(`TestLegalMoves(%d)`, pos.Turn)
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

	pos = engine.FromFEN("k7/8/8/8/2pPp3/5B2/8/rQ2K3 w - - 0 1")

	pos.Turn = engine.White
	gotMoves = pos.LegalMoves()
	wantMoves = []engine.Move{
		// queen (along ray of pin only)
		engine.NewMoveFromStr("b1a1"),
		engine.NewMoveFromStr("b1c1"),
		engine.NewMoveFromStr("b1d1"),
		// king
		engine.NewMoveFromStr("e1d1"),
		engine.NewMoveFromStr("e1d2"),
		engine.NewMoveFromStr("e1e2"),
		engine.NewMoveFromStr("e1f2"),
		engine.NewMoveFromStr("e1f1"),
		// pawn
		engine.NewMoveFromStr("d4d5"),
		// bishop
		engine.NewMoveFromStr("f3e4"),
		engine.NewMoveFromStr("f3e2"),
		engine.NewMoveFromStr("f3d1"),
		engine.NewMoveFromStr("f3g2"),
		engine.NewMoveFromStr("f3h1"),
		engine.NewMoveFromStr("f3g4"),
		engine.NewMoveFromStr("f3h5"),
	}
	if !equalSets(gotMoves, wantMoves) {
		t.Errorf(`TestLegalMoves(%d)`, pos.Turn)
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

	pos.Turn = engine.Black
	pos.EnPassantSquare = engine.SquareD3
	gotMoves = pos.LegalMoves()
	wantMoves = []engine.Move{
		// rook
		engine.NewMoveFromStr("a1a2"),
		engine.NewMoveFromStr("a1a3"),
		engine.NewMoveFromStr("a1a4"),
		engine.NewMoveFromStr("a1a5"),
		engine.NewMoveFromStr("a1a6"),
		engine.NewMoveFromStr("a1a7"),
		engine.NewMoveFromStr("a1b1"),
		// pawn
		engine.NewMoveFromStr("c4c3"),
		engine.NewMoveFromStr("c4d3") | engine.EnPassantFlag,
		engine.NewMoveFromStr("e4f3"),
		// king
		engine.NewMoveFromStr("a8a7"),
	}
	if !equalSets(gotMoves, wantMoves) {
		t.Errorf(`TestLegalMoves(%d)`, pos.Turn)
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

func TestDoUndo(t *testing.T) {
	pos := engine.FromFEN("1k6/7P/8/8/n4p2/8/6P1/R3K3 w Q - 4 29")

	// king move
	move := engine.NewMoveFromStr("e1f2")
	pos.DoMove(move)
	wantPos := engine.FromFEN("1k6/7P/8/8/n4p2/8/5KP1/R7 b - - 5 29")
	if !pos.Equals(wantPos) || wantPos.ToFEN() != pos.ToFEN() {
		t.Errorf(`DoMove(%s)`, move.ToString())
		fmt.Println("Want: " + wantPos.ToFEN())
		fmt.Println("Got: " + pos.ToFEN())
	}

	// undo king move
	pos.UndoMove(move)
	wantPos = engine.FromFEN("1k6/7P/8/8/n4p2/8/6P1/R3K3 w Q - 4 29")
	if !pos.Equals(wantPos) || wantPos.ToFEN() != pos.ToFEN() {
		t.Errorf(`UndoMove(%s)`, move.ToString())
		fmt.Println("Want: " + wantPos.ToFEN())
		fmt.Println("Got: " + pos.ToFEN())
	}

	// promotion to rook and check
	move = engine.NewMoveFromStr("h7h8r")
	pos.DoMove(move)
	wantPos = engine.FromFEN("1k5R/8/8/8/n4p2/8/6P1/R3K3 b Q - 0 29")
	if !pos.Equals(wantPos) || wantPos.ToFEN() != pos.ToFEN() {
		t.Errorf(`DoMove(%s)`, move.ToString())
		fmt.Println("Want: " + wantPos.ToFEN())
		fmt.Println("Got: " + pos.ToFEN())
	}

	// black king evades check, white rook captures black knight
	move2 := engine.NewMoveFromStr("b8c7")
	move3 := engine.NewMoveFromStr("a1a4")
	pos.DoMove(move2)
	pos.DoMove(move3)
	wantPos = engine.FromFEN("7R/2k5/8/8/R4p2/8/6P1/4K3 b - - 0 30")
	if !pos.Equals(wantPos) || wantPos.ToFEN() != pos.ToFEN() {
		t.Errorf(`DoMove(%s %s %s)`, move.ToString(), move2.ToString(), move3.ToString())
		fmt.Println("Want: " + wantPos.ToFEN())
		fmt.Println("Got: " + pos.ToFEN())
	}

	// undo capture of black knight
	pos.UndoMove(move3)
	wantPos = engine.FromFEN("7R/2k5/8/8/n4p2/8/6P1/R3K3 w Q - 1 30")
	if !pos.Equals(wantPos) || wantPos.ToFEN() != pos.ToFEN() {
		t.Errorf(`DoMove(%s %s)`, move.ToString(), move2.ToString())
		fmt.Println("Want: " + wantPos.ToFEN())
		fmt.Println("Got: " + pos.ToFEN())
	}

	// castle queenside
	move3 = engine.NewMoveCastle(engine.WhiteQueenside)
	if !pos.MoveIsLegal(move3) {
		t.Errorf(`%s is illegal`, move3.ToString())
	}
	pos.DoMove(move3)
	wantPos = engine.FromFEN(("7R/2k5/8/8/n4p2/8/6P1/2KR4 b - - 2 30"))
	if !pos.Equals(wantPos) || wantPos.ToFEN() != pos.ToFEN() {
		t.Errorf(`DoMove(%s %s %s)`, move.ToString(), move2.ToString(), move3.ToString())
		fmt.Println("Want: " + wantPos.ToFEN())
		fmt.Println("Got: " + pos.ToFEN())
	}

	// undo all, back to original pos
	pos.UndoMove(move3)
	pos.UndoMove(move2)
	pos.UndoMove(move)
	wantPos = engine.FromFEN("1k6/7P/8/8/n4p2/8/6P1/R3K3 w Q - 4 29")
	if !pos.Equals(wantPos) || wantPos.ToFEN() != pos.ToFEN() {
		t.Errorf(`UndoMove() 3x`)
		fmt.Println("Want: " + wantPos.ToFEN())
		fmt.Println("Got: " + pos.ToFEN())
	}

	// white pawn moves forward 2 squares, allowing en passant
	move = engine.NewMoveFromStr("g2g4")
	pos.DoMove(move)
	wantPos = engine.FromFEN("1k6/7P/8/8/n4pP1/8/8/R3K3 b Q g3 0 29")
	if !pos.Equals(wantPos) || wantPos.ToFEN() != pos.ToFEN() {
		t.Errorf(`DoMove(%s)`, move.ToString())
		fmt.Println("Want: " + wantPos.ToFEN())
		fmt.Println("Got: " + pos.ToFEN())
	}

	// black captures en passant
	move2 = engine.NewMoveFromStr("f4g3") | engine.EnPassantFlag
	pos.DoMove(move2)
	wantPos = engine.FromFEN("1k6/7P/8/8/n7/6p1/8/R3K3 w Q - 0 30")
	if !pos.Equals(wantPos) || wantPos.ToFEN() != pos.ToFEN() {
		t.Errorf(`DoMove(%s %s)`, move.ToString(), move2.ToString())
		fmt.Println("Want: " + wantPos.ToFEN())
		fmt.Println("Got: " + pos.ToFEN())
	}

	// undo black's en passant capture
	pos.UndoMove(move2)
	wantPos = engine.FromFEN("1k6/7P/8/8/n4pP1/8/8/R3K3 b Q g3 0 29")
	if !pos.Equals(wantPos) || wantPos.ToFEN() != pos.ToFEN() {
		t.Errorf(`DoMove(%s)`, move2.ToString())
		fmt.Println("Want: " + wantPos.ToFEN())
		fmt.Println("Got: " + pos.ToFEN())
	}

	// undo white pawn move
	pos.UndoMove(move)
	wantPos = engine.FromFEN("1k6/7P/8/8/n4p2/8/6P1/R3K3 w Q - 4 29")
	if !pos.Equals(wantPos) || wantPos.ToFEN() != pos.ToFEN() {
		t.Errorf(`UndoMove(%s)`, move.ToString())
		fmt.Println("Want: " + wantPos.ToFEN())
		fmt.Println("Got: " + pos.ToFEN())
	}
}
