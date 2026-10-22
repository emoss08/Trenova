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
	Actor             *permissionports.RequestActor
	Operation         string                      // a generated binding's operation name, never query text
	Variables         map[string]any
	DataAccessCeiling permission.FieldSensitivity // the agent's data-access ceiling
	RequestID         string                      // the run's, so logs join
}

type GraphQLReadResult struct {
	Data       map[string]any     // the operation's `data`, decoded; a tool reads Data[rootField]
	Errors     []GraphQLReadError // path, message, extensions.code
	Withheld   []string           // field paths withheld by @protected or the ceiling
	Cost       int                // the complexity the budget charged
	RetryAfter time.Duration      // set when the budget refused the read
}
```

`internal/api/graphql/agentread` implements it.

- **Operation text.** It looks the operation up by name in the embedded persisted manifest
  (`persisted-documents.json`, the same `//go:embed` the handler uses). It accepts only names
  in the generated binding table, so the safelist holds in every environment. This is stricter
  than HTTP outside production, which also accepts raw query text.
- **Executor.** It builds a gqlgen `executor.Executor` over the read executor described
  below, wrapped in the same `gqlext.NewCostLimitedSchema`. It adds the same extensions, in the same
  order: depth limit, `FixedComplexityLimit`, `CostBudgetExtension`, the edition extensions,
  `ReadOnlyExtension`, `FieldGuardExtension` and `ObservabilityExtension`. `NewServer` and the
  reader take that list from one function (`gqlext.Extensions`), so a new extension reaches both.
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
that the generics rules in module-architecture.md exist to prevent. It runs a *read executor* built only from what
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
  writes `agentread.ResolverParams` (an `fx.In` struct) from the same per-package `Deps` analysis
  `task gqlgen` already does, so the worker graph takes, say, the customer and location services,
  and not the shipment import assistant.
- **Startup check.** Instead of the whole-schema check, `gqlexec.NewSchema` runs the registry's
  `validateOperations` whenever `SchemaConfig.Operations` is set, walking each bound operation's
  selection against the registry, through every possible type of an interface or union. A selected field with no executor, or an
  object with no resolver, panics at startup with the list, exactly as `NewExecutableSchema`
  panics for the whole schema.

The executor takes the same extensions as `NewServer`, from one function both call
(`gqlext.Extensions`), so a new extension reaches both.

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

**Self-scoped resolvers** (`My*`, listed in `selfscoped.Resolvers`) act on the caller's own
records. They refuse an agent principal, as `dispatch` refuses a self-scoped tool in an
unattended run.

**Conversations.** In a conversation the actor is the person (`session_user`), and the agent's
ceiling travels beside it, in the context (below).

## Mapping tools to operations

The mapping is derived, not hand-kept. `go run ./internal/api/graphql/agentread/gen` runs as
the last step of `task gqlgen` and in `Codegen Checks`. It reads four sources:

- **`agentread/reads.yml`**: the tools that read through GraphQL. This is the state of the
  migration: one line per converted tool, and nothing else unless an operation is overridden.
- **The schema** (`schema/*.graphqls`).
- **The persisted manifest** (`persisted-documents.json`).
- **The generated executor and resolver root** (`generated/exec`, `resolver/resolver.generated.go`,
  `resolver/services.go`), for the wiring below.

It writes two files:

- `agentread/executor_gen.go`: the read executor, holding the bound operations byte for byte.
- `agentquerytoolservice/graphqlreads_gen.go`: for each tool, its operation, root field and node
  type, the fields the operation selects on the node, the input types its variables take, and a
  constructor for the node's Go entity, read from gqlgen's own binder.

The rule, for each listed tool:

1. **Root field to record kind.** A Query root field returning `XConnection` (a list) or `X` with
   an `id` argument (a get) reads the record kind the permission registry maps `X` to
   ([record-kinds.md](record-kinds.md)). A type no kind claims gets no tool.
2. **Record kind to tool.** The tool is `agentports.ToolNameFor("list_" + plural(kind))` or
   `ToolNameFor("get_" + kind)`. This is the reading-order rule the conventions already use, so
   `tractors` becomes `equipment_list_tractors` and `customer` becomes `customer_get`. A listed
   tool no root field is named for fails generation.
3. **Tool to operation.** Among the persisted queries whose only root field is that field, the
   rule picks the first of:
   - a `Desk*` operation, written for the Desk and compact;
   - `<Type>Table`;
   - the only operation.

   More than one candidate in a tier, or none at all, fails generation, naming the field and the
   candidates.
4. **Named bindings.** Where the rule picks wrong or finds nothing, the tool's line in
   `reads.yml` names the operation and gives a `reason`, and the generator proves the binding
   rather than taking it on trust:
   - the operation is persisted and reads one root field;
   - the field's shape matches the tool's verb: a connection, a list or an `items` container for
     a list tool, one record for a get;
   - the field's resolver checks the tool's own resource for read. The generator reads the check
     from the resolver's code, and `TestGraphQLReadsCheckTheToolsOwnResource` holds every bound
     tool's policy resource to it, so a read meets the same check `authorize` made.

   A named binding with no reason fails generation. So does one that names the operation the rule
   would pick anyway. So does any binding, by rule or by name, whose operation requires a
   variable the tool's verb cannot send: a list sends its `DataTableConnectionInput` and
   `includeTotalCount`, a get sends `id`, and a required `$region` would leave every call
   failing at the server. A list tool never binds to a connection read that selects no `pageInfo`,
   because it could not page. Where the node's type binds to a generated model (`gqlmodel`), the
   line also names the `entity` whose filter configuration the resolver compiles with.

   The rule binds about two in five of the tools with an operation (49 of 119 in the trial
   recorded in the inventory). Tool names follow the product's words, not always their record
   kind's, so the rest are named and proven.

`TestEveryBoundOperationIsPersistedAsItIsRun` holds each bound operation to the manifest byte for
byte, so a client edit that re-hashes an operation fails until the generator runs again.

With the first conversion, `TestEveryReadToolReadsThroughGraphQL` (in `agenttoolcatalog`) holds
three rules:

- Every registered `list` and `get` tool is a generated binding, or appears in `agentread.NoGraphQL`
  with its reason.
- A tool in `NoGraphQL` whose kind gains a root field and an operation fails, so the list only
  shrinks.
- Like the namespace ratchet, the count of unconverted tools may only fall. It is pinned in the
  test and lowered by each module's PR.

When a module's operation changes on the client, `pnpm --filter @trenova/graphql codegen`
re-hashes it and `task gqlgen` (or `go run ./internal/api/graphql/agentread/gen`) rebinds the tool.
A field the operation stops selecting leaves the tool's result and its parameters, and the catalog
snapshot shows it.

## Typed parameters

Each parameter is derived from the operation's variables and the node type, within
[agent-tool-conventions.md](agent-tool-conventions.md).

**Operations on `DataTableConnectionInput`** (most `*Table` operations). Every field of the node
type that the server's filter configuration marks filterable (`FilterableFields`, `NestedFields`)
becomes a parameter. Its type comes from the field's GraphQL type:

| Node field type | Parameter | Compiled to `fieldFilters` |
|---|---|---|
| `status` of any enum | `status`: list of the enum's values | `in` |
| other enum | `<field>`: list of the enum's values | `in` |
| `Timestamp` | `<field>From`, `<field>To`: `agenttoolschema.Date` (or `DateTime` where the field is an instant people name to the minute) | `gte` / `lte` on the organization's day bounds |
| `Boolean` | `<field>`: boolean | `eq` |
| `ID` | `<field>`: `RecordID` of the resource the tool's `references` entry names, with the tool that supplies one; a nullable one also gives `has<Thing>`: boolean | `eq`; `isnotnull` / `isnull` |
| `Int`, `Float`, `Decimal` | `<field>Min`, `<field>Max` | `gte` / `lte` |
| `String` | `<field>`: string | `contains` (`eq` for codes the configuration marks exact) |

- **An id with nowhere to get it is not offered.** An `ID` field the tool's `references` does not
  name is left out, keeping only its `has<Thing>` parameter when it is nullable: a model has no
  honest way to get an id no tool lists (`masterdata_list_fleet_codes` takes no `managerId`,
  because no tool lists users). A reference to a field the record lacks fails construction.
- **Names follow the conventions.** A parameter takes the name
  [agent-tool-conventions.md](agent-tool-conventions.md) gives free text (`notes` becomes `note`),
  and its filter still names the field.
- **A record's own parts are filtered through.** Where the filter configuration allows a path one
  record down (`profile.endorsement`) and the operation selects that part, the part's field
  becomes a parameter named for the path (`profileEndorsement`), typed by the part's own GraphQL
  field and filtered on the path. A part is a type named for the record (`WorkerProfile` of
  `Worker`); a linked record (a state, an equipment type) is filtered by its id instead. A
  filter on a part's field is judged by the part's own kind where it has one, as the field
  guard judges it.
- **Not every field is something a person says.** The tenant, the row's version, its id and the
  Google Places id an address was geocoded to (`placeId`) are never parameters.
- **Free text and sorting.** `query` maps to the input's `query`. `sortBy` is an enum of
  `SortableFields`, and `sortDirection` is `asc|desc`.
- **Limit.** `limit` maps to `first`, `minimum` 1, `maximum` 50, marked `toolschema.Clamped`.
- **Paging.** `after` is the scoped cursor of the previous page (see Paging).
- **Total count.** `includeTotalCount` is sent `false`, unless the tool reports a total.
- **Every filterable field, selected or not.** Parameters come from every field of the record
  type that the filter configuration marks filterable, not only the ones the operation selects:
  the resolver accepts the filter either way, and the old catalogs filtered by fields no table
  shows (`customer_list` by `mcNumber`, the tractor and trailer lists by `ownershipType` and
  `leaseEndDate`). The operation's selection decides only what a result carries.
- **Withheld fields.** A field above the caller's ceiling, or in a `@protected` group the caller
  lacks, is refused as a filter or sort before the read, with the same message today's
  `refuseWithheldFields` gives. A view `report_compose_table_view` composes and previews through the list
  is held to the same check, since its filters reach the same resolver.

**Operations on a table-shaped input** (`WorkerPTOEntriesInput`, `TrainingCoursesInput`, and any
input that holds `first`, `after`, `query`, `fieldFilters` and `sort`). The table's fields are
built as for `DataTableConnectionInput`, and the generator's supplied-variable check accepts the
input under its own name. The input's other fields are the screen's own filters and become
parameters sent on the input (`OnInput`), not as field filters: a single enum takes one value, a
`Timestamp` named for a range end (`startDateFrom`, `startDateTo`) is a day in the organization's
zone, an `ID` follows the reference rule, and a `Boolean` the schema describes is a switch whose
description is the schema's own (`overlapping`). A range end on the input takes the place of the
record's range parameter of the same name, because the resolver reads it with the screen's meaning:
time off's `startDateFrom` with `overlapping` matches leave already under way, which a field filter
on `startDate` cannot. Any other name a record parameter already takes keeps the record's, which can
match several values (`status`). An `include*` toggle the screen sets for its own rendering is left
out. Being arguments the screen sends, they are not judged by a record field's sensitivity. The
near-miss probes drop one field filter at a time and then one input field at a time (never a
switch), so an empty page names the status or the window that kept every row out.

**Defaults.** A spec's `defaults` are the values a call that omits a parameter reads with, named
in that parameter's note: `worker_list_training_courses` reads the active courses unless the call
names a status, as its repository tool always did. `defaultsAt` are defaults that depend on the
day, applied together only when the call gives none of them: `worker_list_time_off` reads the leave
on at any point from today through 14 days out, counted by the calendar in the organization's zone
so a change of clocks moves neither end, and naming either date replaces that window whole.
`inputs` are input fields every call sends as the screen sets them (each request's worker).

**Unpaged lists keyed by a record** (`workerChecklists(workerId:, includeClosed:)`,
`workerTrainingRecords`, `workerPerformanceReviews`: a root field returning `[X!]!`). The tool's
parameters are the operation's own variables, typed by the root field argument each is sent as: an
`ID` is a reference the spec names (`workerId` from `worker_list`), a described `Boolean` is a
switch in the schema's words, a single enum takes one value. A required variable is a required
parameter, and a required id with no reference fails construction. There is no `limit`, `after`,
`query` or sort, and the record's fields are not filters: the list is what the screen shows for
that record, and `defaults` say what a call that omits a switch reads (open records only). The
generator binds such a field by name in `reads.yml` and accepts any scalar, enum or id variable as
supplied; an input object the tool cannot build still fails generation. The node needs no Go
entity, because nothing compiles field filters against it. `newGraphQLReadTool` picks the shape
from the binding, so a spec does not say which it is.

The record a list is keyed by may be optional: `workerInjuries` and `workerDotTests` read one
worker's records, or every worker's when no worker is named, as the repository tools did. Such a
field takes a `limit` argument, which the tool offers with the list bounds (25 unless asked, at
most 100, the server's page), and the resolver reads every row of a named record when no limit is
sent and one default page when no record is named (`base.RecordListLimit`). An unpaged list has
no next page, so when the rows fill the limit the result's `note` says more may exist and how to
narrow the call. Any other `Int` argument is a parameter only when the schema describes it
(`caseYear`). A default that holds only when the call names no record goes in the spec's
`unscoped`, not `defaults`: the safety events list reads what is still open across the
organization when no worker is named, and one worker's record whole.

**Reads alongside.** Where the repository tool returned records of another kind beside the list
because the next call needs their ids (the checklist templates a person starts by hand, the review
templates), the spec names the operation (`alongside`) and the key the result carries the records
under, and `reads.yml` lists it under the tool. The generator proves each one as it proves a named
binding: persisted, one connection root field that selects `pageInfo`, no required variable but
its table input, and a resolver that checks a read permission. The tool makes the read only when
the reader may read that kind (`AgentCeilings.Reads`), so a reader without it gets the list and no
templates, as before, and a refusal is never charged to the budget. It reads at most 100 records
and says so in `note` when there are more. Only an unpaged list reads alongside; a paged list's
next call pages its own records.

A read alongside may instead be an unpaged list keyed by the list's own arguments (a worker's
recognitions beside their safety events). The generator proves that each of its variables is one
the list's operation declares, of the same type, and the tool sends it the value the call sent,
and makes the read only when the call names every one of them: with no worker named, the safety
events list reads every worker's events and no recognitions. It takes no table input and is never
cut short.

**Operations on another bespoke input** (`FuelPurchasesInput`, `WatchtowerItemsInput`,
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
do. A parameter's description is the schema field's description, followed by a sentence its kind
writes ("Only records whose status is one of these."; "… is on or after this day, in the
organization's timezone."). A converted tool's `filtercatalog` note, where it says more than that,
moves to the spec's `notes`, keyed by parameter, and is appended to that parameter's description.
A note for a parameter the tool does not take fails construction. Notes stay with the tool rather
than in the schema because they are written for a model choosing a call ("only an active reason
can be used on a new hold"), and a schema description edit is a schema change for every client.

The tool's description is its summary followed by every typed parameter an agent at the
default data access (Internal) can use, by name ("Filters: active, code, createdAtFrom/To, …"),
as the repository lists' were, so a model choosing among tools, and the ranking that embeds
descriptions, sees what each list narrows by. A part's parameters are named once under it
("profile fields, each named profile and the field (profileCdlClass): cdlClass, …"). A parameter
above Internal stays in the schema, with its own description, for a reader whose access reaches
it. `TestConvertedLists_DescribeEveryParameter` holds each converted tool to it.

Two hooks carry what a repository row did that the screen's resolver does not, and each is
named in the spec rather than hidden in a fetch: `rank` orders a page that matched a free-text
query when the call chose no order (`worker_list` puts the closest name first), and `shape`
applies a domain rule to each row given what was withheld (`worker_list` folds a NonCompliant
credential into `canBeAssigned`, as `worker.Standing` does, and keeps the credential dates
present and null when nothing is on file).

The parameters are derived when the tool is built, not when the code is generated, because the
filter configuration is computed from the entity at runtime (`querybuilder.GetFieldConfiguration`).
The catalog snapshot (`TestToolCatalogSnapshot`) pins the result, so a schema or configuration
change that moves a parameter shows in review. `TestGraphQLReadsKeepEveryFilterTheCatalogOffered`
fails a conversion that loses a filter the tool's catalog entry offered.

The generic `filters` parameter, `filtercatalog` notes duplicated in `listcatalog*.go`, and the
`listDateRule` prose leave with the last module. `FilterCatalog()` stays for the data-table "Ask"
composer, which compiles against the same configuration.

## Paging

Paging is the connection's own, under the cursor contract the paging fix put on every
cursor-paged list (`paging.go`):

- `limit` maps to `first`;
- the result carries `hasMore` from `pageInfo.hasNextPage`, and, when there is a next page,
  `nextCursor`: the connection's `endCursor` wrapped by `pagination.EncodeKeyCursor` in a scope
  hashed from the tool's name and every argument but `limit` and `after`;
- `after` is read by `readCursorWindow`, which unwraps it in the call's own scope and passes the
  connection cursor inside to the operation's `after`.

So a model sees the same shape whichever path a list reads through. A cursor sent with changed
filters, query or sort, an altered cursor, or one the tool never returned is refused before the
read with `errStaleCursor` ("call it again without after"), and `offset` with
`errOffsetOnCursorList`. `TestGraphQLListTool_KeepsTheCursorContract` holds both, and
`TestGraphQLListTool_PagesThroughWhatTheRepositoryToolPagedThrough` walks every page of both paths
and compares them.

Three kinds of root field do not page this way, and their tools say so:

- limit/offset fields (`carrierInvoiceMatches`, `ediCarrierInvoices`, `mySettlements`): offset
  stays internal; the tool hands out `nextCursor` as an opaque encoded offset, so the model learns
  one convention;
- unpaged lists: no `limit` or `after` parameter, and `hasMore` is never set; the list keyed by a
  record (above) returns every record the screen shows for it;
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
   no agent ceiling is set; a person's screen applies reference views
   ([reference-views.md](reference-views.md)) and nothing more. This is the one place an agent sees less than
   the person's screen, by design.
3. **Nested records.** A field's resource is the nearest enclosing type that is a registered
   resource: `WorkerProfile.dob` reads as the worker's `dob`, and a `Worker` nested in a
   `Tractor` is gated by the worker's sensitivities, not the tractor's. A record carried by
   another shows its kind's reference view to every reader, and its other fields only to a
   reader who may read the kind: the same rule people's screens follow, decided by the same
   package (`refview`), described in [reference-views.md](reference-views.md). A nested `User`,
   `Organization` or `BusinessUnit` is one more kind with a view (id, name, avatar), not a
   special case. Whether the nested row is visible at all is still the database's call: the
   loader reads it under the same RLS and `read_scope`, so a worker outside a team scope comes
   back null, as on the screen.

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

**Filters.** A filter or sort on a withheld field is refused before the read. The generator
knows each parameter's source field, and the tool asks the same question the guard does, of the
same function: `fieldsensitivity.Visible` against `fieldsensitivity.AgentCeiling` for the tool's
resource. Every `@protected` field is classified confidential (`fieldguard_test.go` holds that),
so the ceiling refuses it too.

**A record the reader cannot read shows its reference view.** A record of one kind carried by
another (a tractor's drivers are workers) shows the fields of its kind's reference view (a
driver's name) to every reader of the record that carries it; every other field needs the kind's
own read, and is withheld and named in `withheldByAccess` without it. The check is the engine's,
once per kind per read (`AgentCeilings.Reads`), as the person or as the agent, and a failed check
reads as no; the ceiling still applies inside the view. This replaced the id-only gating #852
added, so an agent and a person are told the same thing about a linked record.
`TestConvertedLists_ATractorsDriversShowTheirReferenceView` holds the tractor list to it.

**A record's id is never withheld.** Row-level security has already decided the row may be read,
and the id carries none of the row's content. A tool needs it to name the row in the next call.

## The ceiling now applies to every read

Converting a tool closes a real gap. Of the 37 repository-backed list tools, only nine applied the
agent's data access ceiling (`fetchGated`): `worker_list`, `invoice_list`,
`billing_list_queue_items`, `ledger_list_accounts`, `ledger_list_fiscal_periods`,
`ledger_list_manual_journals`, `ledger_list_journal_reversals`, `order_list` and
`invoice_list_runs`. The other 28 returned every column they projected, whatever the agent's or
the person's tier. Through the guard, every converted read applies the ceiling, so a converted
tool withholds at least what it did before, and sometimes more.

Each module's PR:

- lists, per tool, every field it now withholds that it did not, at the default agent data
  access (Internal) and at a person's role ceiling;
- updates the tool's description wherever it names a newly withheld field, so the model is told
  the field can be withheld rather than finding it missing;
- runs the parity test, which fails a converted tool that withholds less than its old path.

| Module | Tool | Newly withheld at Internal | Notes |
|---|---|---|---|
| Reference data | `equipment_list_manufacturers` | none | every manufacturer field is Internal |
| Reference data | `equipment_list_types` | none | every field is Internal |
| Reference data | `masterdata_list_fleet_codes` | none | every field is Internal; the nested `manager` reduces to `id` and `name` |
| Reference data | `masterdata_list_service_types` | none | every field is Internal |
| Reference data | `masterdata_list_shipment_types` | none | every field is Internal |
| Reference data | `masterdata_list_commodities` | none | every field is Internal |
| Reference data | `masterdata_list_hazardous_materials` | none | every field is Internal, the emergency contact and phone among them |
| Reference data | `masterdata_list_accessorial_charges` | none | every field is Internal |
| Reference data | `document_list_types` | none | every field is Internal |
| Reference data | `location_list_categories` | none | every field is Internal |
| Reference data | `masterdata_list_hold_reasons` | none | every field is Internal |
| Parties | `customer_list` | none | every field the table selects is Internal |
| Parties | `location_list` | none | every field the table selects is Internal |
| Parties | `carrier_list` | none | every field the table selects is Internal |
| Equipment | `equipment_list_tractors` | none | a driver (`primaryWorker`, `secondaryWorker`) shows the worker reference view (the driver's name) to a reader who cannot read workers, and nothing else |
| Equipment | `equipment_list_trailers` | none | every field the table selects is Internal |
| Workforce | `worker_list_training_courses` | none | every field the table selects is Internal |
| Workforce | `worker_list_time_off` | `reason`, `rejectionReason`, `cancellationReason`, `balanceAfterDays` | `worker_pto` declared no field sensitivities, so under its Restricted default every field but `id` would be withheld at Internal; its dates, status, type, days and people are now declared Internal, and a leave's reasons and balance stay Restricted, where the repository row returned the reason and decision ungated. Each request's `worker` shows only the worker reference view to a reader who cannot read workers |
| Workforce | `worker_list_checklists` | none | reads `DeskWorkerChecklists`; every checklist field is Internal. The templates read alongside are root records of their own read |
| Workforce | `worker_list_training` | none | reads `DeskWorkerTrainingRecords`; each assignment's `course` is a linked training course and shows its reference view (`TrainingCourseReference`) to any reader of the assignment, which includes the name the repository row carried |
| Workforce | `worker_list_performance_reviews` | none | reads `DeskWorkerPerformanceReviews`; every review field, the summary and rating comments among them, is Internal, as the repository row gated them |
| Workforce | `worker_list_injuries` | none | reads `DeskWorkerInjuries`. `worker_injury` declared no field sensitivities, so under its Restricted default every field but `id` would be withheld at Internal; what the OSHA 300 log posts (case, classification, illness type, status, dates, days away or restricted, privacy flag, log name, claim status) is now Internal, and treatment, description, body part, harmful agent, location, claim details and notes stay Restricted, as the repository row gated the medical detail. A privacy case's `worker` is left out by the spec's `shape`, as the repository row left out its name |
| Workforce | `worker_list_dot_tests` | `substance`, `result`, `collectionSite` | reads `DeskWorkerDotTests`. `worker_dot_test` declared no field sensitivities and defaulted to Internal, so results showed at Internal; by the owner's decision the test type, reason, status, dates and random-selection fields are Internal, and the result, substance, alcohol reading, collector, lab, MRO, specimen, notes and document are Restricted (49 CFR 40.321). This also changes what a person whose role reaches only Internal sees where sensitivity applies to them. The description says the substance and result can be withheld |
| Workforce | `worker_list_leave_cases` | none | reads `DeskWorkerLeaveCases`. `WorkerLeaveCase` had no kind (it is not named for `worker_leave`), so the guard did not gate it; it now declares `@kind(resource: "worker_leave")`. `worker_leave` declared no field sensitivities, so under its Restricted default every field but `id` would be withheld at Internal; that a worker is on leave, of what kind, when, where the certification stands and the days taken are now Internal, and the reason, notes, certificate, military caregiver flag and eligibility hours stay Restricted, as the repository row gated the reason. The read is gated by `worker_leave` read through the root's own check and the `@kind`, not by the worker family, so it stays correct when `worker_leave` stops inheriting from worker read |
| Workforce | `worker_list_employment_verifications` | `hadDrugAlcoholViolations` | reads `DeskWorkerEmploymentVerifications`. `WorkerEmploymentVerification` had no kind; it now declares `@kind(resource: "qualification")`, and `driverQualificationFile` declares `@authorizes(resource: "qualification")`, so the verifications its file carries are that root's own records. `qualification` declared no field sensitivities; the employer, the dates, status, method, follow-ups and accident history are now Internal, and the employer's contact details, the driver's drug and alcohol history, the findings and notes stay Restricted. The type gains `worker`, so an organization-wide list names each driver by the worker reference view, as the repository row did. `qualification` takes no `ParentResource`: the permission engine resolves a child through its parent's grant, so worker read would have granted every qualification record |
| Workforce | `worker_list_safety_events` | none | reads `DeskWorkerSafetyEvents`, with each worker's recognitions read alongside (`DeskWorkerRecognitions`) when the call names a worker. `worker_safety_event` and `worker_recognition` declare no field sensitivities and default to Internal, as the repository row read them. Each event gains `violations` (a loader, one query per page of events), which the repository tool read beside the events; `WorkerSafetyViolation` now declares `@kind(resource: "worker_safety_event")`, so its fields follow the event's. With no worker the list reads the events still open across the organization, newest first, as the repository tool did (the spec's `unscoped` default); `openOnly: false` lists the closed ones too, and one worker's record is read whole |
| Workforce | `worker_list` | none | reads `DeskWorkers`, which selects only what the Desk reads (no address, contact, personnel or Confidential profile fields); `city` is Restricted and withheld at Internal, as the repository row gated it |

None of the repository-backed reference, party or equipment lists was gated, so at a person's role ceiling below
Internal each now withholds every field but `id`, where it used to return them all. No description
names a field that is newly withheld at Internal, so none changed.

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
- **Cost budget.** Each read is charged to the caller's token bucket: a person's own, and an
  unattended agent's own (`budgetKey`), never one bucket shared by every agent in an
  organization. Buckets are kept per process, as they already are per API replica, so the
  worker's reads and the screen's requests draw on separate buckets of the same size.
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

Each converted tool's tests read through the real reader (`agentread/agentreadtest`, which builds
it as the application does), over the real resolver and service, with the repository faked.
Five tests:

1. **Typed filters reach the operation.** Each typed parameter appears in the filters the
   repository receives, with the right operator and day bounds in the organization's zone.
2. **Refusals match `authorize`.** For a granting and a denying engine, as a person and as an
   unattended agent, the read is refused exactly when `authorize(actor, resource, read)` refuses,
   and a refusal is a `ReadRefusedError`, which dispatch reports as denied, not failed.
3. **Parity with the repository tool.** The old tool and the converted one, over the same rows,
   return the same records for the same call. The converted one withholds at least what the old
   one did.
4. **Withheld stays withheld.** At a ceiling below a field's tier, the field, including a
   non-null one, never reaches the tool's result, and is named in `withheldByAccess`. Filtering on
   it is refused before the read.
5. **Two pages, no repeats.** Page one, then `after: nextCursor`, return disjoint rows, and
   `hasMore` is false at the end.

Each module's PR also adds an integration test on Postgres (build tag `integration`,
`seedtest.SetupTestDBWithDSN`), over the real repositories under an RLS connection, with two
tenants written by `agentreadtest.SeedTenant`. For every converted tool it:

- walks every page through the tool and gets each of the tenant's rows once and none of the
  other's;
- walks every page of the repository tool and gets the same rows;
- narrows by a typed filter and gets what the repository tool's filter got;
- reads at a role ceiling of Public and finds every withheld field absent from every row.

For kinds that declare an "own" or "team" scope (shipments, orders, invoices, workers), the
module's test also reads as a person narrowed by `readscope` and asserts the GraphQL path returns
the rows the repository path returned for that person
(`TestConvertedLists_WorkerListIsNarrowedToTheTeamOnBothPaths` for workers).

Every list converted by a spec is listed once, in `graphQLListSpecs`, which registers them and
which the tests range over: `TestConvertedLists_ReadWhatTheRepositoryToolsReadUnderRowLevelSecurity`
holds each to the above, and `TestConvertedLists_RefuseExactlyWhatAuthorizeRefuses` covers refusal
for each, as a person and as an unattended agent. A module's PR adds its specs and its cases.

A repository-backed list that another tool composes (`shipment_get` reads its customers and
locations through `newListCustomersTool` and `newListLocationsTool`) keeps that use until the
composing tool's own module converts it.

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
5. **Nested records are gated by their own kind's reference view, read permission and
   sensitivities.** A record carried by another shows its kind's reference view; its other
   fields need the kind's read. This decision first added no read check the screen does not
   make; the tractor list showed that the repository tools did make one (a driver's name needed
   worker read), so an agent read briefly showed only the id of a record of a kind it could not
   read. Reference views replaced that for agents and people alike: one rule, in `refview`.
6. **The `Desk*` operation is preferred.** A list with none reads its table's operation, unless
   its module's PR adds a `Desk<Plural>` operation, backed by the result-size figures it records.
   `worker_list` reads `DeskWorkers` rather than `WorkerTable`, which selects about thirty fields
   an agent at Internal is never shown. `withheldByAccess` is one list per response, by path
   without row indices, however many rows a page holds.
7. **Bindings the rule cannot find are named in `reads.yml` and proven** by shape and by the
   resource their resolver checks, never trusted. No resource-only rule was possible: most
   resources are checked by several root fields (`edi` by thirteen).
8. **Filter notes live on the tool spec, not the schema.** They are prose for a model, edited
   against traces, and the schema is the screen's contract.
9. **An id parameter needs a source.** A list offers an id filter only where a tool supplies that
   id; otherwise only `has<Thing>`.
