BINARY := dewbu
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X main.Version=$(VERSION) -X main.GitCommit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)

.PHONY: build install clean

build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/dewbu

install: build
	install -m 755 $(BINARY) ~/.local/bin/$(BINARY)

clean:
	rm -f $(BINARY)
	rm -rf dist/
