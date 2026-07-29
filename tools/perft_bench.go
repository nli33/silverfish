//go:build ignore

package main

import (
	"fmt"
	"time"

	"silverfish/engine"
)

func main() {
	engine.Init()
	pos := engine.StartingPosition()
	depth := 6
	start := time.Now()
	nodes := engine.Perft(&pos, depth, false)
	elapsed := time.Since(start)
	fmt.Printf("depth=%d nodes=%d elapsed=%s nps=%.0f\n", depth, nodes, elapsed, float64(nodes)/elapsed.Seconds())
}
