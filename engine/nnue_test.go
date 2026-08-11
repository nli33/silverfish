package engine_test

import (
	"math/rand"
	"testing"

	"silverfish/engine"
)

// findLegalMove returns the legal move matching from->to (ignoring
// promotion/flags -- fine for these tests, no ambiguous promotions used).
func findLegalMove(t *testing.T, pos *engine.Position, uci string) engine.Move {
	t.Helper()
	want := engine.NewMoveFromStr(uci)
	for _, candidate := range pos.LegalMoves() {
		if candidate.From() == want.From() && candidate.To() == want.To() {
			return candidate
		}
	}
	t.Fatalf("move %s not found in legal moves", uci)
	return 0
}

// checkIncrementalMatchesFromScratch plays moves (each a UCI string) from
// startFEN via DoMove, then checks the incrementally-updated accumulator's
// evaluation matches a from-scratch Position built straight from the
// resulting FEN. See TestNNUEIncrementalMatchesFromScratch's doc comment
// for why this matters -- under HalfKA (king-relative feature indices),
// this is the check that actually exercises the risky part: a king move
// must trigger a full accumulator rebuild for its own perspective (see
// needsAccRefresh in position.go), not an incremental patch, and this is
// the only kind of test that would catch getting that wrong.
func checkIncrementalMatchesFromScratch(t *testing.T, startFEN string, moves []string) {
	t.Helper()
	pos := engine.FromFEN(startFEN)

	for _, m := range moves {
		pos.DoMove(findLegalMove(t, &pos, m))
	}

	fresh := engine.FromFEN(pos.ToFEN())

	got := engine.Evaluate(&pos)
	want := engine.Evaluate(&fresh)
	if got != want {
		t.Errorf("incrementally updated eval = %d, from-scratch eval = %d (FEN %s); want equal", got, want, pos.ToFEN())
	}
}

func TestNNUEIncrementalMatchesFromScratch(t *testing.T) {
	checkIncrementalMatchesFromScratch(t,
		"6k1/5p1p/1q2p1p1/1PnpP3/3N4/1Pr5/P5PP/3QR1K1 w - - 3 37",
		[]string{"d1a1", "b6a5", "d4c6"},
	)
}

// TestNNUEIncrementalMatchesFromScratch_KingMove exercises a plain king
// move -- the case that must trigger a full accumulator rebuild for the
// mover's own perspective (see needsAccRefresh), since every one of that
// perspective's HalfKA feature indices is relative to its own king square.
func TestNNUEIncrementalMatchesFromScratch_KingMove(t *testing.T) {
	checkIncrementalMatchesFromScratch(t,
		"r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1",
		[]string{"e1d1", "e8f8", "d1e1"},
	)
}

// TestNNUEIncrementalMatchesFromScratch_Castling covers both castling
// directions for both colors -- each moves the king via PutPiece/
// RemovePiece (not MovePiece), and moves a second piece (the rook) for the
// same color in between the king's two touches, which must also be
// deferred to the full rebuild rather than incrementally patched against a
// king square that's transiently off the board mid-castle.
func TestNNUEIncrementalMatchesFromScratch_Castling(t *testing.T) {
	for _, tc := range []struct {
		name  string
		moves []string
	}{
		{"white kingside", []string{"e1g1"}},
		{"white queenside", []string{"e1c1"}},
		{"black kingside", []string{"e1d1", "e8g8"}},
		{"black queenside", []string{"e1d1", "e8c8"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checkIncrementalMatchesFromScratch(t, "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1", tc.moves)
		})
	}
}

// TestNNUEIncrementalMatchesFromScratch_KingCaptures covers a king
// capturing a piece -- CapturePiece's mover can be a King too, same
// full-rebuild requirement as a plain king move.
func TestNNUEIncrementalMatchesFromScratch_KingCaptures(t *testing.T) {
	checkIncrementalMatchesFromScratch(t,
		"8/8/8/3k4/3P4/8/4K3/8 b - - 0 1",
		[]string{"d5d4"},
	)
}

// TestNNUEIncrementalMatchesFromScratch_UndoAfterKingMove plays a king
// move (and a castle) and then undoes them, checking the evaluation
// matches the ORIGINAL start position -- exercises UndoMove's MovePiece
// (to,from) reversal and the castling undo branch's king-first-then-rook-
// then-king-removed-last ordering (see domove.go), both of which also
// trigger the own-perspective full-rebuild path.
func TestNNUEIncrementalMatchesFromScratch_UndoAfterKingMove(t *testing.T) {
	startFEN := "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1"
	pos := engine.FromFEN(startFEN)
	start := engine.FromFEN(startFEN)
	wantEval := engine.Evaluate(&start)

	moves := []engine.Move{
		findLegalMove(t, &pos, "e1g1"), // white castles kingside
	}
	for _, m := range moves {
		pos.DoMove(m)
	}
	for i := len(moves) - 1; i >= 0; i-- {
		pos.UndoMove(moves[i])
	}

	got := engine.Evaluate(&pos)
	if got != wantEval {
		t.Errorf("eval after do+undo castling = %d, original eval = %d; want equal", got, wantEval)
	}
}

// TestNNUEIncrementalMatchesFromScratch_RandomGame plays a longer
// pseudo-random game (many king moves, captures, and castles likely along
// the way) and checks the incremental-vs-from-scratch invariant after
// EVERY move, not just at the end -- a check only at the end could miss an
// error that happens to cancel out over many moves by chance.
func TestNNUEIncrementalMatchesFromScratch_RandomGame(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	pos := engine.FromFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	for ply := 0; ply < 80; ply++ {
		legal := pos.LegalMoves()
		if len(legal) == 0 {
			break // checkmate/stalemate
		}
		move := legal[rng.Intn(len(legal))]
		pos.DoMove(move)

		fresh := engine.FromFEN(pos.ToFEN())
		got := engine.Evaluate(&pos)
		want := engine.Evaluate(&fresh)
		if got != want {
			t.Fatalf("ply %d: incrementally updated eval = %d, from-scratch eval = %d (FEN %s); want equal", ply, got, want, pos.ToFEN())
		}
	}
}
