##
## Copyright 2023-2025 Eric Moss
## Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
## Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md##

FROM traefik:v3.2

# Create necessary directories
RUN mkdir -p /etc/traefik/dynamic /letsencrypt /var/log/traefik

# Copy configuration files
COPY traefik/traefik.yml /etc/traefik/traefik.yml
COPY traefik/dynamic.yml /etc/traefik/dynamic/dynamic.yml

# Set proper permissions for Let's Encrypt storage
RUN touch /letsencrypt/acme.json && chmod 600 /letsencrypt/acme.json

# Health check
HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=3 \
    CMD traefik healthcheck

EXPOSE 80 443 8082

# Run as non-root user (Traefik image already includes this user)
USER traefik

ENTRYPOINT ["traefik"]