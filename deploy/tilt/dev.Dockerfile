FROM golang:1.25-alpine

RUN go install github.com/air-verse/air@latest

WORKDIR /app
COPY go.work go.work.sum ./
COPY apps/ apps/
COPY pkg/ pkg/
COPY proto/ proto/

WORKDIR /app/apps/identity-svc
RUN go mod download

EXPOSE 8080 9090

CMD ["air", "-c", ".air.toml"]
