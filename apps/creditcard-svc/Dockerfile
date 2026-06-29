FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY apps/creditcard-svc/ apps/creditcard-svc/
COPY pkg/ pkg/
COPY proto/ proto/

WORKDIR /build/apps/creditcard-svc
RUN go mod download
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o /app/bin/server ./cmd/server

FROM alpine:3.24

WORKDIR /app

COPY --from=builder /app/bin/server ./server
COPY --from=builder /build/apps/creditcard-svc/migrations ./migrations

EXPOSE 50055

ENTRYPOINT ["./server"]
