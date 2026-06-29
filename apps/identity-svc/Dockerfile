FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY apps/identity-svc/ apps/identity-svc/
COPY pkg/ pkg/
COPY proto/ proto/

WORKDIR /build/apps/identity-svc
RUN go mod download
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o /app/bin/server ./cmd/server

FROM alpine:3.24

WORKDIR /app

COPY --from=builder /app/bin/server ./server
COPY --from=builder /build/apps/identity-svc/migrations ./migrations

EXPOSE 8080 9090

ENTRYPOINT ["./server"]
