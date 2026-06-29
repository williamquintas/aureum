FROM golang:1.26-alpine AS builder

WORKDIR /build

COPY apps/transaction-svc/ apps/transaction-svc/
COPY pkg/ pkg/
COPY proto/ proto/

WORKDIR /build/apps/transaction-svc
RUN go mod download
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o /app/bin/server ./cmd/server

FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/bin/server ./server
COPY --from=builder /build/apps/transaction-svc/migrations ./migrations

EXPOSE 50054

ENTRYPOINT ["./server"]
