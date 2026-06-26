FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY apps/budget-svc/ apps/budget-svc/
COPY pkg/ pkg/
COPY proto/ proto/

WORKDIR /build/apps/budget-svc
RUN go mod download
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o /app/bin/server ./cmd/server

FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/bin/server ./server
COPY --from=builder /build/apps/budget-svc/migrations ./migrations

EXPOSE 50055

ENTRYPOINT ["./server"]
