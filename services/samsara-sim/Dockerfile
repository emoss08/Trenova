FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY shared/go.mod shared/go.sum ./shared/
COPY services/samsara-sim/go.mod services/samsara-sim/go.sum ./services/samsara-sim/
WORKDIR /src/services/samsara-sim
RUN GOWORK=off go mod download

WORKDIR /src
COPY shared ./shared
COPY services/samsara-sim ./services/samsara-sim
WORKDIR /src/services/samsara-sim
RUN GOWORK=off CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/samsara-sim ./cmd/samsara-sim

FROM alpine:3.21

RUN addgroup -S sim && adduser -S sim -G sim

WORKDIR /app

COPY --from=builder /out/samsara-sim /app/samsara-sim
COPY services/samsara-sim/config /app/config
RUN cp /app/config/config.example.yaml /app/config/config.yaml && chown -R sim:sim /app

USER sim

EXPOSE 8091

ENTRYPOINT ["/app/samsara-sim", "-config", "/app/config/config.yaml"]
