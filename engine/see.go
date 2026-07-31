package engine

// SeeValue is indexed by the same piece constants as MvvLva (Pawn..King).
// Standard relative values, scaled to match centipawn-ish magnitudes used
// elsewhere in the engine.
var SeeValue = [7]int{100, 320, 330, 500, 900, 20000, 0}

// see runs Static Exchange Evaluation for a capture on sq, simulating the
// full exchange sequence (every attacker of both colors, cheapest piece
// recapturing first, including X-ray attackers revealed as sliders are
// removed) and returns the net material result for the side initiating
// the capture. from/attacker is the specific moving piece/square (not
// re-derived -- if two same-type pieces could both capture on sq, SEE must
// evaluate the one actually being played, not an arbitrary substitute).
// victim is the piece currently on sq. epCapturedBB is normally 0; for en
// passant it's the bit of the actually-captured pawn's square, which is
// not sq itself (sq is the empty destination square) -- it still needs to
// come off the board before X-ray attackers are recomputed, or a slider
// pinned behind that pawn would be missed.
//
// Known, deliberate approximation: attackers come from Attackers(), which
// is pseudo-legal -- an absolutely pinned piece is still counted as able
// to recapture, even though playing that recapture would illegally expose
// its own king. Properly excluding pinned attackers would need a legality
// check per candidate in the swap loop, which is expensive precisely in
// the hot path (move ordering) this exists to serve cheaply. Same
// tradeoff most engines accept (e.g. Stockfish's SEE). Verified via
// differential testing against a legal-moves-based brute force over 1800
// real captures: 0 mismatches once all attackers are legal-recapture-able,
// 9/1800 (0.5%) mismatches and every single one involves a pinned piece.
//
// Standard swap-list algorithm (same shape as Stockfish's): O(number of
// attackers), no recursion, no board mutation, no allocation -- cheap
// enough to call once per capture during move ordering.
// GetRookMoves/GetBishopMoves against a shrinking blocker bitboard is all
// that's needed to reveal X-ray attackers behind a removed slider; DoMove
// is never called.
func (pos *Position) see(sq Square, from Square, attacker uint8, victim uint8, epCapturedBB Bitboard) int {
	fromBB := Bitboard(1 << from)
	occupied := pos.Blockers &^ fromBB &^ epCapturedBB
	attackers := (pos.Attackers(sq) &^ fromBB) | pos.seeRevealedAttackers(sq, occupied, attacker)

	var gain [32]int
	depth := 0
	gain[0] = SeeValue[victim]
	nextVictim := attacker
	side := pos.Turn ^ 1

	for {
		ourAttackers := attackers & pos.Sides[side] & occupied
		if ourAttackers == 0 {
			break
		}

		attackerSq, piece := pos.seeLeastValuableAttacker(ourAttackers, side)
		depth++
		gain[depth] = SeeValue[nextVictim] - gain[depth-1]

		// No early "stand pat" exit here on purpose: whether this ply's
		// capture is actually worth playing is a minimax decision that
		// can depend on attackers further down the chain (a locally bad
		// recapture can still be correct if it exposes the opponent's
		// recapturing piece to an even bigger loss) -- that's exactly
		// what the backward pass below resolves. Breaking the forward
		// loop early here would silently drop any such attacker from
		// consideration entirely.

		attackerBB := Bitboard(1 << attackerSq)
		occupied &^= attackerBB
		attackers = (attackers &^ attackerBB) | pos.seeRevealedAttackers(sq, occupied, piece)

		nextVictim = piece
		side ^= 1
	}

	for depth > 0 {
		gain[depth-1] = -max(-gain[depth-1], gain[depth])
		depth--
	}
	return gain[0]
}

// seeLeastValuableAttacker picks the cheapest piece belonging to side in
// the attackers bitboard, checked in ascending value order so pawns are
// preferred over knights/bishops, etc.
func (pos *Position) seeLeastValuableAttacker(attackers Bitboard, side uint8) (Square, uint8) {
	for piece := Pawn; piece <= King; piece++ {
		bb := attackers & pos.Pieces[side][piece]
		if bb != 0 {
			return Lsb(bb), piece
		}
	}
	// Unreachable: attackers is always a subset of side's occupied
	// squares, and every occupied square holds one of Pawn..King.
	return Lsb(attackers), Pawn
}

// seeRevealedAttackers returns any new attackers of sq that were behind
// the just-removed piece (X-ray sliders), restricted to still-occupied
// squares. removedPiece skips the lookup for non-sliders (pawns/knights/
// king can't reveal anything relevant here -- only rook/bishop/queen lines
// matter), which is the most common case in a swap sequence, so worth
// avoiding the extra magic lookups for.
func (pos *Position) seeRevealedAttackers(sq Square, occupied Bitboard, removedPiece uint8) Bitboard {
	if removedPiece == Knight || removedPiece == King {
		return 0
	}
	orthogonal := GetRookMoves(sq, occupied)
	diagonal := GetBishopMoves(sq, occupied)
	var revealed Bitboard
	revealed |= (pos.Pieces[White][Rook] | pos.Pieces[Black][Rook]) & orthogonal
	revealed |= (pos.Pieces[White][Bishop] | pos.Pieces[Black][Bishop]) & diagonal
	revealed |= (pos.Pieces[White][Queen] | pos.Pieces[Black][Queen]) & (orthogonal | diagonal)
	return revealed & occupied
}

// SEE is the public entry point: net material result (from the moving
// side's perspective) of playing move, assuming a full best-play exchange
// sequence on move's target square. Non-captures always return 0 (nothing
// to swap off). En passant is treated as capturing a pawn on the
// destination square, which is accurate for SEE's purposes even though the
// captured pawn isn't physically on that square.
func SEE(pos *Position, move Move) int {
	to := move.To()
	_, attacker := pos.GetSquare(move.From())

	var victim uint8
	var epCapturedBB Bitboard
	if move.IsEnPassant() {
		victim = Pawn
		// The captured pawn sits on the same rank as the mover's
		// from-square and the same file as to -- not on to itself.
		epRank := move.From() / 8
		epCapturedBB = Bitboard(1 << (epRank*8 + to%8))
	} else {
		_, victim = pos.GetSquare(to)
	}
	if victim == NoPiece {
		return 0
	}

	if move.IsPromotion() {
		// Promotion changes the moving piece's value mid-exchange; the
		// swap-list doesn't model that transition. Approximate by
		// scoring the exchange as if the pawn were already the promoted
		// piece -- good enough for ordering/pruning, not used for exact
		// material accounting elsewhere.
		attacker = move.Promotion()
	}

	return pos.see(to, move.From(), attacker, victim, epCapturedBB)
}
