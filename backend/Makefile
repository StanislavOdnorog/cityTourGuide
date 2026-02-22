.PHONY: lint fmt test build clean

BINARY_DIR := bin
GOPATH := $(shell go env GOPATH)
GOLANGCI_LINT := $(GOPATH)/bin/golangci-lint

lint:
	$(GOLANGCI_LINT) run ./...

fmt:
	gofmt -w .
	$(GOPATH)/bin/goimports -w .

test:
	go test -race -cover ./...

build: $(BINARY_DIR)/api $(BINARY_DIR)/worker

$(BINARY_DIR)/api:
	@mkdir -p $(BINARY_DIR)
	go build -o $(BINARY_DIR)/api ./cmd/api

$(BINARY_DIR)/worker:
	@mkdir -p $(BINARY_DIR)
	go build -o $(BINARY_DIR)/worker ./cmd/worker

clean:
	rm -rf $(BINARY_DIR)
