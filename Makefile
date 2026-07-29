BIN_DIR := bin
BINARY := silverfish

.PHONY: build run test perft clean

build:
	go build -o $(BIN_DIR)/$(BINARY) silverfish/cmd/$(BINARY)

run:
	go run ./cmd/silverfish

test:
	go test ./engine

perft:
	go run tools/perft_bench.go

clean:
	go clean
	rm -rf $(BIN_DIR)
