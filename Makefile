.PHONY: all build run clean test

BINARY_NAME := netdispatch
MAIN_PATH := ./cmd/netdispatch

# Build-time variables (can be overridden)
VERSION ?= 0.1.0
API_PORT ?= 9090

# Linker flags for embedding build info (must use string types for -X)
# -H=windowsgui: Build as GUI application (no console window on Windows)
LDFLAGS := -H=windowsgui -X "main.version=$(VERSION)" -X "main.defaultAPIPortStr=$(API_PORT)"

# Detect OS
UNAME_S := $(shell uname -s 2>/dev/null || echo "Windows")

all: build

build: web-build
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME).exe $(MAIN_PATH)

run:
	go run $(MAIN_PATH)/main.go start -c configs/config.yaml

dev: build
	./bin/$(BINARY_NAME).exe start -c configs/config.yaml

clean:
	rm -rf bin/
	rm -f $(BINARY_NAME)

test:
	go test -v ./...

deps:
	go mod download
	go mod tidy

web-deps:
	cd web && npm install

web-dev:
	cd web && npm run dev

web-build:
	@if [ ! -d "web/node_modules" ]; then \
		echo "Installing web dependencies..."; \
		cd web && npm install; \
	fi
	cd web && npm run build

# Build with custom API port
# Example: make build-custom API_PORT=8080
build-custom: web-build
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME)-port$(API_PORT).exe $(MAIN_PATH)

docker-build:
	docker build -t netdispatch:latest -f deployments/docker/Dockerfile .

.PHONY: lint
lint:
	golangci-lint run ./...
