# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the service
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o edi-service ./cmd/edi-service

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/edi-service .

# Copy test data and schemas if needed
COPY --from=builder /app/testdata ./testdata
COPY --from=builder /app/schemas ./schemas

EXPOSE 8080

CMD ["./edi-service"]