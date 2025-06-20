package engine

// based on negamax (flip sign), each player maximizes their own score
// alpha: best score guaranteed for max-player. can prune branches that give less than this
// beta: upper limit that min-player will tolerate. min-player will prune lines exceeding this

func AlphaBeta(pos Position, depth int) (int32, Move) {
	alpha := -Infinity // best score of max-player that is guaranteed
	beta := Infinity   // worst score that minimizing player tolerates
	bestScore := -Infinity
	nodes := 0
	var bestMove Move

	for _, move := range pos.LegalMoves() {
		pos.DoMove(move)
		nodes += 1
		score := -alphaBetaInner(pos, -beta, -alpha, depth-1, &nodes)
		pos.UndoMove(move)

		UciInfo(UciInfoMessage{
			nodes:       nodes,
			hasNodes:    true,
			currmove:    move,
			hasCurrmove: true,
			score:       bestScore,
			hasScore:    true,
		})

		if score > bestScore {
			bestScore = score
			bestMove = move
		}
		if score > alpha {
			alpha = score
		}
	}

	return bestScore, bestMove
}

func alphaBetaInner(pos Position, alpha int32, beta int32, depth int, nodes *int) int32 {
	if depth == 0 {
		return Evaluate(&pos)
	}

	bestScore := -Infinity

	for _, move := range pos.LegalMoves() {
		pos.DoMove(move)
		*nodes += 1
		score := -alphaBetaInner(pos, -beta, -alpha, depth-1, nodes)
		pos.UndoMove(move)

		if depth%7 == 0 {
			UciInfo(UciInfoMessage{
				nodes:       *nodes,
				hasNodes:    true,
				currmove:    move,
				hasCurrmove: true,
				score:       bestScore,
				hasScore:    true,
			})
		}

		if score >= beta {
			return score
		}
		if score > bestScore {
			bestScore = score
		}
		if score > alpha {
			alpha = score
		}
	}

	return bestScore
}
