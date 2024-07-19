FROM golang:1.22-alpine AS builder

# Install necessary packages and librdkafka
RUN apk add --update --no-cache alpine-sdk bash ca-certificates \
    libressl \
    tar \
    git openssh openssl yajl-dev zlib-dev gcc cyrus-sasl-dev openssl-dev build-base coreutils librdkafka-dev pkgconf musl-dev

# Set the working directory
WORKDIR /app

# Environment variables for Go build
ENV GOOS=linux
ENV GOARCH=amd64

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Explicitly download the Kafka package with musl tag
RUN go get -tags musl -u github.com/confluentinc/confluent-kafka-go/v2/kafka

# Download all dependencies
RUN go mod tidy
RUN go mod download

# Copy the entire source code
COPY . .

# Debugging: List the contents of the /app directory to verify source code is copied
RUN ls -al /app

# Build the Go application with musl tag
RUN go build -tags musl -o main .

# Use a minimal base image for the final stage
FROM alpine:latest

# Install necessary packages for running the Go application
RUN apk --no-cache add ca-certificates librdkafka

# Set the working directory
WORKDIR /root/

# Copy the built binary and necessary directories from the builder stage
COPY --from=builder /app/main .
COPY --from=builder /app/migrate/migrations /root/migrate/migrations
COPY --from=builder /app/private_key.pem /root/private_key.pem
COPY --from=builder /app/public_key.pem /root/public_key.pem
COPY --from=builder /app/config.prod.yaml /root/config.prod.yaml
COPY --from=builder /app/model.conf /root/model.conf

# Expose the application port
EXPOSE 3001

# Command to run the executable with arguments
CMD ["./main", "serve", "--env", "prod"]