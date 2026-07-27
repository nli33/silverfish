package engine

import (
	"math/rand"
)

var Rng *rand.Rand = rand.New(rand.NewSource(123))

var defaultNet *Network

func Init() {
	InitBitboard()
	if err := LoadDefaultNetwork(""); err != nil {
		panic("engine: failed to load default NNUE network: " + err.Error())
	}
}

// LoadDefaultNetwork loads path ("" for the embedded default) and installs
// it as the network used by newly-created Positions (FromFEN, NewPosition).
// Positions already holding a *Network keep using it -- swapping this only
// affects positions created afterward. Must only be called from the UCI
// main loop's idle state (not while a search goroutine is running) since
// there is no synchronization on the read in NewPosition/FromFEN.
func LoadDefaultNetwork(path string) error {
	net, err := LoadNNUE(path)
	if err != nil {
		return err
	}
	defaultNet = net
	return nil
}

// DefaultNetwork returns the network currently installed by Init/LoadDefaultNetwork.
func DefaultNetwork() *Network {
	return defaultNet
}
