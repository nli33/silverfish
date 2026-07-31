package engine

// MaxKillerPly bounds the killers/history-indexed ply depth. Search depth is
// user-controllable (`go depth N`), so this is a defensive cap, not an
// expected real depth -- ply beyond it just skips killer lookups/stores
// rather than growing the table unboundedly.
const MaxKillerPly = 128

// isQuietMove reports whether move is neither a capture nor a promotion.
// Must be called with pos in the state move would be played from (i.e.
// before DoMove, or after the matching UndoMove) -- it inspects the
// pre-move board to find move's victim, if any. En passant's victim isn't
// on move's own To() square, so it's special-cased directly.
func isQuietMove(pos *Position, move Move) bool {
	if move.IsPromotion() || move.IsEnPassant() {
		return false
	}
	_, victim := pos.GetSquare(move.To())
	return victim == NoPiece
}

// recordKiller stores move as the newest killer at ply, keeping the
// previous killers[ply][0] as the second slot (so the two most recent
// distinct cutoff moves at this ply are remembered). A no-op if move is
// already the top killer here, to avoid the two slots collapsing to
// duplicates of the same move.
func (search *Search) recordKiller(move Move, ply int) {
	if ply >= MaxKillerPly {
		return
	}
	m := move & 0xffff
	if search.killers[ply][0]&0xffff == m {
		return
	}
	search.killers[ply][1] = search.killers[ply][0]
	search.killers[ply][0] = m
}

// recordHistory increments the history weight for a quiet move that caused
// a beta cutoff, by depth^2. Deliberately unbounded/unclamped here --
// scoreQuiets clamps at read time (a move's score field is only 16 bits, so
// the raw accumulator must never be written into it directly).
func (search *Search) recordHistory(move Move, depth int) {
	search.history[search.Pos.Turn][move.From()][move.To()] += int32(depth * depth)
}

// maxQuietScore is the highest score scoreQuiets ever assigns, kept below
// captureScoreFloor (see ScoreMoves) so a killer or history-favored quiet
// can never be ordered ahead of a capture, regardless of that capture's
// SEE value.
const maxQuietScore = 9

// scoreQuiets scores every not-yet-scored (quiet) move in moveList using
// search's killers and history tables, leaving moves ScoreMoves already
// scored (captures/promotions, all nonzero) untouched. Killers at this ply
// take the top two quiet scores; everything else gets a history-derived
// score clamped to fit below them.
func (search *Search) scoreQuiets(moveList *MoveList, ply int) {
	var killer1, killer2 Move
	if ply < MaxKillerPly {
		killer1, killer2 = search.killers[ply][0], search.killers[ply][1]
	}

	for i := 0; i < int(moveList.Count); i++ {
		move := &moveList.Moves[i]
		if move.Score() != 0 {
			continue
		}

		masked := *move & 0xffff
		switch {
		case killer1 != 0 && masked == killer1:
			move.GiveScore(maxQuietScore)
		case killer2 != 0 && masked == killer2:
			move.GiveScore(maxQuietScore - 1)
		default:
			h := search.history[search.Pos.Turn][move.From()][move.To()]
			if h > maxQuietScore-2 {
				h = maxQuietScore - 2
			}
			move.GiveScore(int(h))
		}
	}
}

// captureScoreFloor/captureScoreCeiling bound the score a capture can get
// from its (clamped) SEE value, chosen so every capture -- winning, even,
// or losing -- always scores above maxQuietScore. This keeps captures'
// existing "always tried before quiets" placement from the old MVV-LVA
// scheme; only the ranking *among* captures changes, from MVV-LVA's crude
// victim/attacker-value heuristic to SEE's actual exchange outcome. A
// separate "deprioritize clearly-losing captures below quiets" tier is a
// distinct idea with its own tradeoffs, deliberately left for a later,
// separately-SPRT'd change rather than bundled in here.
const (
	seeClampMin         = -2000
	seeClampMax         = 4000
	captureScoreFloor   = maxQuietScore + 1
	captureScoreCeiling = captureScoreFloor + (seeClampMax - seeClampMin)
)

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// ScoreMoves scores every capture (including en passant, whose victim
// isn't on the destination square) in moveList using SEE. Non-captures
// (including quiet promotions) are left unscored here, same as before --
// they're picked up by scoreQuiets. Used for the main search (root and
// alphaBetaInner) only -- see ScoreMovesFast for why quiescence uses a
// cheaper heuristic instead.
func ScoreMoves(pos *Position, moveList *MoveList) {
	for i := 0; i < int(moveList.Count); i++ {
		move := &moveList.Moves[i]
		_, victim := pos.GetSquare(move.To())
		if victim == NoPiece && !move.IsEnPassant() {
			continue
		}
		value := clampInt(SEE(pos, *move), seeClampMin, seeClampMax)
		move.GiveScore(captureScoreFloor + (value - seeClampMin))
	}
}

// Indexed by the real piece constants (Pawn=0, Knight=1, Bishop=2, Rook=3,
// Queen=4, King=5, NoPiece=6) on both axes. Row = victim, column = attacker;
// higher score for a more valuable victim taken by a cheaper attacker. Used
// only by ScoreMovesFast (quiescence) -- the main search uses real SEE
// instead (see ScoreMoves).
var MvvLva = [7][7]int{
	{15, 14, 13, 12, 11, 10, 0}, // victim P, attacker P, N, B, R, Q, K, None
	{25, 24, 23, 22, 21, 20, 0}, // victim N, attacker P, N, B, R, Q, K, None
	{35, 34, 33, 32, 31, 30, 0}, // victim B, attacker P, N, B, R, Q, K, None
	{45, 44, 43, 42, 41, 40, 0}, // victim R, attacker P, N, B, R, Q, K, None
	{55, 54, 53, 52, 51, 50, 0}, // victim Q, attacker P, N, B, R, Q, K, None
	{0, 0, 0, 0, 0, 0, 0},       // victim K, attacker P, N, B, R, Q, K, None (should not occur)
	{0, 0, 0, 0, 0, 0, 0},       // victim None (quiet move)
}

// ScoreMovesFast scores captures with MVV-LVA (an O(1) table lookup)
// instead of SEE. Quiescence calls this, not ScoreMoves: quiescence nodes
// dominate total node count in alpha-beta search and its movelist is
// nearly all captures, so SEE's per-move cost (several bitboard ops and
// magic-lookup-based X-ray checks per capture, versus one table read)
// compounds heavily there in a way it doesn't in the main search, where
// nodes are comparatively scarce and precise ordering matters more for
// cutoffs than raw per-node speed. An initial attempt using SEE
// everywhere, including quiescence, measured flat-to-better nps on a
// handful of synthetic fixed-depth positions but regressed real games by
// roughly -18 to -22 Elo in SPRT (reproduced twice, including after
// fixing a real SEE bug) -- this split is the fix being tested for that.
func ScoreMovesFast(pos *Position, moveList *MoveList) {
	for i := 0; i < int(moveList.Count); i++ {
		move := &moveList.Moves[i]
		_, attacker := pos.GetSquare(move.From())
		_, victim := pos.GetSquare(move.To())
		move.GiveScore(MvvLva[victim][attacker])
	}
}

// swap the highest score move to the front, leaving everything else untouched
// OrderMoves sorts moveList by score descending (insertion sort: move lists
// here are small -- at most a few dozen moves -- so this is cheap and
// needs no allocation). A full sort, not just a best-to-front swap, matters
// once ordering has more than one signal below the very top: MVV-LVA always
// outranks killers/history (see maxQuietScore), so a swap-only pass could
// only ever place a killer/history-favored quiet first in capture-free
// positions, and even then left every other move in raw movegen order --
// making killer/history scores irrelevant to move 2 onward, including to
// which moves LMR treats as "late".
func OrderMoves(pos *Position, moveList *MoveList) {
	for i := 1; i < int(moveList.Count); i++ {
		move := moveList.Moves[i]
		score := move.Score()
		j := i - 1
		for j >= 0 && moveList.Moves[j].Score() < score {
			moveList.Moves[j+1] = moveList.Moves[j]
			j--
		}
		moveList.Moves[j+1] = move
	}
}

// orderMoveFirst swaps the move matching target to the front of moveList,
// if present. Move's top 16 bits are a mutable score field the TT never
// stores, so target is compared on its low 16 bits only. A target of 0 or
// one that doesn't match any generated move (a stale or index-collided TT
// entry, or a move that's simply no longer legal here) is a safe no-op --
// this only ever reorders, it's never relied on for correctness.
func orderMoveFirst(moveList *MoveList, target Move) {
	if target == 0 {
		return
	}
	target &= 0xffff
	for i := uint8(0); i < moveList.Count; i++ {
		if moveList.Moves[i]&0xffff == target {
			if i != 0 {
				moveList.Moves[0], moveList.Moves[i] = moveList.Moves[i], moveList.Moves[0]
			}
			return
		}
	}
}
