package engine

import (
	"fmt"
	"math/rand"
)

var Rng *rand.Rand = rand.New(rand.NewSource(123))

func Init() {
	fmt.Println("HELLO")
}
