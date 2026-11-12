FROM golang:1.27.1-alpine AS builder

WORKDIR /src/dev/mail-sim
COPY dev/mail-sim ./
RUN GOWORK=off CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/mail-sim ./cmd/mail-sim

FROM alpine:3.21

RUN addgroup -S sim && adduser -S sim -G sim
USER sim
COPY --from=builder /out/mail-sim /app/mail-sim

EXPOSE 8092

ENTRYPOINT ["/app/mail-sim"]
