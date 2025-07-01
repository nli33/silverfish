package engine

type MoveList struct {
	Moves [256]Move
	Count uint8
}

func (moveList *MoveList) Add(move Move) {
	moveList.Moves[moveList.Count] = move
	moveList.Count++
}

func GenMoves(pos *Position) MoveList {
	var moves MoveList

	us := pos.Turn // our color

	for piece := Pawn; piece <= King; piece++ {
		pieceBB := pos.Pieces[us][piece]
		for pieceBB != 0 {
			from := PopLsb(&pieceBB)
			movesBB := GetPieceMoves(piece, from, pos.Blockers, us)
			for movesBB != 0 {
				to := PopLsb(&movesBB)
				moves.Add(NewMove(from, to))
			}
		}
	}

	GenPawnMoves(pos, &moves)
	GenCastlingMoves(pos, &moves)

	return moves
}

func GenPawnMoves(pos *Position, moves *MoveList) {
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
			if pos.Blockers&(1<<toSq) != 0 {
				if rank == PawnPromotionRank(us) { // promotion
					AddPromotions(moves, fromSq, toSq)
				} else { // normal capture
					moves.Add(NewMove(fromSq, toSq))
				}
			} else if pos.EnPassantSquare == toSq && pos.Blockers&(1<<pos.EnPassantSquare) == 0 {
				// en passant
				moves.Add(NewMove(fromSq, pos.EnPassantSquare) | EnPassantFlag)
			}
		}

		// moving forward
		nextSq := int(fromSq) + nextRank
		if pos.Blockers&(1<<nextSq) != 0 {
			continue
		}
		if rank == PawnPromotionRank(us) { // promotion
			AddPromotions(moves, fromSq, Square(nextSq))
		} else { // normal move 1 square forward
			moves.Add(NewMove(fromSq, Square(nextSq)))
		}

		// moving forward 2 squares
		if rank == pawnStartingRank {
			nextSq += nextRank
			if pos.Blockers&(1<<nextSq) != 0 {
				continue
			}
			moves.Add(NewMove(fromSq, Square(nextSq)))
		}
	}
}

// add all piece promotion moves
func AddPromotions(moves *MoveList, fromSq Square, toSq Square) {
	for piece := Knight; piece <= Queen; piece++ {
		moves.Add(NewPromotionMove(fromSq, toSq, piece))
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

func GenCastlingMoves(pos *Position, moves *MoveList) {
	if pos.Turn == White && pos.CanWhiteCastleKingside(pos.Blockers) {
		moves.Add(NewMoveCastle(WhiteKingside))
	}

	if pos.Turn == Black && pos.CanBlackCastleKingside(pos.Blockers) {
		moves.Add(NewMoveCastle(BlackKingside))
	}

	if pos.Turn == White && pos.CanWhiteCastleQueenside(pos.Blockers) {
		moves.Add(NewMoveCastle(WhiteQueenside))
	}

	if pos.Turn == Black && pos.CanBlackCastleQueenside(pos.Blockers) {
		moves.Add(NewMoveCastle(BlackQueenside))
	}
}

func GetKingMoves(square Square) Bitboard {
	return KingMoves[square]
}
