package engine

import "embed"

//go:embed nnue/*.nnue
var nnueFS embed.FS

const defaultNNUEName = "nnue/256.nnue"

func embeddedNNUEBytes() ([]byte, error) {
	return nnueFS.ReadFile(defaultNNUEName)
}
