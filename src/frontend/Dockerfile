# Tahap 1: Build aplikasi frontend ...
FROM node:20-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
# Jika Anda menggunakan VITE_API_BASE_URL, set ke "" atau "/"
ENV VITE_API_BASE_URL="" 
RUN npm run build

# Tahap 2: Sajikan aplikasi menggunakan server web ringan (Nginx)
FROM nginx:1.27-alpine

COPY --from=builder /app/dist /usr/share/nginx/html

# ---- TAMBAHKAN BARIS INI ----
# Salin file konfigurasi Nginx kustom Anda
COPY nginx.conf /etc/nginx/conf.d/default.conf
# -----------------------------

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]