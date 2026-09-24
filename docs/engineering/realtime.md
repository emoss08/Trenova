# Realtime

How a change made anywhere in Trenova reaches every screen that shows it, how
presence and typing work, and what to keep in mind when you publish an event or
change the stream. Live updates are server-sent events from the API; there is no
third-party realtime vendor and no WebSocket outside the AI chat.

## The shape of it

```
domain write ── PublishResourceInvalidation ──▶ in-memory queue ──▶ pipelined XADD
   (API or worker)            (never blocks)       (Publisher)      trenova:realtime:stream:{shard}
                                                                             │
      every API replica: one XREAD over every shard ◀────────────────────────┘
                 │
                 ▼  local hub: tenant → open streams, filtered per reader
      GET /api/v1/realtime/stream/  (text/event-stream, one per tab)
```

| Piece | Where |
|---|---|
| Port (events, envelopes, gateway, broker) | `internal/core/ports/services/realtime.go` |
| Publishing an invalidation | `internal/core/services/realtimeservice/service.go` |
| Opening streams, presence and typing rules | `internal/core/services/realtimeservice/gateway.go` |
| Redis bus, fan-out, replay, presence register | `internal/infrastructure/realtimebroker/` |
| HTTP endpoints | `internal/api/handlers/realtimehandler/handler.go` |
| Wiring (publisher in every process, gateway in the API) | `internal/bootstrap/modules/infrastructure/realtime.go` |
| Browser client | `client/packages/shared/src/services/realtime.ts` (`realtimeClient`) |
| Configuration and defaults | `realtime:` in `config/config.example.yaml`, `config.RealtimeConfig` |

## Publishing

Call `services.RealtimeService.PublishResourceInvalidation` after a write
commits. It validates the request, encodes the event once (plus a redacted copy
for portal readers) and puts it on a bounded queue. It does not wait on Redis
or anything else, so it is safe on a hot path: a bulk operation that publishes
several events per record pays microseconds for them. It used to be a blocking
HTTPS call to the vendor per event, which is what made a 21-shipment billing
transfer take two minutes.

A single writer goroutine per process drains the queue in pipelined batches, so
one process's events reach Redis in the order it published them. When the queue
is full the event is dropped, counted (`trenova_realtime_publish_dropped_total`)
and reported as `ErrPublishBackpressure`. The write has already happened. The
only loss is the notice, and the affected screens refetch on their next reconnect
or reset.

- **One reader.** Set `AudienceUserID` when an event is about one person
  (notifications with a target user, assistant turns). It is then delivered only
  to that person's connections, filtered on the server.
- **Large entities.** An event whose encoded entity is larger than
  `maxEntityBytes` is sent without it; the reader refetches instead of patching.
- **Portal readers** (drivers in Dash) receive an event not addressed to them
  without `entity` and `fields`: they learn that a kind of record changed, never
  what it now says.

## The bus

Events are appended to one of `shardCount` Redis Streams, chosen by an FNV hash of
`orgID:buID`, so all of a tenant's events share one stream and one order. Each
stream is trimmed to about `streamMaxLen` entries. **`shardCount` must be the same
on every instance sharing a Redis**, or a writer and a reader will disagree about
where a tenant's events live.

Each API replica runs one reader that blocks on every shard at once, starting
from each shard's newest entry when the replica starts. It decodes each entry
once and hands it to the streams open on that replica for the entry's tenant.

## A stream

`GET /api/v1/realtime/stream/` (add `?presence=users` to join the tenant's online
users; Dash does not). Only a signed-in user can open one; API keys cannot. Every
frame is `event:` + `data:` JSON, and anything that can be resumed from also has
an `id:`.

| Event | When | Payload |
|---|---|---|
| `ready` | first, always | `connectionId`, `heartbeatIntervalMs`, `resumed`, and when not resumed a `cursor` to resume from later |
| `reset` | a resume was asked for and cannot be served exactly | `{}`, meaning refetch everything |
| `presence.snapshot` | after `ready` with `?presence=users` | `{scope, members[]}` |
| `resource.invalidation` | a record changed | `ResourceInvalidationEvent` |
| `presence` | someone joined or left a scope this connection is in | `{scope, action, userId, connectionId, name}` |
| `typing` | someone in a joined scope is typing | `{scope, userId, name, stop}` |
| `heartbeat` | every `heartbeatInterval` | `{}` |
| `close` | the server is ending the stream | `{reason: rotate \| shutdown \| overflow}` |

**Resuming.** The client sends the id of the last event it *applied* as
`Last-Event-ID`. The replica replays that tenant's entries after the cursor from
the shard, up to `replayScanLimit` entries back. If the cursor is malformed, from
another shard layout, older than the stream still holds, or too far behind, the
reader gets `reset` and refetches. Live entries that reached the new stream while
the replay ran are deduplicated against the replay's position. Presence and typing
are ephemeral and are never replayed; the snapshot covers presence.

**Ending.** A stream is recycled after `maxStreamLifetime` (±10%) with
`close: rotate`, so a long session proves its authentication again and a fleet
restart does not bring every reader back in the same second. A reader whose
buffer (`subscriberBuffer`) fills is closed with `overflow` instead of being
waited on, so one slow browser cannot hold up everyone else on the replica; it
resumes from its cursor. Shutdown drains every stream with `close: shutdown`
through `http.Server.RegisterOnShutdown`, so readers move to another replica
while this one finishes. Per-replica limits (`maxConnections`,
`maxConnectionsPerUser`) answer `429` with `Retry-After`.

Keep the stream out of anything that buffers. The timeout middleware skips
`/stream` paths, the gzip middleware excludes this route, the handler clears the
write deadline, and Caddy flushes `text/event-stream` as it arrives. A new proxy
in front of the API needs the same.

## Presence and typing

Presence is tied to the stream's connection, not to page requests:

- The connection is recorded in Redis (`conn:{id}`) with its user, tenant and
  display name, and expires after `presenceTTL` unless refreshed.
- A scope's members are a sorted set of connection ids scored by expiry, plus a
  hash of their names. The replica holding the connection refreshes every scope
  it has joined on each heartbeat.
- Closing the stream withdraws every membership and announces each departure.
  A replica that dies without closing leaves its members to expire. On each
  heartbeat one replica, holding a short lock, sweeps expired members with a Lua
  script that removes and returns each one once, and announces those departures.
- A join is written and announced **before** the snapshot is read, so a member
  who arrives between the two is either in the snapshot or reaches the reader
  live, never neither.

Scoped presence and typing go through endpoints that carry their own
authorization. Today that is the shipment comment thread:

| Route | Purpose |
|---|---|
| `POST /api/v1/shipments/:shipmentID/comments/presence/` | join, returns the snapshot |
| `DELETE /api/v1/shipments/:shipmentID/comments/presence/?connectionId=` | leave |
| `POST /api/v1/shipments/:shipmentID/comments/typing/` | typing, or `stop: true` |

Each takes the caller's `connectionId`, and the gateway checks that the
connection belongs to the caller in the caller's current tenant. The scope is
built on the server from the path. Typing is stamped with the caller's identity
from the connection record, never from the request body, and is throttled per
connection and scope (`typingThrottle`). A stop always goes through. Scoped events
reach only connections that joined the scope; the joining replica learns about a
join served by another replica from the join's own `presence` event.

To add a scope for another record, add a route next to these with that resource's
permission check and a scope prefix of its own. Add the route to the owning
feature in `platformcatalog/provider_routes.go`, then add a hook that calls
`realtimeClient.joinScope` / `subscribePresence`.

## The browser client

`realtimeClient` in `@trenova/shared/services/realtime` is the only way the web
app and Dash reach the stream. It reads the stream with `fetch` (not
`EventSource`) so reconnection is under its control:

- Backoff with full jitter up to 30s after a failure, a short spread after
  `rotate` or `shutdown`, and an immediate retry when the browser comes back
  online. `401`/`403` stop it for good, and `429` honours `Retry-After`.
- A watchdog aborts a stream that has been silent for 2.5 heartbeats, whatever
  the socket says.
- Joins are reference-counted per scope and re-sent on every new connection;
  subscribers receive the full member list on each change.
- `connect({ identity, joinUsers })` reopens the stream when the identity changes,
  such as another user or a tenant switch, and forgets the previous session's
  cursor and presence. `disconnect()` runs on logout.

The web app handles events in `hooks/use-realtime-connection.ts`
(`RESOURCE_QUERY_KEY_MAP` decides what each resource invalidates) and refetches
its core keys on `reset`.

## Operating it

Prometheus metrics are under `trenova_realtime_*`: open connections, opens by
result, closes by reason, publishes and drops, flush size and latency, deliveries
by event, replays by result, fan-out read errors, and swept presence members. A
rising `closed_total{reason="overflow"}` means readers cannot keep up. A rising
`replays_total{result="reset_trimmed"}` means `streamMaxLen` is shorter than
typical disconnects.

Redis memory is bounded by `shardCount × streamMaxLen` stream entries plus one
small hash per open connection and one sorted set per active scope.
