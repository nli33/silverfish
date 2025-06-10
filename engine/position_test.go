package engine_test

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

	square := engine.SquareD1
	gotColor, gotPiece := p.GetSquare(square)
	wantColor := engine.White
	wantPiece := engine.Queen
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

func TestAttackers(t *testing.T) {
	pieces := [2][6]engine.Bitboard{
		{
			engine.Bitboard(0x0000000000200000),
			engine.Bitboard(0x0000000000000800),
			engine.Bitboard(0x0000400000000080),
			engine.Bitboard(0x0000000080000000),
			engine.Bitboard(0x0002000001000000),
			engine.Bitboard(0x0000000000080000),
		},
		{
			engine.Bitboard(0x0000000800000000),
			engine.Bitboard(0x0000000400000000),
			engine.Bitboard(0x0080000000000000),
			engine.Bitboard(0x0000000000001000),
			engine.Bitboard(0),
			engine.Bitboard(0),
		},
	}
	pos := engine.Position{
		Pieces: pieces,
	}
	square := engine.SquareE4

	color := engine.White
	gotAttackers := pos.AttackersFrom(square, color)
	wantAttackers := engine.Bitboard(0x0000400081280800)
	if gotAttackers != wantAttackers {
		t.Errorf(`AttackersFrom(%d, %d)`, square, color)
		fmt.Println("Got:")
		fmt.Println(gotAttackers.ToString())
		fmt.Println("Want:")
		fmt.Println(wantAttackers.ToString())
	}

	color = engine.Black
	gotAttackers = pos.AttackersFrom(square, color)
	wantAttackers = engine.Bitboard(0x0000000c00001000)
	if gotAttackers != wantAttackers {
		t.Errorf(`AttackersFrom(%d, %d)`, square, color)
		fmt.Println("Got:")
		fmt.Println(gotAttackers.ToString())
		fmt.Println("Want:")
		fmt.Println(wantAttackers.ToString())
	}

	gotAttackers = pos.Attackers(square)
	wantAttackers = engine.Bitboard(0x0000400c81281800)
	if gotAttackers != wantAttackers {
		t.Errorf(`Attackers(%d)`, square)
		fmt.Println("Got:")
		fmt.Println(gotAttackers.ToString())
		fmt.Println("Want:")
		fmt.Println(wantAttackers.ToString())
	}
}
