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

func (pos *Position) PutPiece(sq Square, piece uint8, color uint8) {
	sqBB := Bitboard(1 << sq)
	pos.Pieces[color][piece] |= sqBB

	// nnue operations come before this adjustment by 10 (0-5 expected; FeatureIndex performs adjustment by 6)
	fWhite := FeatureIndex(White, color, piece, sq)
	fBlack := FeatureIndex(Black, color, piece, sq)
	pos.Acc.Add(pos.Net, fWhite, White)
	pos.Acc.Add(pos.Net, fBlack, Black)

	if color == Black {
		piece += 10
	}
	pos.Board[sq] = piece
	pos.Blockers |= sqBB
	pos.Sides[color] |= sqBB

	pos.Hash ^= pieceSqKey(sq, piece)
}

func (pos *Position) PutPiecesBB(pieces [2][6]Bitboard) {
	// this check is for testing purposes only
	// .. well this whole function is for testing purposes only
	if pos.Net == nil {
		pos.Net = defaultNet
		pos.Acc = NewAccumulator(pos.Net)
	}

	for sq := SquareA1; sq <= SquareH8; sq++ {
		// RemovePiece panics on an already-empty square (NoColor indexes
		// pos.Sides out of bounds), so only call it where there's something
		// to remove -- relevant when reusing the same Position across calls.
		if pos.Board[sq] != NoPiece {
			pos.RemovePiece(sq)
		}
		for piece := Pawn; piece <= King; piece++ {
			for color := White; color <= Black; color++ {
				if pieces[color][piece]&(1<<sq) != 0 {
					pos.PutPiece(sq, piece, color)
				}
			}
		}
	}
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

	fWhiteFrom := FeatureIndex(White, color, piece, from)
	fBlackFrom := FeatureIndex(Black, color, piece, from)
	fWhiteTo := FeatureIndex(White, color, piece, to)
	fBlackTo := FeatureIndex(Black, color, piece, to)
	pos.Acc.AddSub(pos.Net, fWhiteTo, fWhiteFrom, White)
	pos.Acc.AddSub(pos.Net, fBlackTo, fBlackFrom, Black)
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

	fWhiteFrom := FeatureIndex(White, color, piece, from)
	fBlackFrom := FeatureIndex(Black, color, piece, from)
	fWhiteTo := FeatureIndex(White, color, piece, to)
	fBlackTo := FeatureIndex(Black, color, piece, to)
	fWhiteCaptured := FeatureIndex(White, theirColor, capturedPiece, to)
	fBlackCaptured := FeatureIndex(Black, theirColor, capturedPiece, to)

	pos.Acc.AddSubSub(pos.Net, fWhiteTo, fWhiteFrom, fWhiteCaptured, White)
	pos.Acc.AddSubSub(pos.Net, fBlackTo, fBlackFrom, fBlackCaptured, Black)
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

	fWhiteFrom := FeatureIndex(White, color, piece, from)
	fBlackFrom := FeatureIndex(Black, color, piece, from)
	fWhiteTo := FeatureIndex(White, color, piece, to)
	fBlackTo := FeatureIndex(Black, color, piece, to)
	fWhiteCaptured := FeatureIndex(White, theirColor, capturedPiece, to)
	fBlackCaptured := FeatureIndex(Black, theirColor, capturedPiece, to)

	pos.Acc.AddAddSub(pos.Net, fWhiteFrom, fWhiteCaptured, fWhiteTo, White)
	pos.Acc.AddAddSub(pos.Net, fBlackFrom, fBlackCaptured, fBlackTo, Black)
}

func (pos *Position) RemovePiece(sq Square) {
	boardPiece := pos.Board[sq] // pre-adjustment (0-5 white, 10-15 black), for the hash key
	piece := boardPiece
	color := ColorOf(piece)
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

	// nnue operations come after this adjustment by 10 (0-5 expected; FeatureIndex performs adjustment by 6)
	fWhite := FeatureIndex(White, color, piece, sq)
	fBlack := FeatureIndex(Black, color, piece, sq)
	pos.Acc.Remove(pos.Net, fWhite, White)
	pos.Acc.Remove(pos.Net, fBlack, Black)
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
