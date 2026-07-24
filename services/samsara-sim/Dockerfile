FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/samsara-sim ./cmd/samsara-sim

FROM alpine:3.21

RUN addgroup -S sim && adduser -S sim -G sim

WORKDIR /app

COPY --from=builder /out/samsara-sim /app/samsara-sim
COPY config /app/config
RUN cp /app/config/config.example.yaml /app/config/config.yaml && chown -R sim:sim /app

USER sim

EXPOSE 8091

ENTRYPOINT ["/app/samsara-sim", "-config", "/app/config/config.yaml"]
