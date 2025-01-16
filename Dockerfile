FROM golang:1.23-alpine AS builder

# Install necessary packages and librdkafka
RUN apk add --update --no-cache alpine-sdk bash ca-certificates \
    libressl \
    tar \
    git openssh openssl yajl-dev zlib-dev gcc cyrus-sasl-dev openssl-dev build-base coreutils librdkafka-dev pkgconf tzdata

# Set the working directory
WORKDIR /app

# Environment variables for Go build
ENV GOOS=linux
ENV GOARCH=amd64
ENV HATCHET_CLIENT_TLS_STRATEGY=none

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod tidy
RUN go mod download

# Copy the entire source code
COPY . .

# Build the Go application
RUN go build -o main .

FROM alpine:latest

# Install necessary packages for running the Go application
RUN apk --no-cache add ca-certificates librdkafka tzdata bash

# Set the working directory
WORKDIR /root/

# Copy the built binary and necessary directories from the builder stage
COPY --from=builder /app/main .
COPY --from=builder /app/migrate/migrations /root/migrate/migrations
COPY --from=builder /app/testdata/fixtures /root/testdata/fixtures
COPY --from=builder /app/docs /root/docs
COPY --from=builder /app/private_key.pem /root/private_key.pem
COPY --from=builder /app/public_key.pem /root/public_key.pem
COPY --from=builder /app/config.prod.yaml /root/config.prod.yaml
COPY --from=builder /app/model.conf /root/model.conf

# Copy the entrypoint script
COPY bin/entrypoint.sh /root/entrypoint.sh

# Create a simple wait-for script
RUN echo '#!/bin/sh\n\
    host="$1"\n\
    port="$2"\n\
    timeout="$3"\n\
    \n\
    start_time=$(date +%s)\n\
    \n\
    until nc -z "$host" "$port" 2>/dev/null\n\
    do\n\
    current_time=$(date +%s)\n\
    elapsed_time=$((current_time - start_time))\n\
    \n\
    if [ "$elapsed_time" -ge "$timeout" ]; then\n\
    echo "Timeout reached. Exit."\n\
    exit 1\n\
    fi\n\
    \n\
    echo "Waiting for $host:$port..."\n\
    sleep 1\n\
    done\n\
    \n\
    echo "$host:$port is available"' > /usr/local/bin/wait-for

RUN chmod +x /usr/local/bin/wait-for
RUN chmod +x /root/entrypoint.sh

# Modify the entrypoint script to use the new wait-for script
RUN sed -i 's|/usr/local/bin/wait-for-it|/usr/local/bin/wait-for|g' /root/entrypoint.sh

# Expose the application port
EXPOSE 3001

# Set the entrypoint
ENTRYPOINT ["/bin/sh", "/root/entrypoint.sh"]

# Command to run the executable with arguments (this will be passed to the entrypoint)
CMD ["./main", "serve", "--env", "prod"]
