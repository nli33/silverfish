package engine

import "fmt"

func Perft(pos *Position, depth int, verbose bool) uint64 {
	legalMoves := pos.LegalMoves()
	var ans uint64 = 0

	if depth == 0 {
		return 1
	}

	for _, move := range legalMoves {
		pos.DoMove(move)
		count := Perft(pos, depth-1, false)
		ans += count
		pos.UndoMove(move)
		if verbose {
			fmt.Printf("%s: %d\n", move.ToString(), count)
		}
	}

	return ans
}
