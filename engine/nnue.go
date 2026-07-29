package engine

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	Magic   = "NNUE"
	Version = 1
)

// Network holds NNUE weights loaded from a file. It is immutable once
// loaded and safe to share across Positions; per-position evaluation state
// lives in Accumulator instead.
type Network struct {
	NumInputs int
	L1        int

	WInput  []float32 // [L1 * NumInputs]
	BInput  []float32 // [L1]
	WOutput []float32 // [2 * L1]
	BOutput float32

	// FeatureCols is WInput transposed into per-feature columns, laid out
	// as one flat contiguous slice ([NumInputs * L1]) rather than a slice
	// of slices, so column access avoids an extra pointer indirection and
	// keeps neighboring columns cache-adjacent.
	FeatureCols []float32
}

// featureCol returns the L1-length column of weights for feature f.
func (net *Network) featureCol(f uint16) []float32 {
	start := int(f) * net.L1
	return net.FeatureCols[start : start+net.L1]
}

// Accumulator is the mutable per-position NNUE evaluation state: one
// running sum of active feature columns per perspective.
type Accumulator struct {
	Values [2][]float32
}

// TODO: Accumulator stack

// for black's perspective, the board is flipped for evaluation purposes
// this way the first layer parameters only has to "learn" how to play one perspective, which helps with generalization (?)
func FeatureIndex(perspective uint8, pieceColor uint8, pieceType uint8, sq Square) uint16 {
	friendly := perspective == pieceColor
	pieceIdx := pieceType
	if !friendly {
		pieceIdx += 6
	}
	if perspective == Black {
		sq ^= FlipVertical
	}
	return 64*uint16(pieceIdx) + uint16(sq)
}

// LoadNNUEFile loads a network from a file on disk.
func LoadNNUEFile(path string) (*Network, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return loadNNUEFromReader(bufio.NewReader(f))
}

// LoadEmbeddedNNUE loads the default network built into the binary.
func LoadEmbeddedNNUE() (*Network, error) {
	data, err := embeddedNNUEBytes()
	if err != nil {
		return nil, err
	}
	return loadNNUEFromReader(bytes.NewReader(data))
}

// LoadNNUE loads the network at path, or the embedded default if path is empty.
func LoadNNUE(path string) (*Network, error) {
	if path == "" {
		return LoadEmbeddedNNUE()
	}
	return LoadNNUEFile(path)
}

func loadNNUEFromReader(f io.Reader) (*Network, error) {
	// header
	magic := make([]byte, 4)
	if _, err := io.ReadFull(f, magic); err != nil {
		return nil, err
	}
	if string(magic) != Magic {
		return nil, errors.New("invalid NNUE magic")
	}

	var version uint32
	if err := binary.Read(f, binary.LittleEndian, &version); err != nil {
		return nil, err
	}
	if version != Version {
		return nil, fmt.Errorf("unsupported NNUE version %d", version)
	}

	// read as uint32 since python wrote 32 bit ints
	var numInputs32, l132 uint32
	if err := binary.Read(f, binary.LittleEndian, &numInputs32); err != nil {
		return nil, err
	}
	if err := binary.Read(f, binary.LittleEndian, &l132); err != nil {
		return nil, err
	}

	numInputs := int(numInputs32)
	l1 := int(l132)

	// the accumulator update loops are unrolled in steps of 16
	if numInputs <= 0 {
		return nil, fmt.Errorf("invalid NNUE header: numInputs must be positive, got %d", numInputs)
	}
	if l1 <= 0 || l1%16 != 0 {
		return nil, fmt.Errorf("invalid NNUE header: L1 must be a positive multiple of 16, got %d", l1)
	}

	net := &Network{
		NumInputs: numInputs,
		L1:        l1,
	}

	// read network parameters
	net.WInput = make([]float32, l1*numInputs)
	net.BInput = make([]float32, l1)
	net.WOutput = make([]float32, 2*l1)

	if err := binary.Read(f, binary.LittleEndian, &net.WInput); err != nil {
		return nil, err
	}
	if err := binary.Read(f, binary.LittleEndian, &net.BInput); err != nil {
		return nil, err
	}
	if err := binary.Read(f, binary.LittleEndian, &net.WOutput); err != nil {
		return nil, err
	}
	if err := binary.Read(f, binary.LittleEndian, &net.BOutput); err != nil {
		return nil, err
	}

	// build helper representation of first layer's weights
	net.buildFeatureCols()

	return net, nil
}

func (net *Network) buildFeatureCols() {
	numInputs := net.NumInputs
	l1 := net.L1
	flat := make([]float32, numInputs*l1)
	for f := 0; f < numInputs; f++ {
		for o := 0; o < l1; o++ {
			flat[f*l1+o] = net.WInput[o*numInputs+f]
		}
	}
	net.FeatureCols = flat
}

// NewAccumulator allocates a zeroed accumulator sized for net.
func NewAccumulator(net *Network) Accumulator {
	return Accumulator{
		Values: [2][]float32{
			make([]float32, net.L1),
			make([]float32, net.L1),
		},
	}
}

// Clone returns a deep copy, safe to mutate independently of the original.
func (acc Accumulator) Clone() Accumulator {
	if acc.Values[White] == nil && acc.Values[Black] == nil {
		return Accumulator{}
	}
	clone := Accumulator{
		Values: [2][]float32{
			make([]float32, len(acc.Values[White])),
			make([]float32, len(acc.Values[Black])),
		},
	}
	copy(clone.Values[White], acc.Values[White])
	copy(clone.Values[Black], acc.Values[Black])
	return clone
}

// Reset installs net's input bias into both perspectives, discarding any
// active features. This must be done before the first Add/Remove call --
// Add/Remove alone never add bias.
func (acc *Accumulator) Reset(net *Network) {
	copy(acc.Values[White], net.BInput)
	copy(acc.Values[Black], net.BInput)
}

// Refresh overwrites one perspective with net's bias plus the given active features.
func (acc *Accumulator) Refresh(net *Network, features []uint16, perspective uint8) {
	a := acc.Values[perspective]
	copy(a, net.BInput)
	for _, f := range features {
		col := net.featureCol(f)
		for o := 0; o < len(a); o += 16 {
			a[o] += col[o]
			a[o+1] += col[o+1]
			a[o+2] += col[o+2]
			a[o+3] += col[o+3]
			a[o+4] += col[o+4]
			a[o+5] += col[o+5]
			a[o+6] += col[o+6]
			a[o+7] += col[o+7]
			a[o+8] += col[o+8]
			a[o+9] += col[o+9]
			a[o+10] += col[o+10]
			a[o+11] += col[o+11]
			a[o+12] += col[o+12]
			a[o+13] += col[o+13]
			a[o+14] += col[o+14]
			a[o+15] += col[o+15]
		}
	}
}

// RefreshAll overwrites both perspectives with net's bias plus their active features.
func (acc *Accumulator) RefreshAll(net *Network, featuresW, featuresB []uint16) {
	acc.Refresh(net, featuresW, White)
	acc.Refresh(net, featuresB, Black)
}

// Add incrementally adds a feature. NOTE: only using Add() is incorrect
// since no bias is added -- Reset() (or RefreshAll) must run first.
func (acc *Accumulator) Add(net *Network, feature uint16, perspective uint8) {
	col := net.featureCol(feature)
	a := acc.Values[perspective]
	for o := 0; o < len(a); o += 16 {
		a[o] += col[o]
		a[o+1] += col[o+1]
		a[o+2] += col[o+2]
		a[o+3] += col[o+3]
		a[o+4] += col[o+4]
		a[o+5] += col[o+5]
		a[o+6] += col[o+6]
		a[o+7] += col[o+7]
		a[o+8] += col[o+8]
		a[o+9] += col[o+9]
		a[o+10] += col[o+10]
		a[o+11] += col[o+11]
		a[o+12] += col[o+12]
		a[o+13] += col[o+13]
		a[o+14] += col[o+14]
		a[o+15] += col[o+15]
	}
}

// Remove incrementally removes a feature.
func (acc *Accumulator) Remove(net *Network, feature uint16, perspective uint8) {
	col := net.featureCol(feature)
	a := acc.Values[perspective]
	for o := 0; o < len(a); o += 16 {
		a[o] -= col[o]
		a[o+1] -= col[o+1]
		a[o+2] -= col[o+2]
		a[o+3] -= col[o+3]
		a[o+4] -= col[o+4]
		a[o+5] -= col[o+5]
		a[o+6] -= col[o+6]
		a[o+7] -= col[o+7]
		a[o+8] -= col[o+8]
		a[o+9] -= col[o+9]
		a[o+10] -= col[o+10]
		a[o+11] -= col[o+11]
		a[o+12] -= col[o+12]
		a[o+13] -= col[o+13]
		a[o+14] -= col[o+14]
		a[o+15] -= col[o+15]
	}
}

// AddSub applies one added feature and one removed feature in a single pass,
// halving the memory traffic of a separate Add+Remove for the common case of
// relocating a piece (quiet move) from one square to another.
func (acc *Accumulator) AddSub(net *Network, addFeature, subFeature uint16, perspective uint8) {
	addCol := net.featureCol(addFeature)
	subCol := net.featureCol(subFeature)
	a := acc.Values[perspective]
	for o := 0; o < len(a); o += 16 {
		a[o] += addCol[o] - subCol[o]
		a[o+1] += addCol[o+1] - subCol[o+1]
		a[o+2] += addCol[o+2] - subCol[o+2]
		a[o+3] += addCol[o+3] - subCol[o+3]
		a[o+4] += addCol[o+4] - subCol[o+4]
		a[o+5] += addCol[o+5] - subCol[o+5]
		a[o+6] += addCol[o+6] - subCol[o+6]
		a[o+7] += addCol[o+7] - subCol[o+7]
		a[o+8] += addCol[o+8] - subCol[o+8]
		a[o+9] += addCol[o+9] - subCol[o+9]
		a[o+10] += addCol[o+10] - subCol[o+10]
		a[o+11] += addCol[o+11] - subCol[o+11]
		a[o+12] += addCol[o+12] - subCol[o+12]
		a[o+13] += addCol[o+13] - subCol[o+13]
		a[o+14] += addCol[o+14] - subCol[o+14]
		a[o+15] += addCol[o+15] - subCol[o+15]
	}
}

// AddSubSub applies one added feature and two removed features in a single
// pass -- used for captures, where the mover relocates (add at `to`, sub at
// `from`) and the captured piece disappears (sub at `to`).
func (acc *Accumulator) AddSubSub(net *Network, addFeature, subFeature1, subFeature2 uint16, perspective uint8) {
	addCol := net.featureCol(addFeature)
	sub1Col := net.featureCol(subFeature1)
	sub2Col := net.featureCol(subFeature2)
	a := acc.Values[perspective]
	for o := 0; o < len(a); o += 16 {
		a[o] += addCol[o] - sub1Col[o] - sub2Col[o]
		a[o+1] += addCol[o+1] - sub1Col[o+1] - sub2Col[o+1]
		a[o+2] += addCol[o+2] - sub1Col[o+2] - sub2Col[o+2]
		a[o+3] += addCol[o+3] - sub1Col[o+3] - sub2Col[o+3]
		a[o+4] += addCol[o+4] - sub1Col[o+4] - sub2Col[o+4]
		a[o+5] += addCol[o+5] - sub1Col[o+5] - sub2Col[o+5]
		a[o+6] += addCol[o+6] - sub1Col[o+6] - sub2Col[o+6]
		a[o+7] += addCol[o+7] - sub1Col[o+7] - sub2Col[o+7]
		a[o+8] += addCol[o+8] - sub1Col[o+8] - sub2Col[o+8]
		a[o+9] += addCol[o+9] - sub1Col[o+9] - sub2Col[o+9]
		a[o+10] += addCol[o+10] - sub1Col[o+10] - sub2Col[o+10]
		a[o+11] += addCol[o+11] - sub1Col[o+11] - sub2Col[o+11]
		a[o+12] += addCol[o+12] - sub1Col[o+12] - sub2Col[o+12]
		a[o+13] += addCol[o+13] - sub1Col[o+13] - sub2Col[o+13]
		a[o+14] += addCol[o+14] - sub1Col[o+14] - sub2Col[o+14]
		a[o+15] += addCol[o+15] - sub1Col[o+15] - sub2Col[o+15]
	}
}

// AddAddSub applies two added features and one removed feature in a single
// pass -- the inverse of AddSubSub, used to undo a capture: the mover moves
// back (add at `from`, sub at `to`) and the captured piece reappears (add at
// `to`).
func (acc *Accumulator) AddAddSub(net *Network, addFeature1, addFeature2, subFeature uint16, perspective uint8) {
	add1Col := net.featureCol(addFeature1)
	add2Col := net.featureCol(addFeature2)
	subCol := net.featureCol(subFeature)
	a := acc.Values[perspective]
	for o := 0; o < len(a); o += 16 {
		a[o] += add1Col[o] + add2Col[o] - subCol[o]
		a[o+1] += add1Col[o+1] + add2Col[o+1] - subCol[o+1]
		a[o+2] += add1Col[o+2] + add2Col[o+2] - subCol[o+2]
		a[o+3] += add1Col[o+3] + add2Col[o+3] - subCol[o+3]
		a[o+4] += add1Col[o+4] + add2Col[o+4] - subCol[o+4]
		a[o+5] += add1Col[o+5] + add2Col[o+5] - subCol[o+5]
		a[o+6] += add1Col[o+6] + add2Col[o+6] - subCol[o+6]
		a[o+7] += add1Col[o+7] + add2Col[o+7] - subCol[o+7]
		a[o+8] += add1Col[o+8] + add2Col[o+8] - subCol[o+8]
		a[o+9] += add1Col[o+9] + add2Col[o+9] - subCol[o+9]
		a[o+10] += add1Col[o+10] + add2Col[o+10] - subCol[o+10]
		a[o+11] += add1Col[o+11] + add2Col[o+11] - subCol[o+11]
		a[o+12] += add1Col[o+12] + add2Col[o+12] - subCol[o+12]
		a[o+13] += add1Col[o+13] + add2Col[o+13] - subCol[o+13]
		a[o+14] += add1Col[o+14] + add2Col[o+14] - subCol[o+14]
		a[o+15] += add1Col[o+15] + add2Col[o+15] - subCol[o+15]
	}
}

func (acc *Accumulator) Evaluate(net *Network, side uint8) float32 {
	ourAcc := acc.Values[side]
	theirAcc := acc.Values[1-side]
	var result float32 = 0.0

	// ReLU before output layer
	for i := 0; i < net.L1; i++ {
		result += net.WOutput[i] * max(ourAcc[i], 0.0)
	}
	for j := 0; j < net.L1; j++ {
		result += net.WOutput[net.L1+j] * max(theirAcc[j], 0.0)
	}
	result += net.BOutput
	return result
}
