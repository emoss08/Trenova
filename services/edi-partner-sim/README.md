# EDI Partner Simulator

A standalone AS2 trading-partner simulator plus an end-to-end runner for exercising
Trenova's EDI stack against a real partner over the wire.

The simulator behaves like an external trading partner:

- Receives AS2 documents from Trenova (decrypts, verifies signatures, returns a
  signed synchronous MDN).
- Auto-generates and delivers a `997` functional acknowledgment back to Trenova for
  every received transaction set (so ack reconciliation can be tested).
- Sends inbound `204` load tenders (and arbitrary X12 payloads) into Trenova's AS2
  inbound receiver on demand.
- Exposes a small control API to configure certificates/identity and inspect what it
  has received and sent.

## Binaries

- `cmd/edi-partner-sim` — the long-running simulator HTTP server.
- `cmd/edi-e2e` — a scripted end-to-end runner that provisions a partner, an AS2
  communication profile, and an outbound 204 document profile through the Trenova
  REST API, generates and delivers a 204, and asserts the full round trip (delivery,
  MDN, 997 reconciliation, the unsigned-inbound security gate, and an inbound 204
  tender).

## Running the flow

Requires the Trenova API + Temporal worker running (`task run-watch` and
`task worker` in `services/tms`), the local infra up (`task docker-up`), migrations
applied (`task db-migrate`), and the database seeded (`task db-seed`) so a shipment
exists.

```bash
# terminal 1 — start the simulator
task sim              # from services/edi-partner-sim, or:
go run ./cmd/edi-partner-sim -listen :9210

# terminal 2 — run the end-to-end scenario
task e2e              # or:
go run ./cmd/edi-e2e
```

The runner generates fresh AS2 certificates and a unique AS2 identifier per run, so it
can be run repeatedly without colliding with earlier partners.

## Simulator control API

| Method | Path | Purpose |
| --- | --- | --- |
| `GET`  | `/control/identity` | AS2 identity + the simulator's public certificate (PEM) |
| `POST` | `/control/partner` | Set the Trenova certificate, inbound URL, auto-ack, and per-run AS2 identity (`as2Id`/`remoteAs2Id`) |
| `GET`  | `/control/received` | Documents received from Trenova (with parsed X12 envelope, signed/encrypted flags, ack status) |
| `GET`  | `/control/sent` | Documents sent to Trenova and the resolved MDN status |
| `POST` | `/control/send` | Send a raw X12 payload to Trenova |
| `POST` | `/control/send-tender` | Send an inbound 204 load tender to Trenova |
| `POST` | `/control/reset` | Clear received/sent history |
| `POST` | `/as2` | The AS2 receiver Trenova posts outbound documents to |

## Flags

`cmd/edi-partner-sim`: `-listen`, `-as2-id`, `-remote-as2-id`, `-trenova-inbound`,
`-auto-ack`.

`cmd/edi-e2e`: `-api`, `-sim`, `-email`, `-password`, `-inbound`.
