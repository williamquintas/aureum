GO ?= go

.PHONY: build lint test/unit test/integration migrate/up migrate/down gen docker

build:
	$(GO) build -o bin/server ./cmd/server

lint:
	golangci-lint run

test/unit:
	$(GO) test -short -race ./...

test/integration:
	$(GO) test -tags=integration -race ./...

migrate/up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate/down:
	migrate -path migrations -database "$(DATABASE_URL)" down

dev/run:
	@echo "Starting identity-svc locally..."
	$(GO) run ./cmd/server

gen:
	@echo "TODO: generate proto code"

docker:
	docker build -t aureum/identity-svc:latest .
