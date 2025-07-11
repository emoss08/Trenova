FROM golang:1.24-bookworm AS builder

# Install necessary packages
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    bash \
    ca-certificates \
    libssl-dev \
    tar \
    git \
    openssh-client \
    openssl \
    libyajl-dev \
    zlib1g-dev \
    libsasl2-dev \
    pkg-config \
    tzdata \
    libffi-dev \
    && rm -rf /var/lib/apt/lists/*

# Set the working directory
WORKDIR /app

# Environment variables for Go build
ENV GOOS=linux
ENV GOARCH=amd64
ENV APP_ENV=production


# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the entire source code
COPY . .

# Build the binary
RUN go build -o apiserver cmd/api/main.go

FROM debian:bookworm-slim AS final

# Install runtime dependencies and PostgreSQL client
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    libffi8 \
    wget \
    gnupg \
    lsb-release \
    && wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | apt-key add - \
    && echo "deb http://apt.postgresql.org/pub/repos/apt/ $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends postgresql-client-17 \
    && apt-get remove -y wget gnupg lsb-release \
    && apt-get autoremove -y \
    && rm -rf /var/lib/apt/lists/*


# Set the environment variable
ENV APP_ENV=production

WORKDIR /app

# Copy binary and config files from the builder stage
COPY --from=builder /app/apiserver .
COPY --from=builder /app/config/production/config.production.yaml ./config/production/config.production.yaml
# COPY --from=builder /app/config/development/config.development.yaml ./config/development/config.development.yaml

# Make sure the binary is executable
RUN chmod +x /app/apiserver

# Command to run when starting the container
ENTRYPOINT ["/app/apiserver", "serve"]