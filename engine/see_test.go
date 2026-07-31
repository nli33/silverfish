package engine_test

import (
	"silverfish/engine"
	"testing"
)

func TestSEE(t *testing.T) {
	cases := []struct {
		name string
		fen  string
		move string
		want int
	}{
		{
			// White queen takes a pawn defended by another black pawn:
			// after Qxe5, the pawn on d6 recaptures. Losing exchange:
			// +100 (pawn) then -900 (queen) = -800 net for white.
			name: "queen takes pawn-defended pawn loses the queen",
			fen:  "4k3/8/3p4/4p3/8/8/4Q3/4K3 w - - 0 1",
			move: "e2e5",
			want: 100 - 900,
		},
		{
			// Knight takes a pawn with no black defenders at all --
			// clean, undisputed win of a pawn.
			name: "knight takes fully undefended pawn",
			fen:  "4k3/8/8/4p3/8/3N4/8/4K3 w - - 0 1",
			move: "d3e5",
			want: 100,
		},
		{
			// X-ray: white rook on e4 (front) takes the pawn on e5, then
			// black's rook on e8 recaptures the white rook, then white's
			// second rook on e1 -- invisible to a naive "just check
			// direct attackers" scan, since e4's rook was in the way
			// until it moved -- recaptures black's rook. Net for white:
			// +100 (pawn) - 500 (first rook lost) + 500 (black rook won)
			// = +100, a favorable exchange only visible with X-ray
			// attackers accounted for.
			name: "x-ray attacker behind the first rook wins the exchange",
			fen:  "4r1k1/8/8/4p3/4R3/8/8/4R1K1 w - - 0 1",
			move: "e4e5",
			want: 100,
		},
		{
			name: "en passant capture is a clean pawn win when undefended",
			fen:  "4k3/8/8/8/3pP3/8/8/4K3 b - e3 0 1",
			move: "d4e3",
			want: 100,
		},
		{
			// Regression test for a real bug: an earlier version of see()
			// stopped simulating the exchange as soon as one ply's local
			// gain looked bad, which silently dropped later attackers
			// (here, a second knight and a second rook) from
			// consideration entirely. Only a full forward simulation
			// with a backward minimax pass gets a 4-ply exchange like
			// this right -- verified against a legal-moves-based brute
			// force (see conversation), not hand-derived, since manually
			// tracing a chain this long is exactly the kind of thing
			// that's easy to get wrong by eye.
			name: "full exchange chain matters even when an intermediate ply looks bad",
			fen:  "3rr1k1/8/2n3n1/4p3/8/4R3/8/4R1K1 w - - 0 1",
			move: "e3e5",
			want: -400,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pos := engine.FromFEN(c.fen)
			// NewMoveFromStr can't know a move is en passant/castling
			// from the UCI string alone (those flags come from movegen
			// context) -- find the matching legal move instead, so it
			// carries the right flags.
			want := engine.NewMoveFromStr(c.move)
			var move engine.Move
			found := false
			for _, m := range pos.LegalMoves() {
				// Compare from/to squares only (bits 0-11) -- the flag
				// bits (14-15: castling/en passant/promotion) live in
				// this same low-16 range but NewMoveFromStr never sets
				// them, so a full low-16 match would never find an en
				// passant/castling move.
				if m&0xfff == want&0xfff {
					move = m
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("move %s not found among legal moves for %s", c.move, c.fen)
			}

			got := engine.SEE(&pos, move)
			if got != c.want {
				t.Errorf("SEE(%s, %s) = %d, want %d", c.fen, c.move, got, c.want)
			}
		})
	}
}
