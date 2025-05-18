package board

import (
	"fmt"
	"silverfish/engine"
	"testing"
)

func TestRankFile(t *testing.T) {
	startingSquare := engine.SquareG8
	file := engine.FileOf(startingSquare)
	rank := engine.RankOf(startingSquare)
	wantSquare := engine.SquareD6
	gotSquare := engine.NewSquare(rank-2, file-3)
	if wantSquare != gotSquare {
		t.Errorf(`(r,f) = %d, (r-2, f-1) = %d; want %d`, startingSquare, wantSquare, gotSquare)
	}
}

func TestGetSquare(t *testing.T) {
	p := engine.StartingPosition()
	var wantColor, wantPiece, gotColor, gotPiece uint8
	var square engine.Square

	square = engine.SquareD1
	gotColor, gotPiece = p.GetSquare(square)
	wantColor = engine.White
	wantPiece = engine.Queen
	if gotColor != wantColor || gotPiece != wantPiece {
		t.Errorf(`p.GetSquare(%d) = (%d, %d); want (%d, %d)`, square, gotColor, gotPiece, wantColor, wantPiece)
	}

	square = engine.SquareG8
	gotColor, gotPiece = p.GetSquare(square)
	wantColor = engine.Black
	wantPiece = engine.Knight
	if gotColor != wantColor || gotPiece != wantPiece {
		t.Errorf(`p.GetSquare(%d) = (%d, %d); want (%d, %d)`, square, gotColor, gotPiece, wantColor, wantPiece)
	}

	square = engine.SquareH6
	gotColor, gotPiece = p.GetSquare(square)
	wantColor = engine.NoColor
	wantPiece = engine.NoPiece
	if gotColor != wantColor || gotPiece != wantPiece {
		t.Errorf(`p.GetSquare(%d) = (%d, %d); want (%d, %d)`, square, gotColor, gotPiece, wantColor, wantPiece)
	}
}

func TestSlidingAttack(t *testing.T) {
	var piece uint8
	var square engine.Square
	var gotBB, wantBB engine.Bitboard

	piece = engine.Bishop
	square = engine.SquareD3
	gotBB = engine.SlidingAttack(piece, square)
	wantBB = engine.Bitboard(0b00000000_00000000_01000000_00100010_00010100_00000000_00010100_00000000)
	if gotBB != wantBB {
		t.Errorf(`SlidingAttack(%d, %d)`, piece, square)
		fmt.Println("Got:")
		engine.PrintBB(gotBB)
		fmt.Println("Want:")
		engine.PrintBB(wantBB)
	}

	piece = engine.Rook
	square = engine.SquareH1
	gotBB = engine.SlidingAttack(piece, square)
	wantBB = engine.Bitboard(0b00000000_10000000_10000000_10000000_10000000_10000000_10000000_01111110)
	if gotBB != wantBB {
		t.Errorf(`SlidingAttack(%d, %d)`, piece, square)
		fmt.Println("Got:")
		engine.PrintBB(gotBB)
		fmt.Println("Want:")
		engine.PrintBB(wantBB)
	}
}
