package engine

func (pos *Position) DoMove(move Move) {
	from := move.From()
	to := move.To()

	ourColor, movingPiece := pos.GetSquare(from)
	if ourColor != pos.Turn {
		panic("not our turn")
	}
	// don't use opp color from here, since en passants will yield NoColor
	_, capturedPiece := pos.GetSquare(to)

	// save the current state before moving
	// CapturedPiece will not include pawns captured en passant
	state := State{
		MovedPiece:      movingPiece,
		CapturedPiece:   capturedPiece, // may be NoPiece
		CastlingRights:  pos.CastlingRights,
		Rule50:          pos.Rule50,
		EnPassantSquare: pos.EnPassantSquare,
		Hash:            pos.Hash,
	}

	pos.Rule50++

	// update en passant square
	pos.Hash ^= enPassantKey(pos.EnPassantSquare)
	if movingPiece == Pawn {
		dist := int(to) - int(from)
		pawnDisplacement := PawnDisplacement(ourColor)
		if dist == 2*pawnDisplacement {
			pos.EnPassantSquare = Square(int(to) - pawnDisplacement)
		} else {
			pos.EnPassantSquare = NoSquare
		}
		pos.Rule50 = 0
	} else {
		pos.EnPassantSquare = NoSquare
	}
	pos.Hash ^= enPassantKey(pos.EnPassantSquare)

	// update castling rights
	pos.Hash ^= CastleKeys[pos.CastlingRights]
	switch movingPiece {
	case King:
		if ourColor == White && from == SquareE1 {
			pos.CastlingRights &^= 0b0011
		} else if ourColor == Black && from == SquareE8 {
			pos.CastlingRights &^= 0b1100
		}
	case Rook:
		switch {
		case from == SquareA1 && ourColor == White:
			pos.CastlingRights &^= WhiteQueenside
		case from == SquareA8 && ourColor == Black:
			pos.CastlingRights &^= BlackQueenside
		case from == SquareH1 && ourColor == White:
			pos.CastlingRights &^= WhiteKingside
		case from == SquareH8 && ourColor == Black:
			pos.CastlingRights &^= BlackKingside
		}
	}
	pos.Hash ^= CastleKeys[pos.CastlingRights]

	if move.IsCastling() {
		pos.RemovePiece(from)
		rookFromSquare := RookSquares[move]
		pos.RemovePiece(rookFromSquare)
		rookToSquare := Square(int(to) - KingCastlingDirection(move))
		pos.PutPiece(rookToSquare, Rook, ourColor)
		pos.PutPiece(to, King, ourColor)
	} else if move.IsEnPassant() {
		pos.RemovePiece(from)
		capturedPawnSq := Square(int(to) - PawnDisplacement(ourColor))
		pos.RemovePiece(capturedPawnSq)
		pos.PutPiece(to, movingPiece, ourColor)
	} else if move.IsPromotion() {
		pos.RemovePiece(from)
		if capturedPiece != NoPiece { // is a capture
			pos.RemovePiece(to)
		}
		pos.PutPiece(to, move.Promotion(), ourColor)
	} else if capturedPiece != NoPiece { // is a capture: fuse remove(from)+remove(to)+put(to) into one accumulator pass
		pos.Rule50 = 0
		pos.CapturePiece(from, to, movingPiece, ourColor, capturedPiece)
	} else { // plain quiet move: fuse remove(from)+put(to) into one accumulator pass
		pos.MovePiece(from, to, movingPiece, ourColor)
	}

	pos.Ply++
	pos.Turn ^= 1
	pos.Hash ^= TurnKey
	pos.History = append(pos.History, state)
}

// DoNullMove passes the turn without moving a piece (used by null-move
// pruning). Returns the previous en-passant square, to be passed to
// UndoNullMove. Unlike DoMove, this doesn't push onto pos.History -- the
// caller is expected to restore state itself via UndoNullMove rather than
// via UndoMove/IsRepetition machinery.
func (pos *Position) DoNullMove() Square {
	prevEP := pos.EnPassantSquare
	pos.Hash ^= enPassantKey(pos.EnPassantSquare)
	pos.EnPassantSquare = NoSquare
	pos.Turn ^= 1
	pos.Hash ^= TurnKey
	return prevEP
}

func (pos *Position) UndoNullMove(prevEP Square) {
	pos.Turn ^= 1
	pos.Hash ^= TurnKey
	pos.EnPassantSquare = prevEP
	pos.Hash ^= enPassantKey(pos.EnPassantSquare)
}

func (pos *Position) UndoMove(move Move) {
	from := move.From()
	to := move.To()

	n := len(pos.History)
	if n == 0 {
		panic("no reversible moves")
	}
	lastState := pos.History[n-1]
	pos.History = pos.History[:n-1]

	pos.Turn ^= 1
	pos.Ply--
	// pos.Turn is now the side that did the move

	pos.CastlingRights = lastState.CastlingRights
	pos.EnPassantSquare = lastState.EnPassantSquare
	pos.Rule50 = lastState.Rule50

	if move.IsEnPassant() {
		pos.PutPiece(from, lastState.MovedPiece, pos.Turn) // put moved piece back to origin square
		capturedPawnSq := Square(int(to) - PawnDisplacement(pos.Turn))
		pos.RemovePiece(to)                            // remove capturer
		pos.PutPiece(capturedPawnSq, Pawn, pos.Turn^1) // put captured pawn back
	} else if move.IsCastling() {
		pos.PutPiece(from, lastState.MovedPiece, pos.Turn) // put moved piece back to origin square
		rookFromSquare := RookSquares[move]
		rookToSquare := Square(int(to) - KingCastlingDirection(move))
		pos.RemovePiece(rookToSquare)                // remove rook
		pos.PutPiece(rookFromSquare, Rook, pos.Turn) // place rook
		pos.RemovePiece(to)                          // remove king
	} else if move.IsPromotion() {
		pos.PutPiece(from, lastState.MovedPiece, pos.Turn) // put moved piece back to origin square
		pos.RemovePiece(to)                                // remove promoted piece
		if lastState.CapturedPiece != NoPiece {            // capture
			pos.PutPiece(to, lastState.CapturedPiece, pos.Turn^1) // put piece back
		}
	} else if lastState.CapturedPiece != NoPiece { // capture: fuse put(from)+remove(to)+put(to) into one accumulator pass
		pos.UncapturePiece(from, to, lastState.MovedPiece, pos.Turn, lastState.CapturedPiece)
	} else { // quiet move: fuse put(from)+remove(to) into one accumulator pass
		pos.MovePiece(to, from, lastState.MovedPiece, pos.Turn)
	}

	// Overwrite rather than incrementally undo: the PutPiece/RemovePiece
	// calls above already toggled the piece-square hash components, but
	// turn/castling/en-passant were restored directly (not incrementally)
	// above, so their hash contributions were never toggled back. Simplest
	// correct fix is to just restore the exact pre-move hash wholesale.
	pos.Hash = lastState.Hash
}

// Note: As long as castling rights are updated properly we don't need to check for
// position of King/Rook when generating moves. Just need to check castling rights

// all squares between King and Rook
var WhiteKingsideMask Bitboard = 0x0000000000000060
var WhiteQueensideMask Bitboard = 0x000000000000000e
var BlackKingsideMask Bitboard = 0x6000000000000000
var BlackQueensideMask Bitboard = 0x0e00000000000000

var RookSquares = map[Move]Square{
	NewMove(SquareE1, SquareG1) | CastlingFlag: SquareH1,
	NewMove(SquareE1, SquareC1) | CastlingFlag: SquareA1,
	NewMove(SquareE8, SquareG8) | CastlingFlag: SquareH8,
	NewMove(SquareE8, SquareC8) | CastlingFlag: SquareA8,
}

func KingCastlingDirection(move Move) int {
	if move.From() < move.To() {
		return East
	} else {
		return West
	}
}

func (pos *Position) CanWhiteCastleKingside(blockers Bitboard) bool {
	return pos.CastlingRights&WhiteKingside != 0 && blockers&WhiteKingsideMask == 0
}

func (pos *Position) CanWhiteCastleQueenside(blockers Bitboard) bool {
	return pos.CastlingRights&WhiteQueenside != 0 && blockers&WhiteQueensideMask == 0
}

func (pos *Position) CanBlackCastleKingside(blockers Bitboard) bool {
	return pos.CastlingRights&BlackKingside != 0 && blockers&BlackKingsideMask == 0
}

func (pos *Position) CanBlackCastleQueenside(blockers Bitboard) bool {
	return pos.CastlingRights&BlackQueenside != 0 && blockers&BlackQueensideMask == 0
}
