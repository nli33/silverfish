package engine

import (
	"context"
	"time"
)

const InfiniteDepth = 100000
const InfiniteMovetime = 600000 * time.Millisecond
const MaxMovetime = 3000 * time.Millisecond

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
	estimatedMovesLeft := 50 - pos.FullMoves()
	return min(MaxMovetime, time.Duration(ourTime/int32(estimatedMovesLeft)+ourInc/4))
}

// based on negamax (flip sign), each player maximizes their own score
// alpha: best score guaranteed for max-player. can prune branches that give less than this
// beta: upper limit that min-player will tolerate. min-player will prune lines exceeding this

// pass timeLimit in nanoseconds (default)
func Search(pos *Position, maxDepth int, timeLimit time.Duration) (int32, Move) {
	ctx, cancel := context.WithTimeout(context.Background(), timeLimit*time.Millisecond)
	defer cancel()

	var bestMove Move
	var bestScore int32

	moveList := GenMoves(pos, BB_Full)

	// lastScores := make(map[Move]int32, len(moves))

	nodes := 0

	for depth := 1; depth <= maxDepth; depth++ {
		select {
		case <-ctx.Done():
			return bestScore, bestMove
		default:
		}

		// sort.Slice(moves, func(i, j int) bool {
		// 	return lastScores[moves[i]] > lastScores[moves[j]]
		// })

		alpha := -Infinity
		beta := Infinity
		bestScore = -Infinity

		for i := uint8(0); i < moveList.Count; i++ {
			move := moveList.Moves[i]
			if !pos.MoveIsLegal(move) {
				continue
			}

			nodes++
			pos.DoMove(move)
			score := -alphaBetaInner(pos, -beta, -alpha, depth-1, &nodes, &ctx)
			pos.UndoMove(move)

			// lastScores[move] = score

			if score > bestScore {
				bestScore = score
				bestMove = move
			}
			if score > alpha {
				alpha = score
			}

			UciInfo(UciInfoMessage{
				depth:       depth,
				hasDepth:    true,
				currmove:    move,
				hasCurrmove: true,
				score:       bestScore,
				hasScore:    true,
				nodes:       nodes,
				hasNodes:    true,
			})
		}
	}

	return bestScore, bestMove
}

func alphaBetaInner(pos *Position, alpha, beta int32, depth int, nodes *int, ctx *context.Context) int32 {
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
		return Evaluate(pos)
	}

	bestScore := -Infinity
	for i := uint8(0); i < moveList.Count; i++ {
		select {
		case <-(*ctx).Done():
			return bestScore
		default:
		}

		move := moveList.Moves[i]
		if !pos.MoveIsLegal(move) {
			continue
		}

		*nodes++
		pos.DoMove(move)
		score := -alphaBetaInner(pos, -beta, -alpha, depth-1, nodes, ctx)
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
				depth:       depth,
				hasDepth:    true,
				nodes:       *nodes,
				hasNodes:    true,
				currmove:    move,
				hasCurrmove: true,
				score:       bestScore,
				hasScore:    true,
			})
		}
	}

	return bestScore
}
