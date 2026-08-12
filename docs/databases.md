# Database support

PostgreSQL is the reference database for Trenova. The schema, the query layer
and every performance decision are built around it, and it is the only supported
production target.

SQLite is supported as a **development-only** convenience so contributors can run
the stack without a PostgreSQL/PostGIS toolchain. It is not a supported way to
run Trenova for real freight.

## Choosing a driver

```yaml
database:
  driver: sqlite # postgres | sqlite
  sqlite:
    path: trenova.db
    journalMode: WAL
    synchronous: NORMAL
    busyTimeout: 5s
    foreignKeys: true
    cacheSizeKb: 64000
```

The SQLite driver is `modernc.org/sqlite`, a pure-Go implementation, so no cgo
toolchain is required. Pragmas travel in the DSN so every pooled connection gets
them, and write transactions open with `BEGIN IMMEDIATE` so `busy_timeout`
actually applies instead of surfacing spurious lock errors.

## How the SQLite schema is produced

The SQLite migrations in `services/tms/internal/infrastructure/sqlite/migrations`
are **generated** from the PostgreSQL migrations by
`scripts/dialect-convert/convert.py`:

```bash
task sqlite-convert          # regenerate and verify against a scratch database
task sqlite-convert-report   # regenerate and write a per-statement report
```

The converter is build-time tooling, not part of the application binary. Adding
another dialect means adding a profile to `scripts/dialect-convert/profiles.py`,
not writing a second converter — `profiles.py` already carries a starting sketch
for SQL Server.

Current state: **1471 of 1471 translated statements apply cleanly, producing 273
tables.**

### What the converter translates

| PostgreSQL | SQLite |
|---|---|
| `CREATE TYPE ... AS ENUM` | `TEXT` (values enforced by the Go domain layer) |
| `CREATE DOMAIN` | the underlying base type |
| `jsonb` / `json` | `TEXT` |
| `varchar(n)`, `uuid`, `timestamptz`, ranges | `TEXT` |
| `bigint`, `smallint`, `boolean`, identity columns | `INTEGER` |
| `numeric(p,s)` | `NUMERIC` |
| `bytea` | `BLOB` |
| `EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint` | `(unixepoch())` |
| `now()` | `CURRENT_TIMESTAMP` |
| `TRIM(BOTH FROM x)` | `TRIM(x)` |
| `char_length` | `length` |
| materialized views | plain views (recomputed per query) |
| `GENERATED ALWAYS ... STORED` added via `ALTER` | `VIRTUAL` generated column |

Enum values are deliberately **not** enforced with `CHECK` constraints. Later
migrations extend enums with `ALTER TYPE ... ADD VALUE`, and SQLite cannot alter
a `CHECK`, so a generated constraint would reject values the application
legitimately writes.

### What the converter drops

Every dropped statement is reported by reason rather than silently discarded.

| Dropped | Why |
|---|---|
| `CREATE EXTENSION` | no extension system |
| PL/pgSQL functions, triggers, `DO` blocks | no procedural language |
| GIN / GiST indexes | index methods do not exist |
| `tsvector` columns and their generated expressions | no full-text vectors |
| PostGIS `geography` / `geometry` columns | no spatial types |
| `EXCLUDE USING gist` constraints | no exclusion constraints |
| `ALTER TABLE ... ADD CONSTRAINT` | SQLite cannot add constraints after creation |
| `ALTER TABLE ... ALTER COLUMN` | SQLite cannot alter column definitions |
| `ALTER TABLE ... DROP COLUMN` | rejected whenever an index or `CHECK` still references the column |
| `CREATE STATISTICS`, publications, RLS policies | no planner statistics or replication objects |

The practical consequence: **a SQLite database has weaker integrity guarantees
than the Postgres one.** Constraints that arrived through later `ALTER`
statements are absent, so SQLite will accept some rows Postgres rejects. Test
data-integrity behaviour against Postgres.

## Feature differences at runtime

Capabilities are declared in `pkg/dbdialect` and checked at the call site.

### Unavailable on SQLite — returns HTTP 501

| Feature | Affected area |
|---|---|
| PostGIS geometry writes | location geofences, weather alert geometry |
| PostGIS distance | dispatch console average deadhead (reported as null) |

These return `errortypes.NotImplementedError`, which carries the driver and
capability name, rather than failing with a raw SQL error.

### Degraded on SQLite — narrower but correct

| Feature | Behaviour |
|---|---|
| Shipment comment search | `LIKE` substring match instead of ranked full-text search |
| Document content search | `LIKE` across file name, original name and extracted text; no relevance ranking |

`LIKE` returns a subset of what full-text search would find — matches are
correct, coverage is narrower. Stemming, ranking and multi-word queries are lost.

### Not available at all

- **Change data capture.** `services/gtc` reads the PostgreSQL write-ahead log
  via logical replication. There is no SQLite equivalent, so Meilisearch indexes
  are never populated when running on SQLite.
- **Exact decimal money.** Postgres stores charge columns as `numeric(19,4)`.
  SQLite's `NUMERIC` affinity falls back to floating point for values it cannot
  represent exactly, so money arithmetic can drift. **Never validate billing or
  settlement behaviour on SQLite.**
- **Advisory locks, `SELECT ... FOR UPDATE`, `SET LOCAL lock_timeout`.** The
  connection layer skips these on SQLite; concurrency is serialised by SQLite's
  single-writer model instead.

## Error handling

`pkg/dberror` maps SQLite result codes onto the SQLSTATE codes the repositories
already check, so unique, foreign key, not-null and check violations — and the
optimistic concurrency retry path — behave identically on both drivers. `SQLITE_BUSY`
maps to `serialization_failure` so the existing retry logic engages.

## Running the checks

```bash
task sqlite-convert   # regenerate the migration set and apply it to a scratch DB
task test-sqlite      # apply the committed migration set through bun's migrator
go test ./pkg/dbdialect/... ./pkg/dberror/... ./internal/infrastructure/config/...
```
