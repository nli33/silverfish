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

func TestIsRepetition(t *testing.T) {
	pos := engine.FromFEN("4k3/8/8/8/8/8/8/4K3 w - - 0 1")

	play := func(uci string) engine.Move {
		m := engine.NewMoveFromStr(uci)
		var legal engine.Move
		for _, lm := range pos.LegalMoves() {
			if lm.From() == m.From() && lm.To() == m.To() {
				legal = lm
				break
			}
		}
		if legal == engine.Move(0) {
			t.Fatalf("%s not legal in %s", uci, pos.ToFEN())
		}
		pos.DoMove(legal)
		return legal
	}

	if pos.IsRepetition() {
		t.Fatalf("starting position flagged as a repetition")
	}

	play("e1d1") // Kd1
	if pos.IsRepetition() {
		t.Errorf("no repetition yet after one king move")
	}
	play("e8d8") // Kd8
	if pos.IsRepetition() {
		t.Errorf("no repetition yet")
	}
	play("d1e1") // Ke1 -- back to a position with the same side to move as start, but not equal (black king moved)
	if pos.IsRepetition() {
		t.Errorf("position is not actually equal to any prior position yet")
	}
	play("d8e8") // Ke8 -- now identical to the starting position, with white to move again
	if !pos.IsRepetition() {
		t.Errorf("expected a repetition of the starting position")
	}
}

// A capture or pawn move is irreversible, so no repetition can span across
// one -- IsRepetition must not report a false positive by scanning past it.
func TestIsRepetitionBoundedByIrreversibleMove(t *testing.T) {
	pos := engine.FromFEN("4k3/8/8/8/8/8/4P3/4K3 w - - 0 1")

	play := func(uci string) {
		m := engine.NewMoveFromStr(uci)
		var legal engine.Move
		for _, lm := range pos.LegalMoves() {
			if lm.From() == m.From() && lm.To() == m.To() {
				legal = lm
				break
			}
		}
		if legal == engine.Move(0) {
			t.Fatalf("%s not legal in %s", uci, pos.ToFEN())
		}
		pos.DoMove(legal)
	}

	play("e1d1") // Kd1
	play("e8d8") // Kd8
	play("e2e4") // pawn push -- irreversible, resets Rule50
	play("d8e8") // Ke8
	play("d1e1") // Ke1 -- side to move matches the position right after the pawn push, but that's the
	// only candidate and it's on the far side of the irreversible move, so this must not be flagged.

	if pos.IsRepetition() {
		t.Errorf("false-positive repetition across an irreversible (pawn) move")
	}
}

// Guards against a real historical bug (the dead `tt` branch): its
// InitZobrist only seeded PieceSqKeys rows 0-5, leaving black's rows (6-11)
// permanently zero, so every black piece contributed nothing to the hash
// and positions differing only in black's material/placement hashed
// identically. The incremental-vs-from-scratch test above can't catch this
// (it would still agree with itself even if every key were zero), so this
// checks the property that actually matters for a transposition table: no
// key is zero, and two positions differing only in a black piece hash
// differently.
func TestZobristKeysAreNonZeroAndDistinctForBlackPieces(t *testing.T) {
	for piece := 0; piece < 12; piece++ {
		for sq := engine.SquareA1; sq <= engine.SquareH8; sq++ {
			if engine.PieceSqKeys[piece][sq] == 0 {
				t.Fatalf("PieceSqKeys[%d][%d] is zero (uninitialized key)", piece, sq)
			}
		}
	}

	withBlackKnight := engine.FromFEN("4k3/8/8/8/8/8/8/n3K3 w - - 0 1")
	withoutBlackKnight := engine.FromFEN("4k3/8/8/8/8/8/8/4K3 w - - 0 1")
	if withBlackKnight.Hash == withoutBlackKnight.Hash {
		t.Fatalf("positions differing only by a black knight hash identically (%x)", withBlackKnight.Hash)
	}

	blackKnightA1 := engine.FromFEN("4k3/8/8/8/8/8/8/n3K3 w - - 0 1")
	blackKnightH1 := engine.FromFEN("4k3/8/8/8/8/8/8/4K2n w - - 0 1")
	if blackKnightA1.Hash == blackKnightH1.Hash {
		t.Fatalf("a black knight on different squares hashes identically (%x)", blackKnightA1.Hash)
	}
}
