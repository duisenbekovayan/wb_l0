# Dockerfile
FROM golang:1.24
WORKDIR /app

# зависимости
COPY go.mod go.sum ./
RUN go mod download

# все исходники
COPY . .

# билд
RUN go build -o app ./cmd/app

# запуск
CMD ["./app"]
