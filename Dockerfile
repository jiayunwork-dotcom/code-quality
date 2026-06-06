FROM golang:1.21-alpine AS builder

WORKDIR /app

ENV GOPROXY=https://goproxy.cn,direct

COPY . .

RUN go mod tidy && go mod download

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /app/code-quality .

FROM alpine:latest

COPY --from=builder /app/code-quality /usr/local/bin/code-quality

WORKDIR /workspace

ENTRYPOINT ["/usr/local/bin/code-quality"]
CMD ["--help"]
