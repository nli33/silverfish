package engine

import (
	"fmt"
	"math/rand"
	"os"
)

var Rng *rand.Rand = rand.New(rand.NewSource(123))

func Init() {
	UciSetEngineName("silverfish")
	UciSetAuthor("silverfish developers")
	UciSetProtocol(2)

	err := UciOk()
	// This should not happen, ideally, but if it does, something is deeply wrong
	// with the program.
	if err != nil {
		fmt.Errorf("WHAT THE FUCK??")
		os.Exit(69)
	}
}
