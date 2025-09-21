package engine

import (
	"time"
)

const InfiniteDepth = 100000                       // arbitrary large number for infinite depth
const InfiniteMovetime = 600000 * time.Millisecond // arbitrary large number for infinite movetime
const MaxMovetime = 2000                           // max movetime for any move if unspecified
const MaxQuiescenceDepth = 8

// return number in milliseconds
func TimeLimit(pos *Position, command *UciGoMessage) time.Duration {
	var ourTime, ourInc int32 //, theirTime, theirInc int32
	switch pos.Turn {
	case White:
		ourTime = command.WTime
		ourInc = command.WInc
		// theirTime = command.BTime
		// theirInc = command.BInc
	case Black:
		ourTime = command.BTime
		ourInc = command.BInc
		// theirTime = command.WTime
		// theirInc = command.WInc
	}

	estimatedMovesLeft := max(10, 100-pos.FullMoves())
	// multiplying time.Miillisecond twice?
	return min(MaxMovetime, time.Duration(ourTime/int32(estimatedMovesLeft)+ourInc/4))
}

var MvvLva = [7][7]int{
	{0, 0, 0, 0, 0, 0, 0},       // victim K, attacker K, Q, R, B, N, P, None
	{50, 51, 52, 53, 54, 55, 0}, // victim Q, attacker K, Q, R, B, N, P, None
	{40, 41, 42, 43, 44, 45, 0}, // victim R, attacker K, Q, R, B, N, P, None
	{30, 31, 32, 33, 34, 35, 0}, // victim B, attacker K, Q, R, B, N, P, None
	{20, 21, 22, 23, 24, 25, 0}, // victim N, attacker K, Q, R, B, N, P, None
	{10, 11, 12, 13, 14, 15, 0}, // victim P, attacker K, Q, R, B, N, P, None
	{0, 0, 0, 0, 0, 0, 0},       // victim None, attacker K, Q, R, B, N, P, None
}

func ScoreMoves(pos *Position, moveList *MoveList) {
	for i := 0; i < int(moveList.Count); i++ {
		move := &moveList.Moves[i]
		_, victim := pos.GetSquare(move.From())
		_, attacker := pos.GetSquare(move.To())
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

// based on negamax (flip sign), each player maximizes their own score
// alpha: best score guaranteed for max-player. can prune branches that give less than this
// beta: upper limit that min-player will tolerate. min-player will prune lines exceeding this

// pass timeLimit in nanoseconds (default)
func Search(pos *Position, maxDepth int, timeLimit time.Duration) (int32, Move) {
	startTime := time.Now()

	var bestMove Move
	bestScore := -Infinity

	moveList := GenMoves(pos, BB_Full)
	nodes := 0

	for depth := 1; depth <= maxDepth; depth++ {
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
				nodes:    nodes,
				hasNodes: true,
			})
		}

		if time.Since(startTime) > timeLimit {
			break
		}

		bestScore = bestScoreCurr
		bestMove = bestMoveCurr
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

	ScoreMoves(pos, &moveList)
	OrderMoves(pos, &moveList)

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
	ScoreMoves(pos, &moveList)
	OrderMoves(pos, &moveList)

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
