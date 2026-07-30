package engine

import (
	"time"
)

const InfiniteDepth = 100000                       // arbitrary large number for infinite depth
const InfiniteMovetime = 600000 * time.Millisecond // arbitrary large number for infinite movetime
const MaxMovetime = 2000                           // max movetime for any move if unspecified
const MaxQuiescenceDepth = 8

// MateScoreThreshold: any score at least this close to Infinity is a mate
// score (see the mate-distance comment on alphaBetaInner), not a real
// evaluation -- real evaluations never get remotely close to Infinity.
const MateScoreThreshold = Infinity - 10000

// mateInfo reports whether score is a forced-mate score and, if so, converts
// it to UCI's "score mate N" convention: N is moves (not plies) to mate,
// positive if the engine's side delivers it and negative if it's being
// mated.
func mateInfo(score int32) (movesToMate int32, isMate bool) {
	abs := score
	if abs < 0 {
		abs = -abs
	}
	if abs < MateScoreThreshold {
		return 0, false
	}
	pliesToMate := Infinity - abs
	movesToMate = (pliesToMate + 1) / 2
	if score < 0 {
		movesToMate = -movesToMate
	}
	return movesToMate, true
}

const NodeReportInterval = 32768

type Search struct {
	Pos   Position
	Nodes int

	// lastReportedNodes is the Nodes value as of the last periodic
	// unscored progress ping (see alphaBetaInner) -- a threshold-crossing
	// check against this, rather than a Nodes%NodeReportInterval==0 check,
	// so a ping is guaranteed at least every NodeReportInterval nodes
	// regardless of exactly which node count a check happens to land on.
	lastReportedNodes int

	// limits
	StartTime time.Time
	TimeLimit time.Duration
	MaxDepth  int
}

// return number in milliseconds
func TimeLimit(pos *Position, command *UciGoMessage) time.Duration {
	var ourTime, ourInc int32 //, theirTime, theirInc int32
	if pos.Turn == White {
		ourTime = command.WTime
		ourInc = command.WInc
		// theirTime = command.BTime
		// theirInc = command.BInc
	} else if pos.Turn == Black {
		ourTime = command.BTime
		ourInc = command.BInc
		// theirTime = command.WTime
		// theirInc = command.WInc
	}
	estimatedMovesLeft := max(10, 100-pos.FullMoves())
	// multiplying time.Miillisecond twice?
	return min(MaxMovetime, time.Duration(ourTime/int32(estimatedMovesLeft)+ourInc/4))
}

// Indexed by the real piece constants (Pawn=0, Knight=1, Bishop=2, Rook=3,
// Queen=4, King=5, NoPiece=6) on both axes. Row = victim, column = attacker;
// higher score for a more valuable victim taken by a cheaper attacker.
var MvvLva = [7][7]int{
	{15, 14, 13, 12, 11, 10, 0}, // victim P, attacker P, N, B, R, Q, K, None
	{25, 24, 23, 22, 21, 20, 0}, // victim N, attacker P, N, B, R, Q, K, None
	{35, 34, 33, 32, 31, 30, 0}, // victim B, attacker P, N, B, R, Q, K, None
	{45, 44, 43, 42, 41, 40, 0}, // victim R, attacker P, N, B, R, Q, K, None
	{55, 54, 53, 52, 51, 50, 0}, // victim Q, attacker P, N, B, R, Q, K, None
	{0, 0, 0, 0, 0, 0, 0},       // victim K, attacker P, N, B, R, Q, K, None (should not occur)
	{0, 0, 0, 0, 0, 0, 0},       // victim None (quiet move)
}

func ScoreMoves(pos *Position, moveList *MoveList) {
	for i := 0; i < int(moveList.Count); i++ {
		move := &moveList.Moves[i]
		_, attacker := pos.GetSquare(move.From())
		_, victim := pos.GetSquare(move.To())
		value := MvvLva[victim][attacker]
		move.GiveScore(value)
	}
}

// swap the highest score move to the front, leaving everything else untouched
func OrderMoves(pos *Position, moveList *MoveList) {
	if moveList.Count <= 1 {
		return
	}

	bestIdx := 0
	bestScore := moveList.Moves[0].Score()

	for j := 1; j < int(moveList.Count); j++ {
		if moveList.Moves[j].Score() > bestScore {
			bestIdx = j
			bestScore = moveList.Moves[j].Score()
		}
	}

	if bestIdx != 0 {
		moveList.Moves[0], moveList.Moves[bestIdx] = moveList.Moves[bestIdx], moveList.Moves[0]
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

func (search *Search) Init(pos *Position) {
	search.Pos = pos.Clone()
}

// based on negamax (flip sign), each player maximizes their own score
// alpha: best score guaranteed for max-player. can prune branches that give less than this
// beta: upper limit that min-player will tolerate. min-player will prune lines exceeding this

// pass TimeLimit in nanoseconds (default)
func (search *Search) Search() (int32, Move) {
	var bestMove Move
	bestScore := -Infinity

	search.StartTime = time.Now()

	moveList := GenMoves(&search.Pos, BB_Full)
	ScoreMoves(&search.Pos, &moveList)
	OrderMoves(&search.Pos, &moveList)

	for depth := 1; depth <= search.MaxDepth; depth++ {
		alpha := -Infinity
		beta := Infinity

		// Put the previous iteration's best move (stored by this same loop,
		// one depth ago) first -- gives PV-move-first ordering across
		// iterative-deepening iterations, not just within a single
		// alphaBetaInner call.
		if entry, ok := TTProbe(search.Pos.Hash); ok {
			orderMoveFirst(&moveList, entry.Move)
		}

		bestScoreCurr := -Infinity
		var bestMoveCurr Move
		timedOut := false

		for i := uint8(0); i < moveList.Count; i++ {
			move := moveList.Moves[i]
			if !search.Pos.MoveIsLegal(move) {
				continue
			}

			search.Pos.DoMove(move)
			score := -search.alphaBetaInner(-beta, -alpha, depth-1, 1)
			search.Pos.UndoMove(move)

			// ensure a null move is not chosen (in case of unavoidable checkmate)
			if score > bestScoreCurr || bestMoveCurr == Move(0) {
				bestScoreCurr = score
				bestMoveCurr = move
			}
			if score > alpha {
				alpha = score
			}

			if time.Since(search.StartTime) > search.TimeLimit {
				timedOut = true
				break
			}
		}

		// A depth that fully explored every root move is strictly better
		// information than any previous (shallower) depth, so always trust
		// it. A depth that timed out partway through only checked some
		// prefix of the (move-ordered, but not perfectly) root list -- if we
		// already have a complete result from a previous depth, that result
		// is more reliable than this partial one and must NOT be
		// overwritten. The only time a partial result is used is when there
		// is nothing else yet at all (typically depth 1 timing out before
		// its first move even finishes) -- a partial answer beats returning
		// an illegal null move.
		if !timedOut || bestMove == Move(0) {
			if bestMoveCurr != Move(0) {
				bestScore = bestScoreCurr
				bestMove = bestMoveCurr

				// Only a fully-completed depth's result is trustworthy
				// enough to mark Exact (a timed-out partial pass didn't
				// finish comparing every root move). Stored so the next
				// iteration's probe above can order this move first.
				if !timedOut {
					TTStore(search.Pos.Hash, bestMove, ScoreToTT(bestScore, 0), depth, BoundExact)
				}

				// Reported once per completed depth, with that depth's own
				// final score -- not per move, and not a stale score left
				// over from the previous depth.
				infoScore := bestScore
				movesToMate, isMate := mateInfo(bestScore)
				if isMate {
					infoScore = movesToMate
				}
				UciInfo(UciInfoMessage{
					depth:    depth,
					hasDepth: true,
					score:    infoScore,
					hasScore: true,
					isMate:   isMate,
					nodes:    search.Nodes,
					hasNodes: true,
				})
			}
		}

		if timedOut {
			break
		}
	}

	return bestScore, bestMove
}

// ply mirrors alphaBetaInner's ply: the number of plies from the search
// root, needed so a checkmate found here scores consistently with one found
// in alphaBetaInner (see the mate-distance comment there).
func (search *Search) Quiescence(alpha, beta int32, qdepth int, ply int) int32 {
	if qdepth > MaxQuiescenceDepth {
		return Evaluate(&search.Pos)
	}

	inCheck := search.Pos.Checkers(search.Pos.Turn) != 0

	// Unlike a capture, check can't be declined -- taking a stand-pat floor
	// while in check can hide that the position is actually lost (or won)
	// tactically, so it's skipped entirely here; every evasion must be
	// searched instead.
	if !inCheck {
		standPat := Evaluate(&search.Pos)
		if standPat >= beta {
			return beta
		}
		if standPat > alpha {
			alpha = standPat
		}
	}

	var moveList MoveList
	if inCheck {
		moveList = GenMoves(&search.Pos, BB_Full)
	} else {
		moveList = GenMoves(&search.Pos, search.Pos.Sides[search.Pos.Turn^1]) // only captures
	}

	ScoreMoves(&search.Pos, &moveList)
	OrderMoves(&search.Pos, &moveList)

	hasLegal := false
	for i := uint8(0); i < moveList.Count; i++ {
		move := moveList.Moves[i]
		if !search.Pos.MoveIsLegal(move) {
			continue
		}
		hasLegal = true

		search.Nodes++

		search.Pos.DoMove(move)
		score := -search.Quiescence(-beta, -alpha, qdepth+1, ply+1)
		search.Pos.UndoMove(move)

		if score >= beta {
			return beta
		}
		if score > alpha {
			alpha = score
		}
	}

	// Running out of captures just means "stand pat" (handled above), but
	// running out of legal evasions while in check is checkmate.
	if inCheck && !hasLegal {
		return -(Infinity - int32(ply))
	}

	return alpha
}

// hasNonPawnMaterial reports whether color has any knight/bishop/rook/queen
// on the board. Used to guard null-move pruning against zugzwang: in bare
// king-and-pawn endgames, passing can be strictly better than any legal
// move, which breaks the "a free move can only help our opponent" assumption
// null-move pruning relies on.
func hasNonPawnMaterial(pos *Position, color uint8) bool {
	return pos.Pieces[color][Knight]|pos.Pieces[color][Bishop]|pos.Pieces[color][Rook]|pos.Pieces[color][Queen] != 0
}

// ply is the number of plies from the root of this search (the move passed
// to alphaBetaInner from Search() is ply 1). Used only to prefer faster
// mates (and defer forced ones): a mate found at a smaller ply scores
// strictly higher than the same mate found deeper in the tree.
func (search *Search) alphaBetaInner(alpha, beta int32, depth int, ply int) int32 {
	search.Nodes++

	// Treat the first repetition as a draw rather than waiting for a literal
	// threefold (standard practice -- see Position.IsRepetition). Checked
	// before move generation so a repeated node also skips that work.
	if search.Pos.IsRepetition() {
		return 0
	}

	alphaOrig := alpha

	var ttMove Move
	if entry, ok := TTProbe(search.Pos.Hash); ok {
		ttMove = entry.Move
		if int(entry.Depth) >= depth {
			s := ScoreFromTT(entry.Score, ply)
			switch {
			case entry.Bound == BoundExact:
				return s
			case entry.Bound == BoundLower && s >= beta:
				return s
			case entry.Bound == BoundUpper && s <= alpha:
				return s
			}
		}
	}

	inCheckEarly := search.Pos.Checkers(search.Pos.Turn) != 0

	// Null-move pruning: let the opponent move twice in a row (i.e. we do
	// nothing) and search at reduced depth. If even a free move for the
	// opponent can't bring the score down to beta, our actual move surely
	// won't either, so prune. Guarded against zugzwang (positions where
	// passing is actually better than any legal move, breaking the "a free
	// move can only help" assumption -- mainly king-and-pawn endgames) by
	// requiring the side to move to have some non-pawn material, and
	// skipped in check (a null move can't escape check, so the reduced
	// search would be meaningless) and near mate scores (verifying a mate
	// score off a reduced, unverified search is unreliable).
	if depth >= 3 && !inCheckEarly && beta < MateScoreThreshold && hasNonPawnMaterial(&search.Pos, search.Pos.Turn) {
		const nullMoveReduction = 2
		prevEP := search.Pos.DoNullMove()
		score := -search.alphaBetaInner(-beta, -beta+1, depth-1-nullMoveReduction, ply+1)
		search.Pos.UndoNullMove(prevEP)
		if score >= beta {
			return score
		}
	}

	moveList := GenMoves(&search.Pos, BB_Full)

	ScoreMoves(&search.Pos, &moveList)
	OrderMoves(&search.Pos, &moveList)
	orderMoveFirst(&moveList, ttMove)

	hasLegal := false

	if moveList.Count == 0 {
		if inCheckEarly {
			// checkmate
			return -(Infinity - int32(ply))
		} else {
			// stalemate
			return 0
		}
	}

	if depth == 0 {
		// return Evaluate(pos)
		return search.Quiescence(alpha, beta, 0, ply)
	}

	inCheck := inCheckEarly

	bestScore := -Infinity
	var bestMove Move
	legalMoveNum := 0
	for i := uint8(0); i < moveList.Count; i++ {
		move := moveList.Moves[i]
		if !search.Pos.MoveIsLegal(move) {
			continue
		}
		hasLegal = true
		legalMoveNum++

		// Late Move Reductions: search moves that are unlikely to matter --
		// late in the (MVV-LVA-then-TT-move) ordering, quiet (score 0: no
		// killer/history heuristic yet to distinguish quiets further),
		// not a promotion, and not while in check -- at reduced depth
		// first. If a reduced search still beats alpha, it wasn't
		// obviously bad, so re-search it at full depth before trusting the
		// score. LeafMoveNum/depth thresholds are conservative (no PVS or
		// killer/history yet to lean on for ordering confidence).
		reduction := 0
		if depth >= 3 && legalMoveNum > 3 && !inCheck && move.Score() == 0 && !move.IsPromotion() {
			reduction = 1
			if legalMoveNum > 6 {
				reduction = 2
			}
			if reduction > depth-1 {
				reduction = depth - 1
			}
		}

		search.Pos.DoMove(move)
		var score int32
		if reduction > 0 {
			score = -search.alphaBetaInner(-alpha-1, -alpha, depth-1-reduction, ply+1)
			if score > alpha {
				score = -search.alphaBetaInner(-beta, -alpha, depth-1, ply+1)
			}
		} else {
			score = -search.alphaBetaInner(-beta, -alpha, depth-1, ply+1)
		}
		search.Pos.UndoMove(move)

		if score >= beta {
			TTStore(search.Pos.Hash, move, ScoreToTT(score, ply), depth, BoundLower)
			return score
		}
		if score > bestScore {
			bestScore = score
			bestMove = move
		}
		if score > alpha {
			alpha = score
		}

		if search.Nodes-search.lastReportedNodes >= NodeReportInterval {
			search.lastReportedNodes = search.Nodes
			// No score here: bestScore is this internal node's own
			// negamax-local value, not the root-relative evaluation UCI's
			// `score` field is supposed to report -- nodes/depth are the
			// only legitimate "still working" signal available this deep in
			// the tree.
			UciInfo(UciInfoMessage{
				depth:    depth,
				hasDepth: true,
				nodes:    search.Nodes,
				hasNodes: true,
			})
		}
	}

	// GenMoves returns a list of valid but possibly illegal (leaves king in check) moves
	// avoid the case where all moves are illegal (stalemate/checkmate) but moveList.Count != 0
	if !hasLegal {
		if search.Pos.Checkers(search.Pos.Turn) != 0 {
			return -(Infinity - int32(ply))
		} else {
			return 0
		}
	}

	bound := BoundExact
	if bestScore <= alphaOrig {
		bound = BoundUpper
	}
	TTStore(search.Pos.Hash, bestMove, ScoreToTT(bestScore, ply), depth, bound)

	return bestScore
}
