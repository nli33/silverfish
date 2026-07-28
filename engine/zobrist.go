package engine

// Zobrist hashing: a random 64-bit key per (piece, square), castling-rights
// state, en-passant file, and side to move, XORed together. XOR is its own
// inverse, so a move can update the hash incrementally (XOR out what
// changed, XOR in what replaced it) instead of recomputing from scratch --
// see PutPiece/RemovePiece and DoMove/UndoMove. Used for repetition
// detection (Position.IsRepetition).

// Indexed by the real piece constants (Pawn=0 .. King=5) for white,
// offset by +6 for black (Board's own +10 black encoding is remapped to
// this tighter 0-11 range by pieceSqKeyIndex).
var PieceSqKeys [12][64]uint64
var CastleKeys [16]uint64 // 2**4 possible castling-rights combinations
var EnPassantKeys [8]uint64
var TurnKey uint64

func InitZobrist() {
	for piece := 0; piece < 12; piece++ {
		for sq := SquareA1; sq <= SquareH8; sq++ {
			PieceSqKeys[piece][sq] = Rng.Uint64()
		}
	}
	for i := range CastleKeys {
		CastleKeys[i] = Rng.Uint64()
	}
	for i := range EnPassantKeys {
		EnPassantKeys[i] = Rng.Uint64()
	}
	TurnKey = Rng.Uint64()
}

// pieceSqKeyIndex remaps Board's piece encoding (0-5 white, 10-15 black) to
// PieceSqKeys' tighter 0-11 range. NoPiece has no key (empty squares don't
// contribute to the hash).
func pieceSqKeyIndex(piece uint8) (index uint8, ok bool) {
	if piece == NoPiece {
		return 0, false
	}
	if piece >= 10 {
		return piece - 4, true // black Pawn=10..King=15 -> 6..11
	}
	return piece, true // white Pawn=0..King=5 -> 0..5
}

func pieceSqKey(sq Square, piece uint8) uint64 {
	index, ok := pieceSqKeyIndex(piece)
	if !ok {
		return 0
	}
	return PieceSqKeys[index][sq]
}

// enPassantKey returns the hash contribution of an en-passant target
// square, or 0 if none is set (NoSquare, or -- defensively -- a square that
// isn't actually a valid EP rank).
func enPassantKey(sq Square) uint64 {
	if sq == NoSquare {
		return 0
	}
	rank := RankOf(sq)
	if rank != Rank3 && rank != Rank6 {
		return 0
	}
	return EnPassantKeys[FileOf(sq)]
}

// Hash computes a position's Zobrist key from scratch. Only needed once, in
// FromFEN -- everywhere else the key is maintained incrementally.
func Hash(pos *Position) uint64 {
	h := uint64(0)
	for sq := SquareA1; sq <= SquareH8; sq++ {
		h ^= pieceSqKey(sq, pos.Board[sq])
	}
	h ^= CastleKeys[pos.CastlingRights]
	h ^= enPassantKey(pos.EnPassantSquare)
	if pos.Turn == Black {
		h ^= TurnKey
	}
	return h
}
