package engine

func GenPieceMoves(pos Position, color uint8) []Move {
	var moveList []Move

	us := pos.Turn
	them := pos.Turn ^ 1
	blockers := Merge(pos.Pieces[them][:])

	for piece := Pawn; piece <= King; piece++ {
		pieceBB := pos.Pieces[us][piece]
		for pieceBB != 0 {
			from := PopLsb(&pieceBB)
			movesBB := GetMoves(piece, from, blockers, color)
			for movesBB != 0 {
				to := PopLsb(&movesBB)
				moveList = append(moveList, NewMove(from, to))
			}
		}
	}

	return moveList
}

func GetMoves(piece uint8, square Square, blockers Bitboard, color uint8) Bitboard {
	switch piece {
	case Pawn:
		// TODO: pawn move generation
		return GetPawnMoves(square, color)
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

func GetPawnMoves(square Square, color uint8) Bitboard {
	bb := BB_Empty

	switch color {
	case White:
		if RankOf(square) == Rank2 {
			rank := Rank3
			file := FileOf(square)
			dest := rank * 8 + file
			bb |= 1 << dest

			rank += 1
			dest = rank * 8 + file
			bb |= 1 << dest
		} else {
			rank := RankOf(square) + 1
			file := FileOf(square)

			dest := rank * 8 + file
			bb |= 1 << dest
		}
	case Black:
		if RankOf(square) == Rank7 {
			rank := Rank6
			file := FileOf(square)
			dest := rank * 8 + file
			bb |= 1 << dest

			rank -= 1
			dest = rank * 8 + file
			bb |= 1 << dest
		} else {
			rank := RankOf(square) - 1
			file := FileOf(square)

			dest := rank * 8 + file
			bb |= 1 << dest
		}
	}

	return bb
}
