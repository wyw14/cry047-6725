# Makefile for the public-facility preventive maintenance collaboration platform.
# Common targets: build, run, test, fmt, vet, tidy, frontend, clean.

GO ?= go
GOFLAGS := -trimpath
APP := baseline-server
BIN_DIR := bin
WEB_DIR := web

.PHONY: all build run server test test-race vet fmt tidy fmt-check clean frontend frontend-install frontend-dev frontend-build frontend-test help

all: build frontend-build

build:
	@echo ">> building Go server"
	$(GO) build $(GOFLAGS) -o $(BIN_DIR)/$(APP) ./cmd/server

run: build
	@echo ">> running server (offline mode)"
	./$(BIN_DIR)/$(APP)

server: build
	./$(BIN_DIR)/$(APP)

test:
	@echo ">> go test ./..."
	$(GO) test ./...

test-race:
	@echo ">> go test -race ./..."
	$(GO) test -race ./...

vet:
	@echo ">> go vet ./..."
	$(GO) vet ./...

fmt:
	@echo ">> gofmt -w ."
	gofmt -w .

fmt-check:
	@echo ">> gofmt -l . (must be empty)"
	@gofmt -l . && echo "gofmt OK"

tidy:
	@echo ">> go mod tidy"
	$(GO) mod tidy

frontend-install:
	@echo ">> installing frontend deps"
	cd $(WEB_DIR) && npm install

frontend: frontend-install frontend-build

frontend-dev:
	@echo ">> vite dev server"
	cd $(WEB_DIR) && npm run dev

frontend-build:
	@echo ">> vite build"
	cd $(WEB_DIR) && npm run build

frontend-test:
	@echo ">> frontend tests"
	cd $(WEB_DIR) && npm run test -- --run

clean:
	@echo ">> cleaning"
	rm -rf $(BIN_DIR) $(WEB_DIR)/dist $(WEB_DIR)/node_modules/.vite
	rm -f coverage.out coverage.txt

help:
	@echo "Available targets:"
	@echo "  build            - build the Go server into bin/"
	@echo "  run              - build then run the server (offline mode)"
	@echo "  test             - go test ./..."
	@echo "  test-race        - go test -race ./..."
	@echo "  vet              - go vet ./..."
	@echo "  fmt              - gofmt -w ."
	@echo "  fmt-check        - gofmt -l . (verify clean)"
	@echo "  tidy             - go mod tidy"
	@echo "  frontend-install - npm install in web/"
	@echo "  frontend-build   - vite build in web/"
	@echo "  frontend-test    - vitest run in web/"
	@echo "  all              - build + frontend-build"
	@echo "  clean            - remove build artifacts"
