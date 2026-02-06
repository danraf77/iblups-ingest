FROM golang:1.21-alpine

# Instalamos FFmpeg
RUN apk add --no-cache ffmpeg

WORKDIR /app

# Copiamos solo go.mod primero
COPY go.mod ./

# Descargamos dependencias (esto generará go.sum automáticamente)
RUN go mod download

# Copiamos el resto del código
COPY . .

# Compilamos
RUN go build -o main .

# Creamos directorio para thumbnails
RUN mkdir -p /app/thumbnails

EXPOSE 3000

CMD ["./main"]