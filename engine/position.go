// turn: 0 white 1 black

package engine

type Position struct {
	// Turn: 0=white 1=black
	Turn   uint8
	Pieces [2][6]Bitboard

	// Castling Rights:
	// 0 - can white castle kingside?
	// 1 - can white castle queenside?
	// 2 - can black castle kingside?
	// 3 - can black castle queenside?
	CastlingRights uint8
	Rule50         uint8

	// square that is available for en passant, or NoSquare if no enpassant available
	// basically the square that the last pawn skipped over (if it moved forward 2 squares)
	EnPassantSquare Square
}

const (
	WhiteKingside uint8 = 1 << iota
	WhiteQueenside
	BlackKingside
	BlackQueenside
)

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

// Note: As long as castling rights are updated properly we don't need to check for
// position of King/Rook when generating moves. Just need to check castling rights

var WhiteKingsideMask Bitboard = 0x0000000000000060
var WhiteQueensideMask Bitboard = 0x000000000000000e
var BlackKingsideMask Bitboard = 0x6000000000000000
var BlackQueensideMask Bitboard = 0x0e00000000000000

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

func (pos *Position) KingInCheck(color uint8) bool {
	kingBB := pos.Pieces[color][King]
	kingSquare := PopLsb(&kingBB)

	return pos.AttackersFrom(kingSquare, color) != 0
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

// checking for attackers:
// just check the 8 knight squares, diagonals, and horizontal/vertical

// return bitboard of a specific side's pieces that attack a square
func (pos *Position) AttackersFrom(sq Square, color uint8) Bitboard {
	var attackers Bitboard
	blockers := pos.Blockers()

	orthogonal := GetRookMoves(sq, blockers)
	diagonal := GetBishopMoves(sq, blockers)
	knightMoves := GetKnightMoves(sq)
	kingMoves := GetKingMoves(sq)
	pawnCaptures := PawnCaptures[color^1][sq]

	attackers |= pos.Pieces[color][Rook] & orthogonal
	attackers |= pos.Pieces[color][Bishop] & diagonal
	attackers |= pos.Pieces[color][Queen] & (orthogonal | diagonal)
	attackers |= pos.Pieces[color][Knight] & knightMoves
	attackers |= pos.Pieces[color][King] & kingMoves
	attackers |= pos.Pieces[color][Pawn] & pawnCaptures

	return attackers
}

// return bitboard of all pieces attacking a square
func (pos *Position) Attackers(sq Square) Bitboard {
	return pos.AttackersFrom(sq, White) | pos.AttackersFrom(sq, Black)
}
