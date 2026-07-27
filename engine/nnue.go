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

type NNUE struct {
	NumInputs int
	L1        int

	WInput  []float32 // [L1 * NumInputs]
	BInput  []float32 // [L1]
	WOutput []float32 // [2 * L1]
	BOutput float32

	FeatureCols [][]float32

	Acc Accumulator
}

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
func LoadNNUEFile(path string) (*NNUE, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return loadNNUEFromReader(bufio.NewReader(f))
}

// LoadEmbeddedNNUE loads the default network built into the binary.
func LoadEmbeddedNNUE() (*NNUE, error) {
	data, err := embeddedNNUEBytes()
	if err != nil {
		return nil, err
	}
	return loadNNUEFromReader(bytes.NewReader(data))
}

// LoadNNUE loads the network at path, or the embedded default if path is empty.
func LoadNNUE(path string) (*NNUE, error) {
	if path == "" {
		return LoadEmbeddedNNUE()
	}
	return LoadNNUEFile(path)
}

func loadNNUEFromReader(f io.Reader) (*NNUE, error) {
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

	nnue := &NNUE{
		NumInputs: numInputs,
		L1:        l1,
	}

	// read network parameters
	nnue.WInput = make([]float32, l1*numInputs)
	nnue.BInput = make([]float32, l1)
	nnue.WOutput = make([]float32, 2*l1)

	if err := binary.Read(f, binary.LittleEndian, &nnue.WInput); err != nil {
		return nil, err
	}
	if err := binary.Read(f, binary.LittleEndian, &nnue.BInput); err != nil {
		return nil, err
	}
	if err := binary.Read(f, binary.LittleEndian, &nnue.WOutput); err != nil {
		return nil, err
	}
	if err := binary.Read(f, binary.LittleEndian, &nnue.BOutput); err != nil {
		return nil, err
	}

	// build helper representation of first layer's weights
	nnue.BuildFeatureCols()

	// allocate accumulators for both sides
	nnue.Acc.Values[0] = make([]float32, l1)
	nnue.Acc.Values[1] = make([]float32, l1)

	return nnue, nil
}

// must be called first before using anything
func (nnue *NNUE) BuildFeatureCols() {
	numInputs := int(nnue.NumInputs)
	l1 := int(nnue.L1)
	cols := make([][]float32, numInputs)
	for f := 0; f < numInputs; f++ {
		col := make([]float32, l1)
		for o := 0; o < l1; o++ {
			col[o] = nnue.WInput[o*numInputs+f]
		}
		cols[f] = col
	}
	nnue.FeatureCols = cols
}

// add a whole set of initial features (overwriting existing features).
// this is the only place where input bias is added
func (nnue *NNUE) Refresh(features []uint16, perspective uint8) {
	acc := nnue.Acc.Values[perspective]
	copy(acc, nnue.BInput)
	for _, f := range features {
		col := nnue.FeatureCols[f]
		for o := 0; o < len(acc); o += 16 {
			acc[o] += col[o]
			acc[o+1] += col[o+1]
			acc[o+2] += col[o+2]
			acc[o+3] += col[o+3]
			acc[o+4] += col[o+4]
			acc[o+5] += col[o+5]
			acc[o+6] += col[o+6]
			acc[o+7] += col[o+7]
			acc[o+8] += col[o+8]
			acc[o+9] += col[o+9]
			acc[o+10] += col[o+10]
			acc[o+11] += col[o+11]
			acc[o+12] += col[o+12]
			acc[o+13] += col[o+13]
			acc[o+14] += col[o+14]
			acc[o+15] += col[o+15]
		}
	}
}

// refresh both perspectives
func (nnue *NNUE) RefreshAll(featuresW, featuresB []uint16) {
	nnue.Refresh(featuresW, White)
	nnue.Refresh(featuresB, Black)
}

// incrementally add a feature
// NOTE: only using Add() is incorrect, since no bias is added
// remember to also to perform an empty refresh? (ex: in FromFEN)
func (nnue *NNUE) Add(feature uint16, perspective uint8) {
	col := nnue.FeatureCols[feature]
	acc := nnue.Acc.Values[perspective]
	for o := 0; o < len(acc); o += 16 {
		acc[o] += col[o]
		acc[o+1] += col[o+1]
		acc[o+2] += col[o+2]
		acc[o+3] += col[o+3]
		acc[o+4] += col[o+4]
		acc[o+5] += col[o+5]
		acc[o+6] += col[o+6]
		acc[o+7] += col[o+7]
		acc[o+8] += col[o+8]
		acc[o+9] += col[o+9]
		acc[o+10] += col[o+10]
		acc[o+11] += col[o+11]
		acc[o+12] += col[o+12]
		acc[o+13] += col[o+13]
		acc[o+14] += col[o+14]
		acc[o+15] += col[o+15]
	}
}

// incrementally remove a feature
func (nnue *NNUE) Remove(feature uint16, perspective uint8) {
	col := nnue.FeatureCols[feature]
	acc := nnue.Acc.Values[perspective]
	for o := 0; o < len(acc); o += 16 {
		acc[o] -= col[o]
		acc[o+1] -= col[o+1]
		acc[o+2] -= col[o+2]
		acc[o+3] -= col[o+3]
		acc[o+4] -= col[o+4]
		acc[o+5] -= col[o+5]
		acc[o+6] -= col[o+6]
		acc[o+7] -= col[o+7]
		acc[o+8] -= col[o+8]
		acc[o+9] -= col[o+9]
		acc[o+10] -= col[o+10]
		acc[o+11] -= col[o+11]
		acc[o+12] -= col[o+12]
		acc[o+13] -= col[o+13]
		acc[o+14] -= col[o+14]
		acc[o+15] -= col[o+15]
	}
}

func (nnue *NNUE) Evaluate(side uint8) float32 {
	ourAcc := nnue.Acc.Values[side]
	theirAcc := nnue.Acc.Values[1-side]
	var result float32 = 0.0

	// ReLU before output layer
	for i := 0; i < nnue.L1; i++ {
		result += nnue.WOutput[i] * max(ourAcc[i], 0.0)
	}
	for j := 0; j < nnue.L1; j++ {
		result += nnue.WOutput[nnue.L1+j] * max(theirAcc[j], 0.0)
	}
	result += nnue.BOutput
	return result
}
