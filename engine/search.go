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

func (search *Search) Init(pos Position) {
	search.Pos = pos
	search.HistoryPly = 0
	search.History[search.HistoryPly] = search.Pos.Hash
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
			score := -search.alphaBetaInner(-beta, -alpha, depth-1)
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

func (search *Search) Quiescence(alpha, beta int32, qdepth int) int32 {
	if qdepth > MaxQuiescenceDepth {
		return Evaluate(&search.Pos)
	}

	standPat := Evaluate(&search.Pos)
	if standPat >= beta {
		return beta
	}
	if standPat > alpha {
		alpha = standPat
	}

	var moveList MoveList
	if search.Pos.Checkers(search.Pos.Turn) != 0 {
		moveList = GenMoves(&search.Pos, BB_Full)
	} else {
		moveList = GenMoves(&search.Pos, search.Pos.Sides[search.Pos.Turn^1]) // only captures
	}

	ScoreMoves(&search.Pos, &moveList)
	OrderMoves(&search.Pos, &moveList)

	for i := uint8(0); i < moveList.Count; i++ {
		move := moveList.Moves[i]
		if !search.Pos.MoveIsLegal(move) {
			continue
		}

		search.Nodes++

		search.Pos.DoMove(move)
		score := -search.Quiescence(-beta, -alpha, qdepth+1)
		search.Pos.UndoMove(move)

		if score >= beta {
			return beta
		}
		if score > alpha {
			alpha = score
		}
	}

	return alpha
}

func (search *Search) alphaBetaInner(alpha, beta int32, depth int) int32 {
	search.Nodes++

	moveList := GenMoves(&search.Pos, BB_Full)

	ScoreMoves(&search.Pos, &moveList)
	OrderMoves(&search.Pos, &moveList)

	hasLegal := false

	if moveList.Count == 0 {
		if search.Pos.Checkers(search.Pos.Turn) != 0 {
			// checkmate
			return -Infinity
		} else {
			// stalemate
			return 0
		}
	}

	if depth == 0 {
		// return Evaluate(pos)
		return search.Quiescence(alpha, beta, 0)
	}

	bestScore := -Infinity
	for i := uint8(0); i < moveList.Count; i++ {
		move := moveList.Moves[i]
		if !search.Pos.MoveIsLegal(move) {
			continue
		}
		hasLegal = true

		search.Pos.DoMove(move)
		score := -search.alphaBetaInner(-beta, -alpha, depth-1)
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
			return -Infinity
		} else {
			return 0
		}
	}

	return bestScore
}
