package test

import (
	"fmt"
	"silverfish/engine"
	"testing"
)

func TestMoveStr(t *testing.T) {
	gotStr := engine.NewMoveFromStr("b2b3").ToString()
	wantStr := "b2b3"
	if gotStr != wantStr {
		t.Errorf("NewMoveFromStr(b2b3).ToString()")
		fmt.Printf("Got: %s\n", gotStr)
		fmt.Printf("Want: %s\n", wantStr)
	}

	gotStr = engine.NewMoveFromStr("b2b1q").ToString()
	wantStr = "b2b1q"
	if gotStr != wantStr {
		t.Errorf("NewMoveFromStr(b2b1q).ToString()")
		fmt.Printf("Got: %s\n", gotStr)
		fmt.Printf("Want: %s\n", wantStr)
	}

	gotMove := engine.NewMoveFromStr("g6h5")
	wantMove := engine.NewMove(engine.SquareG6, engine.SquareH5)
	if gotMove != wantMove {
		t.Errorf("NewMoveFromStr(g6h5)")
		fmt.Printf("Got: %016b\n", gotMove)
		fmt.Printf("Want: %016b\n", wantMove)
	}

	gotMove = engine.NewMoveFromStr("b2b1q")
	wantMove = engine.NewPromotionMove(engine.SquareB2, engine.SquareB1, engine.Queen)
	if gotMove != wantMove {
		t.Errorf("NewMoveFromStr(b2b1q)")
		fmt.Printf("Got: %016b\n", gotMove)
		fmt.Printf("Want: %016b\n", wantMove)
	}
}
