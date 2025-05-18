// turn: 0 white 1 black

package engine

type Position struct {
	Turn           uint8
	Pieces         [2][6]Bitboard
	CastlingRights uint8
	Rule50         uint8
}

func NewPosition() *Position {
	p := Position{}
	return &p
}

func StartingPosition() *Position {
	p := Position{
		0,
		Default,
		0b00001111,
		0,
	}
	return &p
}

func (pos *Position) DoMove(move Move) bool {
	return true
}

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

func (pos *Position) IsLegal(move Move) bool {
	var fromColor, fromPiece = pos.GetSquare(move.FromSquare)
	var toColor, toPiece = pos.GetSquare(move.ToSquare)

	if fromColor != pos.Turn {
		return false
	}

	if fromColor == toColor {
		return false
	}

	fromPiece = fromPiece
	toPiece = toPiece

	return false
}

/*
func (pos *Position) GenMoves() (moves []Move) {
	for piece := Knight; piece <= King; piece++ {
		var a = uint8(piece).pop
	}
	return
}
*/
