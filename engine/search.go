package engine

import (
	"time"
)

const InfiniteDepth = 100000                       // arbitrary large number for infinite depth
const InfiniteMovetime = 600000 * time.Millisecond // arbitrary large number for infinite movetime
const MaxMovetime = 2000                           // max movetime for any move if unspecified
const MaxQuiescenceDepth = 5

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

// based on negamax (flip sign), each player maximizes their own score
// alpha: best score guaranteed for max-player. can prune branches that give less than this
// beta: upper limit that min-player will tolerate. min-player will prune lines exceeding this

// pass timeLimit in nanoseconds (default)
func Search(pos *Position, maxDepth int, timeLimit time.Duration) (int32, Move) {
	startTime := time.Now()

	var bestMove Move
	bestScore := -Infinity

	moveList := GenMoves(pos, BB_Full)

	// lastScores := make(map[Move]int32, len(moves))

	nodes := 0

	for depth := 1; depth <= maxDepth; depth++ {
		// sort.Slice(moves, func(i, j int) bool {
		// 	return lastScores[moves[i]] > lastScores[moves[j]]
		// })

		alpha := -Infinity
		beta := Infinity

		bestScoreCurr := -Infinity
		var bestMoveCurr Move

		for i := uint8(0); i < moveList.Count; i++ {
			move := moveList.Moves[i]
			if !pos.MoveIsLegal(move) {
				continue
			}

			pos.DoMove(move)
			score := -alphaBetaInner(pos, -beta, -alpha, depth-1, &nodes, &startTime, &timeLimit)
			pos.UndoMove(move)

			if time.Since(startTime) > timeLimit {
				break
			}

			// lastScores[move] = score

			if score > bestScoreCurr {
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
				nodes:    nodes,
				hasNodes: true,
			})
		}

		if time.Since(startTime) > timeLimit {
			break
		}

		// NOTE: this may lead to choosing a move that is higher eval at lower depth but is actually worse
		// 	than a move that is slightly lower (but more accurate) eval at higher depth
		if bestScoreCurr > bestScore {
			bestScore = bestScoreCurr
			bestMove = bestMoveCurr
		}
	}

	return bestScore, bestMove
}

func Quiescence(pos *Position, alpha, beta int32, nodes *int, startTime *time.Time, timeLimit *time.Duration, qdepth int) int32 {
	if qdepth > MaxQuiescenceDepth {
		return Evaluate(pos)
	}

	standPat := Evaluate(pos)
	if standPat >= beta {
		return beta
	}
	if standPat > alpha {
		alpha = standPat
	}

	var moveList MoveList
	if pos.Checkers(pos.Turn) != 0 {
		moveList = GenMoves(pos, BB_Full)
	} else {
		moveList = GenMoves(pos, pos.Sides[pos.Turn^1]) // only captures
	}

	for i := uint8(0); i < moveList.Count; i++ {
		move := moveList.Moves[i]
		if !pos.MoveIsLegal(move) {
			continue
		}

		*nodes++

		pos.DoMove(move)
		score := -Quiescence(pos, -beta, -alpha, nodes, startTime, timeLimit, qdepth+1)
		pos.UndoMove(move)

		if score >= beta {
			return beta
		}
		if score > alpha {
			alpha = score
		}
	}

	return alpha
}

func alphaBetaInner(pos *Position, alpha, beta int32, depth int, nodes *int, startTime *time.Time, timeLimit *time.Duration) int32 {
	moveList := GenMoves(pos, BB_Full)

	if moveList.Count == 0 {
		if pos.Checkers(pos.Turn) != 0 {
			// checkmate
			return -Infinity
		} else {
			// stalemate
			return 0
		}
	}

	if depth == 0 {
		// return Evaluate(pos)
		return Quiescence(pos, alpha, beta, nodes, startTime, timeLimit, 0)
	}

	bestScore := -Infinity
	for i := uint8(0); i < moveList.Count; i++ {
		move := moveList.Moves[i]
		if !pos.MoveIsLegal(move) {
			continue
		}

		*nodes++

		pos.DoMove(move)
		score := -alphaBetaInner(pos, -beta, -alpha, depth-1, nodes, startTime, timeLimit)
		pos.UndoMove(move)

		if score >= beta {
			return score
		}
		if score > bestScore {
			bestScore = score
		}
		if score > alpha {
			alpha = score
		}

		if *nodes&32767 == 0 {
			UciInfo(UciInfoMessage{
				depth:    depth,
				hasDepth: true,
				nodes:    *nodes,
				hasNodes: true,
				score:    bestScore,
				hasScore: true,
			})
		}
	}

	return bestScore
}
