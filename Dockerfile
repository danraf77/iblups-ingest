FROM golang:1.21-alpine

# Instalamos FFmpeg para generar los thumbnails
RUN apk add --no-cache ffmpeg git

WORKDIR /app
COPY . .
RUN go mod init srs-backend && go get github.com/supabase-community/supabase-go && go mod tidy
RUN go build -o main .

# Puerto del backend
EXPOSE 3000

CMD ["./main"]