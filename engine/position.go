// turn: 0 white 1 black

package engine

import (
	"math/bits"
)

type Position struct {
	// Turn: 0=white 1=black
	Turn   uint8
	Pieces [2][6]Bitboard

	// Castling Rights:
	// 0 - can white castle kingside?
	// 1 - can white castle queenside?
	// 2 - can black castle kingside?
	// 3 - can black castle queenside?
	CastlingRights uint8

	// half-move clock
	Rule50 uint8

	// number of turns
	Ply uint16

	// square that is available for en passant, or NoSquare if no enpassant available
	// basically the square that the last pawn skipped over (if it moved forward 2 squares)
	EnPassantSquare Square

	// past states
	History []State
}

type State struct {
	CastlingRights  uint8
	EnPassantSquare Square
	Rule50          uint8
	CapturedPiece   uint8
	MovedPiece      uint8
}

const (
	WhiteKingside uint8 = 1 << iota
	WhiteQueenside
	BlackKingside
	BlackQueenside
)

func NewPosition() Position {
	p := Position{}
	return p
}

func StartingPosition() Position {
	p := Position{
		Turn:           White,
		Pieces:         Default,
		CastlingRights: 0b00001111,
		Rule50:         0,
		Ply:            0,
	}
	return p
}

/* func (pos *Position) DoMove(move Move) bool {
	return true
} */

func (pos *Position) GetSquare(square Square) (uint8, uint8) {
	var color, piece uint8
	var mask Bitboard = Bitboard(1 << square)
	for color = 0; color <= 1; color++ {
		for piece = 0; piece <= 5; piece++ {
			if pos.Pieces[color][piece]&mask != 0 {
				return color, piece
			}
		}
	}
	return NoColor, NoPiece
}

// returns position after move, regardless of legality or pseudo-legality
// copies the current board (doesn't modify original)
// potentially works as a "make move" function?
func (pos Position) After(move Move) Position {
	ourColor, piece := pos.GetSquare(move.From())
	oppColor := ourColor ^ 1
	pos.Pieces[ourColor][piece] &^= 1 << move.From()
	if move.Type() == CastlingFlag {
		rookFromSquare := RookSquares[move]
		pos.Pieces[ourColor][Rook] &^= 1 << rookFromSquare
		rookToSquare := Square(int(move.To()) - KingCastlingDirection(move))
		pos.Pieces[ourColor][Rook] |= 1 << rookToSquare
		pos.Pieces[ourColor][King] |= 1 << move.To()
	} else if move.Type() == EnPassantFlag {
		capturedPawnSq := Square(int(move.To()) - PawnDisplacement(ourColor))
		pos.Pieces[oppColor][Pawn] &^= 1 << capturedPawnSq
		pos.Pieces[ourColor][piece] |= 1 << move.To()
	} else if move.Type() == PromotionFlag {
		pos.Pieces[ourColor][move.Promotion()] |= 1 << move.To()
	} else {
		for c := White; c <= Black; c++ {
			for p := Pawn; p <= King; p++ {
				pos.Pieces[c][p] &^= 1 << move.To()
			}
		}
		pos.Pieces[ourColor][piece] |= 1 << move.To()
	}
	return pos
}

// Note: As long as castling rights are updated properly we don't need to check for
// position of King/Rook when generating moves. Just need to check castling rights

// all squares between King and Rook
var WhiteKingsideMask Bitboard = 0x0000000000000060
var WhiteQueensideMask Bitboard = 0x000000000000000e
var BlackKingsideMask Bitboard = 0x6000000000000000
var BlackQueensideMask Bitboard = 0x0e00000000000000

var RookSquares = map[Move]Square{
	NewMove(SquareE1, SquareG1) | CastlingFlag: SquareH1,
	NewMove(SquareE1, SquareC1) | CastlingFlag: SquareA1,
	NewMove(SquareE8, SquareG8) | CastlingFlag: SquareH8,
	NewMove(SquareE8, SquareC8) | CastlingFlag: SquareA8,
}

func KingCastlingDirection(move Move) int {
	if move.From() < move.To() {
		return East
	} else {
		return West
	}
}

func (pos *Position) CanWhiteCastleKingside(blockers Bitboard) bool {
	return pos.CastlingRights&WhiteKingside != 0 && blockers&WhiteKingsideMask == 0
}

func (pos *Position) CanWhiteCastleQueenside(blockers Bitboard) bool {
	return pos.CastlingRights&WhiteQueenside != 0 && blockers&WhiteQueensideMask == 0
}

func (pos *Position) CanBlackCastleKingside(blockers Bitboard) bool {
	return pos.CastlingRights&BlackKingside != 0 && blockers&BlackKingsideMask == 0
}

func (pos *Position) CanBlackCastleQueenside(blockers Bitboard) bool {
	return pos.CastlingRights&BlackQueenside != 0 && blockers&BlackQueensideMask == 0
}

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
	ourColor, _ := pos.GetSquare(move.From())
	oppColor := ourColor ^ 1

	// check if it is our turn
	if pos.Turn != ourColor {
		return false
	}

	destColor, _ := pos.GetSquare(move.To())

	// check if move tries to capture same color piece
	if destColor == ourColor {
		return false
	}

	if move.Type() == CastlingFlag {
		// movegen will only generate castling moves with castling flags allowing, no need to check again
		kingStep := KingCastlingDirection(move)
		for sq := move.From(); sq <= move.To(); sq += Square(kingStep) {
			if pos.AttackersFrom(sq, oppColor) != 0 {
				return false
			}
		}
	}

	// check if the move leaves us in check (inefficient method)
	after := pos.After(move)
	if after.Checkers(ourColor) != 0 {
		return false
	}

	// not sure if this is necessary
	/* if !after.IsLegal() {
		return false
	} */

	return true
}

func (pos *Position) LegalMoves() []Move {
	var moveList []Move
	for _, move := range GenMoves(*pos) {
		if pos.MoveIsLegal(move) {
			moveList = append(moveList, move)
		}
	}
	return moveList
}

func (pos *Position) Blockers() Bitboard {
	return Merge(append(pos.Pieces[0][:], pos.Pieces[1][:]...))
}

// checking for attackers:
// just check the 8 knight squares, diagonals, and horizontal/vertical

// return bitboard of a specific side's pieces that attack a square
func (pos *Position) AttackersFrom(sq Square, color uint8) Bitboard {
	var attackers Bitboard
	blockers := pos.Blockers()

	orthogonal := GetRookMoves(sq, blockers)
	diagonal := GetBishopMoves(sq, blockers)
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
