GO ?= go

.PHONY: build lint test/unit test/integration dev/run gen docker

build:
	$(GO) build -o bin/server ./cmd/server

lint:
	golangci-lint run

test/unit:
	$(GO) test -short -race ./...

test/integration:
	$(GO) test -tags=integration -race ./...

gen:
	$(GO) run github.com/99designs/gqlgen generate

dev/run:
	@echo "Starting graphql-bff locally..."
	$(GO) run ./cmd/server

docker:
	docker build -t aureum/graphql-bff:latest -f Dockerfile ../..
