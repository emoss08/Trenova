FROM golang:1.24rc3-alpine AS builder

# Install necessary packages and librdkafka
RUN apk add --update --no-cache alpine-sdk bash ca-certificates \
    libressl \
    tar \
    git openssh openssl yajl-dev zlib-dev gcc cyrus-sasl-dev openssl-dev build-base coreutils pkgconf tzdata

# Set the working directory
WORKDIR /app

# Environment variables for Go build
ENV GOOS=linux
ENV GOARCH=amd64
ENV HATCHET_CLIENT_TLS_STRATEGY=none
ENV APP_ENV=production


# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the entire source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 go build -o apiserver cmd/api/main.go

FROM alpine:latest

# Install CA certificates
RUN apk --no-cache add ca-certificates

# Set the environment variable
ENV HATCHET_CLIENT_TLS_STRATEGY=none
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