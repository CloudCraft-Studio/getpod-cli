.PHONY: build run test lint clean

BINARY_NAME=getpod
BINARY_DIR=./bin
CMD_DIR=./cmd/getpod
VERSION?=0.1.0
LDFLAGS=-ldflags "-X main.Version=$(VERSION)"

build:
	@mkdir -p $(BINARY_DIR)
	go build $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME) $(CMD_DIR)

run: build
	./$(BINARY_DIR)/$(BINARY_NAME)

test:
	go test ./...

lint:
	golangci-lint run

clean:
	rm -rf $(BINARY_DIR)