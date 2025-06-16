package engine

import (
	"math/rand"
)

var Rng *rand.Rand = rand.New(rand.NewSource(123))

func Init() {
	UciSetEngineName("silverfish")
	UciSetAuthor("silverfish developers")
	UciSetProtocol(2)

}
