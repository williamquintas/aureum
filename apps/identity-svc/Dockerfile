FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /app/bin/server ./cmd/server

FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/bin/server ./server
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080 9090

ENTRYPOINT ["./server"]
