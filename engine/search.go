package engine

import (
	"sync/atomic"
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

	// killers holds up to 2 quiet moves per ply that have caused a beta
	// cutoff there before, tried before other quiets on the assumption
	// that a move good enough to cut off once at this ply is often good
	// again in a sibling node (same ply, different path to it). Stored
	// masked to their low 16 bits (see orderMoveFirst) since a Move's
	// score field is mutable and irrelevant for identity comparison.
	killers [MaxKillerPly][2]Move

	// history accumulates a [side][from][to] weight on every quiet-move
	// beta cutoff, weighted by depth^2 so cutoffs found deeper (more
	// searched-out, so more reliable) count for more. Orders quiet moves
	// that aren't killers at this ply. Unlike killers this isn't
	// ply-indexed -- it's a search-wide "this from/to square pair tends to
	// be strong" signal, not a "strong at this specific ply" one.
	history [2][64][64]int32

	// limits
	StartTime time.Time
	TimeLimit time.Duration
	MaxDepth  int

	// timedOut is set once checkTimeUp first detects the budget has been
	// exceeded, and stays set for the rest of this Search() call. Sticky so
	// every frame on the way back up the call stack can bail out on a cheap
	// field read instead of each re-checking time.Since.
	timedOut bool

	// stopSignal, if set (see SetStopSignal), is a shared cancellation flag
	// used by Lazy SMP (smp.go): once the main search thread finishes, it
	// flips this so any still-running helper threads unwind promptly rather
	// than running out their own full time budget for no benefit.
	stopSignal *int32

	// silent suppresses UciInfo output. Set on Lazy SMP helper threads
	// (smp.go) -- only the main thread's progress/PV is meaningful UCI
	// output; helpers exist purely to enrich the shared TT.
	silent bool
}

// SetStopSignal wires an external cancellation flag into checkTimeUp, in
// addition to this Search's own time/depth budget. Used by Lazy SMP to stop
// helper threads once the main thread concludes.
func (search *Search) SetStopSignal(stop *int32) {
	search.stopSignal = stop
}

// checkTimeUp reports whether the search has exceeded its time budget.
// Checked periodically (every 2048 nodes, via the low bits of Nodes) rather
// than on every node -- time.Since on every node would itself be a
// meaningful overhead. This is what lets alphaBetaInner/Quiescence unwind a
// single oversized subtree instead of only checking between root moves (see
// the "go movetime can hang" note in todo.md): a subtree that runs long
// still gets probed every couple thousand nodes no matter how deep it goes.
func (search *Search) checkTimeUp() bool {
	if search.timedOut {
		return true
	}
	if search.stopSignal != nil && atomic.LoadInt32(search.stopSignal) != 0 {
		search.timedOut = true
		return true
	}
	if search.Nodes&2047 == 0 && time.Since(search.StartTime) > search.TimeLimit {
		search.timedOut = true
	}
	return search.timedOut
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

			// search.timedOut means this move's score is the checkTimeUp
			// sentinel (0), not a real result -- discard it rather than
			// letting it compete with bestScoreCurr.
			if search.timedOut {
				timedOut = true
				break
			}

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
if !search.silent {
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

	if search.checkTimeUp() {
		return 0
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

	if search.checkTimeUp() {
		return 0
	}

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
	search.scoreQuiets(&moveList, ply)
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

	// Futility pruning: this close to the horizon, a quiet, non-check-giving
	// move can only improve the position by a small amount -- if even the
	// static eval plus a generous margin can't reach alpha, this branch is
	// almost certainly not going to raise alpha either, so it's skipped
	// entirely rather than searched. Margins are deliberately more
	// conservative than a well-tuned modern engine would use (real engines
	// go much smaller): this engine's move ordering is comparatively weak
	// (no SEE, no PVS -- both tried and dropped this session after negative
	// SPRTs), so an aggressive margin risks pruning away moves ordering
	// hasn't actually ranked well. Gated off near mate scores (a
	// material-margin argument says nothing reliable about a nearby forced
	// mate) and never applied to a node's first move (that's move ordering's
	// best guess, and always gets searched for real).
	const futilityMaxDepth = 3
	var futilityMargin = [futilityMaxDepth + 1]int32{0, 150, 300, 450}

	canFutilityPrune := false
	if depth >= 1 && depth <= futilityMaxDepth && !inCheck &&
		alpha > -MateScoreThreshold && beta < MateScoreThreshold {
		staticEval := Evaluate(&search.Pos)
		canFutilityPrune = staticEval+futilityMargin[depth] <= alpha
	}

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

		isQuiet := isQuietMove(&search.Pos, move)

		// Late Move Reductions: search moves that are unlikely to matter --
		// late in the (MVV-LVA/killer/history-then-TT-move) ordering, quiet,
		// and not while in check -- at reduced depth first. If a reduced
		// search still beats alpha, it wasn't obviously bad, so re-search it
		// at full depth before trusting the score. LeafMoveNum/depth
		// thresholds are conservative (no PVS yet to lean on for ordering
		// confidence).
		reduction := 0
		if depth >= 3 && legalMoveNum > 3 && !inCheck && isQuiet {
			reduction = 1
			if legalMoveNum > 6 {
				reduction = 2
			}
			if reduction > depth-1 {
				reduction = depth - 1
			}
		}

		search.Pos.DoMove(move)

		// The futility skip check needs the post-move position: a move
		// that looks prunable by material margin alone must still be
		// searched for real if it gives check (a checking "quiet" move can
		// be tactically decisive despite costing no material).
		if canFutilityPrune && legalMoveNum > 1 && isQuiet &&
			search.Pos.Checkers(search.Pos.Turn) == 0 {
			search.Pos.UndoMove(move)
			continue
		}

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
			// A timed-out score is 0 by convention (see checkTimeUp), not a
			// real search result -- storing it would poison the TT with a
			// bogus cutoff for future probes at this position.
			if !search.timedOut {
				TTStore(search.Pos.Hash, move, ScoreToTT(score, ply), depth, BoundLower)
				if isQuietMove(&search.Pos, move) {
					search.recordKiller(move, ply)
					search.recordHistory(move, depth)
				}
			}
			return score
		}
		if score > bestScore {
			bestScore = score
			bestMove = move
		}
		if score > alpha {
			alpha = score
		}

		if !search.silent && search.Nodes-search.lastReportedNodes >= NodeReportInterval {
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

	if !search.timedOut {
		bound := BoundExact
		if bestScore <= alphaOrig {
			bound = BoundUpper
		}
		TTStore(search.Pos.Hash, bestMove, ScoreToTT(bestScore, ply), depth, bound)
	}

	return bestScore
}
