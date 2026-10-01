# Idempotency keys

How a retried write returns the first result instead of being applied twice, what
the server keeps and for how long, and what to do when you add a write that should
be safe to retry. A client opts in per request by sending an `Idempotency-Key`
header. A request without one behaves exactly as it did before.

## The shape of it

```
POST/PUT/PATCH/DELETE + Idempotency-Key
        │  (after auth, rate limit, CSRF, password-change and control-plane checks)
        ▼
IdempotencyMiddleware ── Claim(scoped key, fingerprint) ──▶ Redis  idempotency:v1:{key}
        │                                                    (one hash per key)
        ├─ claimed ─────▶ handler runs ─▶ response captured ─▶ Complete (kept 24h)
        │                                                   └▶ Release (5xx, 408, 429,
        │                                                       transient GraphQL error)
        ├─ completed, same request ─▶ stored response replayed, Idempotent-Replayed: true
        ├─ still running ───────────▶ 409 + Retry-After: 1
        └─ different request ───────▶ 400, "already used for a different request"
```

| Piece | Where |
|---|---|
| Header names, key validation, scoping, fingerprint, context carrier | `services/tms/pkg/idempotency` |
| Store port | `internal/core/ports/repositories/idempotency.go` |
| Redis store (Lua claim / complete / release) | `internal/infrastructure/redis/repositories/idempotency.go` |
| Middleware | `internal/api/middleware/idempotency.go`, registered in `Router.protectedGroup` |
| Shipment create backstop | `shipments.idempotency_key`, `shipmentservice/shipment_idempotency.go` |
| Configuration and defaults | `security.idempotency` in `config/config.example.yaml`, `config.IdempotencyConfig` |
| Browser | `@trenova/shared/lib/idempotency`, `apps/web/src/hooks/use-idempotency-key.ts` |

## What a key means

The header value is the client's own string: 1 to 255 printable ASCII characters with
no spaces. A UUID per logical write is the expected value. The server never stores it
as sent. It stores `sha256(organization, business unit, principal type, principal id,
key)`, so two callers who pick the same string, or one caller in two organizations,
never collide. A leaked key also cannot be used to read someone else's response.

A key names one request. That request is identified by a fingerprint:
`sha256(method, path and query, body)`. Sending the same key with a different body is
refused rather than replayed, because the caller has clearly mixed up two writes. To
compute the fingerprint the middleware reads the whole body, so a keyed request is
capped at `maxRequestBytes` (4 MiB by default). The handler then reads the same bytes
from a restored body.

## What is kept, and what is released

Each key's first request claims it for `lockTTL`. That defaults to 2 minutes and is
never shorter than `server.requestTimeout` + 5s, so a slow handler cannot lose its
claim mid-request. When the handler returns, the response is either kept or the key
is released:

- **Kept for `recordTTL` (24h):** any status below 500 except 408 and 429. This
  includes 4xx answers: a request that failed validation fails the same way when
  replayed, and the browser starts a new key after any answer (below).
- **Released:** 5xx, 408 and 429, and a handler panic. On `/graphql`, which answers
  resolver errors with HTTP 200, the key is also released when any error carries
  `SYSTEM_ERROR`, `TOO_MANY_REQUESTS` or `NOT_IMPLEMENTED`, or carries no code at all.
  A released key can be retried straight away and runs again. That is safe, because a
  write that fails this way has rolled back (shipment create is one transaction for
  exactly this reason).

A response is kept only up to `maxResponseBytes` (1 MiB). Above that, the key still
blocks a second write, but a repeat answers 409 ("already completed with status N")
instead of a replay. Only `Content-Type`, the status and the body are kept; the
replay adds `Idempotent-Replayed: true`, which CORS exposes to the browser.

The Redis record is a hash, and every write goes through a Lua script. Claim is
SET-if-absent, so two concurrent first requests cannot both win. Complete and Release
only act while the stored owner token is still this request's, so a request whose
claim expired cannot overwrite the one that took the key over.

## When Redis is down

The middleware fails open: if the claim errors, it logs the error and serves the
request without replay. It still puts the scoped key in the request context, so a
write with its own backstop stays exactly-once. Shipment create is one: the
GraphQL `createShipment` resolver and `POST /api/v1/shipments/` copy the key onto
`Shipment.IdempotencyKey` (`json:"-"`, never accepted from the body).
`shipmentservice.Create` then:

1. looks up a shipment already booked under that key and returns it, with its
   details, instead of creating another;
2. otherwise creates as normal, and the partial unique index
   `uq_shipments_idempotency_key (organization_id, business_unit_id, idempotency_key)`
   turns a raced second insert into a unique violation, which is answered with the
   shipment the winner booked.

The key is excluded from `Update` and from shipment copies, so an edit keeps the key
the shipment was booked under, and a duplicate is a new booking. The backstop has no
expiry: a key that booked a shipment books that shipment forever, including after it
is canceled.

## The browser

`requestGraphQL` takes an `idempotencyKey` option and sends it as the header.
`useIdempotencyKey()` returns a runner that holds one key per form until the server
settles it:

- the same payload resubmitted after no answer (offline, timeout) or after a 409
  "still running" reuses the key, so the server replays instead of writing twice;
- a success, any other server answer, or a changed payload starts a new key, so
  fixing a validation error and resubmitting is a new request.

Every shipment-create screen uses it: the shipment panel, the document draft review
and the rate confirmation import.

## Adding a retry-safe write

1. Have the client send a key. For a browser write, wrap the call in
   `useIdempotencyKey()` and pass the key through `requestGraphQL({ idempotencyKey })`
   or a REST `headers` option. An API caller sends the header itself.
2. That alone covers retries while Redis is up and for 24 hours. If a duplicate
   would be costly (money moves, a booking, a message sent to a partner), add a
   backstop the way shipment create does: a nullable `idempotency_key varchar(64)`
   column with a partial unique index, the key copied from `idempotency.KeyFrom(ctx)`
   in the handler or resolver (never in the service, because one request can create
   several records), a lookup before the write, and a unique-violation path that
   returns the existing record.
3. Make the write one transaction, so a failure that releases the key leaves nothing
   behind for the retry to trip over.
