package engine_test

import (
	"bytes"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"silverfish/engine"
)

// captureStdout redirects os.Stdout for the duration of fn and returns
// everything written to it -- UciInfo prints directly to stdout, so this is
// the only way to observe the search-progress messages it emits.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	old := os.Stdout
	os.Stdout = w

	fn()

	os.Stdout = old
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

// parseUciInfoLine extracts the fields TestUciInfo* care about from one
// "info ..." line. hasScore is false for the internal periodic ping, which
// deliberately omits the score field.
func parseUciInfoLine(line string) (depth int, score int32, isMate bool, hasScore bool) {
	fields := strings.Fields(line)
	for i, f := range fields {
		switch f {
		case "depth":
			if i+1 < len(fields) {
				d, _ := strconv.Atoi(fields[i+1])
				depth = d
			}
		case "cp":
			if i+1 < len(fields) {
				v, _ := strconv.Atoi(fields[i+1])
				score = int32(v)
				hasScore = true
			}
		case "mate":
			if i+1 < len(fields) {
				v, _ := strconv.Atoi(fields[i+1])
				score = int32(v)
				hasScore = true
				isMate = true
			}
		}
	}
	return
}

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

// Even under a time budget so tight it expires mid-way through the very
// first iterative-deepening iteration, Search() must still return a legal
// move -- never the zero-value null move, which would be sent straight to
// the GUI as an illegal "bestmove".
func TestSearchTightTimeLimitReturnsLegalMove(t *testing.T) {
	fen := "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1"
	pos := engine.FromFEN(fen)

	search := engine.Search{
		MaxDepth:  engine.InfiniteDepth,
		TimeLimit: 0,
	}
	search.Init(&pos)

	_, bestMove := search.Search()

	if bestMove == engine.Move(0) {
		t.Fatalf("Search() returned a null move under a zero time limit")
	}
	if !pos.MoveIsLegal(bestMove) {
		t.Errorf("Search() returned illegal move %s", bestMove.ToString())
	}
}

// A deeper iterative-deepening depth that only partway completes (checked
// some but not all root moves before the clock ran out) must NOT overwrite
// a shallower depth that fully completed -- the complete result is strictly
// more reliable than an incomplete deeper one that may not have reached the
// true best move yet. Regression test for a real bug: an earlier version of
// the timeout fix committed any partial progress unconditionally, which at
// real time controls (where the deepest attempted iteration almost always
// times out mid-way) made the engine routinely discard a solid, fully-
// searched shallow result in favor of whatever the first couple of moves at
// the next depth happened to be -- caught by an SPRT run scoring 0-97 against
// the pre-fix engine.
func TestSearchTimeoutKeepsLastFullyCompletedDepth(t *testing.T) {
	fen := "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1"

	runToCompletion := func(maxDepth int) (int32, engine.Move, time.Duration) {
		pos := engine.FromFEN(fen)
		search := engine.Search{MaxDepth: maxDepth, TimeLimit: engine.InfiniteMovetime}
		search.Init(&pos)
		start := time.Now()
		score, move := search.Search()
		return score, move, time.Since(start)
	}

	wantScore, wantMove, elapsed2 := runToCompletion(2)
	_, _, elapsed3 := runToCompletion(3)

	if elapsed3 <= elapsed2 {
		t.Skip("depth 3 didn't take meaningfully longer than depth 2 on this machine; can't construct a reliable partial-depth timeout")
	}
	// Comfortably past depth 2's own completion time, but well short of
	// depth 3's -- gives the timed run room to start depth 3, check a few
	// moves, and time out mid-way.
	budget := elapsed2 + (elapsed3-elapsed2)/3

	pos := engine.FromFEN(fen)
	search := engine.Search{MaxDepth: 8, TimeLimit: budget}
	search.Init(&pos)
	gotScore, gotMove := search.Search()

	if gotMove != wantMove || gotScore != wantScore {
		t.Errorf("timed search (budget %s, between depth-2 %s and depth-3 %s) = (%d, %s), want the fully-completed depth-2 result (%d, %s)",
			budget, elapsed2, elapsed3, gotScore, gotMove.ToString(), wantScore, wantMove.ToString())
	}
}

// Documents (rather than "fixes") the one case where Search() has no legal
// move to give: a root position that's already checkmate. There's nothing
// to return here, so this pins down the current, well-defined behavior
// (null move, -Infinity score) so it doesn't change by accident.
func TestSearchTerminalRootPositionReturnsNullMove(t *testing.T) {
	pos := engine.FromFEN("R5k1/5ppp/8/8/8/8/8/6K1 b - - 1 1")
	search := engine.Search{
		MaxDepth:  3,
		TimeLimit: engine.InfiniteMovetime,
	}
	search.Init(&pos)

	score, bestMove := search.Search()

	if bestMove != engine.Move(0) {
		t.Errorf("Search() on a checkmated root = %s, want the null move (no legal move exists)", bestMove.ToString())
	}
	if score != -engine.Infinity {
		t.Errorf("Search() on a checkmated root score = %d, want -Infinity", score)
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

// Mate scores are ply-relative, so a TT entry storing one must be re-based
// (via ScoreToTT/ScoreFromTT) when reused from a different ply -- otherwise
// the reported mate distance corrupts. This runs a known mate-in-1 across
// several MaxDepths so the position is reached, cached, and reprobed at
// different plies (and iterations) within iterative-deepening searches,
// and checks the move found is still an immediate, genuine checkmate every
// time -- not just a mate-scored move that happens to not actually mate,
// which is exactly what a corrupted ply rebasing would produce.
func TestSearchMateDistanceCorrectThroughTT(t *testing.T) {
	engine.ClearTT()

	fen := "6k1/5ppp/8/8/8/8/8/R5K1 w - - 0 1" // mate in 1 (Ra8#)
	for _, maxDepth := range []int{2, 3, 4, 5} {
		pos := engine.FromFEN(fen)
		search := engine.Search{MaxDepth: maxDepth, TimeLimit: 4 * time.Second}
		search.Init(&pos)

		score, bestMove := search.Search()
		if score < engine.Infinity-10 {
			t.Fatalf("MaxDepth=%d: score = %d, want a mate score near +Infinity", maxDepth, score)
		}

		pos.DoMove(bestMove)
		if len(pos.LegalMoves()) != 0 || pos.Checkers(pos.Turn) == 0 {
			t.Errorf("MaxDepth=%d: move %s is not checkmate (mate distance likely corrupted by a stale TT ply)", maxDepth, bestMove.ToString())
		}
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

// UciInfo's search-progress messages must report each completed depth's
// real score exactly once -- not repeated per root move with a stale value
// left over from the previous depth (the root-loop bug), and the periodic
// internal-node progress ping must never carry a score field (that value is
// a local negamax number from deep in the tree, not the root-relative score
// UCI's `score` field is supposed to report).
func TestUciInfoReportsScoreOncePerDepth(t *testing.T) {
	fen := "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1"
	pos := engine.FromFEN(fen)
	search := engine.Search{MaxDepth: 4, TimeLimit: engine.InfiniteMovetime}
	search.Init(&pos)

	var finalScore int32
	output := captureStdout(t, func() {
		finalScore, _ = search.Search()
	})

	seenAtDepth := map[int]int{}
	sawUnscoredPing := false
	var lastScore int32
	var lastDepth int

	for _, line := range strings.Split(output, "\n") {
		if !strings.HasPrefix(line, "info") {
			continue
		}
		depth, score, isMate, hasScore := parseUciInfoLine(line)
		if !hasScore {
			sawUnscoredPing = true
			continue
		}
		if isMate {
			t.Fatalf("unexpected mate score in a non-mate position: %q", line)
		}
		seenAtDepth[depth]++
		lastScore = score
		lastDepth = depth
	}

	for d, count := range seenAtDepth {
		if count != 1 {
			t.Errorf("depth %d: got %d scored info lines, want exactly 1", d, count)
		}
	}
	if lastDepth != 4 {
		t.Errorf("last scored info line was depth %d, want 4 (MaxDepth)", lastDepth)
	}
	if lastScore != finalScore {
		t.Errorf("last scored info line score = %d, want %d (Search()'s returned score)", lastScore, finalScore)
	}
	if !sawUnscoredPing {
		t.Errorf("expected at least one unscored internal-node progress ping (search.Nodes should exceed engine.NodeReportInterval at this depth)")
	}
}

// A mate score must be reported as UCI's "score mate N" (moves to mate),
// not "score cp <huge centipawn number>".
func TestUciInfoReportsMateFormat(t *testing.T) {
	pos := engine.FromFEN("6k1/5ppp/8/8/8/8/8/R5K1 w - - 0 1")
	search := engine.Search{MaxDepth: 2, TimeLimit: 4 * time.Second}
	search.Init(&pos)

	output := captureStdout(t, func() {
		search.Search()
	})

	var lastIsMate bool
	var lastMateValue int32
	var sawScoredLine bool
	for _, line := range strings.Split(output, "\n") {
		if !strings.HasPrefix(line, "info") {
			continue
		}
		_, score, isMate, hasScore := parseUciInfoLine(line)
		if !hasScore {
			continue
		}
		sawScoredLine = true
		lastIsMate = isMate
		lastMateValue = score
	}

	if !sawScoredLine {
		t.Fatalf("no scored info line found in output:\n%s", output)
	}
	if !lastIsMate {
		t.Errorf("mate-in-1 position: last scored info line used score cp, want score mate")
	}
	if lastMateValue != 1 {
		t.Errorf("score mate %d, want score mate 1", lastMateValue)
	}
}

// Search must recognize that a candidate move recreating a position already
// reached earlier in the game is a draw, and prefer a genuinely winning
// continuation over it -- not just find *a* legal move that happens to
// avoid the repeat by luck. Primes real game history (via DoMove, exactly
// like main.go replays a UCI "position ... moves ..." command) with a
// K+R vs K waiting-move sequence that returns to the exact starting
// position, then re-offers the same waiting move (h1h2) as a candidate.
func TestSearchAvoidsKnownRepetition(t *testing.T) {
	findMove := func(pos *engine.Position, uci string) engine.Move {
		want := engine.NewMoveFromStr(uci)
		for _, lm := range pos.LegalMoves() {
			if lm.From() == want.From() && lm.To() == want.To() {
				return lm
			}
		}
		return engine.Move(0)
	}

	pos := engine.FromFEN("4k3/8/8/8/2K5/8/8/7R w - - 0 1")
	for _, uci := range []string{"h1h2", "e8e7", "h2h1", "e7e8"} {
		m := findMove(&pos, uci)
		if m == engine.Move(0) {
			t.Fatalf("priming move %s not legal in %s", uci, pos.ToFEN())
		}
		pos.DoMove(m)
	}

	// h1h2 recreates the position from right after the first h1h2 -- same
	// side to move, same everything. Confirmed at the Position level too
	// (TestIsRepetition covers the mechanism directly); this checks it
	// actually changes Search()'s behavior.
	repeat := findMove(&pos, "h1h2")
	clone := pos.Clone()
	clone.DoMove(repeat)
	if !clone.IsRepetition() {
		t.Fatalf("test setup bug: h1h2 was expected to recreate a prior position")
	}

	for _, depth := range []int{1, 2, 3} {
		p := pos.Clone()
		search := engine.Search{MaxDepth: depth, TimeLimit: 4 * time.Second}
		search.Init(&p)
		score, move := search.Search()

		if move == repeat {
			t.Errorf("depth %d: chose h1h2, the known repetition, when other winning moves are available", depth)
		}
		// A rook-up win should score decisively positive, not be dragged
		// toward 0 just because one of the candidate moves is a draw.
		if score < 200 {
			t.Errorf("depth %d: score = %d, want a decisively positive (rook-up) score", depth, score)
		}
	}
}
