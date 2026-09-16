# Realtime Gateway Design

Status: proposed
Owner: platform
Supersedes: the Foony-backed realtime path (`internal/infrastructure/foony`,
`internal/core/services/realtimeservice`)

This document specifies a first-party WebSocket transport for Trenova: the gateway
that carries cache-invalidation events, notifications, presence, and typing
indicators to the web app and the driver portal.

The design optimizes for four properties, in this order:

1. **No cross-tenant or cross-principal leakage.** A subscriber receives an event
   only if the server can prove, from server-side state, that the subscriber is
   entitled to it.
2. **Durability of event production.** If a transaction commits, its event is
   delivered or the client is told to resynchronize. Never silently lost.
3. **Tenant isolation under load.** One tenant's burst degrades that tenant, not
   its neighbors on the same node.
4. **Throughput and connection density**, accepting the trade-offs listed in
   [Accepted trade-offs](#accepted-trade-offs).

## 1. Why replace the current path

Three defects motivate this work. All three survive any vendor swap, which is why
the transport and the event pipeline are redesigned together.

### 1.1 Intra-tenant broadcast leaks data between principals

`realtimeservice.tenantCapability` grants every authenticated actor `subscribe` on
`tenant:{org}:{bu}:*`, and every producer publishes to a single
`tenant:{org}:{bu}:data-events` channel. Thirty-three call sites attach full entity
payloads.

`notificationservice/service.go:72` publishes `Entity: created` — the complete
notification including `targetUserId`, title, and body — to that tenant-wide
channel. The only thing keeping one user's notifications off another user's screen
is a browser-side filter in `use-realtime-connection.ts:165`:

```ts
const isForCurrentUser = !notif.targetUserId || notif.targetUserId === user.id;
```

The message has already crossed the wire. The driver portal subscribes to the same
channel (`use-dash-realtime.ts:46`), so driver-portal sessions receive invoice,
customer, carrier, and settlement entity payloads for the entire tenant.

**Root cause: authorization is expressed as a channel-name wildcard rather than as a
per-topic entitlement check.** Section 4 replaces this.

### 1.2 Event production is a best-effort dual write

`PublishResourceInvalidation` is called synchronously inside the request path, and
every call site discards the failure:

```go
if err != nil {
    s.l.Warn("failed to publish shipment comment invalidation", zap.Error(err))
}
```

A transport blip after `COMMIT` means the event is gone. Clients stay stale until a
reconnect or an unrelated refetch. It also places third-party latency on every
write. Section 5 replaces this with a transactional outbox.

### 1.3 No tenant fairness

There is no per-tenant connection quota, no ingest admission control, and no
per-connection buffer bound. A bulk invoice run or a client reconnect loop is
absorbed by whatever node happens to hold the connections. Section 6 addresses
this.

### 1.4 What is worth keeping

The `services.RealtimeService` port is the right seam. All ~58 producer call sites
route through `realtimeinvalidation.Publish` into that interface, so the
implementation can be replaced without touching them. The client's recovery
strategy — invalidate core query keys on reconnect
(`use-realtime-connection.ts:138`) — is also correct, and this design promotes it
from a blunt reconnect hammer into a first-class `resync` control frame.

## 2. Architecture

```
  ┌─────────────────────────────────────────────────────────────┐
  │ TMS API (write path)                                        │
  │                                                             │
  │   service mutation ──┬──> domain tables      ┐              │
  │                      └──> realtime_outbox    ┘ ONE Bun tx   │
  └─────────────────────────────────────────────────────────────┘
                              │
                    ┌─────────▼─────────┐
                    │ Relay (N replicas)│  FOR UPDATE SKIP LOCKED
                    │                   │  ordered per tenant
                    └─────────┬─────────┘
                              │ XADD
              ┌───────────────▼───────────────┐
              │ Redis Streams: rt:shard:{0..K}│  hash(orgID) % K
              │ MAXLEN ~ retention window     │
              └───────────────┬───────────────┘
                              │ XREAD (no consumer group — true fan-out)
          ┌───────────────────┼───────────────────┐
          │                   │                   │
  ┌───────▼──────┐    ┌───────▼──────┐    ┌───────▼──────┐
  │ Gateway n1   │    │ Gateway n2   │    │ Gateway nN   │
  │ per-tenant   │    │              │    │              │
  │  dispatchers │    │              │    │              │
  │ conn registry│    │              │    │              │
  └───────┬──────┘    └──────────────┘    └──────────────┘
          │ WebSocket
     web / dash clients

  Redis (shared): presence zsets + meta hashes, GCRA rate-limit keys,
                  per-tenant connection counters
```

The gateway is a **separate deployable** in the same Go module. Long-lived sockets
have different memory and deploy characteristics than HTTP requests; an API deploy
must not drop every connection. It imports `services.PermissionEngine`,
`repositories.RateLimitStore`, and the domain types directly — sharing the
authorization code is the entire reason for building rather than buying.

### 2.1 Component responsibilities

| Component | Owns | Holds authoritative state |
| --- | --- | --- |
| Producer (TMS services) | Writing outbox rows inside the business transaction | No |
| Relay | Outbox → Redis Streams, ordered per tenant, at-least-once | Cursor in Postgres |
| Gateway | Connections, subscriptions, authz, fan-out, backpressure | No (recoverable) |
| Redis | Stream buffer, presence, quotas, rate limits | Ephemeral, rebuildable |
| Postgres | Outbox, dead letters | Yes |

The gateway holding **no authoritative state** is what makes node loss a non-event:
clients reconnect elsewhere and re-establish subscriptions.

## 3. Wire protocol

JSON frames over WebSocket (`github.com/coder/websocket`, already a transitive
dependency), encoded with `sonic` per the repository Go rules. `permessage-deflate`
is negotiated; entity payloads compress well.

### 3.1 Client → server

| Type | Fields | Notes |
| --- | --- | --- |
| `subscribe` | `topics[]`, `lastId?` | Batched. `lastId` requests gap replay. |
| `unsubscribe` | `topics[]` | |
| `presence.enter` | `topic`, `meta` | `meta` is server-sanitized; see 8.2 |
| `presence.leave` | `topic` | |
| `presence.beat` | `topics[]` | Every 20s, batched across topics |
| `publish` | `topic`, `name`, `data` | Only on `typing` topics; see 4.4 |
| `auth.refresh` | `token` | In-band re-auth, no reconnect |

### 3.2 Server → client

| Type | Fields | Notes |
| --- | --- | --- |
| `ready` | `connectionId`, `expiresAt` | Post-handshake |
| `subscribed` | `topics[]`, `denied[]` | Denials are explicit, never silent |
| `event` | `topic`, `id`, `name`, `data` | `id` is the stream ID, used for `lastId` |
| `resync` | `scope`, `reason` | Client invalidates `scope` and refetches |
| `presence.delta` | `topic`, `join[]`, `leave[]` | |
| `presence.sync` | `topic`, `members[]` | Full set on subscribe |
| `error` | `code`, `message` | |

`resync` is the universal degradation path. Every place this design chooses to shed
load, drop precision, or recover from failure, it emits `resync` rather than
failing silently. The client already implements the handler — it is the existing
`invalidateCoreKeys()` path, scoped.

## 4. Topic model and authorization

### 4.1 Topic grammar

```
org:{orgID}:bu:{buID}:res:{resource}              tenant-scoped resource class
org:{orgID}:bu:{buID}:user:{userID}               principal-private
org:{orgID}:bu:{buID}:user:{userID}:res:{res}     self-scoped resource class
org:{orgID}:bu:{buID}:ent:{resource}:{entityID}   single entity
org:{orgID}:bu:{buID}:typing:{resource}:{entityID}  client-publishable, ephemeral
org:{orgID}:bu:{buID}:presence:{scope}            presence-only
```

The single `data-events` firehose is retired. This is both the leak fix and the
largest single reduction in fan-out volume — most subscribers care about a handful
of resource classes, not all 30 in `RESOURCE_QUERY_KEY_MAP`.

### 4.2 Authorization at subscribe

Topic names are addressing, **not** security. Every subscribe is authorized against
server-side state:

1. **Tenant binding.** The connection carries an immutable `(orgID, buID, userID,
   principalType)` established at handshake from the authenticated session. A
   subscribe whose topic org/bu does not match the connection's is rejected with
   `ERR_TOPIC_FORBIDDEN`. Client-supplied org/bu is never trusted, including from
   token claims — the claim is validated against the session at handshake.
2. **Entitlement.** `PermissionEngine.GetLightManifest(userID, orgID)` is loaded
   once at handshake and cached on the connection. Subscribe evaluation is an
   in-process manifest lookup — no I/O on the hot path. This is the concrete payoff
   of owning the gateway: with a third-party subscribe-proxy this is a network
   round-trip per subscription.
3. **Scope.** A principal with tenant-wide read on a resource may subscribe to the
   `res:` topic. A self-scoped principal (driver-portal users) may subscribe only to
   its own `user:{self}:res:` topic. A subscribe to another user's private topic is
   rejected regardless of role.

### 4.3 Revocation on a live connection

Long-lived sockets must not keep streaming after a role change. `PermissionEngine`
already exposes `InvalidateUser(ctx, userID, orgID)`. That call additionally
publishes to an internal control topic; every gateway holding a connection for that
user reloads the manifest and **re-evaluates existing subscriptions**, dropping any
that no longer authorize and emitting `resync`. A 5-minute manifest TTL is the
backstop if the control event is missed.

This is the failure mode most homegrown gateways miss: they authorize at subscribe
and never again, so a revoked user keeps receiving data until they close the tab.

### 4.4 Client-published topics

Only `typing:` topics accept client `publish`. Payloads are schema-validated,
size-capped (1 KB), rate-limited per connection, and the server overwrites the actor
identity from the connection — a client cannot forge who is typing. Typing events
are never persisted to the outbox.

### 4.5 Audience is declared by the producer

Resource-level permission is not sufficient for row-scoped data: a driver may read
their own settlement but not a peer's. The producer therefore declares an
**audience**, not just a resource. `realtimeinvalidation.PublishParams` gains:

```go
type Audience struct {
    Scope      AudienceScope // AudienceTenant | AudienceUsers | AudienceEntity
    UserIDs    []pulid.ID
    EntityID   pulid.ID
}
```

Routing rules:

- `AudienceTenant` → `res:` topic. Delivered to principals with tenant-wide read.
- `AudienceUsers` → one `user:{id}` or `user:{id}:res:{r}` topic per listed user.
  Notifications and driver-owned records use this. **Targeted data is never
  broadcast and never filtered client-side.**
- `AudienceEntity` → `ent:` topic, for subscribers already viewing that record.

`Audience` is a required field. A producer that omits it fails validation at
compile time via a constructor rather than defaulting to tenant-wide — defaulting to
the broadest audience is how the current leak happened.

### 4.6 Payload shaping

Two payload variants per event:

- `full` — the entity, for principals entitled to read it.
- `ref` — `{entityId, entityVersion, fields[]}`, enough to invalidate or
  version-check without disclosing content.

Variants are computed **once per event**, not per subscriber, which preserves the
encode-once optimization in 7.2. The variant a subscriber receives is a property of
the topic it subscribed to, so selection is a map lookup at dispatch time.

Resources carrying regulated or sensitive content (worker medical, drug and alcohol,
settlement detail, HR records) are configured `ref`-only regardless of topic; the
client refetches through the authorized REST/GraphQL path. Defaulting these to `ref`
means a future routing bug degrades to a spurious refetch rather than a disclosure.

### 4.7 Defense in depth at the write boundary

Every outbound frame carries its tenant identity, and the connection writer asserts
`frame.orgID == conn.orgID && frame.buID == conn.buID` immediately before writing.

This is one integer comparison per frame and it is deliberately redundant with the
subscribe-time check. It exists to catch *routing* bugs — a corrupted subscriber
list, a map key collision, a slice reused across tenants — which are exactly the
bugs that produce cross-tenant disclosure and which subscribe-time authorization
cannot detect. A failed assertion drops the frame, increments
`rt_tenant_assert_failed_total`, pages, and closes the connection.

Treat any non-zero value of that counter as a Sev-1.

### 4.8 Organization switching

A user switching organizations invalidates the entire authorization context. The
gateway closes the connection with `ERR_CONTEXT_CHANGED`; the client reconnects and
re-handshakes under the new org. Subscriptions are never carried across a context
change.

## 5. Durability

### 5.1 Transactional outbox

The event row is written in the same Bun transaction as the mutation. No dual write,
no lost-after-commit window, no vendor latency in the request path.

```sql
CREATE TABLE realtime_outbox (
    id                  varchar(100) PRIMARY KEY,
    organization_id     varchar(100) NOT NULL,
    business_unit_id    varchar(100) NOT NULL,
    tenant_seq          bigint       NOT NULL,
    topic               text         NOT NULL,
    event_name          text         NOT NULL,
    payload_full        jsonb,
    payload_ref         jsonb        NOT NULL,
    audience_scope      smallint     NOT NULL,
    audience_user_ids   varchar(100)[],
    created_at          timestamptz  NOT NULL DEFAULT now(),
    published_at        timestamptz,
    attempts            smallint     NOT NULL DEFAULT 0
);

CREATE INDEX realtime_outbox_pending_idx
    ON realtime_outbox (organization_id, tenant_seq)
    WHERE published_at IS NULL;
```

`tenant_seq` comes from a per-tenant sequence and establishes total order within a
tenant. The partial index keeps the relay's scan proportional to the backlog, not
the table.

Repository code uses the generated helpers in `pkg/buncolgen/` per the repository
rules.

### 5.2 Relay

Multiple replicas, each claiming work with `FOR UPDATE SKIP LOCKED` batched by
tenant so a tenant's rows are processed in `tenant_seq` order by exactly one relay
at a time. `XADD` to `rt:shard:{hash(orgID) % K}`, then mark `published_at`.

Delivery is **at-least-once**. A crash between `XADD` and the mark replays the
event. Consumers deduplicate on the event ULID; the client already carries
`EventID`. Cache invalidation is idempotent, so duplicates are harmless by
construction.

### 5.3 Poison events and the ordering trade-off

After `maxAttempts`, a row moves to `realtime_dead_letter` and the relay **skips
it** rather than blocking the tenant.

Skipping breaks per-tenant ordering, so the relay immediately emits a synthetic
`resync` for the affected resource scope. Correctness is preserved by the client's
refetch; only precision is lost. Blocking a tenant's entire realtime feed on one bad
row is strictly worse than one extra refetch.

### 5.4 Retention and reaping

Published rows are deleted in batches after 24 hours (kept briefly for incident
forensics). The table is range-partitioned by `created_at` daily so reaping is a
partition drop, not a bulk delete — bulk deletes on a hot table generate the vacuum
pressure that eventually makes the outbox itself the incident.

### 5.5 Redis Stream durability

Streams are trimmed with `XTRIMMAXLEN ~` to a retention window sized for the
reconnect-replay budget (target: 5 minutes of tenant traffic, see 7.3).

If Redis is lost, **no committed event is lost** — unpublished rows remain in the
outbox and the relay resumes. Events already delivered to the stream but not yet to
a client are lost; those clients reconnect, fail replay, and receive `resync`. This
is an explicit trade-off: we do not run Redis as a system of record. Enable
`appendonly everysec` on the realtime instance to narrow the window, and accept the
rest.

## 6. Tenant isolation and noisy-neighbor control

This section answers the design question directly: **multiple tenants share a node;
what stops one from degrading the others?**

Four distinct amplification paths exist, and each needs its own bound.

### 6.1 Ingest burst — one tenant floods the pipeline

*Example: an invoice run posting 10,000 invoices, or a bulk shipment import.*

Per-tenant GCRA admission control at the relay, reusing
`repositories.RateLimitStore` (`infrastructure/ratelimit`). Each tenant has an
events-per-second policy with a burst allowance.

On exceedance the relay does **not** drop events. It **collapses** them: the burst is
replaced by a single `resync` per affected resource scope, emitted at most once per
collapse window. A tenant doing a 10,000-row import produces one "refresh your
shipment list" instead of 10,000 row patches — which is also better UX. Collapse is
correct precisely because the client's fallback is a refetch.

Collapse thresholds are per-tenant configuration, so a known-heavy tenant can be
tuned without code changes.

### 6.2 Connection hogging

Enforced at handshake, before the socket is accepted:

- **Per-tenant connection quota**, tracked in a Redis counter keyed by org, sized
  from the tenant's seat entitlement with headroom. Exceeded → `ERR_QUOTA` and a
  `Retry-After`.
- **Per-user connection cap** (default 5) to contain reconnect-loop bugs and
  runaway tab counts.
- **Per-node global cap** with the node reporting unready to the load balancer on
  exceedance, shifting new connections to other nodes rather than degrading
  existing ones.

Handshake admission is also GCRA-limited per tenant, so a thundering-herd reconnect
from one tenant cannot monopolize accept capacity.

### 6.3 Slow consumers — memory amplification

Each connection has a **byte-bounded** send buffer (default 64 KB), not a
message-count bound. Message counts give non-deterministic memory because payload
sizes vary by two orders of magnitude; byte bounds make the worst case exactly
`maxConnsPerNode × 64 KB`.

On overflow the connection does not grow its buffer and does not block the
dispatcher. It is sent `resync` and closed. The client reconnects and refetches.

**A slow client's cost is bounded at 64 KB and paid by that client alone.** This is
the single most important property for node stability; unbounded per-connection
queues are the standard way a homegrown gateway dies.

### 6.4 Head-of-line blocking — CPU and scheduling amplification

A single dispatch loop shared across tenants lets one tenant's serialization cost
stall every other tenant's delivery.

Each `(node, tenant)` pair gets a **dedicated dispatcher goroutine with a bounded
queue**. A tenant's burst fills that tenant's queue and nothing else. Queue overflow
degrades to `resync` for that tenant's subscribers, exactly as in 6.3.

Go's scheduler round-robins runnable goroutines, so per-tenant dispatchers give
approximately fair CPU distribution without an explicit weighted-fair-queue
implementation. If measurement later shows starvation under extreme skew, the
dispatchers are the correct place to add explicit weighting — the structure is
already in place.

### 6.5 The escape hatch: cell routing

The controls above bound a noisy tenant's blast radius. They do not eliminate
resource contention for a genuinely pathological tenant.

Organizations carry an optional `realtime_cell` label. The load balancer routes by
it, pinning a tenant to a dedicated gateway pool. This is the answer for a single
tenant large enough to warrant isolation, and it is also the **primary scaling
lever** for Redis egress (see 7.3).

Cell routing is Phase 4 — the label and the routing hook ship early so that moving a
tenant is a configuration change during an incident, not a deploy.

### 6.6 Isolation summary

| Amplification path | Bound | Degradation |
| --- | --- | --- |
| Ingest burst | Per-tenant GCRA at relay | Collapse to `resync` |
| Connection count | Per-tenant / per-user / per-node quota | Reject at handshake |
| Slow consumer memory | 64 KB per connection | Shed connection, `resync` |
| Dispatch CPU | Per-tenant goroutine + bounded queue | `resync` that tenant only |
| Sustained heavy tenant | Cell routing | Dedicated pool |

Every bound degrades to `resync`. There is no path where exceeding a limit produces
silent data loss.

## 7. Scale and performance

### 7.1 Fan-out: `XREAD` without consumer groups

This deliberately departs from the existing pattern in
`tablechangealertservice/consumer.go`, which uses `XReadGroup`.

Consumer groups deliver each message to **one** member — correct for work queues,
wrong for fan-out, where every gateway holding connections for a tenant needs every
event for that tenant. Plain `XREAD` from a per-node checkpoint gives true fan-out,
and the checkpoint gives free gap recovery after a node restart.

Sharding is `hash(orgID) % K` with `K = 64`. Hashing (not round-robin) is what
preserves per-tenant ordering, since a tenant's events always land in one shard and
streams are ordered.

### 7.2 Encode once, write many

Each event is serialized **once per variant** (`full`, `ref`) into an immutable
`[]byte`, and that buffer is shared by every subscriber's writer. Fan-out cost
becomes `O(events × variants)` for encoding plus `O(subscribers)` for socket writes,
instead of `O(events × subscribers)` for encoding.

This is the largest single CPU win in the design, and it is why 4.6 constrains
variants to a fixed, small set. Per-subscriber payload personalization would forfeit
it.

The shared buffers are immutable and never pooled across events; recycling them into
a `sync.Pool` reachable from another tenant's write path would reintroduce exactly
the cross-tenant hazard that 4.7 guards against.

### 7.3 Redis egress is the real ceiling

With random tenant placement, each node subscribes to nearly all `K` shards, so
Redis egress is:

```
egress ≈ E × avg_event_size × N_nodes
```

where `E` is total event rate. This grows with node count — it is the binding
constraint at scale, not CPU and not connection memory.

Tenant affinity is the fix. With cell routing (6.5), a node subscribes only to the
shards its cell owns:

```
egress ≈ E × avg_event_size × N_nodes × (shards_per_cell / K)
```

Affinity is therefore a scaling requirement, not an optimization, and the design
ships the hook early even though the routing policy lands in Phase 4.

Nodes subscribe only to shards for which they currently hold at least one
connection, and unsubscribe when the last connection for a shard closes.

### 7.4 Capacity model

Per connection:

| Item | Bytes |
| --- | --- |
| `coder/websocket` read + write buffers | ~8 KB |
| Two goroutine stacks (reader, writer) | ~8 KB |
| Connection struct, subscription set, cached manifest ref | ~4 KB |
| Send buffer worst case | 64 KB |
| **Worst case total** | **~84 KB** |
| **Steady state (buffer near empty)** | **~20 KB** |

Target: **10,000 connections per node** at 4 vCPU / 8 GB. That is ~200 MB steady
state and ~840 MB if every connection is simultaneously at its buffer ceiling — a
state that implies every client is slow at once, and which sheds itself by design.

20,000 goroutines per node is comfortable for the Go scheduler.

### 7.5 Reconnect and gap recovery

Stream IDs are monotonic per shard, so the client's `lastId` is a resumable cursor:

1. Client reconnects, sends `subscribe` with `lastId`.
2. If `lastId` is within the retention window, the gateway `XRANGE`s the gap and
   replays only events matching the (re-authorized) subscriptions.
3. Otherwise → `resync`.

Re-authorization on replay is mandatory: permissions may have changed while the
client was disconnected, and replaying a pre-revocation buffer would leak.

This is strictly better than today's unconditional invalidate-everything, and it
falls out of the Streams design at no extra cost.

### 7.6 Deploy draining

On `SIGTERM`: report unready to the load balancer, stop accepting connections, then
close existing connections in **jittered batches** over the termination grace
period, with a `resync`-on-reconnect hint.

Jitter is essential. Closing 10,000 sockets simultaneously produces a reconnect
thundering herd that lands on the remaining nodes and can cascade. Clients
additionally use exponential backoff with full jitter.

## 8. Presence

### 8.1 Storage

Presence is the part of a homegrown gateway most often built wrong — in-memory
membership leaks ghost members on every rolling deploy and every node crash.

Redis, two keys per presence topic:

- `rt:pres:{topic}` — sorted set, member `clientID`, score = last heartbeat (Unix ms)
- `rt:pres:meta:{topic}` — hash, `clientID` → sanitized member metadata

Operations:

- **enter** → `ZADD` + `HSET`, publish `presence.delta` join
- **beat** → `ZADD` (20s interval, batched across topics in one frame)
- **leave** → `ZREM` + `HDEL`, publish delta
- **read** → `ZRANGEBYSCORE (now - 50s) +inf`, then `HMGET`
- **sweep** → periodic `ZREMRANGEBYSCORE` on `-inf (now - 50s)`, `HDEL` the removed
  members, publish leave deltas

Reads filter by score rather than trusting set membership, so a crashed node's
members disappear from *reads* immediately at the staleness horizon even before the
sweeper runs. Node death requires no cleanup coordination — entries simply age out.

### 8.2 Metadata is server-authored

Presence metadata (name, email) is populated **by the gateway from the authenticated
session**, never from the client frame. The current client sends its own
`userId`/`name`/`email` in `presence.enter` (`use-realtime-connection.ts:119`),
which lets any client claim any identity in a presence list. The `meta` field in the
client frame is reduced to a small allowlist of non-identity fields (e.g. view
context) and is size-capped.

### 8.3 Isolation

Presence keys are namespaced by topic and therefore by tenant, and presence reads go
through the same subscribe-time authorization as events. Presence set cardinality is
bounded per topic; a tenant exceeding it gets truncated reads plus a warning metric
rather than an unbounded `ZRANGEBYSCORE`.

## 9. Failure modes

| Failure | Detection | Behavior | Data loss |
| --- | --- | --- | --- |
| Producer tx rolls back | — | Outbox row rolls back with it | None (correct) |
| Relay crash pre-`XADD` | Lease expiry | Another relay claims by `tenant_seq` | None |
| Relay crash post-`XADD`, pre-mark | Lease expiry | Event replayed, client dedupes on ULID | None |
| Poison event | `attempts > max` | Dead-letter + skip + scoped `resync` | Precision only |
| Relay lag | Outbox depth / oldest-unpublished age | Alert; events still durable | None |
| Redis down | Client errors | Outbox retains; clients `resync` on recovery | In-flight only |
| Gateway node dies | LB health check | Clients reconnect elsewhere, replay or `resync` | None |
| Client slow | Send buffer at cap | Shed + `resync` | None |
| Tenant floods ingest | GCRA | Collapse to `resync` | Precision only |
| Permission revoked mid-session | Control event / manifest TTL | Subscriptions dropped, `resync` | None |
| Routing bug crosses tenants | Write-boundary assertion (4.7) | Frame dropped, connection closed, page | None |

Every row terminates in `resync`, replay, or a page. None terminates in silent
staleness — which is the current behavior for most of this table.

## 10. Observability

Metrics, labeled by tenant where cardinality permits (bucket small tenants into an
`other` label to bound series growth):

- `rt_connections{org}` — gauge; also per-node total
- `rt_subscribe_denied_total{reason}` — a rising rate means a client bug or an
  authorization regression
- `rt_tenant_assert_failed_total` — **must be zero**; non-zero is Sev-1
- `rt_outbox_pending`, `rt_outbox_oldest_seconds` — relay health, primary lag SLI
- `rt_relay_publish_seconds` — histogram
- `rt_fanout_lag_seconds` — `now - event.created_at` at socket write; the
  end-to-end SLI
- `rt_send_buffer_bytes` — histogram; the leading indicator before shedding
- `rt_conn_shed_total{reason}`
- `rt_resync_total{reason}` — a rising rate means a bound is being hit
- `rt_collapse_total{org}` — noisy-neighbor detector, and the signal to consider
  cell routing for that tenant
- `rt_presence_members{topic_class}`

Tracing propagates the producing request's trace ID through the outbox row so a
delivered event links back to the mutation that caused it.

Proposed SLOs:

| SLI | Objective |
| --- | --- |
| p99 end-to-end (commit → client) | < 500 ms |
| p99 outbox age | < 2 s |
| Delivery (event delivered or `resync` issued) | 99.99% |
| Cross-tenant frames | 0 |

## 11. Accepted trade-offs

The brief asked for scale and performance even at a cost. These are the costs.

1. **At-least-once, not exactly-once.** Clients deduplicate by event ULID. Cache
   invalidation is idempotent, so this is free in practice and avoids distributed
   transactions.
2. **Per-tenant ordering only.** No global ordering across tenants. Nothing in the
   product needs it.
3. **Precision degrades under load.** Bursts collapse to `resync`. Correctness is
   preserved by refetch; individual row patches are lost. This is what makes the
   noisy-neighbor bounds safe to enforce aggressively.
4. **Fixed payload variants.** Per-subscriber personalization is forfeited to keep
   encode-once fan-out.
5. **Presence is eventually consistent.** A crashed client lingers up to 50 s.
6. **Bounded recovery window.** Disconnections longer than stream retention get a
   full `resync` rather than replay.
7. **Redis is not a system of record.** Losing Redis loses in-flight events; the
   outbox guarantees nothing committed is lost.
8. **Separate deployable.** An additional service to operate, an additional network
   hop, in exchange for independent scaling and deploys that do not drop sockets.
9. **Producers must declare audience.** A required field on ~58 call sites. This is
   deliberate friction: the alternative — defaulting to tenant-wide — is the current
   leak.

## 12. Migration

`services.RealtimeService` stays as the port, which keeps all ~58 producer call
sites untouched through the transport swap.

**Phase 1 — outbox.** Add the table, write rows inside existing transactions, keep
publishing to Foony. No client change. Delivers the durability fix independently and
is independently revertable.

**Phase 2 — topic model and audience.** Add `Audience` to `PublishParams`, migrate
call sites, split the firehose into resource-class and user-scoped topics, move
notifications to `user:` topics. **This closes the leak** and can ship on the
existing transport. Highest security value, lowest infrastructure risk — it is the
phase to prioritize if anything slips.

**Phase 3 — gateway.** Relay, Redis Streams, gateway service, client transport.
Dual-publish to both transports; flag clients over per tenant. Compare
`rt_fanout_lag_seconds` and event counts across transports in shadow mode before
cutting over.

**Phase 4 — decommission and harden.** Remove `infrastructure/foony` and the
dependency. Enable cell routing. Tune per-tenant quotas from observed traffic.

Phases 1 and 2 are independently valuable and independently revertable. The gateway
is only load-bearing in Phase 3, by which point the event model it carries has been
running in production for two phases.

## 13. Testing

- **Authorization matrix** — table-driven over (principal type, role set, topic
  shape). Must include: driver-portal principal against every tenant-scoped topic, a
  user against another user's private topic, and a subscribe after mid-session
  revocation. These are regression tests for §1.1; write them from the entitlement
  contract, not from the implementation.
- **Tenant isolation** — an integration test with two tenants on one gateway
  asserting `rt_tenant_assert_failed_total == 0` and zero cross-delivery under
  concurrent load.
- **Noisy neighbor** — tenant A at 100× tenant B's event rate; assert B's p99
  `rt_fanout_lag_seconds` stays within SLO. This is the test that proves §6.
- **Durability** — kill the relay between `XADD` and mark; assert redelivery and
  client-side dedupe. Kill Redis; assert outbox drains on recovery.
- **Backpressure** — a client that never reads; assert the connection sheds at the
  buffer cap and the node's memory stays flat.
- **Recovery** — disconnect within and beyond the retention window; assert replay
  and `resync` respectively, and assert replay re-authorizes.
- **Soak** — 10,000 connections per node with rolling deploys; assert no presence
  ghosts and no goroutine or memory growth across restarts.

Go tests use `t.Context()` per the repository rules.

## 14. Open questions

1. **Per-tenant quota defaults.** Should connection quotas derive from seat
   entitlements in `entitlementservice`, or be independently configured? Deriving
   couples realtime capacity to billing state, which is either elegant or
   surprising during an incident.
2. **Stream retention window.** Five minutes is a starting point; the right value is
   the p99 client disconnect duration, which needs measurement.
3. **Does the driver portal need presence at all?** It currently inherits the shared
   client. Dropping it there removes a whole class of fan-out from the highest
   connection-count population.
4. **Relay placement.** A dedicated deployable, or a Temporal worker? Temporal gives
   retries and visibility for free but adds a hop; a bespoke poller is simpler and
   faster. Leaning bespoke, since the workload is a tight `SKIP LOCKED` loop rather
   than a workflow.
5. **`K = 64` shards.** Chosen for headroom; should be validated against the
   projected tenant count, since resharding requires an ordering-safe migration.
