package engine

func GenPieceMoves(pos Position) []Move {
	var moveList []Move

	us := pos.Turn
	them := pos.Turn ^ 1
	blockers := Merge(pos.Pieces[them][:])

	for piece := Pawn; piece <= King; piece++ {
		pieceBB := pos.Pieces[us][piece]
		for pieceBB != 0 {
			from := PopLsb(&pieceBB)
			movesBB := GetMoves(piece, from, blockers)
			for movesBB != 0 {
				to := PopLsb(&movesBB)
				moveList = append(moveList, NewMove(from, to))
			}
		}
	}

	return moveList
}

func GetMoves(piece uint8, square Square, blockers Bitboard) Bitboard {
	switch piece {
	case Pawn:
		// TODO: pawn move generation
		return Bitboard(0)
	case Knight:
		return GetKnightMoves(square)
	case Bishop:
		return GetBishopMoves(square, blockers)
	case Rook:
		return GetRookMoves(square, blockers)
	case Queen:
		return GetBishopMoves(square, blockers) | GetRookMoves(square, blockers)
	case King:
		return GetKingMoves(square)
	}
	return Bitboard(0)
}

func GetRookMoves(square Square, blockers Bitboard) Bitboard {
	magic := RookMagics[square]
	moves := RookMoves[square]
	return moves[MagicIndex(magic, blockers)]
}

func GetBishopMoves(square Square, blockers Bitboard) Bitboard {
	magic := BishopMagics[square]
	moves := BishopMoves[square]
	return moves[MagicIndex(magic, blockers)]
}

func GetKnightMoves(square Square) Bitboard {
	return KnightMoves[square]
}

func GetKingMoves(square Square) Bitboard {
	return KingMoves[square]
}
