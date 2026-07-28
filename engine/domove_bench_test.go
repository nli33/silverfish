package engine_test

import (
	"testing"

	"silverfish/engine"
)

// Isolates DoMove/UndoMove cost specifically (as opposed to perft, which
// also spends time in move generation) -- useful for measuring the direct
// overhead of anything added to the make/unmake path, like incremental
// Zobrist hashing.
func BenchmarkDoUndoMove(b *testing.B) {
	pos := engine.FromFEN("r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1")
	moveList := engine.GenMoves(&pos, engine.BB_Full)
	var moves []engine.Move
	for i := uint8(0); i < moveList.Count; i++ {
		m := moveList.Moves[i]
		if pos.MoveIsLegal(m) {
			moves = append(moves, m)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m := moves[i%len(moves)]
		pos.DoMove(m)
		pos.UndoMove(m)
	}
}
