FROM golang:1.21-alpine

# Instalamos FFmpeg
RUN apk add --no-cache ffmpeg

WORKDIR /app

# Copiamos todo el código primero
COPY . .

# Generamos go.sum y descargamos dependencias
RUN go mod tidy

# Compilamos
RUN go build -o main .

# Creamos directorio para thumbnails
RUN mkdir -p /app/thumbnails

EXPOSE 3000

CMD ["./main"]