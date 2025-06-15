package engine

func Perft(pos *Position, depth int) uint64 {
	legalMoves := pos.LegalMoves()
	var ans uint64 = 0

	if depth == 0 {
		return 1
	}

	for _, move := range legalMoves {
		pos.DoMove(move)
		ans += Perft(pos, depth-1)
		pos.UndoMove(move)
	}

	return ans
}
