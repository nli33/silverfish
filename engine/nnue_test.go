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

	// For 256_test.nnue (one nonzero weight)
	//pos := engine.FromFEN("8/8/8/8/8/8/P7/8 w - - 0 1")

	// var featuresW []uint16
	// var featuresB []uint16
	// for sq := engine.SquareA1; sq <= engine.SquareH8; sq++ {
	// 	if pos.Board[sq] >= 10 {
	// 		featuresW = append(featuresW, engine.FeatureIndex(engine.White, engine.Black, pos.Board[sq]-uint8(10), sq))
	// 		featuresB = append(featuresB, engine.FeatureIndex(engine.Black, engine.Black, pos.Board[sq]-uint8(10), sq))
	// 	} else if pos.Board[sq] != 6 {
	// 		featuresW = append(featuresW, engine.FeatureIndex(engine.White, engine.White, pos.Board[sq], sq))
	// 		featuresB = append(featuresB, engine.FeatureIndex(engine.Black, engine.White, pos.Board[sq], sq))
	// 	}
	// }
	fmt.Println(1000 * pos.Nnue.Evaluate(pos.Turn))

	pos.DoMove(engine.NewMoveFromStr("d1a1"))
	// nnue.Remove(engine.FeatureIndex(engine.White, engine.White, engine.Queen, engine.SquareD1), engine.White)
	// nnue.Remove(engine.FeatureIndex(engine.Black, engine.White, engine.Queen, engine.SquareD1), engine.Black)
	// nnue.Add(engine.FeatureIndex(engine.White, engine.White, engine.Queen, engine.SquareA1), engine.White)
	// nnue.Add(engine.FeatureIndex(engine.Black, engine.White, engine.Queen, engine.SquareA1), engine.Black)

	fmt.Println(1000 * pos.Nnue.Evaluate(pos.Turn))

	pos.DoMove(engine.NewMoveFromStr("b6a5"))
	// nnue.Remove(engine.FeatureIndex(engine.White, engine.Black, engine.Queen, engine.SquareB6), engine.White)
	// nnue.Remove(engine.FeatureIndex(engine.Black, engine.Black, engine.Queen, engine.SquareB6), engine.Black)
	// nnue.Add(engine.FeatureIndex(engine.White, engine.Black, engine.Queen, engine.SquareA5), engine.White)
	// nnue.Add(engine.FeatureIndex(engine.Black, engine.Black, engine.Queen, engine.SquareA5), engine.Black)

	fmt.Println(1000 * pos.Nnue.Evaluate(pos.Turn))

	// t.Errorf("")
}
