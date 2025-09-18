package engine

const (
	Exact = iota
	Lower
	Upper
)

type TTEntry struct {
	Zobrist  uint64
	Depth    int
	Score    int32
	BestMove Move
	// exact, lower, upper
	Type int
}

const TTSize = 1 << 20
const Mask = TTSize - 1

var TT [TTSize]TTEntry

func Probe(key uint64) *TTEntry {
	idx := key & Mask
	entry := &TT[idx]
	if entry.Zobrist == key {
		return entry
	} else {
		return nil
	}
}

func Store(key uint64, depth int, score int32, move Move, entryType int) {
	idx := key & Mask
	entry := &TT[idx]
	if depth >= entry.Depth {
		entry.Zobrist = key
		entry.Depth = depth
		entry.Score = score
		entry.BestMove = move
		entry.Type = entryType
	}
}
