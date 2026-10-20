# Agent reads through GraphQL

Conversion lands one module per pull request. The first one carries this design and the shared
machinery. The list tools convert only after the cursor-paging fix (`after`/`nextCursor`) has
merged, because it rewrites the same files. Design decisions were delegated to the orchestrating
session of the Desk harness review; each is recorded under "Decisions" below.

The read tools keep their names, policies and `dispatch` path. GraphQL is what runs *underneath*
a tool, which is the shape code mode chose ([agent-code-mode.md](agent-code-mode.md), §1): its SDK
is generated from the tool registry, so it picks up the typed parameters with no change, and a
script's reads run through the same operations as the model's.

An agent's list and get tools read through the same safelisted, persisted GraphQL operation the
web client sends for the same records, run in-process as the person (or the unattended agent)
through the gqlexec executor. Typed parameters derived from that operation's filter and sort
inputs replace the generic `filters` array. One path then decides, for the screen and the agent
alike:

- permissions;
- data scopes;
- field gating;
- row-level security;
- paging;
- the operation-cost budget.

A tool can never see or filter what the person's screen cannot.

This is Phase 3's "type the filters" item ([desk-harness-review.md](../desk-harness-review.md), §4
item 4 and §5), widened by the owner's direction to "use GraphQL with everything".

## Why

Today a read tool reaches a repository or service directly (`agentquerytoolservice`, 176 `list`
and `get` tools). That gives four problems.

**Two authorization paths.** The tool checks `resource:read` once in `dispatch`, then reads with
whatever the repository allows. The resolver behind the same screen checks its own permission:
sometimes a narrower resource, sometimes a team scope (`RequireTeamScope`,
`RequirePTOTeamScope`). The two drift.

**Two field gates.** `@protected` fields are nulled by `FieldGuardExtension` on GraphQL. The tools
blank fields themselves, through `fieldGate`, `redactor` and `withhold`.

**Prose filters.** The 37 generated list tools take
`filters: [{field, operator, value, values, days}]`. The fields, their values and the date rule are
written into the description, so:

- the model learns the vocabulary by reading;
- an unknown field or a wrong operator is caught only after the call;
- the catalog in `listcatalog*.go` is a second, hand-kept copy of what the table's filter
  configuration already says.

**No shared budget.** A tool that pages through a large list is not charged against the person's
operation-cost budget, which the screen is.

## How it works

```text
model call ──▶ agentruntime.dispatch ──▶ graphQL read tool (agent module)
                                            │  typed params → variables (generated)
                                            ▼
                                agentports.GraphQLReader (port, agent module)
                                            │
                                            ▼
                      internal/api/graphql/agentread (adapter, API package)
                        persisted op by name → manifest text
                        executor with the server's own extensions
                        auth context, permission memo, loaders, read scope
                                            │
                                            ▼
                                gqlexec → resolvers → services → RLS
```

### The port and the adapter

The agent module may not import `internal/api` (`.go-arch-lint.yml`), so the read is an
inversion port declared low and provided at the composition root:

```go
// agentports
type GraphQLReader interface {
	Read(ctx context.Context, req *GraphQLReadRequest) (*GraphQLReadResult, error)
}

type GraphQLReadRequest struct {
	Actor     *permissionports.RequestActor
	Operation string          // a generated binding's operation name, never query text
	Variables map[string]any
	Ceiling   permission.FieldSensitivity // the agent's data-access ceiling
}

type GraphQLReadResult struct {
	Data     []byte   // the operation's `data`, sonic-encoded
	Errors   []GraphQLReadError // path, message, extensions.code
	Withheld []string // field paths nulled by @protected or the ceiling
	Cost     int      // the complexity the budget charged
}
```

`internal/api/graphql/agentread` implements it.

- **Operation text.** It looks the operation up by name in the embedded persisted manifest
  (`persisted-documents.json`, the same `//go:embed` the handler uses). It accepts only names
  in the generated binding table, so the safelist holds in every environment. This is stricter
  than HTTP outside production, which also accepts raw query text.
- **Executor.** It builds a gqlgen `executor.Executor` over the read executor described
  below, wrapped in the same `newCostLimitedSchema`. It adds the same extensions, in the same
  order: depth limit, `FixedComplexityLimit`, `CostBudgetExtension`, the edition extensions,
  `ReadOnlyExtension`, `FieldGuardExtension` and `ObservabilityExtension`. `NewServer` and the
  reader take that list from one function (`serverExtensions`), so a new extension reaches both.
- **No HTTP.** It runs `CreateOperationContext` and `DispatchOperation` directly. There is no
  request body, so the body-size limit does not apply; the variables are bounded by the tool's
  own clamps instead (`limit`, list lengths, string lengths).

Before each read the adapter builds what `Handler.handle` and the gin middleware build for a
request:

- `gqlctx.WithAuthContext` from the actor: user principal or agent principal; API keys stay
  refused, as on HTTP;
- a fresh `gqlctx.PermissionMemo`;
- `loaders.Factory.Attach` for the tenant;
- the request id, taken from the run, so logs join;
- the `pkg/readscope` holder. `agentflow.runTool` already installs one for every tool call, and
  the adapter refuses to run without it, rather than reading the whole tenant.

### Where it runs: a read executor on the worker

Agent tools run in the worker: one tool call is one `agentflow` activity, and a code-mode
script's broker calls `dispatch` in-process inside its own activity. The reads must therefore run
on the worker too.

The worker does **not** take the API's `GraphQLModule`. That would pull every resolver package,
every service they name and every loader into the worker graph, at the build-time and memory cost
that CLAUDE.md's generics rules exist to prevent. It runs a *read executor* built only from what
the bound operations reach:

- **Validation.** The **parsed schema is the whole schema**, from the same embedded `.graphqls`
  text. Every operation is validated exactly as on HTTP, at no Go dependency cost.
- **Shards.** The **registry holds only the executor shards** that define a type or field the
  bound operations select (`gqlexec.Compile` takes any slice of shards). The generator works the
  set out from the operations' selection sets and fragments, so it grows one module at a time as
  tools convert.
- **Resolvers.** The **resolver map holds only those shards' objects**, keyed by object name as
  `NewExecutableSchema`'s is. `Query` is a generated composite that embeds only the query
  resolver types of the packages the bound root fields live in. Each shard checks only the
  methods it calls, so a partial composite satisfies it.
- **Services.** The **services are only the ones those resolver packages name**. The generator
  writes `agentread.Services` (an `fx.In` struct) from the same per-package `Deps` analysis
  `task gqlgen` already does, so the worker graph takes, say, the customer and location services,
  and not the shipment import assistant.
- **Startup check.** Instead of the whole-schema check, `Registry.ValidateOperations` walks each
  bound operation's selection against the registry. A selected field with no executor, or an
  object with no resolver, panics at startup with the list, exactly as `NewExecutableSchema`
  panics for the whole schema.

The executor takes the same extensions as `NewServer`, from one function both call
(`serverExtensions`), so a new extension reaches both.

**Rejected: routing read tools to a queue the API polls.** `heavyTools` already moves three tools
to another queue, so it fits the loop. But a code-mode script's reads run inside one activity, and
cannot hop queues per call. It would also give the API process a Temporal worker to run and size.

**Rejected: calling the API over HTTP.** It needs a credential that asserts a person, or an
agent, from the worker: a new authentication path. It doubles the hops, and still needs
everything above.

### The agent principal

`gqlctx.AuthContext` knows three principals: `session_user`, `api_key` (refused by the handler)
and `capture_device`. The reader adds a fourth, `authctx.PrincipalTypeAgent = "agent"`, carrying:

- `PrincipalID`: the agent definition;
- `UserID`: the system user, for attribution only;
- `AgentDefinitionID`;
- the organization and business unit.

**Checks.** `middleware.BuildPermissionCheckRequest` maps it to `permissionports.PrincipalTypeAgent`.
The engine already applies the agent ceiling (`IsAgentAllowed`), the organization's unattended
role (`AgentRoleSource`) and the access policies. A resolver's `RequirePermission` therefore
answers for an unattended run as `authorize` does today.

**Who can create it.** Only the reader creates an agent principal; no HTTP path can. The guards:

- `RequireAuth` refuses `agent` on HTTP;
- `authctx` exposes no setter that takes it from a request;
- `TestOnlyTheReaderMakesAnAgentPrincipal` holds both.

**Self-scoped resolvers** (`My*`, listed in `authzlint`'s `selfScoped`) act on the caller's own
records. They refuse an agent principal, as `dispatch` refuses a self-scoped tool in an
unattended run.

**Conversations.** In a conversation the actor is the person (`session_user`), and the agent's
ceiling travels beside it, in the context (below).

## Mapping tools to operations

The mapping is derived, not hand-kept. `internal/api/graphql/agentread/gen` (`go generate
./internal/api/graphql/agentread/...`, a new step in `Codegen Checks`) reads three sources:

- the schema (`schema/*.graphqls`);
- the persisted manifest;
- the server's filter configuration for each node type: `querybuilder.GetFieldConfiguration`
  for the entity that `projection/specs_gen.go` binds the type to.

It writes `agentread/bindings_gen.go` and `agentread/bindings_gen.json`; the JSON is read by the
agent module through the port and kept in the catalog snapshot.

1. **Root field to record kind.** A Query root field returning `XConnection` (a list) or `X` with
   an `id` argument (a get) reads the record kind the permission registry maps `X` to
   ([record-kinds.md](record-kinds.md)). A type no kind claims gets no tool.
2. **Record kind to tool.** The tool is `agentports.ToolNameFor("list_" + plural(kind))` or
   `ToolNameFor("get_" + kind)`. This is the reading-order rule the conventions already use, so
   `tractors` becomes `equipment_list_tractors` and `customer` becomes `customer_get`. A root field
   whose kind has no namespace gets no tool.
3. **Tool to operation.** Among the persisted queries whose only root field is that field, the
   rule picks the first of:
   - the `Desk<Type>Card` / `DeskRecord<Plural>` operation, written for the Desk and compact;
   - `<Type>Table`;
   - the only operation.

   More than one candidate left, or none, fails generation, naming the field and the candidates.
4. **Overrides.** Where the rule picks wrong, `agentread/overrides.go` names the operation and
   says why, in the shape `internal/api/toolcontract` uses for writes. A stale override, one the
   rule would now pick anyway, fails the generator. The rule is the mapping; an override is a
   recorded exception.

`TestEveryReadToolReadsThroughGraphQL` (in `agenttoolcatalog`) holds three rules:

- Every registered `list` and `get` tool is a generated binding, or appears in `agentread.NoGraphQL`
  with its reason.
- A tool in `NoGraphQL` whose kind gains a root field and an operation fails, so the list only
  shrinks.
- Like the namespace ratchet, the count of unconverted tools may only fall. It is pinned in the
  test and lowered by each module's PR.

When a module's operation changes on the client, `pnpm --filter @trenova/graphql codegen`
re-hashes it and `go generate` rebinds the tool. A field the operation stops selecting leaves the
tool's result, and the catalog snapshot shows it.

## Typed parameters

Each parameter is derived from the operation's variables and the node type, within
[agent-tool-conventions.md](agent-tool-conventions.md).

**Operations on `DataTableConnectionInput`** (most `*Table` operations). Every field the server's
filter configuration marks filterable (`FilterableFields`, `NestedFields`) and the node type
selects becomes a parameter. Its type comes from the field's GraphQL type:

| Node field type | Parameter | Compiled to `fieldFilters` |
|---|---|---|
| `status` of any enum | `status`: list of the enum's values | `in` |
| other enum | `<field>`: list of the enum's values | `in` |
| `Timestamp` | `<field>From`, `<field>To`: `agenttoolschema.Date` (or `DateTime` where the field is an instant people name to the minute) | `gte` / `lte` on the organization's day bounds |
| `Boolean` | `<field>`: boolean | `eq` |
| `ID` | `<field>` (or `<field>s` as a list): `RecordIDText` of the referenced resource | `eq` / `in` |
| `Int`, `Float`, `Decimal` | `<field>Min`, `<field>Max` | `gte` / `lte` |
| `String` | `<field>`: string | `contains` (`eq` for codes the configuration marks exact) |

- **Free text and sorting.** `query` maps to the input's `query`. `sortBy` is an enum of
  `SortableFields`, and `sortDirection` is `asc|desc`.
- **Limit.** `limit` maps to `first`, `minimum` 1, `maximum` 50, marked `toolschema.Clamped`.
- **Paging.** `after` maps to `after`.
- **Total count.** `includeTotalCount` is sent `false`, unless the tool reports a total.
- **Hidden from the model.** A filterable field the operation does not select is not offered:
  the screen cannot filter on what it does not show either.
- **Withheld fields.** A field above the caller's ceiling, or in a `@protected` group the caller
  lacks, is refused as a filter or sort before the read, with the same message today's
  `refuseWithheldFields` gives.

**Operations on a bespoke input** (`FuelPurchasesInput`, `WatchtowerItemsInput`,
`FactoringNoaListInput`, the side `filter:` inputs, …). Each input field is a parameter under its
own name:

- enum lists stay lists;
- `Timestamp` becomes `Date`/`DateTime`;
- `first` becomes `limit`;
- `search` becomes `query`; the conventions name free text `query`, and the binding records the
  rename as toolcontract does;
- a wrapped `table: DataTableConnectionInput!` contributes its typed fields by the rule above.

**Get operations.** The get operation's own arguments become parameters: the id as `<record>Id`,
marked with its resource. A human key the person may say instead (a PRO number) is a second,
optional parameter. The generator resolves it through the list operation's exact filter on that
field, so `shipment_get` with `proNumber` reads `shipments` filtered by `proNumber`, then the card.

**What stays hand-written** is model-facing prose: the summary, the "use this rather than …"
sentence and the search terms. Descriptions are edited against traces, which a generator cannot
do. The parameter descriptions come from the schema's field descriptions, then from the
`filtercatalog` note where the schema has none. A parameter whose source has neither fails
generation, so a parameter is never undocumented.

The generic `filters` parameter, `filtercatalog` notes duplicated in `listcatalog*.go`, and the
`listDateRule` prose leave with the last module. `FilterCatalog()` stays for the data-table "Ask"
composer, which compiles against the same configuration.

## Paging

Paging is the connection's own:

- `limit` maps to `first`;
- `after` maps to `after`;
- the result carries `hasMore` from `pageInfo.hasNextPage`, and `nextCursor` from
  `pageInfo.endCursor` when there is a next page.

That is the convention the paging fix introduces for repository-backed lists, so the model sees no
change in shape when a tool converts.

The cursor encodes the sort, so a call that changes `sortBy` and passes an old `after` gets the
server's "Cursor is invalid". The tool turns it into "the cursor belongs to a different sort;
start again without `after`".

Three kinds of root field do not page this way, and their tools say so:

- limit/offset fields (`carrierInvoiceMatches`, `ediCarrierInvoices`, `mySettlements`): offset
  stays internal; the tool hands out `nextCursor` as an opaque encoded offset, so the model learns
  one convention;
- unpaged lists: no `after` parameter, and `hasMore` is never set;
- `DeskMemoryPage` (`items`, `next`): the tool reads `next` as the cursor.

## Field gating

The executor gates fields, so there is one implementation, used by the screen and by every tool
that reads through GraphQL. `FieldGuardExtension` gains the ceiling, so one interceptor decides
three rules.

1. **`@protected` groups**, exactly as on the screen.
2. **The agent's data-access ceiling.** The ceiling travels in the context
   (`gqlctx.WithFieldCeiling`), and only the reader sets it. A field above it is withheld. The
   field's level comes from the permission registry (`fieldsensitivity.Level(resource, field)`),
   where `resource` is the record kind of the field's parent type. The generator writes that
   type-to-resource table from the record-kind registry. Confidential is never shown. On HTTP
   no ceiling is set, so the screen is unchanged. This is the one place an agent sees less than
   the person's screen, by design.
3. **Nested records.** A field's resource is the nearest enclosing type that is a registered
   resource: `WorkerProfile.dob` reads as the worker's `dob`, and a `Worker` nested in a
   `Tractor` is gated by the worker's sensitivities, not the tractor's. That is what today's
   `nestedRedactor` does. A nested `User`, `Organization` or `BusinessUnit` shows only `id` and
   `name`, as the redactor reduces them today. Whether the nested row is visible at all is the
   database's call: the loader reads it under the same RLS and `read_scope`, so a worker outside
   a team scope comes back null, as on the screen.

**One implementation.** The ceiling itself is computed in one place: `fieldAccess.ceiling` moves
to `fieldsensitivity.AgentCeiling` (the definition's data access, lowered to the person's role
ceiling when an agent acts for them). The reader calls it, and so do the tools not yet converted,
so the two never compute it differently. `fieldGate`, `redactor` and `withhold` are deleted
module by module as tools convert. `TestConvertedToolsDoNotGateThemselves` fails a converted tool
that still calls them. The parts retrieval and document access use (`retrievalAccess`,
`readableDocuments`) stay, because they are not GraphQL reads.

**How a field is withheld.** A nullable field is answered `null` without being resolved, as
`@protected` fields already are. A non-null field cannot be nulled without nulling its parent,
so the guard lets it resolve, records its path, and the reader removes the value at that path
before the result leaves the adapter. Either way the tool never holds the value.

**What the tool reports.** The guard collects every withheld path in a per-operation set, and the
reader returns them. The tool reports them as `withheldByAccess`, with today's `withheldNote`, so
the model knows the value exists and is hidden and does not report it as empty.

**Filters.** A filter or sort on a withheld field is refused before the read. The generator knows
each parameter's source field, so the tool asks the same guard, `FieldGuardExtension.Withheld`,
whether that field is withheld.

## Permissions, data scopes and refusals

- **Pre-check.** `dispatch` keeps its `resource:read` check, so a call is refused before a read is
  built, with the advice the model already follows.
- **The resolver's check governs.** A resolver refusal (`FORBIDDEN`, or the team-scope refusals)
  becomes the same denied outcome, `aitrace.OutcomeDenied`, with `deniedAccess` and
  `deniedAdvice` worded from the resolver's resource and operation. It does not become a failed
  tool call.
- **Data scopes.** "Own" and "team" scopes are the restrictive `read_scope` policies, applied by
  RLS to the same statements the screen runs. Nothing in the tool filters by owner.
- **Unattended runs.** They read as `PrincipalTypeAgent`. The engine applies the agent ceiling
  and the organization's unattended role, as it does today.
- **Cost budget.** Each read is charged to the caller's token bucket (organization and principal).
  A read refused for budget (`OPERATION_COST_BUDGET_EXCEEDED`) becomes a failed outcome that
  names the wait (`Retry-After`). It is not retried by the activity: the model decides whether
  the read is still worth making.

## Taint and `readsExternal`

`readsExternal` stays declared per tool (`readSpec`), and keeps its meaning:

- `Always`: EDI files, documents, inbound mail and the web;
- `Marked`: a record that carries outside text, such as an agent run or a note;
- `Never`.

A converted tool keeps the value it had. The generator adds one check: a binding whose node type
selects a field the registry classifies as outside content (an inbound message's `preview` or
`body`, an EDI file's raw text) may not be `Never`.

For `Marked` tools, the records a result carries are the nodes' `id`s under the binding's record
kind. `callTaint` reads them from the result as it reads `TaintedRecords` today.

## Tracing

The read is a child of the tool's `execute_tool <name>` span, opened by the activity and never by
workflow code ([ai-tracing.md](ai-tracing.md)). The adapter opens `graphql.read <operation>` with
these attributes:

- `graphql.operation.name`;
- `trenova.graphql.persisted_hash`;
- `trenova.graphql.root_field`;
- `trenova.graphql.cost`;
- `trenova.graphql.rows`;
- `trenova.graphql.has_next_page`;
- `trenova.graphql.withheld`, a count, not paths.

The `ObservabilityExtension` spans sit under it as on HTTP. On error the span records the error
codes, not variables: variables can carry what the person typed.

## The step ledger

Nothing changes in the ledger. A read's step key covers its arguments, so each page (a different
`after`) is its own step. A retried activity replays the recorded outcome rather than reading
again, so a retry cannot shift a page under a model that already saw it. The ledger keeps up to
64 KB of the result's `Data` for conversations, as today.

## Result shapes

A list returns what `searchOutcome` returns today, with cursor paging:

```json
{
  "count": 25,
  "items": [ { "id": "cus_…", "code": "ACME", "name": "Acme Foods", "status": "Active" } ],
  "columns": ["id", "code", "name", "status"],
  "searchedFor": "acme",
  "hasMore": true,
  "nextCursor": "eyJ…",
  "note": "",
  "withheldByAccess": ["creditLimit"],
  "withheldNote": "…"
}
```

- **Items.** `items` are the connection's `edges[].node`, as the operation selects them. Before
  encoding, nulls, `__typename` and empty lists are dropped. A nested record keeps the fields the
  operation selects for it, usually `id` with `code`/`name`.
- **Columns.** `columns` are the node's top-level fields, in the operation's order.
- **Near-misses.** When page one is empty, `listnearmiss` probes again with the query or one
  typed parameter dropped, through the same operation; each probe is charged to the budget. Then
  it says what to loosen.
- **Gets.** A get returns the operation's record, shaped the same way, with
  `withheldByAccess`. A get for several ids (`invoice_get`) runs one read per id, at most 10
  concurrently, under one span each.

Size: a `*Table` operation selects what a table row shows, which can be large; shipments are
about 3 KB a row. The rule therefore prefers `Desk*` operations, which were written to be read by
the Desk. Where a list has none, its module's PR either adds a `Desk<Plural>` operation in
`client/packages/graphql/src/operations/` (then `pnpm --filter @trenova/graphql codegen`) or
records in the PR why the table's selection is the right one. Each module's PR records the
result's encoded size, before and after, for its largest list.

## Tests

Each converted tool is covered by tests from one table-driven harness,
`agentread/agentreadtest`. The harness runs the tool, and the same operation through the HTTP
handler, against the same resolvers. Four tests:

1. **Typed filters reach the operation.** Each typed parameter appears in the variables (or
   `fieldFilters`) the operation receives, with the right operator and day bounds in the
   organization's zone.
2. **Refusals match the screen.** A person without `resource:read`, and a person whose data scope
   excludes the record, are refused by the tool exactly as the handler refuses them:
   - the same resource and operation;
   - the same rows missing under "own" or "team";
   - the same denied outcome.
3. **Withheld stays withheld.** A `@protected` field, and a field above the agent's ceiling, come
   back null and in `withheldByAccess`. Filtering on either is refused before the read.
4. **Two pages, no repeats.** Page one, then `after: nextCursor`, returns disjoint items and the
   right `hasMore`.

Each module's PR also adds an integration test on Postgres (`TRENOVA_TEST_POSTGRES_DSN`). The
first PR adds the harness's database mode. The test:

- seeds a tenant with RLS in `enforce`;
- pages and filters a real connection through the reader as a person with an "own" scope;
- asserts the rows match the handler's.

## Tools with no GraphQL equivalent

The starting survey of all 176 `list` and `get` tools is
[agent-graphql-reads-inventory.md](agent-graphql-reads-inventory.md). It sorts them into four
classes, each with a rule:

| Class | Tools | What the module's PR does |
|---|---|---|
| `op`: a persisted operation reads the same records | 119 | Converts the tool to the operation. |
| `partial`: the root field reads the records but lacks a parameter the tool has (an `openOnly`, an `asOf`, a list of ids, a per-row value) | 30 | Adds what is missing to the GraphQL input or type, additively (`task gqlschema-diff` must report no breaking change), with the resolver and its permission check, then converts. A tool never loses a capability to convert. |
| `field-no-op`: the root field exists but no persisted query selects it | 6 | Adds a `Desk*` operation in `client/packages/graphql/src/operations/`, runs `pnpm --filter @trenova/graphql codegen`, then converts. |
| `none`: no root field | 21 | Stays on its current path, in `agentread.NoGraphQL` with its reason. |

The `none` reasons:

- **REST-only records**, which have no Query root field (the screen reads them over REST):
  - the billing queue (`billing_list_queue_items`, `billing_get_queue_item`);
  - invoice runs and open statements;
  - GL accounts and fiscal-period close blockers;
  - bank receipts and their exceptions;
  - rate confirmations and rate imports;
  - permits;
  - the EDI tender and transfer change queues.

  Moving one of them is a GraphQL query of its own: a root field, a resolver with its permission
  check, and an operation the screen also uses. It is listed as follow-up work in its module's PR,
  not done inside the conversion, because the screen would have to move to it too.
- **Semantic retrieval**: `document_list`, which ranks by meaning.
- **Document content and extraction**: `document_get_summary`, `intake_get_draft`.
- **Computed candidates**: `invoice_list_share_candidates`.
- **A composite live read**: `shipment_get_tracking` (board moves, stored ETAs, positions and
  hours together).
- **Agent extension state with no record kind**: `web_list_portal_sites`, `web_list_watches`.

Two `partial` tools keep a mode that only they have. `agent_list_memories` and
`inbox_list_inbound_messages` rank by meaning when given a `query`, and GraphQL's `query`
matches text. Each reads through its operation for its filters, and keeps its ranked search for
`query`, listed in `NoGraphQL` under that mode only.

## Rollout

| PR | Contents |
|---|---|
| This one | The design, for approval. No code. |
| 1 | The reader, generator, harness and worker module, plus the first module: reference data (`masterdata`, `equipment_list_types`, `document_list_types`, `location_list_categories`; plain `DataTableConnectionInput` lists). Worker build figures. Conventions doc updated: the Filters row becomes "typed parameters", and the Paging row points here. |
| 2 | Parties: `customer`, `location`, `carrier`. |
| 3 | Equipment: tractors, trailers, manufacturers. |
| 4 | Workforce: `worker`, the HR and safety reads (bespoke inputs). |
| 5 | Operations: `shipment`, `order`, `dispatch`, `detention`, `exception`. |
| 6 | Billing and receivables: `billing`, `invoice`, `receivable`. |
| 7 | Settlements and pay: `settlement`. |
| 8 | Accounting and ledger. |
| 9 | EDI. |
| 10 | Factoring, fuel and IFTA, rates. |
| 11 | Agent, insight, inbox, report, home; the generic `filters` path removed. |

Each PR, per the Phase 3 discipline:

- regenerates the prompt goldens, the catalog snapshot, the safety doc, the embeddings and floors
  (local Ollama `nomic-embed-text`), and the split ledger;
- records held-out recall@5 and top-1 before and after in the Phase 3 table of
  `desk-harness-review.md`, never tuned against the held-out cases;
- runs `task gqlschema-diff` (no breaking change), `authzlint`, and, where operations changed,
  `task gqlgen` and `pnpm --filter @trenova/graphql codegen`.

## Decisions

The owner delegated design decisions to the orchestrating session. These are made, and the
reasoning for each is above.

1. **GraphQL under the tools, not beside them.** Tools keep their names, policies, `dispatch`,
   taint (`callTaint`, `SourcedTaintCarrier`), ledger and spans. Code mode's SDK is generated from
   the tool registry, so it inherits the typed parameters.
2. **The worker runs a read executor over the shards the bound operations reach.** It does not
   run the API's GraphQL module, and does not make an HTTP or queue hop.
3. **An `agent` principal in `AuthContext`**, made only by the reader.
4. **The ceiling is enforced in the executor**, by the guard that already enforces `@protected`.
   The tool-side gates are deleted as tools convert, and the ceiling is computed in one function.
5. **Nested records are gated by their own kind's sensitivities**, and users, organizations and
   business units reduce to `id` and `name`, as the tools do today. No extra read check is added
   that the screen does not make.
6. **The `Desk*` operation is preferred.** A list with none reads its table's operation, unless
   its module's PR adds a `Desk<Plural>` operation, backed by the result-size figures it records.
