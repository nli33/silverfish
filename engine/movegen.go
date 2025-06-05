package engine

func GenMoves(pos Position) []Move {
	var moveList []Move

	us := pos.Turn // our color
	blockers := pos.Blockers()

	for piece := Pawn; piece <= King; piece++ {
		pieceBB := pos.Pieces[us][piece]
		for pieceBB != 0 {
			from := PopLsb(&pieceBB)
			movesBB := GetPieceMoves(piece, from, blockers, us)
			for movesBB != 0 {
				to := PopLsb(&movesBB)
				moveList = append(moveList, NewMove(from, to))
			}
		}
	}

	moveList = append(moveList, GetPawnMoves(pos, blockers)...)

	return moveList
}

func GetPawnMoves(pos Position, blockers Bitboard) []Move {
	var moveList []Move

	us := pos.Turn
	ourPawnsBB := pos.Pieces[us][0]

	nextRank := PawnDisplacement(us)
	pawnStartingRank := PawnStartingRank(us)

	var captureSq Square
	for ourPawnsBB != 0 {
		pawnSq := PopLsb(&ourPawnsBB)
		rank := RankOf(pawnSq)
		file := FileOf(pawnSq)

		// captures
		if file != FileH {
			captureSq = Square(int(pawnSq) + nextRank + 1)
			if blockers&(1<<captureSq) != 0 {
				moveList = append(moveList, NewMove(pawnSq, captureSq))
			}
		}
		if file != FileA {
			captureSq = Square(int(pawnSq) + nextRank - 1)
			if blockers&(1<<captureSq) != 0 {
				moveList = append(moveList, NewMove(pawnSq, captureSq))
			}
		}

		// moving forward
		nextSq := int(pawnSq) + nextRank
		if blockers&(1<<nextSq) != 0 {
			continue
		}
		moveList = append(moveList, NewMove(pawnSq, Square(nextSq)))

		if rank == pawnStartingRank {
			nextSq += nextRank
			if blockers&(1<<nextSq) != 0 {
				continue
			}
			moveList = append(moveList, NewMove(pawnSq, Square(nextSq)))
		}
	}

	return moveList
}

func GetPieceMoves(piece uint8, square Square, blockers Bitboard, color uint8) Bitboard {
	switch piece {
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
