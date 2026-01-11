package engine_test

import (
	"fmt"
	"silverfish/engine"
	"testing"
)

// entirely manual verification against pytorch eval script for now
// TODO: automatically run the torch eval script
func TestRefreshEval(t *testing.T) {
	pos := engine.FromFEN("6k1/5p1p/1q2p1p1/1PnpP3/3N4/1Pr5/P5PP/3QR1K1 w - - 3 37")
	// pos = engine.StartingPosition()
	nnue, err := engine.LoadNNUE("test.nnue")
	if err != nil {
		fmt.Println(err)
		t.Errorf("Error")
	}
	var w_features []uint16
	var b_features []uint16
	for sq := engine.SquareA1; sq <= engine.SquareH8; sq++ {
		if pos.Board[sq] >= 10 {
			w_features = append(w_features, engine.FeatureIndex(engine.White, engine.Black, pos.Board[sq]-uint8(10), sq))
			b_features = append(b_features, engine.FeatureIndex(engine.Black, engine.Black, pos.Board[sq]-uint8(10), sq))
		} else if pos.Board[sq] != 6 {
			w_features = append(w_features, engine.FeatureIndex(engine.White, engine.White, pos.Board[sq], sq))
			b_features = append(b_features, engine.FeatureIndex(engine.Black, engine.White, pos.Board[sq], sq))
		}
	}
	nnue.RefreshAll(w_features, b_features)
	fmt.Println(1000 * nnue.Evaluate(pos.Turn))

	pos.DoMove(engine.NewMoveFromStr("d1a1"))
	nnue.Remove(engine.FeatureIndex(engine.White, engine.White, engine.Queen, engine.SquareD1), engine.White)
	nnue.Remove(engine.FeatureIndex(engine.Black, engine.White, engine.Queen, engine.SquareD1), engine.Black)
	nnue.Add(engine.FeatureIndex(engine.White, engine.White, engine.Queen, engine.SquareA1), engine.White)
	nnue.Add(engine.FeatureIndex(engine.Black, engine.White, engine.Queen, engine.SquareA1), engine.Black)

	fmt.Println(1000 * nnue.Evaluate(pos.Turn))

	pos.DoMove(engine.NewMoveFromStr("b6a5"))
	nnue.Remove(engine.FeatureIndex(engine.White, engine.Black, engine.Queen, engine.SquareB6), engine.White)
	nnue.Remove(engine.FeatureIndex(engine.Black, engine.Black, engine.Queen, engine.SquareB6), engine.Black)
	nnue.Add(engine.FeatureIndex(engine.White, engine.Black, engine.Queen, engine.SquareA5), engine.White)
	nnue.Add(engine.FeatureIndex(engine.Black, engine.Black, engine.Queen, engine.SquareA5), engine.Black)

	fmt.Println(1000 * nnue.Evaluate(pos.Turn))

	// t.Errorf("")
}
