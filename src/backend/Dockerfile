# Tahap 1: Build aplikasi Go
FROM golang:1.24-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY *.go ./
# JANGAN salin direktori 'data' dari host ke builder jika Anda ingin dibuat dari nol oleh skrip
# COPY data ./data/ # <-- Mungkin ini bisa dikomentari jika scrapeonly membuat semuanya

# Jalankan main.go dengan flag -scrapeonly untuk hanya melakukan scraping dan filter
RUN go run . -scrapeonly
# Kita tambahkan ini untuk melihat apakah direktori data dibuat dan apa isinya
RUN echo "Isi direktori /app setelah scrapeonly:" && ls -la /app
RUN echo "Isi direktori /app/data setelah scrapeonly:" && ls -la /app/data || echo "/app/data tidak ditemukan atau kosong"

# Kompilasi aplikasi Go
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o /app/main_backend .

# Tahap 2: Siapkan image runtime yang ringan
FROM alpine:latest
WORKDIR /app

# Salin direktori 'data' yang sudah dibuat di builder
COPY --from=builder /app/data ./data/

COPY --from=builder /app/main_backend .
EXPOSE 8080
CMD ["./main_backend"]