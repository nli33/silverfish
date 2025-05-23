package engine

import (
	"fmt"
	"math/rand"
)

var Rng *rand.Rand = rand.New(rand.NewSource(14))

func Init() {
	fmt.Println("HELLO")
}
