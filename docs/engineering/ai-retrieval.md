# Semantic Retrieval

Agents find memories, documents and inbound email by meaning as well as by words. Keyword
search is the fallback everywhere: an installation without pgvector, or an organization with
no embedding model routed, keeps working exactly as before. This page describes the pieces
and how they fit. Each work package adds its own section.

## Storage

### The Postgres image

pgvector is compiled into the database image, not installed from a package:
`deploy/Dockerfile.postgres` is `postgis/postgis:18-3.6-alpine`, then pg_cron and pgvector
**v0.8.1** built from source. pgvector's Makefile defaults to `-march=native`, which ties the
binary to the CPU that built it, so it is built with `make with_llvm=no OPTFLAGS=""` (the
same portable flags pg_cron uses). `docker-compose-local.yml` builds this file, so local
development has pgvector once the `db` image is rebuilt
(`docker compose -f docker-compose-local.yml up -d --build db`).

0.8 is the minimum: it is the first release with iterative index scans
(`hnsw.iterative_scan`), which a filtered HNSW search needs to return `k` rows after the
tenant filter has discarded most candidates.

### Tests and CI

Every test harness runs Postgres 18, the same major as the image. The throwaway container a
harness starts comes from `TRENOVA_TEST_POSTGRES_IMAGE` and defaults to plain PostGIS,
which runs every test on the keyword-only fallback path. To run the vector tests locally:

```bash
cd services/tms
task test-db-image                                   # builds trenova-postgres:local
task test-db-down                                    # a running test server keeps its image
TRENOVA_TEST_POSTGRES_IMAGE=trenova-postgres:local task test-integration
```

Vector tests call `requireVector` and skip when the capability is missing.
`TRENOVA_TEST_EXPECT_VECTOR=true` turns that skip into a failure, so a job that is meant to
exercise pgvector cannot pass by skipping.

`.github/workflows/test-tms.yml` never publishes an image:

| Job | Postgres | Proves |
| --- | --- | --- |
| Integration Tests | `deploy/Dockerfile.postgres`, built in the job with buildx (`load: true`, GHA layer cache scope `trenova-postgres`) and started with `docker run` | every integration test, with `TRENOVA_TEST_EXPECT_VECTOR=true` |
| Integration Tests (no pgvector) | `postgis/postgis:18-3.6-alpine` service | every migration applies without the extension and storage reports itself unavailable (`TRENOVA_TEST_EXPECT_VECTOR=false`) |
| Race Detection (nightly) | `postgis/postgis:18-3.6-alpine` service | the fallback path under the race detector |

### Migrations

Two migrations, because the second one is also run on its own:

- `20261231006100_ai_retrieval` always applies. It tries `CREATE EXTENSION vector` inside a
  `DO` block that swallows `undefined_file`, `insufficient_privilege` and
  `feature_not_supported`, then creates the tables that need no extension:
  - `ai_index_entries`, the outbox (see below);
  - `ai_retrieval_settings`, one row per organization;
  - statement-level `AFTER DELETE` triggers on `agent_memories`, `documents` and
    `inbound_messages` that delete a source's outbox rows in the same statement.
- `20261231006110_ai_retrieval_vector` is one `DO` block that returns early unless pgvector
  0.8 or newer is installed, and otherwise `EXECUTE`s the vector DDL:
  - `ai_embeddings` and `ai_catalog_embeddings`;
  - one partial HNSW index per allowed dimension;
  - statement-level `AFTER DELETE` triggers on the same three source tables that remove
    their embeddings.

  Everything in it is idempotent, and it must stay one statement (no `--bun:split`).

Rolling back `006110` drops the vector tables but leaves the extension installed. Other
objects may depend on it, and removing it is an operator's decision.

### Installing pgvector after the migrations ran: `trenova db enable-vector`

A server that ran its migrations without pgvector has the outbox and settings tables but no
vector tables. After installing the extension, run:

```bash
trenova db enable-vector            # create/upgrade the extension, then the vector schema
trenova db enable-vector --dry-run  # report the current state only
```

The command runs `migrations.EnableVector`:

1. `CREATE EXTENSION IF NOT EXISTS vector`, which fails loudly (for example, without
   privilege) instead of being swallowed.
2. `ALTER EXTENSION vector UPDATE` when the installed version is older than 0.8.
3. The `006110` file itself, read from the embedded migrations, so the command and the
   migration never diverge.

It is safe to run twice. Running services notice within five minutes, because a
capability probe that found the vector tables missing checks again after five minutes.

### Capability probe

`dbdialect.CapVectorSearch` is the one capability that cannot be read off the driver.
`dbdialect.Kind.Supports` always answers false for it. The answer comes from
`postgres.CapabilityProbe`, which runs at startup (an `fx.Invoke` in the database module) and
reads `pg_extension` and `to_regclass('ai_embeddings')` in one query:

| Probe state | `airetrieval.UnavailableReason` | Operator fix |
| --- | --- | --- |
| Ready | available | none |
| ExtensionMissing | `ExtensionMissing` | install pgvector, then `trenova db enable-vector` |
| ExtensionTooOld | `TooOld` | install pgvector 0.8 or newer, then `trenova db enable-vector` |
| SchemaMissing | `SchemaMissing` | `trenova db enable-vector` |
| SQLite | `ExtensionMissing` | none; SQLite never has vector search |

`NoProvider`, `Disabled` and `BudgetPaused` come from the organization's settings and
routing, not from the database. The retrieval service reports them.

The repository's vector methods check the probe first. When the capability is absent they
return `*airetrieval.UnavailableError`, which matches `errors.Is(err,
airetrieval.ErrVectorUnavailable)`, and they never touch the missing tables. Outbox,
settings and `DeleteSource` keep working without the extension.

### Tables

**`ai_embeddings`**: one row per chunk and embedding model.

- Primary key: `(organization_id, business_unit_id, source_type, source_id, chunk_index,
  model_key)`.
- Other columns: `dimensions`, `content_hash` and an unconstrained `halfvec` embedding.
  CHECKs pin `dimensions` to 768, 1024 or 1536 and require `vector_dims(embedding) =
  dimensions`.
- There is one partial index per dimension:

  ```sql
  CREATE INDEX idx_ai_embeddings_hnsw_768 ON ai_embeddings
      USING hnsw ((embedding::halfvec(768)) halfvec_cosine_ops) WHERE dimensions = 768;
  ```

  A query only reaches it when it casts the same way and states the dimensions as a literal.
  `airetrievalrepository.Search` builds the query that way, and an integration test asserts
  from `EXPLAIN` that the index is chosen.

A search runs in a read-only transaction with `SET LOCAL hnsw.iterative_scan =
relaxed_order` and `hnsw.ef_search` set to at least the number of candidates. The inner
query filters tenant, source type, model key and dimensions in one `WHERE`, orders by
cosine distance and fetches `limit × 4` candidates. The outer queries keep the best chunk per
source (`DISTINCT ON`), order strictly by distance (relaxed order can return rows slightly
out of order) and apply the optional similarity floor.

`ReplaceChunks` writes a source's chunks for one model in a transaction:

- it deletes chunk indexes that are no longer present;
- it skips any chunk whose stored `content_hash` and dimensions already match, so an
  unchanged chunk is never embedded or written again;
- it upserts the rest with a `DO UPDATE ... WHERE` hash guard.

A chunk passed without a vector must already be stored with the same hash. Callers read
`ListChunkHashes` first, so they only embed what changed.

**`ai_catalog_embeddings`**: tool catalog and product guide embeddings, which are the same
for every organization. Primary key `(model_key, corpus, item_key, content_hash)`, filled
lazily per model key. `PruneCatalogEmbeddings` drops hashes that are no longer current.

**`ai_index_entries`**: the outbox, one row per source and model key.

- `status` is `Pending`, `Indexed`, `Failed` or `Skipped`.
- `generation` rises each time the source is marked stale.
- Indexer bookkeeping: `attempts`, `chunk_count`, `last_error`, `lease_expires_at`,
  `next_attempt_at`, `last_attempt_at`, `indexed_at`.

The outbox works like this:

- `MarkStale` upserts `Pending` and bumps the generation. It keeps any active lease, so two
  indexers never work on one source at once.
- `ClaimIndexEntries` leases a batch with `FOR UPDATE SKIP LOCKED`.
- `MarkIndexed`, `MarkFailed` and `MarkSkipped` apply only to the generation that was
  claimed. A source written while it was being indexed stays `Pending`, and those rows come
  back in `Superseded`.
- `MarkFailed` with a `RetryAt` keeps the entry `Pending` until then.
- `FindStaleSources` is the hourly sweep. It compares each source's `updated_at` with
  `indexed_at` (or `last_attempt_at` for failures), and also finds sources that have no
  entry yet.

**`ai_retrieval_settings`**: one row per organization. An organization without a row reads
`airetrieval.DefaultSettings`.

- Per-source toggles: `memory_enabled`, `documents_enabled`, `inbound_messages_enabled`.
- `monthly_indexing_budget_usd`.
- `paused`, `paused_reason` (`Manual` or `Budget`) and `paused_at`.
- The model: `active_model_key` and `dimensions`, `pending_model_key` and
  `pending_dimensions`.

### Changing the embedding model

A model key is the provider host, the model and the dimensions, so a different dimension
is a different model. To change models:

1. Set `pending_model_key` and `pending_dimensions` in the settings.
2. Index every source under the pending key. `MarkStale` takes both keys
   (`Settings.IndexedModelKeys`), and the new rows sit beside the old ones. Searches keep
   using the active key.
3. When everything is indexed, `SwapModel` promotes the pending key atomically. It refuses
   when the pending key moved in the meantime, and returns the retired key.
4. Call `PurgeModel` with the retired key repeatedly until `Done()`. Each call deletes one
   bounded batch of embeddings and outbox rows. It refuses a key that is still active or
   pending.
