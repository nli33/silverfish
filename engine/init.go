package engine

import (
	"fmt"
	"math/rand"
)

var Rng *rand.Rand = rand.New(rand.NewSource(123))

func Init() {
	// Wait for the uci command
	for {
		command := ""
		fmt.Scan(&command)
		if command == "uci" {
			break
		}
	}

	UciSetEngineName("silverfish")
	UciSetAuthor("silverfish developers")
	UciSetProtocol(2)
	UciError("HELLO")
}
