package engine

import (
	"math/bits"
)

func (pos *Position) IsLegal() bool {
	// only one king each
	if bits.OnesCount64(uint64(pos.Pieces[0][King])) != 1 || bits.OnesCount64(uint64(pos.Pieces[1][King])) != 1 {
		return false
	}

	// no pawns on Ranks 1, 8
	if bits.OnesCount64(uint64(pos.Pieces[0][Pawn]&BB_Rank1)) != 0 ||
		bits.OnesCount64(uint64(pos.Pieces[1][Pawn]&BB_Rank1)) != 0 ||
		bits.OnesCount64(uint64(pos.Pieces[0][Pawn]&BB_Rank8)) != 0 ||
		bits.OnesCount64(uint64(pos.Pieces[1][Pawn]&BB_Rank8)) != 0 {
		return false
	}

	// check that only 8 pawns exist
	if bits.OnesCount64(uint64(pos.Pieces[0][Pawn])) > 8 ||
		bits.OnesCount64(uint64(pos.Pieces[1][Pawn])) > 8 {
		return false
	}

	// verify that the side not to move is not in check
	if pos.Checkers(pos.Turn^1) != 0 {
		return false
	}

	// TODO: check castling flags for validity? (not sure if necessary)

	return true
}

func (pos *Position) MoveIsLegal(move Move) bool {
	from := move.From()
	to := move.To()

	ourColor, _ := pos.GetSquare(from)
	oppColor := ourColor ^ 1

	// check if it is our turn
	if pos.Turn != ourColor {
		return false
	}

	destColor, _ := pos.GetSquare(to)

	// check if move tries to capture same color piece
	if destColor == ourColor {
		return false
	}

	// movegen will only generate castling moves with castling flags allowing, no need to check again
	if move.IsCastling() {
		kingStep := Square(KingCastlingDirection(move))
		// check whether our rook is there
		//if pos.Board[RookSquares[move]] !=
		if pos.Pieces[ourColor][Rook]&(1<<RookSquares[move]) == 0 {
			return false
		}
		for sq := from; sq != to+kingStep; sq += kingStep {
			if pos.AttackersFrom(sq, oppColor) != 0 {
				return false
			}
		}
	}

	// check if the move leaves us in check (inefficient method)
	pos.DoMove(move)
	if pos.Checkers(ourColor) != 0 {
		pos.UndoMove(move)
		return false
	}
	pos.UndoMove(move)

	return true
}

// deprecated, only used for testing purposes now
func (pos *Position) LegalMoves() []Move {
	var moves []Move
	moveList := GenMoves(pos, BB_Full)
	for i := 0; i < int(moveList.Count); i++ {
		move := moveList.Moves[i]
		if pos.MoveIsLegal(move) {
			moves = append(moves, move)
		}
	}
	return moves
}

// checking for attackers:
// just check the 8 knight squares, diagonals, and horizontal/vertical

// return bitboard of a specific side's pieces that attack a square
func (pos *Position) AttackersFrom(sq Square, color uint8) Bitboard {
	var attackers Bitboard

	orthogonal := GetRookMoves(sq, pos.Blockers)
	diagonal := GetBishopMoves(sq, pos.Blockers)
	knightMoves := GetKnightMoves(sq)
	kingMoves := GetKingMoves(sq)
	pawnCaptures := PawnCaptures[color^1][sq]

	attackers |= pos.Pieces[color][Rook] & orthogonal
	attackers |= pos.Pieces[color][Bishop] & diagonal
	attackers |= pos.Pieces[color][Queen] & (orthogonal | diagonal)
	attackers |= pos.Pieces[color][Knight] & knightMoves
	attackers |= pos.Pieces[color][King] & kingMoves
	attackers |= pos.Pieces[color][Pawn] & pawnCaptures

	return attackers
}

// return bitboard of all pieces attacking a square
func (pos *Position) Attackers(sq Square) Bitboard {
	return pos.AttackersFrom(sq, White) | pos.AttackersFrom(sq, Black)
}

// return bitboard of pieces checking this side's king
func (pos *Position) Checkers(color uint8) Bitboard {
	kingBB := pos.Pieces[color][King]
	kingSq := Lsb(kingBB)
	return pos.AttackersFrom(kingSq, color^1)
}

// IsRepetition reports whether the current position has occurred before,
// earlier in this same game (History covers the whole game, not just the
// current search: main.go replays every move from the UCI "position"
// command before searching). Treats the first repetition as a draw rather
// than waiting for a literal threefold -- standard practice, since if a
// position can be repeated once under continued play, it can be forced
// again, and there's no benefit to spending nodes proving that a second
// time.
//
// Only positions with the same side to move can repeat, so this steps back
// two plies at a time. The scan is bounded by Rule50: a capture or pawn
// move is irreversible, so nothing before the most recent one can be equal
// to the current position.
func (pos *Position) IsRepetition() bool {
	n := len(pos.History)
	limit := int(pos.Rule50)
	if limit > n {
		limit = n
	}
	for i := 2; i <= limit; i += 2 {
		if pos.History[n-i].Hash == pos.Hash {
			return true
		}
	}
	return false
}
