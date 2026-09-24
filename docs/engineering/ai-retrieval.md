# AI retrieval

How agents find memories, documents and mail by meaning as well as by keyword.

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
