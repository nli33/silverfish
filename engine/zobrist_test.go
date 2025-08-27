package engine_test

import (
	"fmt"
	"silverfish/engine"
	"testing"
)

func TestZobrist(t *testing.T) {
	pos := engine.FromFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	expectedHash := uint64(0xe5578482be7dd47c)
	if pos.Hash != expectedHash {
		t.Errorf("Zobrist hashing: got %x, want %x", pos.Hash, expectedHash)
	}

	pos.DoMove(engine.NewMoveFromStr("e2e4"))
	pos.UndoMove(engine.NewMoveFromStr("e2e4"))
	expectedHash = uint64(0xe5578482be7dd47c)
	if pos.Hash != expectedHash {
		t.Errorf("Zobrist hashing: got %x, want %x", pos.Hash, expectedHash)
	}

	pos.DoMove(engine.NewMoveFromStr("e2e4"))
	pos2 := engine.FromFEN("rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1")
	if pos2.Hash != pos.Hash {
		t.Errorf("Zobrist hashing: pos = %x, pos2 = %x", pos.Hash, pos2.Hash)
		fmt.Println(pos.ToFEN())
		fmt.Println(pos2.ToFEN())
	}

	// promotion
	pos = engine.FromFEN("8/k6P/8/8/8/8/K7/8 w - - 0 1")
	pos.DoMove(engine.NewMoveFromStr("h7h8q"))
	pos2 = engine.FromFEN("7Q/k7/8/8/8/8/K7/8 b - - 0 1")
	if pos2.Hash != pos.Hash {
		t.Errorf("Zobrist hashing: pos = %x, pos2 = %x", pos.Hash, pos2.Hash)
		fmt.Println(pos.ToFEN())
		fmt.Println(pos2.ToFEN())
	}

	// castling
	pos = engine.FromFEN("4k3/8/8/8/8/8/8/4K2R w K - 0 1")
	pos.DoMove(engine.NewMoveCastle(engine.WhiteKingside))
	pos2 = engine.FromFEN("4k3/8/8/8/8/8/8/5RK1 b - - 0 1")
	if pos2.Hash != pos.Hash {
		t.Errorf("Zobrist hashing: pos = %x, pos2 = %x", pos.Hash, pos2.Hash)
		fmt.Println(pos.ToFEN())
		fmt.Println(pos2.ToFEN())
	}

	// en passant
	pos = engine.FromFEN("4k3/8/8/8/1Pp5/8/8/4K3 b - b3 0 1")
	pos.DoMove(engine.NewMoveFromStr("c4b3") | engine.EnPassantFlag)
	pos2 = engine.FromFEN("4k3/8/8/8/8/1p6/8/4K3 w - - 0 1")
	if pos2.Hash != pos.Hash {
		t.Errorf("Zobrist hashing: pos = %x, pos2 = %x", pos.Hash, pos2.Hash)
		fmt.Println(pos.ToFEN())
		fmt.Println(pos2.ToFEN())
	}
}
