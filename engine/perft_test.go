package engine_test

import (
	"fmt"
	"silverfish/engine"
	"testing"
)

func TestPerft(t *testing.T) {
	perft_results := []uint64{1, 20, 400, 8902, 197281, 4865609} //, 119060324, 3195901860}
	pos := engine.StartingPosition()

	// run Perft over a range of depths
	/* for i, want := range perft_results {
		got := engine.Perft(&pos, i)
		if want != got {
			t.Errorf(`Perft(%d)`, i)
			fmt.Printf("Got: %d\n", got)
			fmt.Printf("Want: %d\n", want)
		}
	} */

	depth := 5
	want := perft_results[depth]
	got := engine.Perft(&pos, depth)
	if want != got {
		t.Errorf(`Perft(%d)`, depth)
		fmt.Printf("Got: %d\n", got)
		fmt.Printf("Want: %d\n", want)
	}
}
