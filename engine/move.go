package engine

/*
0-5: from square
6-11: to square
12-13: promotion piece (knight, bishop, rook, queen)
14-15: castling, en passant, promotion
*/
type Move uint16

func NewMove(from Square, to Square) Move {
	return Move(from<<6 | to)
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

func (m Move) Type() uint8 {
	return uint8(m >> 14)
}
