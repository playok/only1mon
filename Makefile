APP_NAME := only1mon
BUILD_DIR := build
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: all build start stop status run clean dev

all: build

build:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME) ./cmd/only1mon

start: build
	./$(BUILD_DIR)/$(APP_NAME) start

stop:
	./$(BUILD_DIR)/$(APP_NAME) stop

status:
	./$(BUILD_DIR)/$(APP_NAME) status

run: build
	./$(BUILD_DIR)/$(APP_NAME) run

dev:
	go run ./cmd/only1mon run

clean:
	rm -rf $(BUILD_DIR) only1mon.db only1mon.pid only1mon.log

# Cross compilation
build-linux-amd64:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 ./cmd/only1mon

build-linux-arm64:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64 ./cmd/only1mon

build-darwin-amd64:
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 ./cmd/only1mon

build-darwin-arm64:
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 ./cmd/only1mon

build-all: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64

# eBPF build (Linux only, requires go generate for BPF bytecode)
build-ebpf:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -tags ebpf -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-ebpf ./cmd/only1mon
