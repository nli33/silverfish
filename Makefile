BIN_DIR := bin
BINARY := silverfish

.PHONY: build run test clean

build:
	go build -o $(BIN_DIR)/$(BINARY) silverfish/cmd/$(BINARY)

run:
	go run ./cmd/silverfish

test:
	go test ./engine

clean:
	go clean
	rm -rf $(BIN_DIR)
