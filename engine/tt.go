package engine

// Transposition table: caches search results keyed by Position.Hash, so a
// position reached by a different move order is not re-searched from
// scratch, and (usually the bigger win) so the best move found last time is
// available to order search at every node on every iterative-deepening
// pass.

const (
	BoundNone uint8 = iota
	BoundExact       // Score is the position's true value.
	BoundLower       // Fail-high: the true value is >= Score.
	BoundUpper       // Fail-low: the true value is <= Score.
)

// TTSizeMB is deliberately modest for a first implementation -- SPRT runs
// many engine instances concurrently on one machine, so this is per-process
// resident memory, not a one-off cost.
const TTSizeMB = 16

type TTEntry struct {
	Key   uint64
	Move  Move // low 16 bits only -- Move's bits 16+ are a mutable score field, never compared/stored.
	Score int32
	Depth int16
	Bound uint8
}

const ttEntrySize = 32 // approx size of TTEntry in bytes, rounded up to a power of 2 for a clean table size

var tt []TTEntry
var ttMask uint64

// allocTT allocates the transposition table. Called from Init().
func allocTT(sizeMB int) {
	numEntries := sizeMB * 1024 * 1024 / ttEntrySize
	// round down to a power of two so key&ttMask is a valid index
	pow := 1
	for pow*2 <= numEntries {
		pow *= 2
	}
	tt = make([]TTEntry, pow)
	ttMask = uint64(pow - 1)
}

// ClearTT resets the transposition table. Should be called on ucinewgame:
// stale entries from a previous game are still key-verified before use, but
// clearing avoids wasting the table on positions that can't recur.
func ClearTT() {
	for i := range tt {
		tt[i] = TTEntry{}
	}
}

// TTProbe returns the entry for key and whether it was found. A non-zero
// Move field is usable for ordering even when the caller can't use the
// score itself (e.g. insufficient stored depth).
func TTProbe(key uint64) (TTEntry, bool) {
	e := tt[key&ttMask]
	if e.Bound == BoundNone || e.Key != key {
		return TTEntry{}, false
	}
	return e, true
}

// TTStore records a search result. Replacement is simple depth-preferred:
// an incoming entry always wins on an empty slot, a different key (a
// collision -- always replacing avoids permanently wedging a stale entry
// from a different position into that slot), or when it comes from at
// least as deep a search as what's already there.
func TTStore(key uint64, move Move, score int32, depth int, bound uint8) {
	idx := key & ttMask
	existing := &tt[idx]
	if existing.Bound == BoundNone || existing.Key != key || int16(depth) >= existing.Depth {
		*existing = TTEntry{
			Key:   key,
			Move:  move & 0xffff,
			Score: score,
			Depth: int16(depth),
			Bound: bound,
		}
	}
}

// ScoreToTT/ScoreFromTT convert between a node-relative score (as used
// throughout the search) and a root-relative score (as stored in the
// table). Mate scores are ply-relative -- "mate in N plies from here" --
// so a mate score stored at one ply and probed at a different ply must be
// re-based by the difference, or the reported mate distance corrupts.
// Non-mate scores are ply-independent and pass through unchanged.
func ScoreToTT(score int32, ply int) int32 {
	if score >= MateScoreThreshold {
		return score + int32(ply)
	}
	if score <= -MateScoreThreshold {
		return score - int32(ply)
	}
	return score
}

func ScoreFromTT(score int32, ply int) int32 {
	if score >= MateScoreThreshold {
		return score - int32(ply)
	}
	if score <= -MateScoreThreshold {
		return score + int32(ply)
	}
	return score
}
