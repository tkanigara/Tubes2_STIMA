# Tahap 1: Build aplikasi frontend ...
FROM node:20-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .

# Set URL API untuk produksi (Railway)
ARG API_URL=""
ENV VITE_API_BASE_URL=$API_URL
RUN echo "Building with API URL: $VITE_API_BASE_URL"

# Build aplikasi
RUN npm run build

# Tahap 2: Sajikan aplikasi menggunakan server web ringan (Nginx)
FROM nginx:1.27-alpine

COPY --from=builder /app/dist /usr/share/nginx/html

# Salin file konfigurasi Nginx kustom untuk produksi
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Tambahkan health check untuk platform deployment
# HEALTHCHECK --interval=30s --timeout=3s CMD wget -qO- http://localhost/ || exit 1

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]