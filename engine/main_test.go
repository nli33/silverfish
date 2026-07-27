package engine_test

import (
	"os"
	"testing"

	"silverfish/engine"
)

func TestMain(m *testing.M) {
	engine.Init()
	os.Exit(m.Run())
}
