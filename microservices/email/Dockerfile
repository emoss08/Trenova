FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install required dependencies for CGO
RUN apk add --no-cache gcc musl-dev

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -o email-service .

# Create a minimal production image
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata bash

# Copy the binary from the builder stage
COPY --from=builder /app/email-service .

# Create directories
RUN mkdir -p /app/templates /app/scripts

# Copy templates and scripts
COPY templates/ /app/templates/
COPY scripts/ /app/scripts/

# Make scripts executable
RUN chmod +x /app/scripts/*.sh

# Expose the service port
EXPOSE 8082

# Run the service
CMD ["./email-service"] 