# Этап сборки
FROM golang:1.24.4 AS builder

WORKDIR /app

# Кэшируем зависимости
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -ldflags="-s -w" -o main ./cmd/url-shortener

# Этап запуска
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /app/main .

# Порт, который слушает приложение
EXPOSE 8080

CMD ["./main"]
