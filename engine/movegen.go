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
	moveList = append(moveList, GetCastlingMoves(pos, blockers)...)

	return moveList
}

func GetPawnMoves(pos Position, blockers Bitboard) []Move {
	var moveList []Move

	us := pos.Turn
	ourPawnsBB := pos.Pieces[us][0]

	nextRank := PawnDisplacement(us)
	pawnStartingRank := PawnStartingRank(us)

	for ourPawnsBB != 0 {
		fromSq := PopLsb(&ourPawnsBB)
		rank := RankOf(fromSq)

		// captures + en passant
		capturesBB := PawnCaptures[us][fromSq]
		for capturesBB != 0 {
			toSq := PopLsb(&capturesBB)
			if blockers&(1<<toSq) != 0 {
				if rank == PawnPromotionRank(us) { // promotion
					AddPromotions(&moveList, fromSq, toSq)
				} else { // normal capture
					moveList = append(moveList, NewMove(fromSq, toSq))
				}
			} else if pos.EnPassantSquare == toSq && blockers&(1<<pos.EnPassantSquare) == 0 {
				// en passant
				moveList = append(moveList, NewMove(fromSq, pos.EnPassantSquare)|EnPassantFlag)
			}
		}

		// moving forward
		nextSq := int(fromSq) + nextRank
		if blockers&(1<<nextSq) != 0 {
			continue
		}
		if rank == PawnPromotionRank(us) { // promotion
			AddPromotions(&moveList, fromSq, Square(nextSq))
		} else { // normal move 1 square forward
			moveList = append(moveList, NewMove(fromSq, Square(nextSq)))
		}

		// moving forward 2 squares
		if rank == pawnStartingRank {
			nextSq += nextRank
			if blockers&(1<<nextSq) != 0 {
				continue
			}
			moveList = append(moveList, NewMove(fromSq, Square(nextSq)))
		}
	}

	return moveList
}

// add a pawn move to a move list
// if a promotion is possible, add promotion moves instead
func AddPromotions(moveList *[]Move, fromSq Square, toSq Square) {
	//if RankOf(toSq) == PawnPromotionRank(color) {
	for piece := Knight; piece <= Queen; piece++ {
		*moveList = append(*moveList, NewPromotionMove(fromSq, toSq, piece))
	}
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

func GetCastlingMoves(pos Position, blockers Bitboard) []Move {
	var moveList []Move

	if pos.Turn == White && pos.CanWhiteCastleKingside(blockers) {
		moveList = append(moveList, NewMoveCastle(WhiteKingside))
	}

	if pos.Turn == Black && pos.CanBlackCastleKingside(blockers) {
		moveList = append(moveList, NewMoveCastle(BlackKingside))
	}

	if pos.Turn == White && pos.CanWhiteCastleQueenside(blockers) {
		moveList = append(moveList, NewMoveCastle(WhiteQueenside))
	}

	if pos.Turn == Black && pos.CanBlackCastleQueenside(blockers) {
		moveList = append(moveList, NewMoveCastle(BlackQueenside))
	}

	return moveList
}

func GetKingMoves(square Square) Bitboard {
	return KingMoves[square]
}
