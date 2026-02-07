FROM golang:1.21-alpine

RUN apk add --no-cache ffmpeg

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# ✅ Compilar desde cmd/server
RUN go build -o main ./cmd/server

RUN mkdir -p /app/thumbnails

EXPOSE 3000

CMD ["./main"]