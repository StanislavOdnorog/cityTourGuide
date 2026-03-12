.PHONY: lint fmt test test-integration test-migrations build clean migrate-up migrate-down migrate-create seed demo-reset generate-api

BINARY_DIR := bin
GOPATH := $(shell go env GOPATH)
GOLANGCI_LINT := $(GOPATH)/bin/golangci-lint
MIGRATE := $(GOPATH)/bin/migrate
MIGRATIONS_DIR := migrations
DATABASE_URL ?= postgres://citystories:citystories_secret@localhost:5433/citystories?sslmode=disable

lint:
	$(GOLANGCI_LINT) run ./...

fmt:
	gofmt -w .
	$(GOPATH)/bin/goimports -w .

TEST_DATABASE_URL ?= postgres://citystories:citystories_secret@localhost:5433/citystories_test?sslmode=disable

test:
	go test -race -cover ./...

test-integration:
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test -race -tags=integration -count=1 ./...

build: $(BINARY_DIR)/api $(BINARY_DIR)/worker

$(BINARY_DIR)/api:
	@mkdir -p $(BINARY_DIR)
	go build -o $(BINARY_DIR)/api ./cmd/api

$(BINARY_DIR)/worker:
	@mkdir -p $(BINARY_DIR)
	go build -o $(BINARY_DIR)/worker ./cmd/worker

clean:
	rm -rf $(BINARY_DIR)

migrate-up:
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" up

migrate-down:
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" down 1

migrate-create:
	@read -p "Migration name: " name; \
	$(MIGRATE) create -ext sql -dir $(MIGRATIONS_DIR) -seq $$name

test-migrations:
	bash scripts/test-migrations.sh

seed:
	go run ./scripts/seed/

demo-reset:
	go run ./scripts/reset/

generate-api:
	bash scripts/generate-clients.sh
