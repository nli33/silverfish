package engine_test

import (
	"testing"

	"silverfish/engine"
)

// The incrementally-maintained Position.Hash must always match a from-
// scratch recompute (engine.Hash), for every kind of move -- quiet,
// capture, castling, en passant, promotion -- and must be restored exactly
// on UndoMove. Rather than hardcoding a magic expected hash constant (which
// would break on any unrelated change to how keys are generated), this
// checks the property that actually matters: incremental == from-scratch.
func TestZobristIncrementalMatchesFromScratch(t *testing.T) {
	cases := []struct {
		name  string
		fen   string
		moves []string
	}{
		{"quiet moves", "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			[]string{"g1f3", "g8f6", "f3g1", "f6g8"}},
		{"pawn double push + capture", "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			[]string{"e2e4", "d7d5", "e4d5"}},
		// white O-O then black O-O-O (and the reverse pairing below) --
		// same-side castling twice would have White's rook, having just
		// landed on f1/d1, illegally block Black's own castling through
		// f8/d8.
		{"white kingside + black queenside castling", "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1",
			[]string{"e1g1", "e8c8"}},
		{"white queenside + black kingside castling", "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1",
			[]string{"e1c1", "e8g8"}},
		{"en passant", "4k3/8/8/8/1Pp5/8/8/4K3 b - b3 0 1",
			[]string{"c4b3"}},
		{"promotion", "8/k6P/8/8/8/8/K7/8 w - - 0 1",
			[]string{"h7h8q"}},
		{"capture-promotion", "1n5k/6P1/8/8/8/8/K7/8 w - - 0 1",
			[]string{"g7h8q"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pos := engine.FromFEN(tc.fen)
			if pos.Hash != engine.Hash(&pos) {
				t.Fatalf("initial hash mismatch: incremental=%x fromScratch=%x", pos.Hash, engine.Hash(&pos))
			}

			var moves []engine.Move
			for _, ms := range tc.moves {
				m := engine.NewMoveFromStr(ms)
				var legal engine.Move
				for _, lm := range pos.LegalMoves() {
					if lm.From() == m.From() && lm.To() == m.To() && lm.Promotion() == m.Promotion() {
						legal = lm
						break
					}
				}
				if legal == engine.Move(0) {
					t.Fatalf("%s is not legal in %s", ms, pos.ToFEN())
				}
				moves = append(moves, legal)
				pos.DoMove(legal)

				want := engine.Hash(&pos)
				if pos.Hash != want {
					t.Fatalf("after %s: incremental=%x fromScratch=%x (fen %s)", ms, pos.Hash, want, pos.ToFEN())
				}
			}

			// Undo everything and check the hash is restored exactly, at
			// every step, not just the very end.
			for i := len(moves) - 1; i >= 0; i-- {
				pos.UndoMove(moves[i])
				want := engine.Hash(&pos)
				if pos.Hash != want {
					t.Fatalf("after undoing move %d: incremental=%x fromScratch=%x", i, pos.Hash, want)
				}
			}
		})
	}
}
