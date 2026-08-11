// turn: 0 white 1 black

package engine

type Position struct {
	// Turn: 0=white 1=black
	Turn uint8

	// 0-5: white pieces
	// 10-15: black pieces
	// 5: NoSquare
	Board [64]uint8

	Pieces   [2][6]Bitboard
	Sides    [2]Bitboard
	Blockers Bitboard

	// Castling Rights:
	// 0 - can white castle kingside?
	// 1 - can white castle queenside?
	// 2 - can black castle kingside?
	// 3 - can black castle queenside?
	CastlingRights uint8

	// half-move clock
	Rule50 uint8

	// number of turns
	Ply uint16

	// square that is available for en passant, or NoSquare if no enpassant available
	// basically the square that the last pawn skipped over (if it moved forward 2 squares)
	//
	// note: field should only be set for the halfmove right after a pawn moves forward 2 squares
	// example: after a2a4, EPsq = a3. After black moves (not EP), it is NoSquare
	EnPassantSquare Square

	// past states
	History []State

	// Net is the immutable network shared across positions; Acc is this
	// position's mutable per-perspective evaluation state.
	Net *Network
	Acc Accumulator

	// Hash is this position's Zobrist key, maintained incrementally by
	// PutPiece/RemovePiece and DoMove/UndoMove. Used for repetition
	// detection (IsRepetition).
	Hash uint64

	// needsAccRefresh[color] is set (by PutPiece/RemovePiece/MovePiece/
	// CapturePiece/UncapturePiece) when color's own king moves during the
	// in-progress DoMove/UndoMove call -- HalfKA feature indices are
	// relative to the owner's own king square, so a king move changes
	// every one of that perspective's active features at once, not just
	// the king's own slot, and can't be patched incrementally. Once set,
	// any other piece touched for that same perspective during the same
	// call is also skipped (see maybeUpdateOwnAcc) since a full rebuild is
	// coming anyway and would just redo the work. DoMove/UndoMove call
	// flushAccRefresh() before returning; other direct callers of those
	// five piece-mutation functions must do the same or risk a stale
	// accumulator. FromFEN and PutPiecesBB don't use this mechanism at all
	// -- they build a position from an empty board in arbitrary order, so
	// they go through putPieceBoardOnly/removePieceBoardOnly (no
	// accumulator work) and an unconditional refreshAccPerspective call
	// for both colors once the board is complete instead.
	needsAccRefresh [2]bool
}

type State struct {
	CastlingRights  uint8
	EnPassantSquare Square
	Rule50          uint8
	CapturedPiece   uint8
	MovedPiece      uint8
	Hash            uint64
}

const (
	WhiteKingside uint8 = 1 << iota
	WhiteQueenside
	BlackKingside
	BlackQueenside
)

func StartingPosition() Position {
	return FromFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
}

// Clone returns a deep copy. Position must not be copied by plain assignment
// (`p2 := p1` / passing by value) -- Acc and History alias mutable state via
// slices, so a plain copy shares them with the original. Net is the sole
// exception: it's immutable, so sharing the pointer is safe.
func (pos *Position) Clone() Position {
	clone := *pos
	clone.Acc = pos.Acc.Clone()
	clone.History = append([]State(nil), pos.History...)
	return clone
}

// ownKingSq returns perspective's own king square. Only ever called when
// that king is actually on the board (exactly one bit set in
// Pieces[perspective][King]) -- see maybeUpdateOwnAcc, which is what
// guarantees that.
func (pos *Position) ownKingSq(perspective uint8) Square {
	return Lsb(pos.Pieces[perspective][King])
}

// maybeUpdateOwnAcc reports whether the caller should still do a normal
// incremental accumulator update for perspective's own half, given a piece
// of color == perspective is being placed/removed/moved. If piece is a
// King, perspective's own king is moving -- every one of perspective's
// active HalfKA feature indices changes at once (they're all relative to
// perspective's own king square), so this can't be patched incrementally;
// instead it flags perspective for a full rebuild (see needsAccRefresh,
// flushAccRefresh) and returns false. If a rebuild is already pending for
// perspective (e.g. a rook moved as part of the same castling move that
// already touched the king), also returns false -- any incremental work
// here would just be redone by the eventual full refresh, and doing it
// anyway would need perspective's king square, which may be transiently
// off the board mid-castle.
func (pos *Position) maybeUpdateOwnAcc(perspective uint8, piece uint8) bool {
	if piece == King {
		pos.needsAccRefresh[perspective] = true
		return false
	}
	return !pos.needsAccRefresh[perspective]
}

// flushAccRefresh performs any accumulator rebuilds queued by
// maybeUpdateOwnAcc (own-king moves) since the last flush. Must be called
// once after every sequence of board mutations that can move a king --
// DoMove, UndoMove, FromFEN, and PutPiecesBB all call this before
// returning; other direct callers of PutPiece/RemovePiece/MovePiece/
// CapturePiece/UncapturePiece must do the same or risk a stale accumulator.
func (pos *Position) flushAccRefresh() {
	for color := White; color <= Black; color++ {
		if !pos.needsAccRefresh[color] {
			continue
		}
		pos.refreshAccPerspective(color)
		pos.needsAccRefresh[color] = false
	}
}

// refreshAccPerspective rebuilds perspective's accumulator half from
// scratch off the current board state (all pieces, both colors, excluding
// perspective's own king -- never a tracked feature, see FeatureIndex).
func (pos *Position) refreshAccPerspective(perspective uint8) {
	// Real game positions always have exactly one king per side, but some
	// existing tests build deliberately incomplete positions (e.g. a
	// movegen fixture with no king at all) via FromFEN/PutPiecesBB purely
	// to exercise movegen in isolation -- never intending to call Evaluate
	// on them. There's no valid king bucket without a king, so there's no
	// valid NNUE evaluation either; leave this perspective at bias-only
	// (same as a fresh Reset) rather than computing on a bogus Square(64).
	if pos.Pieces[perspective][King] == 0 {
		copy(pos.Acc.Values[perspective], pos.Net.BInput)
		return
	}

	kingSq := pos.ownKingSq(perspective)
	features := make([]uint16, 0, 32)
	for color := White; color <= Black; color++ {
		for piece := Pawn; piece <= King; piece++ {
			if color == perspective && piece == King {
				continue
			}
			bb := pos.Pieces[color][piece]
			for bb != 0 {
				sq := PopLsb(&bb)
				features = append(features, FeatureIndex(perspective, kingSq, color, piece, sq))
			}
		}
	}
	pos.Acc.Refresh(pos.Net, features, perspective)
}

// putPieceBoardOnly mutates board/bitboard/hash state only -- no NNUE
// accumulator work. Used directly by FromFEN and PutPiecesBB, which build a
// position from an empty board one piece at a time in whatever order the
// input happens to list pieces in: under HalfKA, a piece placed for one
// color may need the *opponent's* king square before that king has even
// been placed yet, which is invalid mid-construction (see
// refreshAccPerspective) -- unlike DoMove/UndoMove, where both kings are
// always already on the board (a single piece moves at a time, never from
// an empty position). Callers must place every piece via this (or
// PutPiece) and then rebuild the accumulator once at the end from the
// complete board, after both kings are guaranteed present.
func (pos *Position) putPieceBoardOnly(sq Square, piece uint8, color uint8) {
	sqBB := Bitboard(1 << sq)
	pos.Pieces[color][piece] |= sqBB

	if color == Black {
		piece += 10
	}
	pos.Board[sq] = piece
	pos.Blockers |= sqBB
	pos.Sides[color] |= sqBB

	pos.Hash ^= pieceSqKey(sq, piece)
}

func (pos *Position) PutPiece(sq Square, piece uint8, color uint8) {
	other := color ^ 1
	if pos.maybeUpdateOwnAcc(color, piece) {
		f := FeatureIndex(color, pos.ownKingSq(color), color, piece, sq)
		pos.Acc.Add(pos.Net, f, color)
	}
	fOther := FeatureIndex(other, pos.ownKingSq(other), color, piece, sq)
	pos.Acc.Add(pos.Net, fOther, other)

	pos.putPieceBoardOnly(sq, piece, color)
}

func (pos *Position) PutPiecesBB(pieces [2][6]Bitboard) {
	// this check is for testing purposes only
	// .. well this whole function is for testing purposes only
	if pos.Net == nil {
		pos.Net = defaultNet
		pos.Acc = NewAccumulator(pos.Net)
	}

	// Board-only placement (see putPieceBoardOnly/removePieceBoardOnly):
	// this rebuilds the whole board from scratch, square by square, so
	// under HalfKA a piece placed for one color may need the opponent's
	// king square before that king has even been placed yet -- invalid
	// mid-construction. Rebuild the accumulator once at the end instead,
	// once both kings are guaranteed present.
	for sq := SquareA1; sq <= SquareH8; sq++ {
		// removePieceBoardOnly on an already-empty square would read
		// NoColor/garbage piece data, so only call it where there's
		// something to remove -- relevant when reusing the same Position
		// across calls.
		if pos.Board[sq] != NoPiece {
			pos.removePieceBoardOnly(sq)
		}
		for piece := Pawn; piece <= King; piece++ {
			for color := White; color <= Black; color++ {
				if pieces[color][piece]&(1<<sq) != 0 {
					pos.putPieceBoardOnly(sq, piece, color)
				}
			}
		}
	}

	pos.refreshAccPerspective(White)
	pos.refreshAccPerspective(Black)
}

// MovePiece relocates a piece from one square to another (no capture, no
// promotion), equivalent to RemovePiece(from) followed by PutPiece(to, piece,
// color) but doing the NNUE accumulator update as a single fused AddSub pass
// per perspective instead of two separate passes.
func (pos *Position) MovePiece(from, to Square, piece, color uint8) {
	fromBB := Bitboard(1 << from)
	toBB := Bitboard(1 << to)

	boardPieceFrom := pos.Board[from] // pre-adjustment, for the hash key
	pos.Pieces[color][piece] = pos.Pieces[color][piece]&^fromBB | toBB
	pos.Blockers = pos.Blockers&^fromBB | toBB
	pos.Sides[color] = pos.Sides[color]&^fromBB | toBB

	boardPieceTo := piece
	if color == Black {
		boardPieceTo += 10
	}
	pos.Board[from] = NoPiece
	pos.Board[to] = boardPieceTo

	pos.Hash ^= pieceSqKey(from, boardPieceFrom)
	pos.Hash ^= pieceSqKey(to, boardPieceTo)

	other := color ^ 1
	if pos.maybeUpdateOwnAcc(color, piece) {
		kingSq := pos.ownKingSq(color)
		fFrom := FeatureIndex(color, kingSq, color, piece, from)
		fTo := FeatureIndex(color, kingSq, color, piece, to)
		pos.Acc.AddSub(pos.Net, fTo, fFrom, color)
	}
	otherKingSq := pos.ownKingSq(other)
	fOtherFrom := FeatureIndex(other, otherKingSq, color, piece, from)
	fOtherTo := FeatureIndex(other, otherKingSq, color, piece, to)
	pos.Acc.AddSub(pos.Net, fOtherTo, fOtherFrom, other)
}

// CapturePiece relocates a piece from one square to another while capturing
// an enemy piece on the destination square -- equivalent to RemovePiece(from)
// + RemovePiece(to) + PutPiece(to, piece, color) but doing the NNUE
// accumulator update as a single fused AddSubSub pass per perspective.
func (pos *Position) CapturePiece(from, to Square, piece, color, capturedPiece uint8) {
	theirColor := color ^ 1
	fromBB := Bitboard(1 << from)
	toBB := Bitboard(1 << to)

	boardPieceFrom := pos.Board[from] // pre-adjustment, for the hash key
	boardPieceTo := pos.Board[to]     // captured piece, pre-adjustment

	pos.Pieces[color][piece] = pos.Pieces[color][piece]&^fromBB | toBB
	pos.Pieces[theirColor][capturedPiece] &^= toBB
	pos.Sides[color] = pos.Sides[color]&^fromBB | toBB
	pos.Sides[theirColor] &^= toBB
	pos.Blockers &^= fromBB // `to` stays occupied throughout

	newBoardPieceTo := piece
	if color == Black {
		newBoardPieceTo += 10
	}
	pos.Board[from] = NoPiece
	pos.Board[to] = newBoardPieceTo

	pos.Hash ^= pieceSqKey(from, boardPieceFrom)
	pos.Hash ^= pieceSqKey(to, boardPieceTo)
	pos.Hash ^= pieceSqKey(to, newBoardPieceTo)

	// capturedPiece is never King (kings can't be captured in a legal
	// game), so only the mover (piece) can trigger a refresh here.
	if pos.maybeUpdateOwnAcc(color, piece) {
		kingSq := pos.ownKingSq(color)
		fFrom := FeatureIndex(color, kingSq, color, piece, from)
		fTo := FeatureIndex(color, kingSq, color, piece, to)
		fCaptured := FeatureIndex(color, kingSq, theirColor, capturedPiece, to)
		pos.Acc.AddSubSub(pos.Net, fTo, fFrom, fCaptured, color)
	}
	theirKingSq := pos.ownKingSq(theirColor)
	fTheirFrom := FeatureIndex(theirColor, theirKingSq, color, piece, from)
	fTheirTo := FeatureIndex(theirColor, theirKingSq, color, piece, to)
	fTheirCaptured := FeatureIndex(theirColor, theirKingSq, theirColor, capturedPiece, to)
	pos.Acc.AddSubSub(pos.Net, fTheirTo, fTheirFrom, fTheirCaptured, theirColor)
}

// UncapturePiece is the inverse of CapturePiece: it moves a piece from `to`
// back to `from` and restores a previously-captured enemy piece on `to`,
// fusing the accumulator update (AddAddSub) into one pass per perspective.
func (pos *Position) UncapturePiece(from, to Square, piece, color, capturedPiece uint8) {
	theirColor := color ^ 1
	fromBB := Bitboard(1 << from)
	toBB := Bitboard(1 << to)

	boardPieceTo := pos.Board[to] // mover currently on `to`, pre-adjustment

	pos.Pieces[color][piece] = pos.Pieces[color][piece]&^toBB | fromBB
	pos.Pieces[theirColor][capturedPiece] |= toBB
	pos.Sides[color] = pos.Sides[color]&^toBB | fromBB
	pos.Sides[theirColor] |= toBB
	pos.Blockers |= fromBB // `to` stays occupied throughout

	newBoardPieceFrom := piece
	if color == Black {
		newBoardPieceFrom += 10
	}
	newBoardPieceTo := capturedPiece
	if theirColor == Black {
		newBoardPieceTo += 10
	}
	pos.Board[from] = newBoardPieceFrom
	pos.Board[to] = newBoardPieceTo

	pos.Hash ^= pieceSqKey(to, boardPieceTo)
	pos.Hash ^= pieceSqKey(from, newBoardPieceFrom)
	pos.Hash ^= pieceSqKey(to, newBoardPieceTo)

	// capturedPiece is never King, so only the mover (piece) can trigger a
	// refresh here -- same reasoning as CapturePiece, which this undoes.
	if pos.maybeUpdateOwnAcc(color, piece) {
		kingSq := pos.ownKingSq(color)
		fFrom := FeatureIndex(color, kingSq, color, piece, from)
		fTo := FeatureIndex(color, kingSq, color, piece, to)
		fCaptured := FeatureIndex(color, kingSq, theirColor, capturedPiece, to)
		pos.Acc.AddAddSub(pos.Net, fFrom, fCaptured, fTo, color)
	}
	theirKingSq := pos.ownKingSq(theirColor)
	fTheirFrom := FeatureIndex(theirColor, theirKingSq, color, piece, from)
	fTheirTo := FeatureIndex(theirColor, theirKingSq, color, piece, to)
	fTheirCaptured := FeatureIndex(theirColor, theirKingSq, theirColor, capturedPiece, to)
	pos.Acc.AddAddSub(pos.Net, fTheirFrom, fTheirCaptured, fTheirTo, theirColor)
}

// removePieceBoardOnly mutates board/bitboard/hash state only -- no NNUE
// accumulator work. Returns the removed piece's (color, piece) so the
// caller can do accumulator work separately if it wants to (see
// RemovePiece), or skip it entirely (see FromFEN/PutPiecesBB's use --
// same reasoning as putPieceBoardOnly).
func (pos *Position) removePieceBoardOnly(sq Square) (color uint8, piece uint8) {
	boardPiece := pos.Board[sq] // pre-adjustment (0-5 white, 10-15 black), for the hash key
	piece = boardPiece
	color = ColorOf(piece)
	if color == Black {
		piece -= 10
	}
	sqBB := Bitboard(1 << sq)
	pos.Board[sq] = NoPiece
	pos.Blockers &^= sqBB
	pos.Sides[color] &^= sqBB
	if color != NoColor {
		pos.Pieces[color][piece] &^= sqBB
	}

	pos.Hash ^= pieceSqKey(sq, boardPiece)
	return color, piece
}

func (pos *Position) RemovePiece(sq Square) {
	color, piece := pos.removePieceBoardOnly(sq)

	// Guarded like the Pieces bitboard update above: color can be NoColor
	// if this were ever called on an already-empty square (not reachable
	// from DoMove/UndoMove -- see PutPiecesBB's guard, the one call site
	// that could reach it).
	if color != NoColor {
		other := color ^ 1
		if pos.maybeUpdateOwnAcc(color, piece) {
			f := FeatureIndex(color, pos.ownKingSq(color), color, piece, sq)
			pos.Acc.Remove(pos.Net, f, color)
		}
		fOther := FeatureIndex(other, pos.ownKingSq(other), color, piece, sq)
		pos.Acc.Remove(pos.Net, fOther, other)
	}
}

func (pos *Position) Equals(otherPos Position) bool {
	return pos.Turn == otherPos.Turn &&
		pos.Pieces == otherPos.Pieces &&
		pos.CastlingRights == otherPos.CastlingRights &&
		pos.Rule50 == otherPos.Rule50 &&
		pos.Ply == otherPos.Ply &&
		pos.EnPassantSquare == otherPos.EnPassantSquare
}

// (color, piece)
// see engine.Position.Board documentation for encoding
func (pos *Position) GetSquare(sq Square) (uint8, uint8) {
	p := pos.Board[sq]
	if p == NoPiece {
		return NoColor, NoPiece
	}

	if pos.Board[sq] >= 10 {
		return Black, p - 10
	} else {
		return White, p
	}
}

func (pos *Position) FullMoves() uint16 {
	return (pos.Ply)/2 + 1
}
