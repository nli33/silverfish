package engine

import (
	"embed"
	"fmt"
	"os"
)

//go:embed nnue/*.nnue
var nnueFS embed.FS

// resolveNNUEPath returns a filesystem path to open for the NNUE data
//
// If `path` is empty: it writes the embedded default NNUE to a temp file and
// returns that temp path and a cleanup func that should be deferred by caller
//
// If `path` is non-empty it is returned as-is (no cleanup function)
//
// error - failure if neither the provided filepath nor the embedded NNUE is available
func resolveNNUEPath(path string) (string, func(), error) {
	// caller provides path
	if path != "" {
		// caller is responsible for not deleting this file
		return path, func() {}, nil
	}
	// embedded default
	return writeEmbeddedToTemp("nnue/256.nnue")
}

// write the embedded file `embeddedName` to a temporary
// file and returns the temp path and a cleanup func that removes it
func writeEmbeddedToTemp(embeddedName string) (string, func(), error) {
	data, err := nnueFS.ReadFile(embeddedName)
	if err != nil {
		return "", nil, fmt.Errorf("embedded default nnue not found (%s): %w", embeddedName, err)
	}

	// tmp file at /tmp/nnue-xxxxxx.nnue
	tmp, err := os.CreateTemp("", "nnue-*.nnue")
	if err != nil {
		return "", nil, err
	}
	// close tmp file at the end
	tmpName := tmp.Name()
	defer func() {
		_ = tmp.Close()
	}()

	// write NNUE data to tmp file
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return "", nil, err
	}
	// force data to disk
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return "", nil, err
	}
	// close tmp file
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return "", nil, err
	}
	// cleanup function -- deleting tmp file
	cleanup := func() {
		_ = os.Remove(tmpName)
	}
	return tmpName, cleanup, nil
}
