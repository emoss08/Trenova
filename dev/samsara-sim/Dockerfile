FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY shared/go.mod shared/go.sum ./shared/
COPY dev/samsara-sim/go.mod dev/samsara-sim/go.sum ./dev/samsara-sim/
WORKDIR /src/dev/samsara-sim
RUN GOWORK=off go mod download

WORKDIR /src
COPY shared ./shared
COPY dev/samsara-sim ./dev/samsara-sim
WORKDIR /src/dev/samsara-sim
RUN GOWORK=off CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/samsara-sim ./cmd/samsara-sim

FROM alpine:3.21

RUN addgroup -S sim && adduser -S sim -G sim

WORKDIR /app

COPY --from=builder /out/samsara-sim /app/samsara-sim
COPY dev/samsara-sim/config /app/config
RUN cp /app/config/config.example.yaml /app/config/config.yaml \
    && mkdir -p /app/data \
    && chown -R sim:sim /app

USER sim

EXPOSE 8091

ENTRYPOINT ["/app/samsara-sim", "-config", "/app/config/config.yaml"]
