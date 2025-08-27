package engine

var PieceSqKeys [12][64]uint64
var CastleKeys [16]uint64 // 2**4 possible castling rights
var EnPassantKeys [8]uint64
var TurnKey uint64

func InitZobrist() {
	for piece := Pawn; piece <= King; piece++ {
		for sq := SquareA1; sq <= SquareH8; sq++ {
			PieceSqKeys[piece][sq] = Rng.Uint64()
		}
	}
	for i := 0; i < 16; i++ {
		CastleKeys[i] = Rng.Uint64()
	}
	for i := 0; i < 8; i++ {
		EnPassantKeys[i] = Rng.Uint64()
	}
	TurnKey = Rng.Uint64()
}

func GetEPKey(sq Square) uint64 {
	file := FileOf(sq)
	rank := RankOf(sq)
	if rank != Rank6 && rank != Rank3 {
		return 0
	} else {
		return EnPassantKeys[file]
	}
}

func GetPieceSqKey(sq Square, piece uint8) uint64 {
	// black pieces are offset by 10, change to offset by 6
	if piece == NoPiece {
		return 0
	}
	if piece >= 10 {
		piece -= 4
	}
	return PieceSqKeys[piece][sq]
}

func Hash(pos *Position) uint64 {
	h := uint64(0)
	for sq := SquareA1; sq <= SquareH8; sq++ {
		piece := pos.Board[sq]
		h ^= GetPieceSqKey(sq, piece)
	}
	h ^= CastleKeys[pos.CastlingRights]
	h ^= GetEPKey(pos.EnPassantSquare)
	if pos.Turn == Black {
		h ^= TurnKey
	}
	return h
}
