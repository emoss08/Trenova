# GTC

GTC is a PostgreSQL change-data-capture connector written in Go. It snapshots configured tables, tails PostgreSQL logical replication, and projects row changes into external sinks such as Meilisearch, Redis JSON, and Redis Streams.

The service is designed to be generic enough for third-party use while still shipping well with Trenova. It supports composite primary keys, resumable snapshots, transaction-aware WAL checkpoints, bounded worker queues, retry with backoff, and dead-letter handling.

## Features

- PostgreSQL logical replication via `pgoutput`
- Automatic snapshot + tail startup flow
- Composite primary key support
- Config-first projection model
- Meilisearch indexing sink
- Redis JSON materialized-view sink
- Redis Stream change-feed sink
- Ordered, at-least-once transaction processing
- Durable WAL and snapshot checkpoints stored in PostgreSQL
- Retry with backoff and Redis-backed dead-letter queue
- Recovery commands for config validation, backfill, and DLQ replay

## How It Works

GTC runs in two phases:

1. It captures a bootstrap WAL position and snapshots the configured source tables.
2. It starts logical replication from the saved WAL position and applies committed transactions to the configured projections.

For each configured projection, GTC:

- resolves the source table metadata from PostgreSQL
- validates the table primary key shape
- routes matching records to the sink associated with the projection
- preserves checkpoint order by committed transaction

Snapshot progress and the last applied WAL commit position are stored in PostgreSQL so the connector can resume after restart.

## Supported Sinks

### Meilisearch

Use Meilisearch when you want searchable document projections.

- `kind: meilisearch`
- `index` is required
- `searchable_fields` is optional but recommended
- documents are upserted using a synthetic `_pk` field derived from the configured primary key columns
- deletes remove the document from the index

### Redis JSON

Use Redis JSON when you want a materialized latest-state record.

- `kind: redis_json`
- `key_template` is required
- inserts and updates overwrite the current JSON document
- deletes remove the key

### Redis Stream

Use Redis Streams when you want an append-only change feed.

- `kind: redis_stream`
- `stream` is required
- each record is emitted as a JSON payload with operation, source metadata, and row data
- snapshot records are not part of WAL commits, so their metadata does not include commit LSN or transaction ID

## Configuration

GTC uses two configuration layers:

- environment variables for runtime, connections, and checkpoints
- a YAML projection file for source-to-sink routing

### Environment variables

Core variables:

| Variable | Required | Default | Purpose |
|---|---:|---|---|
| `DATABASE_URL` | yes | none | PostgreSQL connection string. Must include `?replication=database` for WAL streaming. |
| `REDIS_URL` | yes for Redis sinks / DLQ | `redis://localhost:6379/0` | Redis connection for Redis JSON, Redis Streams, and DLQ. |
| `MEILISEARCH_URL` | yes for Meilisearch sinks | `http://localhost:7700` | Meilisearch base URL. |
| `MEILISEARCH_API_KEY` | recommended | empty | Meilisearch API key. Required when Meilisearch auth is enabled. |
| `GTC_CONFIG_FILE` | no | `config/gtc.yaml` | Path to the projection config file. |

Replication and runtime:

| Variable | Default |
|---|---|
| `CDC_SLOT_NAME` | `gtc_slot` |
| `CDC_PUBLICATION_NAME` | `gtc_publication` |
| `CDC_AUTO_CREATE_SLOT` | `false` |
| `CDC_AUTO_CREATE_PUBLICATION` | `true` |
| `CDC_INACTIVE_SLOT_ACTION` | `fail` |
| `CDC_MAX_LAG_BYTES` | `5368709120` |
| `CDC_STANDBY_TIMEOUT` | `10s` |
| `CDC_SNAPSHOT_BATCH_SIZE` | `500` |
| `CDC_SNAPSHOT_CONCURRENCY` | `2` |
| `CDC_PROCESS_TIMEOUT` | `15s` |
| `CDC_WORKER_COUNT` | `4` |
| `CDC_WORKER_QUEUE_SIZE` | `128` |
| `CDC_RETRY_MAX_ATTEMPTS` | `3` |
| `CDC_RETRY_BACKOFF` | `500ms` |
| `CDC_HEALTH_POLL_INTERVAL` | `10s` |
| `CDC_CHECKPOINT_SCHEMA` | `public` |
| `CDC_CHECKPOINT_TABLE` | `gtc_checkpoints` |
| `CDC_DLQ_STREAM` | `gtc:dlq` |

A current example env file is in [`.env.example`](/home/wolfred/projects/trenova-2/services/gtc/.env.example).

Operational guidance:

- local development may use `CDC_AUTO_CREATE_SLOT=true`
- production should pre-create the replication slot and set `CDC_AUTO_CREATE_SLOT=false`
- production PostgreSQL should size `logical_decoding_work_mem` for the widest replicated rows; Trenova uses `256MB`

### Projection config

Projection config is YAML under `projections:`.

Each projection defines:

- `name`: unique projection name
- `source_table`: fully qualified PostgreSQL table, such as `public.customers`
- `primary_keys`: optional explicit key column list
- `fields`: optional allowlist of fields to send to the sink
- `searchable_fields`: Meilisearch searchable attributes
- `ignored_updates`: fields that do not trigger stream output when they are the only changes
- `destination`: sink configuration

Example:

```yaml
projections:
  - name: customer-search
    source_table: public.customers
    primary_keys: [id, organization_id, business_unit_id]
    searchable_fields: [code, name]
    destination:
      kind: meilisearch
      index: customers

  - name: customer-cache
    source_table: public.customers
    primary_keys: [id, organization_id, business_unit_id]
    destination:
      kind: redis_json
      key_template: 'cache:customers:{{ key .PrimaryKeys .New .Old }}'

  - name: customer-stream
    source_table: public.customers
    primary_keys: [id, organization_id, business_unit_id]
    ignored_updates: [updated_at, version]
    destination:
      kind: redis_stream
      stream: 'cdc:customers'
```

Notes:

- `primary_keys` may be omitted; GTC will auto-discover the primary key columns from PostgreSQL and validate any configured override.
- For multi-column Redis keys, prefer the `key` helper over manually concatenating `value`.

### Redis key template helpers

Redis JSON and Redis Stream destinations support Go text templates with these helpers:

- `field "id" .New`
- `value "id" .New .Old`
- `key .PrimaryKeys .New .Old`

Recommended patterns:

```yaml
key_template: 'cache:customers:{{ value "id" .New .Old }}'
key_template: 'cache:shipments:{{ key .PrimaryKeys .New .Old }}'
stream: 'cdc:shipments'
```

## Running GTC

### Local Go run

```bash
cd services/gtc
go run ./cmd/gateway run
```

### Task

```bash
task gtc:run
task gtc:validate-config
task gtc:backfill projection=shipment-search
task gtc:replay-dlq
```

### Docker compose

```bash
docker compose -f docker-compose-local.yml up -d --build gtc
docker compose -f docker-compose-local.yml logs -f gtc
```

## CLI Commands

GTC exposes a small operational CLI from the same binary.

### `run`

Starts the HTTP server, snapshot phase, and WAL tailer.

```bash
go run ./cmd/gateway run
```

### `validate-config`

Validates:

- environment config
- projection YAML
- sink initialization
- source table metadata and primary key resolution

```bash
go run ./cmd/gateway validate-config
```

### `backfill`

Runs a snapshot-only rebuild through the configured sink pipeline without starting WAL streaming.

Use it to:

- rebuild a Meilisearch index
- repopulate Redis JSON projections
- repair sink drift after an outage

Examples:

```bash
go run ./cmd/gateway backfill --projection shipment-search
go run ./cmd/gateway backfill --table public.shipments
go run ./cmd/gateway backfill --projection shipment-search,shipment-cache
```

### `replay-dlq`

Reads failed entries from the Redis DLQ stream and retries them through the normal sink path.

Examples:

```bash
go run ./cmd/gateway replay-dlq
go run ./cmd/gateway replay-dlq --limit 50
go run ./cmd/gateway replay-dlq --delete=false
```

When `--delete=true` (default), successfully replayed DLQ entries are removed from the DLQ stream.

## Recovery Model

GTC is designed for at-least-once delivery.

This means:

- a committed transaction is checkpointed only after its records have been processed successfully
- PostgreSQL replication feedback advances only after that WAL checkpoint has been saved
- duplicates may be replayed after failure or restart
- sinks should be idempotent

Built-in sink behavior:

- Meilisearch uses upsert-by-key semantics
- Redis JSON overwrites the latest state by key
- Redis Streams remain append-only and may include duplicates after replay

## Publication Scoping

GTC does not need `FOR ALL TABLES`.

When publication auto-create is enabled, it now scopes the publication to the unique `source_table` values in the configured projection set. This reduces unnecessary replication traffic and is safer for multi-service databases.

If you pre-provision publications in production:

- create a publication with the same name as `CDC_PUBLICATION_NAME`
- include every table referenced by the projection config
- optionally set `CDC_AUTO_CREATE_PUBLICATION=false`

For replication slots in production:

- create a dedicated slot with the same name as `CDC_SLOT_NAME`
- set `CDC_AUTO_CREATE_SLOT=false` so unexpected slot loss fails loudly instead of silently recreating from a newer consistent point

## Checkpoints and Internal Tables

GTC stores runtime state in PostgreSQL:

- WAL checkpoint position
- bootstrap LSN
- snapshot progress cursor by table

By default it uses:

- schema: `public`
- table: `gtc_checkpoints`
- snapshot progress table: `gtc_checkpoints_snapshot_progress`

You can change the schema/table prefix with:

- `CDC_CHECKPOINT_SCHEMA`
- `CDC_CHECKPOINT_TABLE`

## HTTP Endpoints

GTC exposes:

- `/health`
- `/readiness`
- `/metrics`

Readiness reflects actual dependency state rather than just process liveness.

## Development

Common commands:

```bash
task gtc:test
task gtc:test-race
task gtc:docker-up
task gtc:docker-logs
```

Or directly:

```bash
cd services/gtc
go test ./...
go test -race ./...
docker build -t gtc .
```

## Caveats

- PostgreSQL must have logical replication enabled.
- `DATABASE_URL` for WAL streaming must include `?replication=database`.
- Meilisearch backfill and replay commands need a valid `MEILISEARCH_API_KEY` when auth is enabled.
- Snapshot records do not carry WAL commit metadata.
- Redis Streams are not exactly-once and may contain duplicates after replay or restart.
- Publication scoping is derived from the configured projection set. If you add a table to config, restart or rerun validation so publication state is updated.
- Per-table ordering is preserved; the runtime is not yet partitioning by record key.

## Production Notes

For production use, prefer:

- a dedicated replication slot and publication for GTC
- explicit publication provisioning if your environment restricts DDL
- dedicated Redis and Meilisearch credentials
- running `validate-config` as part of deploy checks
- using `backfill` rather than ad hoc manual sink repair

## License

See [LICENSE](/home/wolfred/projects/trenova-2/services/gtc/LICENSE).
