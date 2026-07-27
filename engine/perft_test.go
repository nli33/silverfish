package engine_test

import (
	"testing"

	"silverfish/engine"
)

// Node counts are the standard perft reference values (also used by
// tools/run_perft.sh against known-good engines) for the startpos and the
// Kiwipete/CPW test positions covering castling, en passant, promotion, and
// discovered check.
var perftCases = []struct {
	name  string
	fen   string
	depth int
	want  uint64
}{
	{"startpos", "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", 4, 197281},
	{"kiwipete", "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1", 3, 97862},
	{"position 3", "8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1", 5, 674624},
	{"position 4", "r3k2r/Pppp1ppp/1b3nbN/nP6/BBP1P3/q4N2/Pp1P2PP/R2Q1RK1 w kq - 0 1", 3, 9467},
	{"position 5", "rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8", 3, 62379},
	{"position 6", "r4rk1/1pp1qppp/p1np1n2/2b1p1B1/2B1P1b1/P1NP1N2/1PP1QPPP/R4RK1 w - - 0 10", 3, 89890},
}

func TestPerft(t *testing.T) {
	for _, tc := range perftCases {
		t.Run(tc.name, func(t *testing.T) {
			pos := engine.FromFEN(tc.fen)
			got := engine.Perft(&pos, tc.depth, false)
			if got != tc.want {
				t.Errorf("Perft(%q, %d) = %d, want %d", tc.fen, tc.depth, got, tc.want)
			}
		})
	}
}

// Deeper perft runs are slow (kiwipete@4 alone takes several seconds); keep
// them out of the default `go test` loop but available via -short=false.
func TestPerftDeep(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping deep perft in -short mode")
	}

	cases := []struct {
		name  string
		fen   string
		depth int
		want  uint64
	}{
		{"startpos", "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", 5, 4865609},
		{"kiwipete", "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1", 4, 4085603},
		{"position 4", "r3k2r/Pppp1ppp/1b3nbN/nP6/BBP1P3/q4N2/Pp1P2PP/R2Q1RK1 w kq - 0 1", 4, 422333},
		{"position 6", "r4rk1/1pp1qppp/p1np1n2/2b1p1B1/2B1P1b1/P1NP1N2/1PP1QPPP/R4RK1 w - - 0 10", 4, 3894594},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pos := engine.FromFEN(tc.fen)
			got := engine.Perft(&pos, tc.depth, false)
			if got != tc.want {
				t.Errorf("Perft(%q, %d) = %d, want %d", tc.fen, tc.depth, got, tc.want)
			}
		})
	}
}
