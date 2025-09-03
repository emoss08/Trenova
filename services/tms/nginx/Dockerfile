##
## Copyright 2023-2025 Eric Moss
## Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
## Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md##

FROM nginx:alpine

# Install additional packages
RUN apk add --no-cache \
    curl \
    openssl \
    ca-certificates \
    tzdata

# Remove default nginx config
RUN rm -f /etc/nginx/conf.d/default.conf

# Create necessary directories
RUN mkdir -p /var/log/nginx /var/cache/nginx /etc/nginx/ssl /var/www/certbot

# Copy nginx configuration
COPY nginx/nginx.conf /etc/nginx/nginx.conf
COPY nginx/locations.conf /etc/nginx/conf.d/locations.conf

# Create a self-signed certificate for local development (will be replaced in production)
RUN openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
    -keyout /etc/nginx/ssl/key.pem \
    -out /etc/nginx/ssl/cert.pem \
    -subj "/C=US/ST=State/L=City/O=Organization/CN=localhost"

# Set proper permissions
RUN chown -R nginx:nginx /var/log/nginx /var/cache/nginx

# Health check
HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=3 \
    CMD curl -f http://localhost/health || exit 1

EXPOSE 80 443

CMD ["nginx", "-g", "daemon off;"]