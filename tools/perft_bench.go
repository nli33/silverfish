//go:build ignore

// Consolidated perft correctness + speed check. Runs a fixed set of known
// positions (a mix of startpos and hand-picked tactical/edge-case positions
// that have historically caught move-generation bugs startpos alone
// wouldn't) directly against engine.Perft -- no UCI, no subprocess, no
// shell polling required.
//
// Usage:
//
//	go run tools/perft_bench.go             # run all cases
//	go run tools/perft_bench.go -depth 7     # override depth for all cases
//	go run tools/perft_bench.go -case 2      # run only case index 2 (0-based)
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"silverfish/engine"
)

type testCase struct {
	name     string
	fen      string // empty means starting position
	depth    int
	expected uint64
}

var testCases = []testCase{
	{"startpos", "", 5, 4865609},
	{"castling rights", "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1", 4, 4085603},
	{"en passant / promotion heavy", "8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1", 6, 11030083},
	{"promotion + castling", "r3k2r/Pppp1ppp/1b3nbN/nP6/BBP1P3/q4N2/Pp1P2PP/R2Q1RK1 w kq - 0 1", 5, 15833292},
	{"pinned pieces / checks", "rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8", 5, 89941194},
	{"quiet middlegame", "r4rk1/1pp1qppp/p1np1n2/2b1p1B1/2B1P1b1/P1NP1N2/1PP1QPPP/R4RK1 w - - 0 10", 4, 3894594},
}

func main() {
	depthOverride := flag.Int("depth", 0, "override depth for all cases (0 = use each case's default)")
	caseIdx := flag.Int("case", -1, "run only this case index (0-based); -1 = run all")
	flag.Parse()

	engine.Init()

	cases := testCases
	if *caseIdx >= 0 {
		if *caseIdx >= len(testCases) {
			fmt.Printf("error: -case %d out of range (0..%d)\n", *caseIdx, len(testCases)-1)
			os.Exit(1)
		}
		cases = testCases[*caseIdx : *caseIdx+1]
	}

	failed := false
	for _, tc := range cases {
		var pos engine.Position
		if tc.fen == "" {
			pos = engine.StartingPosition()
		} else {
			pos = engine.FromFEN(tc.fen)
		}

		depth := tc.depth
		if *depthOverride != 0 {
			depth = *depthOverride
		}

		start := time.Now()
		nodes := engine.Perft(&pos, depth, false)
		elapsed := time.Since(start)
		nps := float64(nodes) / elapsed.Seconds()

		if depth == tc.depth && nodes != tc.expected {
			fmt.Printf("FAIL: [%s] depth=%d got=%d expected=%d (%s, %.0f nps)\n",
				tc.name, depth, nodes, tc.expected, elapsed, nps)
			failed = true
			continue
		}

		note := ""
		if depth != tc.depth {
			note = " (depth overridden, no correctness check)"
		}
		fmt.Printf("PASS: [%s] depth=%d nodes=%d %s (%.0f nps)%s\n",
			tc.name, depth, nodes, elapsed, nps, note)
	}

	if failed {
		os.Exit(1)
	}
}
