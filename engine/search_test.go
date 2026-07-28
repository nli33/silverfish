package engine_test

import (
	"testing"
	"time"

	"silverfish/engine"
)

// Structural invariants that must hold regardless of how the evaluation is
// tuned: the search returns a legal move, doesn't mutate the position it
// started from, and explores at least one node.
func TestSearchInvariants(t *testing.T) {
	fen := "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1"
	pos := engine.FromFEN(fen)
	before := pos.ToFEN()

	search := engine.Search{
		MaxDepth:  3,
		TimeLimit: engine.InfiniteMovetime,
	}
	search.Init(&pos)

	_, bestMove := search.Search()

	if bestMove == engine.Move(0) {
		t.Fatalf("Search() returned a null move")
	}
	if !pos.MoveIsLegal(bestMove) {
		t.Errorf("Search() returned illegal move %s", bestMove.ToString())
	}
	if pos.ToFEN() != before {
		t.Errorf("Search() mutated the position: before %q, after %q", before, pos.ToFEN())
	}
	if search.Nodes == 0 {
		t.Errorf("Search() explored 0 nodes")
	}
}

// A fixed-depth search with no time pressure must be reproducible.
func TestSearchReproducible(t *testing.T) {
	fen := "r1bqkbnr/pppp1ppp/2n5/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq - 2 3"

	run := func() (int32, engine.Move) {
		pos := engine.FromFEN(fen)
		search := engine.Search{
			MaxDepth:  3,
			TimeLimit: engine.InfiniteMovetime,
		}
		search.Init(&pos)
		return search.Search()
	}

	wantScore, wantMove := run()
	for i := 0; i < 3; i++ {
		gotScore, gotMove := run()
		if gotScore != wantScore || gotMove != wantMove {
			t.Errorf("run %d: got (%d, %s), want (%d, %s)", i, gotScore, gotMove.ToString(), wantScore, wantMove.ToString())
		}
	}
}

// Quiescence must recognize checkmate reached via a capture, not silently
// fall back to a stand-pat-derived evaluation. MaxDepth=1 means the position
// right after White's move is evaluated by Quiescence itself (depth 0), so
// this specifically exercises Quiescence's own mate detection, not
// alphaBetaInner's. g1g7 (Qxg7#) is a real forced mate: the queen captures
// the only black pawn with check, defended by the bishop on h6, and the
// black king has no flight square or recapture -- verified with
// tools/chess_check.py rather than by hand.
func TestQuiescenceDetectsCheckmate(t *testing.T) {
	pos := engine.FromFEN("7k/6p1/7B/8/8/8/8/4K1Q1 w - - 0 1")
	search := engine.Search{
		MaxDepth:  1,
		TimeLimit: 4 * time.Second,
	}
	search.Init(&pos)

	score, bestMove := search.Search()
	if score < engine.Infinity-10 {
		t.Errorf("score = %d, want a mate score near +Infinity", score)
	}
	if bestMove.ToString() != "g1g7" {
		t.Errorf("bestMove = %s, want g1g7 (Qxg7#)", bestMove.ToString())
	}
}

// Forced mates: any correct search must find them, independent of eval tuning.
func TestSearchFindsMateInOne(t *testing.T) {
	cases := []struct {
		name string
		fen  string
	}{
		{"back rank mate", "6k1/5ppp/8/8/8/8/8/R5K1 w - - 0 1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pos := engine.FromFEN(tc.fen)
			search := engine.Search{
				MaxDepth:  2,
				TimeLimit: 4 * time.Second,
			}
			search.Init(&pos)

			score, bestMove := search.Search()
			if score < engine.Infinity-10 {
				t.Errorf("%s: score = %d, want a mate score near +Infinity", tc.name, score)
			}

			pos.DoMove(bestMove)
			if len(pos.LegalMoves()) != 0 || pos.Checkers(pos.Turn) == 0 {
				t.Errorf("%s: move %s is not checkmate", tc.name, bestMove.ToString())
			}
		})
	}
}

// MVV-LVA move ordering must rank capturing a more valuable piece with a
// cheaper one above the reverse: PxQ (pawn takes a queen) is a great trade
// and must outscore QxP (queen takes a pawn), a poor one, even though both
// are available in the same position.
func TestScoreMovesRanksCapturesByValue(t *testing.T) {
	// White pawn e3 attacks the black queen on d4 (PxQ);
	// white queen h4 attacks the black pawn on g5 (QxP).
	pos := engine.FromFEN("4k3/8/8/6p1/3q3Q/4P3/8/4K3 w - - 0 1")
	moveList := engine.GenMoves(&pos, engine.BB_Full)
	engine.ScoreMoves(&pos, &moveList)

	var pxq, qxp engine.Move
	for i := uint8(0); i < moveList.Count; i++ {
		move := moveList.Moves[i]
		switch {
		case move.From() == engine.SquareE3 && move.To() == engine.SquareD4:
			pxq = move
		case move.From() == engine.SquareH4 && move.To() == engine.SquareG5:
			qxp = move
		}
	}
	if pxq == engine.Move(0) {
		t.Fatalf("expected exd4 (PxQ) to be a generated move")
	}
	if qxp == engine.Move(0) {
		t.Fatalf("expected Qxg5 (QxP) to be a generated move")
	}
	if pxq.Score() <= qxp.Score() {
		t.Errorf("PxQ (score %d) should outrank QxP (score %d)", pxq.Score(), qxp.Score())
	}
}

// Mate-distance scoring: a forced mate found closer to the root must score
// strictly higher than the same kind of mate found deeper in the tree, so
// the engine prefers the faster mate when it has a choice.
func TestSearchPrefersFasterMate(t *testing.T) {
	mateIn1 := engine.FromFEN("6k1/5ppp/8/8/8/8/8/R5K1 w - - 0 1")
	searchIn1 := engine.Search{MaxDepth: 3, TimeLimit: 4 * time.Second}
	searchIn1.Init(&mateIn1)
	scoreIn1, _ := searchIn1.Search()

	mateIn2 := engine.FromFEN("k7/8/2K5/8/8/8/8/7Q w - - 0 1")
	searchIn2 := engine.Search{MaxDepth: 5, TimeLimit: 4 * time.Second}
	searchIn2.Init(&mateIn2)
	scoreIn2, _ := searchIn2.Search()

	if scoreIn1 <= scoreIn2 {
		t.Errorf("mate-in-1 score (%d) should exceed mate-in-2 score (%d)", scoreIn1, scoreIn2)
	}
	if scoreIn2 < engine.Infinity-10 {
		t.Errorf("mate-in-2 score = %d, want a mate score near +Infinity", scoreIn2)
	}
}

// A free, undefended piece with no counterplay must be captured by any
// correct search -- independent of eval tuning, since forfeiting it is a
// clear material loss under any reasonable evaluation.
func TestSearchCapturesHangingPiece(t *testing.T) {
	pos := engine.FromFEN("4k3/8/8/7q/5N2/8/8/4K3 w - - 0 1")
	search := engine.Search{
		MaxDepth:  3,
		TimeLimit: 4 * time.Second,
	}
	search.Init(&pos)

	_, bestMove := search.Search()
	if bestMove.To() != engine.SquareH5 {
		t.Errorf("Search() = %s, want a move to h5 capturing the undefended queen", bestMove.ToString())
	}
}
