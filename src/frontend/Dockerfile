# Tahap 1: Build aplikasi frontend
FROM node:20-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .

# Atur variabel environment untuk build
# Default ke localhost untuk development (akan dioveride dengan build arg)
ARG VITE_API_BASE_URL="http://localhost:8080"
ARG DEPLOYMENT_ENV="local"
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}
ENV DEPLOYMENT_ENV=${DEPLOYMENT_ENV}

# Debug - tampilkan nilai variabel lingkungan
RUN echo "Building with VITE_API_BASE_URL=${VITE_API_BASE_URL}"
RUN echo "Building with DEPLOYMENT_ENV=${DEPLOYMENT_ENV}"

# Create .env file for Vite
RUN echo "VITE_API_BASE_URL=${VITE_API_BASE_URL}" > .env
RUN echo "DEPLOYMENT_ENV=${DEPLOYMENT_ENV}" >> .env

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
    echo '' >> /docker-entrypoint.sh && \
    echo '# Replace backend_upstream in configuration if BACKEND_HOST is set' >> /docker-entrypoint.sh && \
    echo 'if [ -n "$BACKEND_HOST" ]; then' >> /docker-entrypoint.sh && \
    echo '    echo "Setting backend upstream to $BACKEND_HOST"' >> /docker-entrypoint.sh && \
    echo '    sed -i "s/backend:8080/$BACKEND_HOST/g" /etc/nginx/conf.d/default.conf' >> /docker-entrypoint.sh && \
    echo 'fi' >> /docker-entrypoint.sh && \
    echo '' >> /docker-entrypoint.sh && \
    echo 'nginx -g "daemon off;"' >> /docker-entrypoint.sh && \
    chmod +x /docker-entrypoint.sh

EXPOSE 80
CMD ["/docker-entrypoint.sh"]
