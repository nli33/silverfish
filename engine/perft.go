package engine

import "fmt"

func Perft(pos *Position, depth int, verbose bool) uint64 {
	var ans uint64 = 0

	if depth == 0 {
		return 1
	}

	moveList := GenMoves(pos, BB_Full)
	for i := uint8(0); i < moveList.Count; i++ {
		move := moveList.Moves[i]
		if !pos.MoveIsLegal(move) {
			continue
		}

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
