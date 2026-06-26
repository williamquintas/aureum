GO ?= go

.PHONY: build lint test/unit test/integration migrate/up migrate/down dev/run docker

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
	@echo "Starting report-svc locally..."
	$(GO) run ./cmd/server

docker:
	docker build -t aureum/report-svc:latest -f Dockerfile ../..
