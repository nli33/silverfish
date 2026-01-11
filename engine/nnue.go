package engine

import (
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

func LoadNNUE(path string) (*NNUE, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

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
			col[o] = nnue.WInput[f*l1+o]
		}
		cols[f] = col
	}
	nnue.FeatureCols = cols
}

// add a whole set of initial features (overwriting existing features)
func (nnue *NNUE) Refresh(features []uint16, perspective uint8) {
	acc := nnue.Acc.Values[perspective]
	copy(acc, nnue.BInput)
	for _, f := range features {
		col := nnue.FeatureCols[f]
		for o := 0; o < nnue.L1; o++ {
			acc[o] += col[o]
		}
	}
}

// refresh both perspectives
func (nnue *NNUE) RefreshAll(featuresW, featuresB []uint16) {
	nnue.Refresh(featuresW, White)
	nnue.Refresh(featuresB, Black)
}

// incrementally add a feature
func (nnue *NNUE) Add(feature uint16, perspective uint8) {
	col := nnue.FeatureCols[feature]
	for o := 0; o < nnue.L1; o++ {
		nnue.Acc.Values[perspective][o] += col[o]
	}
}

// incrementally remove a feature
func (nnue *NNUE) Remove(feature uint16, perspective uint8) {
	col := nnue.FeatureCols[feature]
	for o := 0; o < nnue.L1; o++ {
		nnue.Acc.Values[perspective][o] -= col[o]
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
