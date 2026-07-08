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
- Runs an embedded **SSH/SFTP mailbox** (host-key pinned, password auth) so Trenova
  can push outbound files to it and poll inbound files from it — the mailbox pickup
  path, not just the AS2 HTTP push path.
- Persists its AS2 keypair and SFTP host key to disk (`-identity-dir`) so restarts
  keep the same identity and don't invalidate communication profiles created earlier.
- Exposes a small control API to configure certificates/identity, drop/inspect SFTP
  mailbox files, and inspect what it has received and sent.

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
go run ./cmd/edi-partner-sim -listen :9210 -sftp-listen :9222 -identity-dir ./.sim-identity

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
| `GET`  | `/control/sftp` | SFTP host/port/credentials, host key (for `knownHostKey`), and the mailbox directory paths |
| `POST` | `/control/sftp/drop` | Drop a file into the SFTP inbound directory for Trenova to poll |
| `GET`  | `/control/sftp/inbound` | List the SFTP inbound + archive directories |
| `GET`  | `/control/sftp/outbound` | List files Trenova pushed to the SFTP outbound directory |

### Where the SFTP files land

With `task sim` the mailbox is a stable `./.sim-sftp/` directory
(`inbound/`, `outbound/`, `archive/`), so you can watch it directly:

```bash
ls -la ./.sim-sftp/outbound      # 204s Trenova delivered over SFTP
cat ./.sim-sftp/outbound/*.x12   # the raw X12
ls -la ./.sim-sftp/archive       # inbound files Trenova polled and archived
```

Or over the control API without touching disk:

```bash
curl -s localhost:9210/control/sftp/outbound | jq
curl -s localhost:9210/control/sftp/inbound  | jq
```

## Flags

`cmd/edi-partner-sim`: `-listen`, `-as2-id`, `-remote-as2-id`, `-trenova-inbound`,
`-auto-ack`, `-identity-dir` (persist keys), `-sftp-listen`, `-sftp-user`,
`-sftp-password`, `-sftp-root`.

`cmd/edi-e2e`: `-api`, `-sim`, `-email`, `-password`, `-inbound`.
