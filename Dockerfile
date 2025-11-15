FROM golang:1.24-alpine

WORKDIR /app

# Копируем исходный код backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./

# Копируем frontend файлы внутрь контейнера
COPY frontend/ ./frontend/

# Собираем приложение
RUN go build -o main .

EXPOSE 8080

CMD ["./main"]