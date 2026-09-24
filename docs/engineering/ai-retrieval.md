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

## Providers

Retrieval is off until an organization routes the **Embedding** task to a provider that can
embed. There is no default embedding provider; without one, every search falls back to keywords.

### The Embedding task

`aiprovider.TaskEmbedding` is routed like every other task, with three differences the domain
enforces (`Provider.validateEmbedding`, and the `ck_ai_providers_embedding_*` check constraints
from migration `20261231006300`):

- **The protocol must have an embedding endpoint.** `Kind.SupportsEmbedding()` is false for
  `AnthropicMessages`, so an Anthropic provider cannot be assigned the task, and the router skips
  one that somehow was (`CanServeTask`).
- **An embedding model serves nothing else.** A provider row is one model, and a model either
  writes text or returns vectors, so Embedding cannot share a provider with another task.
- **The dimension is required** and is one of 768, 1024 or 1536 — the sizes the vector indexes
  are built for (`aiprovider.AllowedEmbeddingDimensions`).

`Task.Generates()` is false for Embedding: `SamplingForTask` returns no sampling for it (an
explicit exemption, checked by `task_wiring_test.go`), and it is priced on input tokens alone.
`WritesToLedger()` stays false.

### Provider fields

| Field | Column | Meaning |
|---|---|---|
| `EmbeddingDimensions` | `embedding_dimensions` (nullable) | Vector size the model returns. A reply of any other size is refused. |
| `EmbeddingInputStyle` | `embedding_input_style` (default `None`) | How a stored document is told from a search query on the wire. |

Input styles:

| Style | Document | Query |
|---|---|---|
| `None` | sent as is | sent as is |
| `VoyageInputType` | `input_type: "document"` | `input_type: "query"` |
| `NomicPrefix` | `search_document: ` prefix | `search_query: ` prefix |

Both are on the REST save request (`POST /ai-providers/`, `PUT /ai-providers/:providerID/`,
which replace the whole provider) and on the `AIProvider` GraphQL type (`embeddingDimensions: Int`,
`embeddingInputStyle: AIEmbeddingInputStyle!`). `GET /ai-providers/catalog/` lists the allowed
sizes (`embeddingDimensions`), the styles (`embeddingInputStyles`), whether each protocol
`supportsEmbedding`, and the embedding presets.

### Model key

`Provider.EmbeddingModelKey()` is `host/model@dimensions`, for example
`api.voyageai.com/voyage-3.5@1024`. Two vectors are comparable only when their model keys match,
so the key is stored beside every vector, and failover never crosses it: a second provider for
the same model on the same host (another key, another region of the same gateway) backs the first
up; a provider for any other model is never used in its place.

### Wire protocols

| Kind | Endpoint | Size parameter |
|---|---|---|
| `OpenAIResponses` | `{base}/v1/embeddings` | `dimensions` |
| `OpenAIChat` | `{base}/embeddings` (Voyage, Gemini's OpenAI-compatible endpoint, vLLM, Together, …) | `dimensions`, or `output_dimension` with `VoyageInputType` |
| `Ollama` | `{base}/api/embed` with an array `input` | `dimensions` |

The OpenAI-shaped adapters merge the provider's extra request fields under the fields the adapter
sets. A server that refuses `dimensions` (vLLM rejects it for a model without Matryoshka
training) is configured with `{"dimensions": null}` in **Extra request fields**: when the extra
body names `dimensions`, the adapter leaves it to the extra body. The reply is still checked
against the configured size.

Replies are ordered by their `index` (arrival order when a server omits it). A reply whose vector
count or size is wrong, or that holds a non-finite component, is a `BusinessError` wrapping
`ErrEmbeddingDimensionMismatch` or `ErrEmbeddingResponseInvalid`: a configuration error that is
never retried (`modeladapter.IsRetryable` is false for business errors, and `modelcall.Classify`
maps them to `ModelRejected`).

### Routing

`completionrouter.Service` implements `serviceports.EmbeddingService` alongside
`CompletionService`, sharing candidate resolution (`usableFor`), health resting (`awake`), the
retry loop (`retrying`, which honours `Retry-After` within the busy budget), and usage recording.

```go
type EmbeddingService interface {
	Embed(ctx context.Context, req EmbedRequest) (EmbedResult, error)
	ConfiguredModelKey(ctx context.Context, tenant pagination.TenantInfo) (string, error)
}
```

- `EmbedRequest.Purpose` is `Document` (indexing) or `Query` (search). It drives the input
  style, PII scrubbing and the usage surface.
- `EmbedRequest.ModelKey` pins the model. Leave it empty to use the highest-priority usable
  embedding provider's key; pass the organization's active key when searching an index, so a
  query is never embedded by a model the index was not built with. A pinned key no enabled
  provider serves fails with `ErrNoProviderConfigured`.
- `Query` calls run under `EmbedRequest.QueryTimeout`, default
  `serviceports.DefaultQueryEmbeddingTimeout` (1.5 s). A query that misses it returns an error
  wrapping `context.DeadlineExceeded`, and the caller falls back to keywords.
- Inputs are batched: at most 96 inputs or about 60,000 estimated tokens per request (100 inputs
  for Gemini's endpoint), at most 10,000 inputs per call. Each batch fails over independently
  among the same-key providers; `EmbedResult.Vectors` keeps input order.
- `ConfiguredModelKey` reports the key the next unpinned call would use, or
  `ErrNoProviderConfigured` when no provider embeds.

### What is sent

`Document` inputs pass through `shared/piiscrub` before they leave: SSNs (formatted, or labelled
as such), Luhn-valid card numbers, IBANs, and labelled bank routing (ABA-checksummed) and account
numbers are replaced with `[SSN]`, `[CARD]`, `[ROUTING]` and `[ACCOUNT]`. Queries are the person's
own words and are sent as typed. Deciding which document text may be embedded at all is the
indexer's job, not the provider's.

### Usage and cost

Every attempt on every batch writes one `ai_usage_records` row with task `Embedding` and surface
`Indexing` (documents) or `Retrieval` (queries, attributed to the run, thread, user and agent the
caller names, so they count against the agent's monthly budget). An evaluation attribution still
records as `Evaluation`. Cost is `InputCostPerMillion` times the input tokens the provider
reported, or the `shared/llmtokens` estimate when it reported none; a provider without an input
price records an unknown cost, never zero.

### Testing a provider

`aiproviderservice.Prober` sends a provider that serves only Embedding one short query-purpose
embedding instead of the JSON-schema probe, and fails the test when the size differs from
`EmbeddingDimensions`. `SchemaHonoured` on the outcome means the vector had the configured shape.

### Configuring one

**AI control → Providers → New provider.** The Voyage AI, Gemini, OpenAI and Ollama embedding
presets fill in the endpoint, model, task, dimension and input style. The Embedding task is shown
disabled, with its reason, for the Anthropic protocol; choosing it opens the **Embedding** section
(**Dimensions**, **Input style**) and hides the settings that only apply to text generation.

## Ranking

Tools and product guide pages are ranked by meaning as well as by words once an organization
can embed a query. Keyword ranking is unchanged underneath, and it is the whole answer
whenever a vector is not available.

### Query vectors

`serviceports.QueryVectorizer` (implemented by `retrievalquery.Service`) turns a search text
into a vector under the organization's active embedding model:

```go
type QueryVectorizer interface {
	Vectorize(ctx context.Context, req QueryVectorRequest) (QueryVector, error)
	Availability(ctx context.Context, tenant pagination.TenantInfo) (airetrieval.Availability, error)
}
```

`Vectorize` embeds the text (trimmed, at most `MaxQueryTextRunes` = 2000 runes) with purpose
`Query`, the model pinned to `active_model_key`, and the router's 1.5 s query budget. The
usage row is surface `Retrieval`, attributed to whatever `QueryVectorRequest.Attribution`
names. When retrieval cannot answer, it returns `Available=false` with a reason and a **nil
error**, and the caller ranks by keyword:

| Reason | When |
| --- | --- |
| `ExtensionMissing`, `SchemaMissing`, `TooOld` | the storage probe says so |
| `Disabled` | `ai_retrieval_settings.paused` with reason `Manual` |
| `BudgetPaused` | paused with reason `Budget` |
| `NoProvider` | no embedding service, no enabled provider for the task, or none serving the active key |
| `NotIndexed` | a provider exists but the settings have no `active_model_key` yet |
| `QueryTimeout` | the provider missed the query budget |
| `ProviderFailed` | any other provider failure, or a reply of the wrong size or model |

A timeout or provider failure is logged. Only a blank text, a missing tenant, a failed read of
the storage probe or the settings, and the caller's own cancellation come back as errors.
`UnavailableReason.Transient()` is true for `QueryTimeout` and `ProviderFailed`.

Successful vectors are kept in-process for 15 minutes (1024 entries), keyed by organization,
business unit, model key and the text's SHA-256. Asking twice in one activity embeds once.
`Availability` is the same check without embedding, for status screens.

### One vector per turn

A turn embeds its request once, in the activity that opens it (`OpenTurn`, called by the
chat prepare activity, `OpenRunActivity`, the delegate opening and the evaluation replay).
The text is `serviceports.TurnQueryText`: the new message and the person's previous one,
which is also what keyword preselection ranks.

The vector rides on the turn as data: `ToolSetState.Query` holds the model key, the
dimensions and the vector packed as little-endian float32 (4 KB for 768 dimensions, 8 KB
base64 in the payload). `FindToolsActivity` reads it back from its input, so `find_tools`
ranks with the same vector on any worker, a retried activity never embeds again, and a
replay reads the recorded vector. The alternative, recomputing in each activity with a
per-turn in-process cache, would re-embed on every cache miss (another worker, a restart),
charge the turn again, and could rank differently if the provider changed in between. A turn
whose state has no vector (opened before this release, or retrieval unavailable) ranks by
keyword. Only a disclosed turn (more than 12 tools) embeds anything.

**Hook for other rankers.** Anything ranking by the turn's request in the same activity asks
the vectorizer for the same text and gets the memoized vector:

```go
vector, err := vectorizer.Vectorize(ctx, serviceports.TurnQueryRequest(runRequest))
// or, from the runtime itself:
vector := runtime.TurnQueryVector(ctx, runRequest)
```

`TurnQueryRequest` builds the tenant, text and attribution (user, agent, thread, run,
evaluation purpose) from a `RunRequest`. Memory ranking in the context builder can use it:
the context is built in the same activity as the turn, so the turn's own vectorize is a
cache hit.

### Catalog embeddings

`serviceports.CatalogVectorIndex` (`retrievalquery.CatalogIndex`) compares a query vector
with a fixed corpus kept in `ai_catalog_embeddings`:

| Corpus | Item key | Text |
| --- | --- | --- |
| `Tools` | tool name | `agenttoolcatalog.DescriptorText`: the name in words, the description, the search terms |
| `ProductGuide` | page path | `productguideservice.PageText`: name, location, description, summary, aliases, task titles |

Each item is keyed by the SHA-256 of its text, so an edited description is a new row and is
embedded again; nothing else is. On the first request for a model key and corpus, the index
reads the stored rows, embeds what is missing (purpose `Document`, surface `Indexing`, the
same pinned model key, paid by the organization whose request found it missing), stores it
with `PutCatalogEmbeddings`, and keeps the unit vectors in memory (16 model-key and corpus
pairs). From then on a turn reads nothing. A request waits at most 5 s for that load, then
ranks by keyword while the load finishes in the background (45 s budget); a failed embed
backs off for a minute. The first complete load in a process prunes rows whose hash is no
longer current.

### Hybrid ranking

`agenttoolcatalog.Catalog.RankHybrid` (preselection) and `FindHybrid` (`find_tools`) take an
optional `Semantic` (similarity per tool). With none they are `Rank` and `Find`, byte for
byte. With one:

1. The keyword leg is the keyword ranking of the tools that matched at all (`Find` keeps its
   half-of-the-best cutoff).
2. The vector leg is the tools by similarity, at most `max(4 × limit, 24)`.
3. The legs are fused by reciprocal rank (`shared/rankfusion`, k = 60); ties break on
   catalog position.
4. A tool found only by meaning is dropped below the similarity floor
   (`serviceports.DefaultCatalogSimilarityFloor`, 0.5), so nonsense still finds nothing. A
   keyword hit is kept whatever its similarity.
5. Preselection fills any remaining slots in keyword order, as before; `find_tools` appends
   families after its matches, as before.

The vector is the turn's request, which preselection has already spent, so `find_tools`'
vector leg skips the tools the turn already carries and adds the next closest ones.
`unheldRefusal` runs in workflow code and stays keyword-only.

`find_in_trenova` does the same through `productguideservice`: it embeds the question (the
tool passes the user and agent for attribution), fuses the keyword and vector page rankings,
and drops pages found only by meaning below the floor. A page found only by meaning names no
task unless the question's words match one. A question about one page (`page` given) is
answered from that page alone and embeds nothing.

The floor is one value for every model, a starting point for `nomic-embed-text` that has not
yet been checked against recorded vectors. A model whose similarities run lower overall
mostly loses its vector-only hits, which falls back toward keyword ranking; one whose
similarities run higher lets more through. Tune it with the recorded fixture: the hybrid gate
fails if a nonsense request finds anything, and its log lists every miss.

### Carrying a turn's tools over

`find_tools`' result message records the tools the search found
(`assistant_messages.found_tools`, `conversation.Message.FoundTools`, migration
`20261231006480`). The runtime's answer carries them (`FindAnswer.Found`,
`FindToolsResult.Found`, `ToolOutcome.Found`) into the message the finish activity saves.
`carryOver` reloads those names for the recent turns' searches instead of searching again,
because the ranking now depends on the vector and on the model. A result saved without
names (before this release, or a search that found nothing) is searched again by keyword, as
it always was. `load` still checks every name against what the agent holds and the person
may use, so a recorded name never widens the grant.

None of this took a `GetVersion` gate. The vector and the found names are optional data on
activity inputs and results; no command depends on them, and a recorded history without them
replays with keyword ranking and issues the same commands.

### Evaluation

`agentevalgate` gates ranking offline in the `Deterministic` job:

| Suite | Floors | Gate |
| --- | --- | --- |
| `toolselection.yaml` | `toolselection.floors.json` `keyword` | `TestToolSelectionAgainstFloors` |
| `toolselection.yaml` + `toolselection.paraphrase.yaml` | `toolselection.floors.json` `hybrid` | `TestToolSelectionAgainstFloorsHybrid` |
| `guideselection.yaml` | `guideselection.floors.json` `keyword` / `hybrid` | `TestGuideSelectionAgainstFloors`, `…Hybrid` |

The paraphrase suite holds requests that share no word with their tool's name
(`TestParaphrasesShareNoWordWithTheirTool` checks). Each suite's `nothing` list must find
nothing, by keyword and by meaning.

The hybrid gates rank from `evals/embeddings/nomic-embed-text.json`: Ollama
`nomic-embed-text` vectors (768, `NomicPrefix` style), packed float32 in base64, keyed by
content hash for every tool descriptor and guide page and by SHA-256 for every eval request.
A missing or unused hash fails with the command that re-records it. **The fixture has not
been recorded yet**, so the hybrid gates skip, saying so, and the hybrid floors are absent. To
record it and set the floors, on a machine running Ollama:

```bash
cd services/tms && ollama pull nomic-embed-text && TRENOVA_EVAL_OLLAMA_URL=http://localhost:11434 go test -tags nofitz -count=1 -run 'TestRecordEmbeddingFixture' ./internal/core/services/agentevalgate/ -record && go test -tags nofitz -count=1 -run 'AgainstFloors' ./internal/core/services/agentevalgate/ -update
```

Then commit the fixture and both floors files, and set the repository variable
`TRENOVA_EVAL_REQUIRE_HYBRID` to `true`: with it, a missing fixture fails the job instead
of skipping. The recorder goes through the production Ollama adapter, so the prefixes are
the ones a Nomic provider sends.

## Indexing

Memories, documents and inbound email are embedded in the background, per organization, so a
search never waits for an embedding call on the corpus. The code is
`internal/core/services/retrievalservice` (the indexer and the pipeline) and
`internal/core/temporaljobs/retrievaljobs` (the workflows).

### Marking a source stale

```go
type RetrievalIndexer interface {
	MarkStale(ctx context.Context, tenant pagination.TenantInfo, sourceType airetrieval.SourceType, ids ...pulid.ID) error
	DeleteSource(ctx context.Context, tenant pagination.TenantInfo, sourceType airetrieval.SourceType, id pulid.ID) error
	Reindex(ctx context.Context, tenant pagination.TenantInfo, sourceType airetrieval.SourceType) error
}
```

`MarkStale` upserts a `Pending` outbox entry for each indexed model key (the active one, and
the pending one during a model change), then signals `IndexOrganizationWorkflow` with
SignalWithStart (workflow id `ai-index:<organization>`, signal `retrieval-index-stale`). The
signal is best effort: when Temporal cannot be reached it is logged, the entry stays in the
outbox, and the hourly sweep picks it up. A source type the organization turned off, or an
organization with no active model yet, is not marked. `DeleteSource` drops a source's
entries and embeddings at once; the `AFTER DELETE` triggers already do that for a deleted row.

The writes that mark:

| Source | Where | When |
| --- | --- | --- |
| Memory | `agentmemoryservice` | `Remember` (a new row), `Update`, `SetStatus`, `ApproveSuggestion`, `RecordCorrection` |
| Document | `documentsearchprojectionservice` | `Upsert` marks the document; `Delete` (a superseded version) drops it |
| Inbound message | `inboundmessageservice` | once `ProcessMessage` settles it, or `MarkFailed` puts it in review |

Every path that writes document text goes through the search projection: upload, the
extraction activities (success, failure, async AI extraction), re-extraction, versioning and
moving a lineage. A write that only changes diagnostics in `structured_data` does not, and
the sweep catches any metadata edit through `documents.updated_at`. No hook ever fails the
write that triggered it; it logs and carries on.

### The workflows

All of them run on `system-queue` at priority 4 with the organization as fairness key.

- **`IndexOrganizationWorkflow`** plans (an activity), purges rows of retired model keys, and
  indexes one batch per model key per round (`DefaultBatchSize` 25, a ten-minute lease).
  When every key is drained it completes a pending model change if the pending key is fully
  indexed, then sweeps once more; when nothing is left it waits five minutes for a signal,
  checks once more after the wait (for failures whose retry was short), and ends. It
  continues as new after 200 activities. Every read, chunk, hash and embed is inside an
  activity, and the workflow takes no decision from anything but activity results.
- **`RetrievalIndexSweepWorkflow`**, scheduled hourly (`retrieval-index-sweep`, minute 17),
  fans out over organizations with `temporaljobs.RunTenantFanOut`. Each organization's sweep
  runs `FindStaleSources` for every enabled source type and indexed key (sources changed
  since they were indexed, and sources with no entry yet), marks them, counts entries still
  `Pending` (retries waiting out their backoff), and wakes the indexer when there is work.
- **`ReindexRetrievalSourceWorkflow`** (`ai-reindex:<organization>:<source type>`, started by
  `RetrievalIndexer.Reindex` for the AI Control screen) marks every source of one type stale,
  500 per page. Hash diffing then re-embeds only what changed, so a re-index after a chunker
  change re-embeds everything and one after nothing changed costs nothing.

### Planning a round

`Plan` decides whether the organization can be indexed right now:

1. The vector storage must be available (`VectorAvailability`).
2. At least one source type must be on, and indexing must not be paused by a person
   (`Disabled`).
3. The embedding router must name a configured model key (`NoProvider` otherwise), whose
   dimensions are read from the key (`airetrieval.ModelKeyDimensions`).
4. With no active key, the configured key becomes active. When the configured key differs
   from the active one it becomes pending, and the next rounds index every source under it
   beside the old rows (searches keep using the active key). Configuring the active key
   again drops the pending one.
5. The month's `Indexing` cost (`AIUsageRepository.SurfaceCost` since the first of the month,
   UTC) is compared with `monthly_indexing_budget_usd`. At or over it, indexing is paused
   with reason `Budget` and the workflow stops; a later round under the budget (a new month,
   or a raised budget) lifts that pause. A pause a person set is never lifted here.
6. Model keys that still have rows but are neither active nor pending are listed for purging.

`CompleteModelChange` swaps the pending key in only when no source of an enabled type is
stale under it and none of its entries is `Pending`; `PurgeRetiredModel` then deletes the old
key's embeddings and entries a thousand rows at a time, twenty batches per activity, until
the repository reports it done.

### A batch

`IndexBatch` rechecks the settings and the budget, claims entries of the enabled source types
(`ClaimIndexEntries`), and for each one:

1. reads the live row: a missing row is dropped (`DeleteSource`); a retired or expired memory,
   a superseded or rejected document, a document whose owning record's default sensitivity
   is above **Internal** (or whose owner is not a registered resource), and a document with
   no text read yet are **skipped**: their embeddings are removed and the entry says why;
2. chunks it with the versioned chunker for its type;
3. compares each chunk's hash with `ListChunkHashes` and embeds only the chunks that changed,
   in one `EmbeddingService.Embed` call per batch (purpose `Document`, surface `Indexing`, the
   batch's model key pinned);
4. writes the source with `ReplaceChunks` (unchanged chunks are passed without a vector) and
   records the outcome for the generation it claimed (`MarkIndexed`, `MarkSkipped`,
   `MarkFailed`).

A failed embed marks the batch's entries `Failed` with a retry: one minute, doubling, at most
six hours, and after eight attempts no more retries. A business error (a wrong dimension, a
refused request) is not retried at all. After the batch, when the month's cost has reached the
budget, the organization is paused and the workflow stops; what was already paid for is kept.

### Chunking

Every chunk's hash is SHA-256 of the chunker version and the chunk text, so bumping a version
re-embeds that source type and nothing else.

| Source | Version | Chunks |
| --- | --- | --- |
| Memory | `memory/1` | one: `[Kind] about: content` |
| Document | `document/1` | an optional first chunk of the extracted fields (sorted, flattened, at most two windows long), then each page of `document_content_pages` (or the content text when there are no pages) in windows of about 350 tokens with 15% overlap, each under a header naming the file, its kind, what it is attached to and the page; at most 200 chunks |
| Inbound message | `email/1` | the sender's own words (`stringutils.MailBody`: quoted lines, a quoted reply, a forwarded header and a signature removed) in the same windows under a header naming the sender and subject; at most 20; a message with no words of its own is one chunk of its header |

Tokens are estimated with `shared/llmtokens` (four runes per token). A single word longer than
two windows is cut.

### What is sent

Memory content is embedded. Document text (and the extracted fields) is embedded only when
the owning record type's default sensitivity is at most Internal, from the permission
registry the query tools read field sensitivity from: a shipment's rate confirmation is; a
worker's medical card is not, and is only ever found by its words. Email subject, sender and
own words are embedded; the quoted thread never is. Everything sent passes the provider's
PII scrubbing (`shared/piiscrub`).

## Search

`retrievalservice.Searcher` answers `search_documents`, `search_inbound_messages` and the
vector side of `recall_memory` and memory ranking:

```go
type RetrievalSearcher interface {
	SearchDocuments(ctx context.Context, req RetrievalSearchRequest) (*DocumentSearchResult, error)
	SearchInboundMessages(ctx context.Context, req RetrievalSearchRequest) (*InboundMessageSearchResult, error)
}

type MemoryVectorSearcher interface {
	SimilarMemories(ctx context.Context, req SimilarMemoriesRequest) (SimilarMemories, error)
}
```

`RetrievalSearchRequest` carries the tenant, the query, the limit (default 5, at most 50;
the tools cap it at 10), the usage attribution and a `RetrievalAccess`, which is required: a
search that cannot say who is asking is refused.

### Two legs

- **Keyword.** Documents match `documents.search_vector` (file name, description, tags; the
  `simple` configuration) or `document_contents.search_vector` (the text and detected kind;
  `english`), current versions only and never a rejected one. Inbound mail matches the new
  generated `inbound_messages.search_vector` (migration `20261231006400`): the subject
  (weight A), the sender (B) and `mail_own_words(text_body)` (C), a SQL function that strips
  the quoted history the same way `stringutils.MailBody` does, with a GIN index. Any word of
  the query may match (the `websearch_to_tsquery` conjunctions become disjunctions, a
  negation stays a negation) and `ts_rank_cd` orders the result. SQLite falls back to an
  escaped `LIKE` on any word.
- **Vector.** When the source type is on and the query can be embedded
  (`QueryVectorizer.Vectorize`), `AIRetrievalRepository.Search` returns the best chunk per
  source under the active key, tenant, source type and dimensions in one `WHERE`.

Both legs ask for four times the limit (at least 20 candidates) and run concurrently. They are
fused by reciprocal rank (`shared/rankfusion`, k = 60); ties go to the keyword rank. A source
found only by meaning must clear the similarity floor (`DefaultCatalogSimilarityFloor`, 0.5,
shared with catalog ranking and tunable with `Searcher.WithTuning`). Each hit says how it was
found: `words`, `meaning` or `both`. When the vector leg cannot run the result says why
(`RetrievalSemantics.Reason`) and the search is keyword only; the tools say "words only" and
the reason.

### Filtering before anything reaches the model

1. **Tenant** in SQL on both legs.
2. **Every hit is joined back to its live row**; a hit whose row is gone, superseded or
   rejected is dropped.
3. **Read access.** The tool's policy gates the whole call on `Document` or
   `InboundMessage`; the searcher checks that resource again, then each document's **owning
   record** (`document.OwnerResource` maps `resource_type` to a permission resource:
   `shipment`, `Worker`, `invoice_adjustment` → invoice, `assistant_thread` → assistant) with
   the record id, so a grant scoped to the caller's own or their team's records is honoured
   per row. An owner that is not a registered resource is never returned.
4. **Field ceiling.** A passage is built only when the caller may see the document's text
   field and the owning record's default sensitivity is within their ceiling
   (`fieldAccess`, where nothing Confidential ever reaches a model and an agent reads at
   Internal); the file name only when its field is visible; a message's subject, sender and
   passage each by their own field.
5. **Taint.** Both tools read outside text (`ExternalReadAlways`, sources `document` and
   `inbound_message`) and name every record they return (`TaintedRecords()`), so the turn
   gets one mark per record, or one for the call when nothing came back.

The passage is the chunk that matched: the vector's chunk for a hit found only by meaning,
otherwise the chunk with the most query words, cut to 600 characters around the first of
them. A document's page comes with it.

Output goes through the runtime's `<untrusted_data>` fence and its 12,000-character limit like
every tool result.

### The tools

| Tool | Returns |
| --- | --- |
| `search_documents` | up to 10 (default 5): `documentId`, `fileName`, `looksLike`, `page`, `snippet`, `attachedTo`, `match`, and `searchedBy` for the whole result |
| `search_inbound_messages` | up to 10 (default 5), best first: `id`, `receivedAt`, `from`, `subject`, `snippet`, `status`, `needsReview`, `classification`, `match` |
| `get_document_summary` | now takes `page` to read one page, reports a failed content read as an error instead of "nothing read yet", and refuses a document whose owning record the caller may not read |
| `list_inbound_messages` | unchanged ordering (newest first); an unknown `status` or `classification` is now refused with the values it accepts instead of being dropped |

The intake desk template carries both search tools.

### Memories

`recall_memory` fuses WP4's tsvector keyword leg with a vector leg: `SimilarMemories` returns
up to four times the limit of memory ids by similarity (embedding the query, or reusing a
vector the caller already has); ids found only by meaning are read back through the same
repository search with the same scope, agent, activity, expiry, kind, subject and tool
filters, so a memory kept for another agent, retired or expired is never recalled by meaning
either. Rows gain `match` (`words`, `meaning` or `both`).

The prompt's memory context ranks by meaning too. Chat turns, delegated tasks and background
runs name the turn's text on `RuntimeContextRequest.Query` (`serviceports.ContextQuery`); the
context builder embeds it (the same text the turn itself embeds, so the vectorizer's cache
answers the second ask) and passes the vector on `MemoryContextRequest.Query` to the
`MemoryRanker`. `retrievalservice.MemoryRanker` fuses the similarity ranking of the
candidates (those above the floor) with WP4's recency-and-use ranking by reciprocal rank,
and falls back to recency and use when there is no vector.

### Evaluation

| Suite | Floors | Gate |
| --- | --- | --- |
| `evals/documentretrieval.yaml` | `documentretrieval.floors.json` `keyword` / `hybrid` | `TestRetrievalAgainstFloors/search_documents` |
| `evals/inboxretrieval.yaml` | `inboxretrieval.floors.json` | `TestRetrievalAgainstFloors/search_inbound_messages` |
| `evals/memoryretrieval.yaml` | `memoryretrieval.floors.json` | `TestRetrievalAgainstFloors/recall_memory` |

The suites are synthetic corpora with relevance judgements, scored by recall@5, MRR and
nDCG@10 (`agentevalgate.EvaluateRetrieval`). They run against Postgres in the integration job
(`retrievalservice/eval_integration_test.go`, build tag `integration`): the keyword leg
always, the hybrid leg when pgvector is present and
`evals/embeddings/retrieval-nomic-embed-text.json` has been recorded (chunk vectors keyed by
chunk hash, query vectors by SHA-256). `TestRetrievalNeverLeaks` and the tools'
`TestSearchToolsNeverLeakAndMarkEveryRecord` seed the same corpus for a second organization,
on worker records the caller may not read, and as memories kept for another agent, retired or
expired, and fail on any of them coming back by either leg, on a returned record without a
taint mark, or on quoted mail history being searchable.

**Neither the fixture nor the floors have been recorded yet.** Until they are, the floor
checks log their measurements and pass (set `TRENOVA_EVAL_REQUIRE_FLOORS=true` to fail
instead, and `TRENOVA_EVAL_REQUIRE_HYBRID=true` to fail without the fixture); the leak tests
always gate. To record:

```bash
cd services/tms && ollama pull nomic-embed-text && TRENOVA_EVAL_OLLAMA_URL=http://localhost:11434 go test -tags nofitz -count=1 -run 'TestRecordRetrievalEmbeddingFixture' ./internal/core/services/retrievalservice/ -record
task test-db-image && TRENOVA_TEST_POSTGRES_IMAGE=trenova-postgres:local go test -tags 'integration nofitz' -count=1 -run 'TestRetrievalAgainstFloors' ./internal/core/services/retrievalservice/ -update
```
