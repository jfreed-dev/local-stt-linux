BINARY := local-stt
BUILD_DIR := bin
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build clean install run vet test

build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/local-stt/

run: build
	./$(BUILD_DIR)/$(BINARY)

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -rf $(BUILD_DIR)

install: build
	install -Dm755 $(BUILD_DIR)/$(BINARY) $(HOME)/.local/bin/$(BINARY)
	install -Dm644 config.example.toml $(HOME)/.config/local-stt-linux/config.toml.example
