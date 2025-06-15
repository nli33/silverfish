// turn: 0 white 1 black

package engine

import (
	"fmt"
	"math/bits"
	"strconv"
	"strings"
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
	//
	// note: field should only be set for the halfmove right after a pawn moves forward 2 squares
	// example: after a2a4, EPsq = a3. After black moves (not EP), it is NoSquare
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

func (pos *Position) Equals(otherPos Position) bool {
	return pos.Turn == otherPos.Turn &&
		pos.Pieces == otherPos.Pieces &&
		pos.CastlingRights == otherPos.CastlingRights &&
		pos.Rule50 == otherPos.Rule50 &&
		pos.Ply == otherPos.Ply &&
		pos.EnPassantSquare == otherPos.EnPassantSquare
}

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

func (pos *Position) FullMoves() uint16 {
	return (pos.Ply)/2 + 1
}

func (pos *Position) ToFEN() string {
	fen := ""
	spaceCounter := 0
	for rank := Rank8; Rank8 >= rank && rank >= Rank1; rank-- {
		for file := FileA; file <= FileH; file++ {
			sq := NewSquare(rank, file)
			color, piece := pos.GetSquare(sq)

			if piece == NoPiece {
				spaceCounter++
				continue
			}
			if spaceCounter > 0 {
				fen += fmt.Sprint(spaceCounter)
				spaceCounter = 0
			}

			char := PieceToChar[piece]
			if color == White { // capitalize
				char -= 32
			}
			fen += string(char)
		}
		if spaceCounter > 0 {
			fen += fmt.Sprint(spaceCounter)
			spaceCounter = 0
		}
		if rank != Rank1 {
			fen += "/"
		}
	}

	if pos.Turn == White {
		fen += " w "
	} else {
		fen += " b "
	}

	// only checking castling rights, not blockers
	if pos.CastlingRights&WhiteKingside > 0 {
		fen += "K"
	}
	if pos.CastlingRights&WhiteQueenside > 0 {
		fen += "Q"
	}
	if pos.CastlingRights&BlackKingside > 0 {
		fen += "k"
	}
	if pos.CastlingRights&BlackQueenside > 0 {
		fen += "q"
	}
	if pos.CastlingRights == 0 {
		fen += "-"
	}

	fen += " "
	if pos.EnPassantSquare != NoSquare &&
		(RankOf(pos.EnPassantSquare) == Rank3 ||
			RankOf(pos.EnPassantSquare) == Rank6) {
		fen += pos.EnPassantSquare.ToString()
	} else {
		fen += "-"
	}

	fen += " " + fmt.Sprint(pos.Rule50)
	fen += " " + fmt.Sprint(pos.FullMoves())

	return fen
}

func FromFEN(fen string) Position {
	var pos Position

	parts := strings.Split(fen, " ")
	if len(parts) < 6 {
		panic("invalid FEN: not enough parts")
	}

	boardPart := parts[0]
	turnPart := parts[1]
	castlingPart := parts[2]
	enPassantPart := parts[3]
	rule50Part := parts[4]
	fullmovePart := parts[5]

	rank := Rank8
	file := FileA

	for _, char := range boardPart {
		if char == '/' {
			rank--
			file = FileA
		} else if char >= '1' && char <= '8' {
			file += uint8(char - '0')
		} else {
			var color uint8
			if char >= 'A' && char <= 'Z' {
				color = White
				char += 32 // make lowercase
			} else {
				color = Black
			}

			piece, exists := CharToPiece[byte(char)]

			if !exists {
				panic("invalid piece character: " + string(char))
			}

			sq := NewSquare(rank, file)
			pos.Pieces[color][piece] |= 1 << sq
			file++
		}
	}

	if turnPart == "w" {
		pos.Turn = White
	} else if turnPart == "b" {
		pos.Turn = Black
	} else {
		panic("invalid turn field")
	}

	pos.CastlingRights = 0
	if castlingPart != "-" {
		for _, c := range castlingPart {
			switch c {
			case 'K':
				pos.CastlingRights |= WhiteKingside
			case 'Q':
				pos.CastlingRights |= WhiteQueenside
			case 'k':
				pos.CastlingRights |= BlackKingside
			case 'q':
				pos.CastlingRights |= BlackQueenside
			default:
				panic("invalid castling character")
			}
		}
	}

	if enPassantPart == "-" {
		pos.EnPassantSquare = NoSquare
	} else {
		pos.EnPassantSquare = NewSquareFromStr(enPassantPart)
	}

	rule50, err := strconv.Atoi(rule50Part)
	if err != nil {
		panic("invalid rule50 field")
	}
	pos.Rule50 = uint8(rule50)

	fullmove, err := strconv.Atoi(fullmovePart)
	if err != nil {
		panic("invalid fullmove field")
	}
	pos.Ply = uint16((fullmove-1)*2 + int(pos.Turn))

	return pos
}

// returns position after move, regardless of legality or pseudo-legality
// copies the current board (doesn't modify original)
// keeping for now, since it might be a bit faster than DoMove ? (no state saving needed, no undo needed)
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

func (pos *Position) DoMove(move Move) {
	ourColor, movingPiece := pos.GetSquare(move.From())
	if ourColor != pos.Turn {
		panic("not our turn")
	}
	// don't use opp color from here, since en passants will yield NoColor
	_, capturedPiece := pos.GetSquare(move.To())

	// save the current state before moving
	// CapturedPiece will not include pawns captured en passant
	state := State{
		MovedPiece:      movingPiece,
		CapturedPiece:   capturedPiece, // may be NoPiece
		CastlingRights:  pos.CastlingRights,
		Rule50:          pos.Rule50,
		EnPassantSquare: pos.EnPassantSquare,
	}

	pos.Rule50++

	// update en passant square
	if movingPiece == Pawn {
		dist := int(move.To()) - int(move.From())
		pawnDisplacement := PawnDisplacement(ourColor)
		if dist == 2*pawnDisplacement {
			pos.EnPassantSquare = Square(int(move.To()) - pawnDisplacement)
		} else {
			pos.EnPassantSquare = NoSquare
		}
		pos.Rule50 = 0
	} else {
		pos.EnPassantSquare = NoSquare
	}

	// update castling rights
	if movingPiece == King {
		if ourColor == White {
			pos.CastlingRights &^= 0b0011
		} else if ourColor == Black {
			pos.CastlingRights &^= 0b1100
		}
	} else if movingPiece == Rook {
		switch move.From() {
		case SquareA1:
			pos.CastlingRights &^= 0b0010
		case SquareA8:
			pos.CastlingRights &^= 0b0001
		case SquareH1:
			pos.CastlingRights &^= 0b1000
		case SquareH8:
			pos.CastlingRights &^= 0b0100
		}
	}

	pos.Pieces[ourColor][movingPiece] &^= 1 << move.From()

	if move.IsCastling() {
		rookFromSquare := RookSquares[move]
		pos.Pieces[ourColor][Rook] &^= 1 << rookFromSquare
		rookToSquare := Square(int(move.To()) - KingCastlingDirection(move))
		pos.Pieces[ourColor][Rook] |= 1 << rookToSquare
		pos.Pieces[ourColor][King] |= 1 << move.To()
	} else if move.IsEnPassant() {
		capturedPawnSq := Square(int(move.To()) - PawnDisplacement(ourColor))
		pos.Pieces[ourColor^1][Pawn] &^= 1 << capturedPawnSq
		pos.Pieces[ourColor][movingPiece] |= 1 << move.To()
	} else if move.IsPromotion() {
		pos.Pieces[ourColor][move.Promotion()] |= 1 << move.To()
	} else {
		if capturedPiece != NoPiece { // is a capture
			pos.Pieces[ourColor^1][capturedPiece] &^= 1 << move.To()
			pos.Rule50 = 0
		}
		pos.Pieces[ourColor][movingPiece] |= 1 << move.To()
	}

	pos.Ply++
	pos.Turn ^= 1
	pos.History = append(pos.History, state)
}

func (pos *Position) UndoMove(move Move) {
	n := len(pos.History)
	if n == 0 {
		panic("no reversible moves")
	}
	lastState := pos.History[n-1]
	pos.History = pos.History[:n-1]

	pos.Turn ^= 1
	pos.Ply--
	// pos.Turn is now the side that did the move

	pos.CastlingRights = lastState.CastlingRights
	pos.EnPassantSquare = lastState.EnPassantSquare
	pos.Rule50 = lastState.Rule50

	// put moved piece back to origin square
	pos.Pieces[pos.Turn][lastState.MovedPiece] |= 1 << move.From()

	if move.IsEnPassant() {
		capturedPawnSq := Square(int(move.To()) - PawnDisplacement(pos.Turn))
		pos.Pieces[pos.Turn][lastState.MovedPiece] &^= 1 << move.To() // remove capturer
		pos.Pieces[pos.Turn^1][Pawn] |= 1 << capturedPawnSq           // put captured pawn back
	} else if move.IsCastling() {
		rookFromSquare := RookSquares[move]
		rookToSquare := Square(int(move.To()) - KingCastlingDirection(move))
		pos.Pieces[pos.Turn][Rook] &^= 1 << rookToSquare  // remove rook
		pos.Pieces[pos.Turn][Rook] |= 1 << rookFromSquare // place rook
		pos.Pieces[pos.Turn][King] &^= 1 << move.To()     // remove king
	} else if move.IsPromotion() {
		pos.Pieces[pos.Turn][move.Promotion()] &^= 1 << move.To() // remove promoted piece
	} else { // quiet move
		pos.Pieces[pos.Turn][lastState.MovedPiece] &^= 1 << move.To() // remove piece
	}

	if lastState.CapturedPiece != NoPiece { // capture
		pos.Pieces[pos.Turn^1][lastState.CapturedPiece] |= 1 << move.To() // put piece back
	}
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
