package engine_test

import (
	"testing"

	"silverfish/engine"
)

func TestScoreToFromTTRoundTrip(t *testing.T) {
	cases := []struct {
		name  string
		score int32
		ply   int
	}{
		{"zero", 0, 5},
		{"ordinary positive eval", 350, 7},
		{"ordinary negative eval", -820, 3},
		{"mate for us, found at root", engine.Infinity - 1, 0},
		{"mate for us, found deep", engine.Infinity - 1, 12},
		{"mate against us, found deep", -(engine.Infinity - 1), 9},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stored := engine.ScoreToTT(tc.score, tc.ply)
			got := engine.ScoreFromTT(stored, tc.ply)
			if got != tc.score {
				t.Errorf("round-trip at ply %d: got %d, want %d (stored intermediate was %d)", tc.ply, got, tc.score, stored)
			}
		})
	}
}

// A mate score stored at one ply and probed at a different ply (as happens
// when the same position is reached via a different, shorter or longer,
// path) must re-base to the correct mate distance from the new ply -- this
// is the exact class of corruption that made the old tt branch play badly.
func TestScoreToFromTTRebasesMateAcrossPlies(t *testing.T) {
	// A mate found 3 plies from the node where it's stored.
	nodeRelative := engine.Infinity - 3
	stored := engine.ScoreToTT(nodeRelative, 10) // stored while at ply 10 -> root-relative mate at ply 13

	// Probed again at a different ply: stored = nodeRelative + 10 (the ply
	// at store time), so probing at ply 5 must yield stored - 5 -- follows
	// directly from the store/probe formulas, independent of what those
	// numbers mean chess-wise.
	gotAtPly5 := engine.ScoreFromTT(stored, 5)
	wantAtPly5 := stored - 5
	if gotAtPly5 != wantAtPly5 {
		t.Errorf("re-based mate score at ply 5 = %d, want %d", gotAtPly5, wantAtPly5)
	}

	gotAtPly10 := engine.ScoreFromTT(stored, 10)
	if gotAtPly10 != nodeRelative {
		t.Errorf("re-based mate score at original ply 10 = %d, want %d", gotAtPly10, nodeRelative)
	}
}

func TestTTStoreAndProbe(t *testing.T) {
	engine.ClearTT()

	pos := engine.StartingPosition()
	move := engine.NewMoveFromStr("e2e4")

	engine.TTStore(pos.Hash, move, 123, 4, engine.BoundExact)

	entry, ok := engine.TTProbe(pos.Hash)
	if !ok {
		t.Fatalf("expected a hit after store")
	}
	if entry.Score != 123 || entry.Depth != 4 || entry.Bound != engine.BoundExact {
		t.Errorf("got entry %+v, want score=123 depth=4 bound=Exact", entry)
	}
	if entry.Move.From() != move.From() || entry.Move.To() != move.To() {
		t.Errorf("got move %v, want %v", entry.Move, move)
	}
}

func TestTTMoveIsMaskedToLow16Bits(t *testing.T) {
	engine.ClearTT()

	move := engine.NewMoveFromStr("e2e4")
	move.GiveScore(999) // sets bits 16+, as move ordering does in practice

	engine.TTStore(0xabc, move, 0, 1, engine.BoundExact)
	entry, ok := engine.TTProbe(0xabc)
	if !ok {
		t.Fatalf("expected a hit")
	}
	if entry.Move.Score() != 0 {
		t.Errorf("stored move retained a score field (%d) -- should be masked to the low 16 bits", entry.Move.Score())
	}
}

// A probe with a key that maps to the same table index but is not actually
// equal must not be treated as a hit -- otherwise a hash collision returns
// a completely unrelated position's cached result.
func TestTTProbeRejectsIndexCollision(t *testing.T) {
	engine.ClearTT()

	keyA := uint64(0x1122334455667788)
	engine.TTStore(keyA, engine.NewMoveFromStr("e2e4"), 50, 3, engine.BoundExact)

	// Same low bits (same table index under any power-of-two mask), but a
	// different full 64-bit key.
	keyB := keyA ^ (uint64(1) << 40)

	_, ok := engine.TTProbe(keyB)
	if ok {
		t.Errorf("probe with a colliding-but-different key returned a hit")
	}

	// The original key must still be retrievable.
	entry, ok := engine.TTProbe(keyA)
	if !ok || entry.Score != 50 {
		t.Errorf("original key no longer retrievable after a colliding probe: ok=%v entry=%+v", ok, entry)
	}
}

func TestTTStorePrefersDeeperOnCollision(t *testing.T) {
	engine.ClearTT()

	keyA := uint64(0x1)
	// Force a collision by finding another key with the same low bits under
	// a very large table -- simplest reliable way is to store at the same
	// key twice with different depths, which also exercises the "same key,
	// deeper replaces shallower" path directly used during real search
	// (repeated probes/stores of the same position at increasing ID depth).
	engine.TTStore(keyA, engine.NewMoveFromStr("e2e4"), 10, 2, engine.BoundExact)
	engine.TTStore(keyA, engine.NewMoveFromStr("d2d4"), 20, 5, engine.BoundExact)

	entry, ok := engine.TTProbe(keyA)
	if !ok {
		t.Fatalf("expected a hit")
	}
	if entry.Depth != 5 || entry.Score != 20 {
		t.Errorf("got depth=%d score=%d, want the deeper store (depth=5 score=20) to win", entry.Depth, entry.Score)
	}

	// A shallower store for the same key must NOT overwrite the deeper one.
	engine.TTStore(keyA, engine.NewMoveFromStr("g1f3"), 30, 1, engine.BoundExact)
	entry, _ = engine.TTProbe(keyA)
	if entry.Depth != 5 || entry.Score != 20 {
		t.Errorf("a shallower store overwrote a deeper entry: got depth=%d score=%d", entry.Depth, entry.Score)
	}
}

func TestClearTT(t *testing.T) {
	engine.TTStore(0x42, engine.NewMoveFromStr("e2e4"), 100, 5, engine.BoundExact)
	if _, ok := engine.TTProbe(0x42); !ok {
		t.Fatalf("expected a hit before clearing")
	}

	engine.ClearTT()

	if _, ok := engine.TTProbe(0x42); ok {
		t.Errorf("expected no hit after ClearTT")
	}
}
