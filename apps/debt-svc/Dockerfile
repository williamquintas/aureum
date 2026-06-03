FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY apps/debt-svc/ apps/debt-svc/
COPY pkg/ pkg/
COPY proto/ proto/

WORKDIR /build/apps/debt-svc
RUN go mod download
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o /app/bin/server ./cmd/server

FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/bin/server ./server
COPY --from=builder /build/apps/debt-svc/migrations ./migrations

EXPOSE 50055

ENTRYPOINT ["./server"]
