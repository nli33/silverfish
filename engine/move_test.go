package engine_test

import (
	"silverfish/engine"
	"testing"
)

// rn2kb1r/pp3ppp/2p1pn2/3p3b/8/1P1P1NPP/PBPqPPB1/RN2K2R w KQkq - 0 9

func TestGiveScore(t *testing.T) {
	move1 := engine.NewMoveFromStr("g2d2")
	move2 := move1
	move2.GiveScore(100)
	if move2.Score() != 100 {
		t.Errorf(`TestGiveScore: "Score": expected %d, got %d`, 100, move2.Score())
	}
	if move1.To() != move2.To() {
		t.Errorf(`TestGiveScore: "To": expected %d, got %d`, move1.To(), move2.To())
	}
}
