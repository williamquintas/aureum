FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY apps/graphql-bff/ apps/graphql-bff/
COPY pkg/ pkg/
COPY proto/ proto/

WORKDIR /build/apps/graphql-bff
RUN go mod download
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o /app/bin/server ./cmd/server

FROM alpine:3.24

WORKDIR /app

COPY --from=builder /app/bin/server ./server

EXPOSE 8082

ENTRYPOINT ["./server"]
