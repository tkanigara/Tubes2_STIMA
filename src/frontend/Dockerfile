# Tahap 1: Build aplikasi frontend
FROM node:20-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .

# Atur variabel environment untuk build
# Default ke Railway URL, tapi bisa dioveride dengan build arg
ARG VITE_API_BASE_URL="https://cozy-contentment-production-7b75.up.railway.app"
ARG DEPLOYMENT_ENV="production"
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}
ENV DEPLOYMENT_ENV=${DEPLOYMENT_ENV}

# Build aplikasi
RUN npm run build

# (Debug opsional) Cek isi folder dist
RUN ls -la /app/dist

# Tahap 2: Sajikan hasil build dengan Nginx
FROM nginx:1.27-alpine

# Copy hasil build ke direktori default Nginx
COPY --from=builder /app/dist /usr/share/nginx/html

# Copy konfigurasi nginx custom berdasarkan environment
COPY nginx.conf /etc/nginx/conf.d/default.conf.local
COPY nginx.railway.conf /etc/nginx/conf.d/default.conf.railway

# Script untuk memilih konfigurasi Nginx yang sesuai
RUN echo '#!/bin/sh' > /docker-entrypoint.sh && \
    echo 'if [ "$DEPLOYMENT_ENV" = "production" ]; then' >> /docker-entrypoint.sh && \
    echo '    echo "Using Railway configuration"' >> /docker-entrypoint.sh && \
    echo '    cp /etc/nginx/conf.d/default.conf.railway /etc/nginx/conf.d/default.conf' >> /docker-entrypoint.sh && \
    echo 'else' >> /docker-entrypoint.sh && \
    echo '    echo "Using local configuration"' >> /docker-entrypoint.sh && \
    echo '    cp /etc/nginx/conf.d/default.conf.local /etc/nginx/conf.d/default.conf' >> /docker-entrypoint.sh && \
    echo 'fi' >> /docker-entrypoint.sh && \
    echo 'nginx -g "daemon off;"' >> /docker-entrypoint.sh && \
    chmod +x /docker-entrypoint.sh

EXPOSE 80
CMD ["/docker-entrypoint.sh"]
