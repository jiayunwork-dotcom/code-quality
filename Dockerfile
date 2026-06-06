FROM golang:1.21-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /app/code-quality .

FROM alpine:latest

RUN apk add --no-cache git ca-certificates

COPY --from=builder /app/code-quality /usr/local/bin/code-quality

WORKDIR /workspace

ENTRYPOINT ["/usr/local/bin/code-quality"]
CMD ["--help"]
