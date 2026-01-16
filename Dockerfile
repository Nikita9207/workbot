# Multi-stage build для минимального размера образа
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Устанавливаем зависимости для сборки
RUN apk add --no-cache git ca-certificates tzdata

# Копируем go.mod и go.sum для кэширования зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем бинарник
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /workbot ./cmd/main.go

# Финальный образ
FROM alpine:3.19

WORKDIR /app

# Устанавливаем необходимые пакеты
RUN apk add --no-cache ca-certificates tzdata

# Копируем бинарник из builder
COPY --from=builder /workbot /app/workbot

# Копируем миграции
COPY --from=builder /app/migrations /app/migrations

# Создаём директорию для данных
RUN mkdir -p /data/Клиенты

# Устанавливаем timezone
ENV TZ=Europe/Moscow

# Запускаем бота
CMD ["/app/workbot"]
