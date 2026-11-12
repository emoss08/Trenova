# Mail simulator

Stands in for Resend on a development machine. It answers the send call, keeps the message and
shows it at <http://localhost:8092/>. Nothing is delivered to anyone.

Without it no write that ends in an email can finish in development: a plan's "email the
customer" step is refused, and the steps after it never run.

## Run it

```sh
docker compose -f dev/compose.yml --profile mail-sim up -d mail-sim
```

or, without Docker, `cd dev/mail-sim && GOWORK=off go run ./cmd/mail-sim`.

## Point Trenova at it

- `services/tms/config/config.yaml` needs `integrations.allowPrivateNetworks: true`, because the
  simulator listens on localhost (`config.example.yaml` has it).
- The `outbound_channels` seed adds a Resend integration whose base URL is the simulator and
  assigns the seeded email profile to every purpose. Apply it with
  `./build/trenova-devdb seed apply -force outbound_channels`.

`GET /_sim/messages` returns what was kept as JSON; `GET /_sim/health` answers `ok`.
