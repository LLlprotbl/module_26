# ==================== Stage 1: Build ====================
FROM golang:1.21-alpine AS builder

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем зависимости (для лучшего кэширования)
COPY go.mod ./
RUN go mod download

# Копируем исходный код
COPY . .

# Компилируем статический бинарник (без зависимостей от glibc)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o pipeline main.go

# ==================== Stage 2: Runtime ====================
FROM alpine:latest 

# Метаданные
LABEL version="1.0.0" \
      maintainer="Test Student<test@test.ru>"

# Копируем бинарник из builder-стадии
COPY --from=builder /app/pipeline /pipeline

# Запускаем приложение (JSON-формат!)
ENTRYPOINT ["/pipeline"]