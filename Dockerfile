FROM golang:1.21-alpine

# Instalamos FFmpeg
RUN apk add --no-cache ffmpeg

WORKDIR /app

# Copiamos archivos de dependencias primero (mejor caching de Docker)
COPY go.mod go.sum ./
RUN go mod download

# Copiamos el código fuente
COPY . .

# Compilamos la aplicación
RUN go build -o main .

# Creamos directorio para thumbnails
RUN mkdir -p /app/thumbnails

EXPOSE 3000

CMD ["./main"]