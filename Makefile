BIN_DIR := bin
BINARY := silverfish

.PHONY: build run test

build:
	go build -o $(BIN_DIR)/$(BINARY) silverfish/cmd/$(BINARY)

run:
	go run cmd/silverfish/main.go

test:
	go test silverfish/test

clean:
	go clean
	rm -rf $(BIN_DIR)