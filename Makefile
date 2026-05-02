SHELL := /usr/bin/env bash
.ONESHELL:

APP        := ms-storage
PKG        := github.com/KryptoStorage/ms-storage
BIN_DIR    := bin
BIN        := $(BIN_DIR)/$(APP)
COVER_OUT  := coverage.out
GOFLAGS    := -trimpath
LDFLAGS    := -s -w

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS=":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

.PHONY: tidy
tidy: ## go mod tidy
	go mod tidy

.PHONY: build
build: ## Build the server binary into ./bin
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BIN) ./cmd/server

.PHONY: run
run: ## Run the server with console logging
	LOG_FORMAT=console LOG_LEVEL=debug go run ./cmd/server

.PHONY: test
test: ## Run unit tests
	go test -race -count=1 ./...

.PHONY: cover
cover: ## Run tests with coverage report
	go test -race -count=1 -coverprofile=$(COVER_OUT) ./...
	go tool cover -func=$(COVER_OUT) | tail -n 1

.PHONY: cover-html
cover-html: cover ## Open coverage HTML report
	go tool cover -html=$(COVER_OUT)

.PHONY: vet
vet: ## go vet
	go vet ./...

.PHONY: lint
lint: ## golangci-lint
	golangci-lint run ./...

.PHONY: fmt
fmt: ## gofmt + goimports
	gofmt -w -s .
	@command -v goimports >/dev/null && goimports -w . || true

.PHONY: docker-build
docker-build: ## Build the Docker image
	docker build -t $(APP):latest .

.PHONY: docker-up
docker-up: ## docker compose up -d --build
	docker compose up -d --build

.PHONY: docker-down
docker-down: ## docker compose down
	docker compose down

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(BIN_DIR) $(COVER_OUT)

.DEFAULT_GOAL := help
