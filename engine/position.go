// turn: 0 white 1 black

package engine

type Position struct {
	Turn   uint8
	Pieces [2][6]Bitboard
	// Castling Rights:
	// (0-1) boolean - does white have castling rights?
	// 0 - can white castle kingside?
	// 1 - can white castle queenside?
	// (2-3) boolean - does black have castling rights?
	// 0 - can black castle kingside?
	// 1 - can black castle queenside?
	CastlingRights  uint8
	Rule50          uint8
	EnPassantSquare Square
}

func NewPosition() *Position {
	p := Position{}
	return &p
}

func StartingPosition() *Position {
	p := Position{
		Turn:           0,
		Pieces:         Default,
		CastlingRights: 0b00001111,
		Rule50:         0,
	}
	return &p
}

/* func (pos *Position) DoMove(move Move) bool {
	return true
} */

func (pos *Position) GetSquare(square Square) (uint8, uint8) {
	var color, piece uint8
	var mask Bitboard = Bitboard(1 << square)
	for color = 0; color <= 1; color++ {
		for piece = 0; piece <= 5; piece++ {
			if pos.Pieces[color][piece]&mask != 0 {
				return color, piece
			}
		}
	}
	return NoColor, NoPiece
}

func (pos *Position) CanWhiteCastleKingside() bool {
	return pos.CastlingRights&WhiteKingside != 0
}

func (pos *Position) CanWhiteCastleQueenside() bool {
	return pos.CastlingRights&WhiteQueenside != 0
}

func (pos *Position) CanBlackCastleKingside() bool {
	return pos.CastlingRights&BlackKingside != 0
}

func (pos *Position) CanBlackCastleQueenside() bool {
	return pos.CastlingRights&BlackQueenside != 0
}

/* func (pos *Position) IsLegal(move Move) bool {
	var fromColor, fromPiece = pos.GetSquare(move.From())
	var toColor, toPiece = pos.GetSquare(move.To())

	if fromColor != pos.Turn {
		return false
	}

	if fromColor == toColor {
		return false
	}

	fromPiece = fromPiece
	toPiece = toPiece

	return false
} */

func (pos *Position) Blockers() Bitboard {
	return Merge(append(pos.Pieces[0][:], pos.Pieces[1][:]...))
}
