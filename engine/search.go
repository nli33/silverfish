package engine

import (
	"time"
)

const InfiniteDepth = 100000                       // arbitrary large number for infinite depth
const InfiniteMovetime = 600000 * time.Millisecond // arbitrary large number for infinite movetime
const MaxMovetime = 2000                           // max movetime for any move if unspecified
const MaxQuiescenceDepth = 8

type Search struct {
	Pos   Position
	Nodes int

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

	for depth := 1; depth <= search.MaxDepth; depth++ {
		alpha := -Infinity
		beta := Infinity

		bestScoreCurr := -Infinity
		var bestMoveCurr Move

		for i := uint8(0); i < moveList.Count; i++ {
			move := moveList.Moves[i]
			if !search.Pos.MoveIsLegal(move) {
				continue
			}

			search.Pos.DoMove(move)
			score := -search.alphaBetaInner(-beta, -alpha, depth-1, 1)
			search.Pos.UndoMove(move)

			if time.Since(search.StartTime) > search.TimeLimit {
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

			UciInfo(UciInfoMessage{
				depth:    depth,
				hasDepth: true,
				score:    bestScore,
				hasScore: true,
				nodes:    search.Nodes,
				hasNodes: true,
			})
		}

		if time.Since(search.StartTime) > search.TimeLimit {
			break
		}

		bestScore = bestScoreCurr
		bestMove = bestMoveCurr
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

// ply is the number of plies from the root of this search (the move passed
// to alphaBetaInner from Search() is ply 1). Used only to prefer faster
// mates (and defer forced ones): a mate found at a smaller ply scores
// strictly higher than the same mate found deeper in the tree.
func (search *Search) alphaBetaInner(alpha, beta int32, depth int, ply int) int32 {
	search.Nodes++

	moveList := GenMoves(&search.Pos, BB_Full)

	ScoreMoves(&search.Pos, &moveList)
	OrderMoves(&search.Pos, &moveList)

	hasLegal := false

	if moveList.Count == 0 {
		if search.Pos.Checkers(search.Pos.Turn) != 0 {
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

	bestScore := -Infinity
	for i := uint8(0); i < moveList.Count; i++ {
		move := moveList.Moves[i]
		if !search.Pos.MoveIsLegal(move) {
			continue
		}
		hasLegal = true

		search.Pos.DoMove(move)
		score := -search.alphaBetaInner(-beta, -alpha, depth-1, ply+1)
		search.Pos.UndoMove(move)

		if score >= beta {
			return score
		}
		if score > bestScore {
			bestScore = score
		}
		if score > alpha {
			alpha = score
		}

		if search.Nodes&32767 == 0 {
			UciInfo(UciInfoMessage{
				depth:    depth,
				hasDepth: true,
				nodes:    search.Nodes,
				hasNodes: true,
				score:    bestScore,
				hasScore: true,
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

	return bestScore
}
