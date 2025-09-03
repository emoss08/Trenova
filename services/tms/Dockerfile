##
## Copyright 2023-2025 Eric Moss
## Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
## Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md##

FROM golang:1.25.0-bookworm AS builder

# Install necessary packages
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    ca-certificates \
    git \
    && rm -rf /var/lib/apt/lists/*

# Set the working directory
WORKDIR /app

# Environment variables for Go build
ENV GOOS=linux
ENV GOARCH=amd64
ENV APP_ENV=production

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the binary with optimizations
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o apiserver cmd/api/main.go

# Final stage - minimal image
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Set the environment variable
ENV APP_ENV=production
ENV TRENOVA_APP_ENVIRONMENT=production
ENV TRENOVA_CONFIG_PATH=/app/config/production/config.production.yaml

WORKDIR /app

# Copy binary and config from builder
COPY --from=builder /app/apiserver .
COPY config/production/config.production.yaml ./config/production/

# Create logs directory
RUN mkdir -p /app/logs

# Make sure the binary is executable
RUN chmod +x /app/apiserver

# Run as non-root user
RUN adduser -D -H -s /bin/sh appuser && chown -R appuser:appuser /app
USER appuser

# Health check
HEALTHCHECK --interval=10s --timeout=3s --start-period=30s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:3001/health || exit 1

EXPOSE 3001

ENTRYPOINT ["/app/apiserver", "serve"]