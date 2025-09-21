package engine

/*
0-5: from square
6-11: to square
12-13: promotion piece (knight, bishop, rook, queen)
14-15: castling (01), en passant (10), promotion (11)
16+: move score
*/
type Move uint32

const (
	NoneFlag = iota << 14
	CastlingFlag
	EnPassantFlag
	PromotionFlag
)

func NewMove(from Square, to Square) Move {
	return Move(uint16(from) | uint16(to)<<6)
}

func NewPromotionMove(from Square, to Square, promotion uint8) Move {
	return Move(uint16(from) | uint16(to)<<6 | uint16(promotion-1)<<12 | PromotionFlag)
}

func NewMoveCastle(side uint8) Move {
	switch side {
	case WhiteKingside:
		return NewMove(SquareE1, SquareG1) | CastlingFlag
	case WhiteQueenside:
		return NewMove(SquareE1, SquareC1) | CastlingFlag
	case BlackKingside:
		return NewMove(SquareE8, SquareG8) | CastlingFlag
	case BlackQueenside:
		return NewMove(SquareE8, SquareC8) | CastlingFlag
	}
	return Move(0)
}

// ex: c2c1q
// ex: b2b4
func NewMoveFromStr(moveStr string) Move {
	from := NewSquareFromStr(moveStr[0:2])
	to := NewSquareFromStr(moveStr[2:4])

	if len(moveStr) == 5 {
		promotion := CharToPiece[moveStr[4]]
		return NewPromotionMove(from, to, promotion)
	} else {
		return NewMove(from, to)
	}
}

func (m Move) From() Square {
	return Square(m & 0b111111)
}

func (m Move) To() Square {
	return Square(m >> 6 & 0b111111)
}

func (m Move) Promotion() uint8 {
	return uint8(m>>12&0b11 + 1)
}

func (m Move) IsPromotion() bool {
	return m&PromotionFlag == PromotionFlag
}

func (m Move) IsCastling() bool {
	return m&PromotionFlag == CastlingFlag
}

func (m Move) IsEnPassant() bool {
	return m&PromotionFlag == EnPassantFlag
}

func (m Move) Type() int {
	return int(m & PromotionFlag)
}

func (m Move) Score() int {
	return int(m&0xffff0000) >> 16
}

func (m *Move) GiveScore(score int) {
	*m &= 0xffff
	*m |= Move(score << 16)
}

func (m Move) ToString() string {
	if m.IsPromotion() {
		return m.From().ToString() + m.To().ToString() + string(PieceToChar[m.Promotion()])
	} else {
		return m.From().ToString() + m.To().ToString()
	}
}
