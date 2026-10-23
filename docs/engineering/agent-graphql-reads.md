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
     a list tool, one record for a get. A get tool may also read the whole list one record holds,
     keyed by that record's required id (`worker_get_schedule` reads
     `workerShiftAssignments(workerId:)`): it binds as an unpaged list keyed by the id its call
     names, and reads beside it as any such list does;
   - the field's resolver checks the tool's own resource for read. The generator reads the check
     from the resolver's code, and `TestGraphQLReadsCheckTheToolsOwnResource` holds every bound
     tool's policy resource to it, so a read meets the same check `authorize` made.

   A named binding with no reason fails generation. So does one that names the operation the rule
   would pick anyway. So does any binding, by rule or by name, whose operation requires a
   variable the tool cannot send: a list sends its `DataTableConnectionInput` and
   `includeTotalCount`, a get sends `id` (or, for a record read by the id of the one it belongs to,
   the operation's only required id: a driver's hours by `$workerId`), any tool takes a scalar,
   enum or id variable as a parameter of its own (a required `$integrationType` is a required parameter), and a paged list
   builds a filter input whose required fields are scalars, enums or ids. A get never takes an
   input object. A list tool never binds to a connection read that selects no `pageInfo`,
   because it could not page. Where the node's type binds to a generated model (`gqlmodel`), the
   line also names the `entity` whose filter configuration the resolver compiles with. Only a
   paged list compiles field filters, so only a paged list's binding carries an entity; a get and
   an unpaged list read by their arguments alone, and may read a generated model
   (`WorkerHosState`).

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
| `Int` named `<amount>Minor` (minor units), where the spec sets `amounts` | `<amount>Min`, `<amount>Max`: a decimal string such as `1250.00` | `gte` / `lte` on the minor units (`money.MinorUnits`) |
| `String` | `<field>`: string | `contains` (`eq` for codes the configuration marks exact) |

- **A record kept under another resource names its kind.** An id reference may name a record
  kind as well as its resource (`kind` on `idReference`): an IFTA jurisdiction is kept under fuel
  purchases, so `jurisdictionId` is held to the `ifj_` prefix rather than to the purchase's.
- **A Confidential field is not a parameter.** No reader's data access reaches it (a card
  provider's own card id), so offering it would only lead to a refusal.
- **An id with nowhere to get it is not offered.** An `ID` field the tool's `references` does not
  name is left out, keeping only its `has<Thing>` parameter when it is nullable: a model has no
  honest way to get an id no tool lists (`masterdata_list_fleet_codes` takes no `managerId`,
  because no tool lists users). A reference to a field the record lacks fails construction.
- **Names follow the conventions.** A parameter takes the name
  [agent-tool-conventions.md](agent-tool-conventions.md) gives free text (`notes` becomes `note`,
  `cancelReason` becomes `reason`), and its filter still names the field.
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

A `String` argument that takes a day says so in its schema description (`as YYYY-MM-DD`, the fuel
index price history's `from` and `to`); the tool offers it as an `agenttoolschema.Date`, refuses
anything that is not a calendar day before the read, and sends the day as the text the argument
takes. An id kept inside another resource's records (a fuel index inside fuel surcharge programs)
is a reference by its record `kinds` rather than its resource, so the runtime still holds it to a
prefix.

A root that takes no argument at all (`fuelProgramCurrentRates`, a computed rate per active
program) is an unpaged list with no parameters of its own. Where its repository tool matched a
free-text query in what it read, the spec keeps that capability with `matches`, the row fields by
dotted path (`program.code`, `program.name`) a case-insensitive `query` is matched against after
the read, described by `queryNote`. The query never reaches the server, which has no search for
it, and only an unpaged list may declare it: a paged list's query is the server's.

**Reads alongside.** Where the repository tool returned records beside its own, because the next
call needs their ids (the checklist templates a person starts by hand, the review templates) or
because they say what the tool's records mean (the reconciliation tolerance a drift is judged by,
the queue beside an accounting connection, a sync record's tries), the spec names the operation
(`alongside`) and the key the result carries the records under, and `reads.yml` lists it under the
tool. Any tool reads alongside: an unpaged list, a paged list on every page, and a get beside its
record. The generator proves each read as it proves a named binding: persisted, one root field,
and a resolver that checks a read permission. The tool makes the read only when the reader may
read that kind (`AgentCeilings.Reads`); a reader without it gets the tool's own records, and a
spec's `withheld` note says what is missing ("The queue is not shown: it needs permission to read
the accounting sync ledger"). A refusal is never charged to the budget.

A read alongside takes one of two shapes:

- **Keyed by the call.** An unpaged list or one record (an overview, a summary) whose every
  variable the tool's own operation declares, of the same type. The tool sends it the value the
  call sent, under the variable's name, and makes the read only when the call names every one of
  them: with no worker named, the safety events list reads every worker's events and no
  recognitions. A get's are keyed by the get's own `id` (`DeskAccountingSyncAttempts($id)` reads
  `accountingSyncAttempts(recordId: $id)`). It takes no table input and is never cut short. A read
  alongside marked `scoped` is made only when the call names one of the list's records: the
  credential types come beside one worker's credentials, as the repository tool read them, and
  not beside what falls due across the organization. A read that declares no variable at all is
  about the whole organization and is made on every call: the factoring queue reads its totals
  (`FactoringSummary`, the root `factoringSummary` takes no argument) beside every page, as the
  repository tool did. The generator accepts a root that takes no argument and returns one object
  (not a connection) as such a record.
- **A connection beside an unpaged list**, read through its table input with no other required
  variable, selecting `pageInfo`: at most 100 records, with a `note` when there are more. A paged
  list reads no connection alongside, as its next call pages its own records.

Each read alongside may `shape` its records (`setupPath` on a mapping summary). A list's `combine`
hook applies what they say to its page: the drift list marks each money difference
`withinTolerance` by the overview's tolerance, as the repository row did.

**Reads instead.** A list whose repository tool had a second mode keyed by records of another
shape (whether named documents reached the accounting system, beside listing the ledger) names
that mode's operation under `instead` in `reads.yml` and in its spec. The generator proves it as
it proves a read alongside, and its variables the list's operation does not declare become
parameters of the tool's own (`objectIds`, named `documentIds`); the ones it shares with the list
(`integrationType`) take the list's parameter. Naming one of its own parameters chooses the read:
the tool reads only that operation, refuses any of the list's filters beside it ("parameter
"status" lists records and does not apply to "documentIds""), and returns its records under the
spec's key with its `note` ("A document missing from states was never sent"). A parameter the
list requires and the read made instead does not (`system`, required to list the ledger and
optional for documents) is required by the list's call, and not in the schema's `required`.

**Root arguments beside the table input.** A paged list's operation may take its root field's
own arguments beside its table input: a required `integrationType`, an EDI transfer's
`direction`. Each scalar, enum or id variable is a parameter sent as that variable (`OnInput`, so
not judged by a record field's sensitivity, and not a field filter), typed by the root argument
and described in the schema's words; a required one is required in the schema and refused before
the read when missing. Keyed reads alongside take it as their key. A spec's `names` give a derived
parameter the name the tool takes it by: every accounting read calls `integrationType` `system`,
as the accounting writes do. Where a root argument and a record field derive the same parameter
(a service failure's `shipmentId`), the root argument wins (`withRootArgs`): it is how the screen
itself narrows the list.

**Connections paged by their root.** A connection whose root takes its own `first` and `after`
(`accountingSyncRecords(integrationType:, first:, after:, filter:)`) pages as a table input does:
`limit` is sent as `first`, the unwrapped cursor as `after`. Its filter input's fields are
parameters sent inside it (`statuses`, named `status`), and the input's `search` is the tool's
`query`. Its record type's fields are not filters, so it needs no Go entity and offers no sort;
the near-miss probes drop one filter input field at a time.
A read alongside may instead be an unpaged list keyed by the list's own arguments (a worker's
recognitions beside their safety events). The generator proves that each of its variables is one
the list's operation declares, of the same type, and the tool sends it the value the call sent,
and makes the read only when the call names every one of them: with no worker named, the safety
events list reads every worker's events and no recognitions. It takes no table input and is never
cut short. An optional variable of its own that the list's operation does not declare is the
spec's to set as the screen does (`inputs`): a worker's schedule reads their swaps with
`openOnly: true` and `limit: 20`, as the repository tool did. The generator records those
variables, and a spec that sets any other is refused at construction; a required one the list
does not declare fails generation. A read alongside marked `scoped` is made only when the call names one of the list's
records: the credential types come beside one worker's credentials, as the repository tool read
them, and not beside what falls due across the organization.

**Operations on another bespoke input** (`FuelPurchasesInput`, `WatchtowerItemsInput`,
`FactoringNoaListInput`, the side `filter:` inputs, …). Each input field is a parameter under its
own name, sent inside the input (a side `filter:` beside a table input or a root's own page, as
above):

- enum lists stay lists;
- `Timestamp` becomes `Date`/`DateTime`;
- `first` becomes `limit`;
- `search` becomes `query`; the conventions name free text `query`, and the binding records the
  rename as toolcontract does;
- a wrapped `table: DataTableConnectionInput!` contributes its typed fields by the rule above.

A root whose own input wraps the table (`factoringNoas(input: FactoringNoaListInput!)`, whose
`table` sits beside `customerId` and `statuses`) is read through an operation that spreads the
table input into it, as the screen's table operations do: `DeskFactoringNoas($input:
DataTableConnectionInput!)` sends `factoringNoas(input: {table: $input})`. The tool then builds
the table input and filters by the record's own fields, which already cover what the wrapper's
fields narrow by, so it needs no machinery of its own.

**A list of ids** an unpaged list's argument takes (`vehiclePositions(tractorIds: [ID!])`) is a
`paramReferences` parameter: typed by the spec's `references` as a list of that resource's ids,
each checked before the read, and sent as the list.

**A default for a required argument.** A spec's `defaults` may name a root argument the schema
requires (`ediTransfers(direction:)`, Inbound unless the call names Outbound, as the repository
tool read it). The call that omits it is sent the default, so the tool's schema does not require
it, and the parameter's note says what the default is.

**Instants.** A record's `Timestamp` field gives a day range (`createdAtFrom`/`To`) by default.
Where people bound it to the minute ("messages created since 08:30") the spec names the field in
`instants`, and the two ends take `agenttoolschema.DateTime` (RFC 3339 with its offset) and compile
to `gte`/`lte` on that second. A day sent where an instant is wanted is refused before the read.
A name in `instants` that is none of the record's dated fields fails construction.

**Fields no one narrows by.** Every filterable field is a parameter, and the description names
every one an agent at Internal may use. Where that would bury the filters people use under ones
no one asks for (an EDI message's template, the remote path it was dropped at, its row's
`updatedAt`), the spec's `unoffered` leaves those fields out, and a name in it that is none of the
record's filters fails construction. The resolver still accepts them from the screen.

**Text arguments.** A `String` root argument an unpaged list or a get sends as it is (a
credential type's code, a reconciliation status) takes the schema's description alone, not a
field filter's "contains this text", since the resolver matches it exactly.

**Include switches.** A get whose operation reads a field only on request
(`raw @include(if: $includeRaw)`, a message's raw X12) takes the variable as a switch, described
by the field's own schema description ("true also returns raw: …"), so a call that leaves it off
never reads the field, and a reader whose ceiling withholds it is told so in `withheldByAccess`.

**Numbers.** A model's numbers arrive as `float64`, which gqlgen's `Int` refuses where the HTTP
transport hands it a JSON number, so the reader sends every whole number as an integer.

**Get operations.** A root that takes an `id` and returns one record is a get even when the
record carries a list under `items` (a schedule of accounts and its invoices); only a root with
no `id` reads such a type as an `items` page. The get operation's own arguments become
parameters: the id as `<record>Id`, marked with its resource, and every other scalar, enum or id
variable as it is for a list's root arguments, named by the spec's `names`, typed by its
`references` and noted by its `notes`. A get read by the id of the record it belongs to (an
invoice's factoring readiness by `invoiceId`, a factor's reserve balance by `factoringCompanyId`)
names that parameter in `paramName`, and a `references` entry under the same name types it as the
other record's id rather than the get's own. A get need not take an id: `accounting_get_connection` reads `accountingSyncStatus(integrationType:)`,
and `accounting_get_mapping` names a mapping by its id or by its target type with a record or a
key. A get's `shape` applies to its record what the screen's resolver does not (what a sync
record's status means). A get keyed by the id of another kind's record (`settlement_get_driver_earnings` reads `workerEarningsSummary(workerId:)`) names that kind in `keyResource`, so its parameter is typed as a worker id from `worker_list`. A human key the person may say instead (a PRO number) is a second,
optional parameter. The generator resolves it through the list operation's exact filter on that
field, so `shipment_get` with `proNumber` reads `shipments` filtered by `proNumber`, then the card.
A get whose operation takes a required input of optional fields only (`todaysBriefing(input:
TodaysBriefingInput!)`) takes those fields as parameters, renamed by the spec's `names`, and sends
the input empty when the call names none of them; the generator refuses one with a required field.
A described enum or text field of such an input says only its own description, since it picks the
record rather than filtering rows.
A root with no arguments at all that returns one record is a get too: `home_get_layout` reads
`homeLayout`, the caller's own page, and takes no parameter. Such a root is usually the caller's
own record, whose resolver checks no resource and is listed in `selfscoped.Resolvers`; the
generator binds it as `SelfScoped`, and its tool's policy must be scoped to the caller
(`scope: agent.ToolScopeSelf` on the spec), since that policy is the only check the read meets.
A get whose operation reads one page of a list the record holds (a rate matrix's
`cells(offset: $cellOffset)`) names it in `pages`: the result adds `cellsHasMore` and, while more
follow, `nextCellOffset`. A variable sent to a field's argument below the root is described by
that argument's schema description, as a root argument's is.
A variable the operation sends to a field of the record rather than to its root is a parameter
the same way: `factoring_get_remittance` takes `matchStatuses`, which `DeskFactoringRemittance`
sends to the remittance's `lines(matchStatuses:)`, so one read returns the record and only the
lines the call asked for.

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

A person's table pages the same list again through the Desk's rows endpoint (see "A table that
pages" in [desk-artifacts.md](desk-artifacts.md)), with the cursors the tool handed out. Only that
read sets `QueryToolParams.IncludeTotal`, which sends `includeTotalCount: true` and returns the
connection's `totalCount` as `total`; the model's reads never pay for the count, and the
parameter is not in any tool's schema.

Three kinds of root field do not page this way, and their tools say so:

- limit/offset fields (`carrierInvoiceMatches`, `ediCarrierInvoices`, `mySettlements`,
  `workerCredentials`): offset stays internal; the tool hands out `nextCursor` as an opaque encoded
  offset, so the model learns one convention. An unpaged list whose root takes both a `limit` and
  an `offset` pages this way through `readOffsetWindow`: `offset` is never a parameter, `after`
  is, the read asks for one row past the page, and the cursor is bound to the call's other
  arguments as a connection cursor is. A root that returns its records under `items`
  (`ediCarrierInvoices: EdiCarrierInvoiceList`) is read as an unpaged list of those items
  (`graphQLRead.Items`), and the rule finds its operation as it does a list's; such a root must
  declare `@authorizes` for its kind, since the field guard reads a record under a container that
  is not a connection as carried by it;
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

**A field that carries a record or a part is a field too.** `worker`, `lines` or `exceptions` on a
settlement is judged by its own name on the kind holding it before anything inside it is: a kind
with a Restricted default that does not declare it withholds the whole driver, every line or every
exception at Internal, however its inner fields are ranked. Declaring a kind's sensitivities
therefore declares its relation and part fields (Internal, since what they carry is judged again
by its own kind or field), as `driver_settlement` and the other pay kinds do.

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
| Parties | `customer_get` | none | reads `DeskCustomerCard`, named in `reads.yml` because `DeskCustomerUpdatePreferences` reads the same root: the customer, its state and its billing profile with the documents each invoice needs. Every field it selects is Internal |
| Parties | `customer_get_update_preferences` | none | reads `DeskCustomerUpdatePreferences`, a new operation on `customer(id)` selecting the preference, the recipients and the notice profile's recipients; every field is Internal. The spec's `shape` (new on gets) words what the repository tool computed: `wantsArrivals`, `wantsDepartures`, the warning for a customer set to None, and `fallbackRecipients` when the customer names none, and says none of it where the preference was withheld |
| Parties | `location_get` | none | reads `DeskLocationCard`; every field it selects is Internal |
| Parties | `carrier_get` | none | reads `DeskCarrierCard`, named in `reads.yml` because `DeskCarrierIntel` reads the same root; every field it selects is Internal |
| Parties | `carrier_get_intelligence_event` | none at Internal | reads `CarrierIntelEvent`. `CarrierIntelEvent` had no kind (it is not named for `carrier_intelligence`), so the guard did not gate it and a reader below Internal got every field; it now declares `@kind(resource: "carrier_intelligence")`, so its fields follow `carrier_intelligence`'s Internal default on every read of the type, the carrier intelligence screens' included, as the repository tool gated them. Who acknowledged or resolved it shows the user reference view |
| Equipment | `equipment_list_tractors` | none | a driver (`primaryWorker`, `secondaryWorker`) shows the worker reference view (the driver's name) to a reader who cannot read workers, and nothing else |
| Equipment | `equipment_list_trailers` | none | every field the table selects is Internal |
| Equipment | `equipment_get_tractor` | none | reads `DeskTractorCard`; its drivers and owner show the worker reference view (their names) to a reader who cannot read workers, and nothing else |
| Equipment | `equipment_get_trailer` | none | reads `DeskTrailerCard`; every field it selects is Internal, and its owner shows the worker reference view |
| Equipment | `equipment_list_positions` | none | reads `DeskVehiclePositions`, written for it: each tractor's latest reading. `vehiclePositions` gains `tractorIds`, which `TelematicsService.ListVehiclePositions` passes to the repository's own filter (the shipment tracking tool now names its tractors too, rather than reading the fleet and keeping one). `maxAgeMinutes` becomes the argument's `maxAgeSeconds`, four hours when left out as before. Each row gains `ageMinutes` from its `recordedAt` and the call's clock (`shape` now takes the clock). The repository tool left out the driver's name for a reader who cannot read workers; the row's `primaryWorkerName` is the name the worker reference view shows on the tractor list and card to any tractor reader, so it names the driver as they do |
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
| Workforce | `worker_list_credentials` | none | reads `DeskWorkerCredentialList`, with the active credential types read alongside (`DeskWorkerCredentialTypes`) when the call names a worker. `workerCredentials` takes an optional worker and the repository tool's filters (`expiringWithinDays`, `includeExpired`, `credentialTypeCode`, `requiredOnly`, `limit`), read by `WorkerCredentialService.List`: one worker's credentials, or with no worker or an expiry window the active ones falling due, soonest first, walked in the query. The window counts days from now, as the screen's expiry forecast does and the description says, where the repository tool counted calendar days in the caller's zone: GraphQL carries no caller's day. It pages by cursor over the root's `limit` and `offset`, as the repository tool paged by offset. `worker_credential` defaulted to Restricted with only `number` declared (Confidential), so every field outside its reference view would be withheld at Internal; what a credential is, whose, its dates, health, issuer, verification and whether a scan is on file are now Internal, and the notes and archive reason stay Restricted. The operation never selects `number`, which the repository tool withheld too |
| Workforce | `worker_list` | none | reads `DeskWorkers`, which selects only what the Desk reads (no address, contact, personnel or Confidential profile fields); `city` is Restricted and withheld at Internal, as the repository row gated it |
| Operations | `order_list` | none | reads `OrderTable`; every order field is Internal, as the repository row gated its amounts. `order` declares an "own" scope, and a person narrowed to their own orders reads the same rows on both paths (`TestConvertedLists_OrderListIsNarrowedToTheOwnerOnBothPaths`) |
| Operations | `order_get` | none | reads `DeskOrder`, written for it: the card's fields with every leg the order groups (`legs`), named in `reads.yml` because `DeskOrderCard` pages its legs through shipments. The repository tool cut the legs at 50 and the charges at 100; the card reads them whole, as the order page does. The customer shows its reference view |
| Operations | `shipment_list_recurring` | none | reads `DeskRecurringShipments`, written for it, which adds the last shipment a series made (`lastGeneratedShipmentId`) to what the table shows; every field is Internal |
| Operations | `exception_list_service_failure_reason_codes` | none | reads `ServiceFailureReasonCodeTable`; it reads the active codes unless the call names `active`, as the repository tool always did, and pages where that tool read one page of 50 |
| Operations | `exception_list_service_failures` | none | reads `ServiceFailureTable`, named in `reads.yml` because its row carries the `version` resolving a failure takes and `DeskRecordFailures` does not; the shipment and stop show their reference views |
| Operations | `exception_get_service_failure` | none | reads `DeskServiceFailureCard`; who created, reviewed, resolved or voided it show the user reference view, which is what the repository tool reduced them to |
| Operations | `detention_get_occurrence` | none at Internal | reads `DeskDetentionCard`, with the evidence, notices, collectability and the price now. `DetentionOccurrence`, `DetentionOccurrenceDetail` and `DetentionDeskEntry` had no kind, so the guard did not gate them; each now declares `@kind(resource: "detention")`, so their fields follow `detention`'s Internal default on every read, the detention screens' included |
| Operations | `detention_list_occurrences` | none at Internal | reads `DeskDetentionOccurrences`, written for it. `detentionDesk` gains `includeAwaitingApproval`, `urgency`, `shipmentId` and `proNumber` (a PRO number or part of it), read by `DetentionService.ListDeskEntries`, and `DetentionDeskUrgency` gains `AwaitingApproval`, so the charges holding an invoice that the repository tool read beside the board are on the board's own read; the spec reads them by default. Each entry gains `billableMinutes`, priced as of now for a running clock. The tool no longer says which other urgencies a load is at when the one asked for matches nothing; the empty note names the filters instead |
| Operations | `dispatch_list_tenders` | none | reads `DeskShipmentTenders`, written for it, which leaves out what a carrier wrote (a decline reason, a counter note) and where an offer was delivered, as the repository tool did, so the tool still reads nothing from outside. `tendersByShipment` gains `shipmentMoveId`, filtered in the repository within the shipment's tenders. The spec's `shape` adds `live`, each offer's `answerable` and `backOutAllowed`, and keeps `counterNeedsApproval` on a standing counter only, from the tender rules (`tender.Tender.IsLive`, `CanBounce`) |
| Accounting | `accounting_list_drift_findings` | none | reads `DeskAccountingDriftFindings` with the overview alongside (`DeskAccountingDriftOverview`), which gives the tolerance each money difference is marked `withinTolerance` by. `AccountingDriftFinding` and `AccountingDriftOverview` had no kind (a record kind is not a resource), so the guard did not gate them; both now declare `@kind(resource: "accounting_sync")`, whose fields default to Internal, as the repository row read them |
| Accounting | `accounting_list_inbound_changes` | none | reads `DeskAccountingInboundChanges`; `AccountingInboundChange` now declares `@kind(resource: "accounting_sync")` |
| Accounting | `accounting_list_sync_records` | none | reads `DeskAccountingSyncRecords`, or, when the call names documents, `DeskAccountingSyncObjectStates` instead. Neither selects `errorMessage` or `errorCode`, the accounting system's own words, which the repository row never returned. `AccountingSyncRecord` and `AccountingSyncObjectState` now declare `@kind(resource: "accounting_sync")`. `accountingSyncObjectStates` gains an optional `integrationType` (one safe change), as the repository tool narrowed by system |
| Accounting | `accounting_get_sync_record` | none | reads `DeskAccountingSyncRecord` with its tries alongside (`DeskAccountingSyncAttempts`, no error message); `AccountingSyncAttempt` now declares `@kind(resource: "accounting_sync")` |
| Accounting | `accounting_get_connection` | none | reads `DeskAccountingConnection` with the queue alongside (`DeskAccountingSyncSummary`) only for a reader of `accounting_sync`, as the repository tool did. `AccountingSyncStatus` now declares `@kind(resource: "accounting_integration")` and `AccountingSyncSummary` `@kind(resource: "accounting_sync")`. `accounting_integration` defaults to Restricted with ten fields declared, so every connection field but those would be withheld at Internal; the connection's operational state (status, setup step, sending mode and dates, failure count and category, certificate expiry, pause) is now declared Internal, and its app keys and every field not declared stay Restricted. A failure the accounting system worded itself (`Unknown`) is replaced by a pointer to the connection page, as `AgentErrorSummary` did |
| Accounting | `accounting_list_mapping_gaps` | none | reads `DeskAccountingMappings` (states Unmatched and Proposed unless the call names others) with the summary alongside (`DeskAccountingMappingSummary`). `AccountingMapping` and `AccountingMappingSummary` now declare `@kind(resource: "accounting_integration")`; what a mapping maps, its state, the chosen record and the candidates are declared Internal |
| Accounting | `accounting_get_mapping` | none | `partial`: no root read one mapping. `accountingMappingDetail` (one safe addition) names it by id or by target type with a record or a key, as `FindMapping` does, and returns the usable records it could be set to for an `optionsQuery`, as the repository tool searched them; checked by `accounting_integration` read. `AccountingMappingDetail` declares `@kind(resource: "accounting_integration")` |
| EDI | `edi_list_partners` | none | reads `DeskEdiPartners`, written for it: what a partner row shows, without its contact details or settings. `edi` defaulted to Restricted with no field sensitivities, so every field but `id` would be withheld at Internal; a partner's, message's, file's and tender's operational fields are now declared Internal, and a partner's contact email and phone, the raw X12 and file text, a tender's payload and rate and the mapping snapshot stay Restricted, as the repository tools gated them |
| EDI | `edi_list_inbound_files` | none | reads `DeskEdiInboundFiles`. `EdiInboundFile` had no kind (it is not named for `edi`), so the guard did not gate it; it now declares `@kind(resource: "edi")`. The ISA sender and receiver ids are parameters named `isaSender` and `isaReceiver`, since they are a partner's identifiers and no record's id |
| EDI | `edi_list_transfers` | none | reads `DeskEdiTransfers`. `EdiTransfer` now declares `@kind(resource: "edi")` and gains what the repository row read from the tender (BOL, customer, service type, stop count) as fields of its own, with its `direction` and `counterparty` judged from the caller's organization, so a row never reads the Restricted payload. `ediTransfers` gains an optional `inboundFileId` (one safe change), as the repository tool narrowed by the file; `direction` is Inbound unless the call names one, as before |
| EDI | `edi_list_messages` | none | `partial`: reads `DeskEdiMessages` (named, since `DeskEdiPartnerMessages` reads the same root for one partner's card). `EdiMessage` now declares `@kind(resource: "edi")` and gains `invoiceId` (a safe addition), which, as a filterable field, narrows by invoice as the repository tool did; the creation window is `createdAtFrom`/`To` to the minute (`instants`), where the repository tool took Unix seconds |
| EDI | `edi_list_carrier_invoices` | none | reads `DeskEdiCarrierInvoices`, an `items` list paged by its root's limit and offset. `EdiCarrierInvoice` had no kind; it now declares `@kind(resource: "carrier_invoice_match")`, whose root `ediCarrierInvoices` now declares `@authorizes` for it, and the kind's fields are declared Internal but the totals, which stay Restricted as the repository row gated them. The suggested carrier is the type's `suggestedCarrier`, resolved only for an invoice no carrier is linked to, from the invoice already read (one carrier lookup per such row, where the repository tool read the invoice again first), and shows the carrier reference view. An unknown reconciliation status is refused |
| EDI | `edi_get_partner` | none | reads `DeskEdiPartnerCard`, named because no rule finds it from the record kind; its readiness items gain their `label` (a safe addition), as the repository tool listed them. The card also carries each transport's settings, which the repository tool never returned: `EdiCommunicationProfile` now declares `@kind(resource: "edi")` and its `config` is Confidential, so no agent is ever shown it; its credentials are named and said to be on file, never read |
| EDI | `edi_get_inbound_file` | none | `partial`: reads `DeskEdiInboundFile`, through `ediInboundFile` (one safe addition) and its `EdiInboundFileDetail`, which carries the file, its profile's name, the first fifty transaction sets parsed from it and, on `includeRaw`, its first 16 KB (`raw`, Restricted) |
| EDI | `edi_get_message` | none | `partial`: reads `DeskEdiMessage`, through `ediMessage` and its `EdiMessageDetail`: the message, the document it decoded to, the first fifty findings with their count and, on `includeRaw`, the raw X12's first 16 KB (`raw`, Restricted) |
| EDI | `edi_get_transfer` | none | `partial`: reads `DeskEdiTransfer`, through `ediTransfer` and its `EdiTransferDetail`: the tender's stops, commodities and rate (`charges`, Restricted, as the repository row gated it), whether it can be decided and accepted, and, for an inbound tender still open, the codes accepting it waits on (`MappingPreviewOf`, from the tender already read) |
| EDI | `edi_get_ack_health` | none | `partial`: reads `DeskEdiAckHealth`, through `ediAckHealth` (one safe addition), which takes the partner, transaction set, overdue age and window the repository tool took and refuses one out of bounds; a 997 or 999 counts every set it answers in the service, and the tool's `shape` words each partner's answer and how long each overdue message has waited from the read's `asOf` |
| Ledger | `ledger_list_journal_entries` | none | reads `DeskJournalEntries` on the new `journalEntries` root (journal_entry read), which lists every entry whatever its status, newest accounting date first, as `JournalEntryService.List` did; `journalReviewTable` reads only the entries still waiting to be approved or posted, so it could not serve the tool. The totals stay Restricted and withheld at Internal, as the repository row gated them. `JournalEntry` gains `entryDate`, `approvalNotes`, `rejectedAt` and `rejectionReason`, which the repository tools returned |
| Ledger | `ledger_list_fiscal_periods` | none | reads `DeskFiscalPeriods` on the new `fiscalPeriods` root (fiscal_period read), latest start first; periods were reachable only under `FiscalYear.periods`, with no filter, sort or paging of their own. `FiscalPeriod` gains `reopenedAt` and `reopenReason`; the reason stays Restricted |
| Ledger | `ledger_list_manual_journals` | none | reads `DeskManualJournals`; totals, reason and the rejection and cancel reasons stay Restricted, as the repository row gated them. `ManualJournal` gains `isBalanced` (Internal), which the repository row returned ungated |
| Ledger | `ledger_list_journal_reversals` | none | reads `DeskJournalReversals`; the reason text and the rejection and cancel reasons stay Restricted, as the repository row gated them |
| Ledger | `ledger_get_journal_entry` | none of what it returned | reads `DeskJournalEntryCard`. `JournalEntryCard` had no kind, so its lines' debits and credits were not gated on any read; it now declares `@kind(resource: "journal_entry")`, so they are Restricted, as the repository tool gated them. The card's own fields (`entry`, `lines`, `period`, `posting`, `source`, the people and the related entries) and a line's account, category and party names, and the posting's refusal, are declared Internal on journal_entry; a record the card carries is still decided by its own kind. The card adds what the repository tool did not read: the reversal request against the entry, whose reason text and rejection and cancel reasons are journal_reversal's and Restricted, and a line's customer and location only to a reader of those kinds. Lines gain `isReconciled` |
| Ledger | `ledger_get_manual_journal` | none of what it returned | reads `DeskManualJournalCard`. `ManualJournalCard` now declares `@kind(resource: "manual_journal")` for the same reason; its own fields (`journal`, `lines`, `requestedPeriod`, `posting`, the people, `entries`), the lines' account and party names, their customer and location, `postedAt` and the posting's refusal are declared Internal, and their amounts stay Restricted. The entries it became are journal entries carried by the card, shown whole only to a reader of journal entries |
| Rates | `rate_list_agreements` | none | reads `DeskRateAgreements`, which selects what the repository row carried and nothing more; the agreement's tariff `inputs` are Restricted and withheld at Internal, as the row gated them. The margin guardrails are `@protected` and never selected |
| Rates | `rate_get_agreement` | none | reads `DeskRateAgreementCard` on the new `rateAgreement(id)` root. Its lanes are the new `lanes(first: 50)` field, the live lanes (current and starting later) the Desk's card pages, with `laneCount`, where the repository tool loaded every rule in force and cut the result at 50. The agreement's and each lane's tariff `inputs`, an accessorial's amount, cap and condition, the review comment and the version's change message stay Restricted, as the repository view gated them; the container fields (`lanes`, `laneCount`, `accessorials`, `fuelBinding`, `currentVersion`) and `fuelSurchargeProgramId` are declared Internal so their Restricted leaves are judged one by one. Inputs come as Tariff wire values, where the repository view wrote them as display text |
| Rates | `rate_get_matrix` | none | reads `DeskRateMatrixCard`, 100 cells at a time from `cellOffset` (the repository tool read 200), with `cellsHasMore` and `nextCellOffset` from the spec's `pages`; the spec's `shape` folds each cell's per-axis columns into `keys` and `ranges` along the matrix's own axes, as the repository row did. A cell's `value` and `minCharge` stay Restricted under `rate_matrix` and are withheld at Internal, as the repository view gated them |
| Insights | `insight_get_daily_briefing` | none | reads `TodaysBriefing`, the Desk home's own operation, with `role` and `date` sent on `TodaysBriefingInput` (as `roleKey` and `briefingDate`). The per-section gate the repository tool applied now lives in `BriefingService` (`briefingservice/access.go`) and cuts every read of a page, `todaysBriefing`, `briefing`, `briefings`, `markBriefingRead` and `regenerateBriefing`, with one batched permission check: a section whose records the reader may not read is left out, a watchtower section keeps the lines of the kinds they may read, and while anything is withheld the headline and the model's sentences go too. `Briefing.withheldSections` names what was cut, and the Desk home says so. This narrows the screen, which showed every section to anyone with briefing read; the tool shows exactly what it showed before |
| Home | `home_get_layout` | none | reads `HomeLayout`, the caller's own page, through `homeLayout`, a root with no arguments whose resolver is now listed in `selfscoped.Resolvers` (it was authzlint's own exemption) and checks no resource; the tool's policy stays home layout read, scoped to the caller. The spec's `shape` names each widget by the catalog and says whether the page may be changed, as the repository view did. `HomeLayout` carries no linked kind, so nothing is newly shown |
| Rates | `rate_list_fuel_index_prices` | none | reads `DeskFuelIndexPrices`. `FuelIndexPrice` had no kind, so nothing was gated; it now declares `@kind(resource: "fuel_surcharge_program")` (Internal by default), as do `FuelIndexLatestPrice` and `FuelProgramCurrentRate` that carry it |
| Rates | `rate_list_fuel_surcharge_programs` | none | reads `DeskFuelProgramCurrentRates`; `FuelProgramCurrentRate` declares `@kind(resource: "fuel_surcharge_program")` and gains `id`, the program's. The repository tool's per-row note (no price on file, a price outside the table) is the spec's `shape`, and is left off a row whose price or rate was withheld |
| Workforce | `worker_get` | `leaveType`, `profile.nextTrainingDue`, `profile.safetyRating` | reads `DeskWorkerCard`, which now also selects what the repository row carried (postal code, emergency contact, leave type, termination, last MVR check, next training due, safety rating, compliance status, ELD and short-haul exemptions). `worker` declares none of the three newly withheld fields, so they take its Restricted default, where the repository row returned them ungated; they were not declared Internal, which would widen what a person whose role reaches only Internal sees. The contact, address, hire and termination dates and disqualification reason stay Restricted, as the repository row gated them. `profile.licenseNumber` is Confidential and now named in `withheldByAccess` rather than left out silently. `worker` now declares `profile` Internal: the field that holds the profile took the Restricted default, so the guard withheld the whole profile at Internal (`worker_list` included) even though each of its fields carries its own sensitivity, which still applies. The spec's `shape` folds a NonCompliant credential into `canBeAssigned`, as `worker_list` does |
| Workforce | `worker_get_credential` | `notes`, `archiveReason` | reads the new `DeskWorkerCredential`; `worker_credential` keeps both Restricted, where the repository row returned them ungated. The operation never selects `number`. `worker_credential` now declares its `worker` and `credentialType` links Internal: they took the Restricted default, so at Internal the guard withheld the holder and the type outright (`worker_list_credentials` included); each linked record is still gated by its own kind, so the holder shows the worker reference view to a reader who cannot read workers |
| Workforce | `worker_get_dot_random_draw` | none | reads `DotRandomDraw`. `DOTRandomDraw` and `DOTRandomDrawEntry` had no kind (neither is named for `dot_random_pool`), so the guard did not gate them and a reader below Internal got every field; both now declare `@kind(resource: "dot_random_pool")`, whose fields default to Internal, on every read of the type, the random testing page's included. Each selection's `worker` shows the worker reference view (the name the repository row carried) to a reader who cannot read workers |
| Workforce | `worker_get_hos` | none | reads `WorkerHosState`, the telematics page's single-root read; `DeskWorkerHos` reads the violations beside it, which the tool never returned. `WorkerHosState` has no kind and stays ungated, as the repository tool and the page read it: declaring `worker` would put every clock under the worker's Restricted default |
| Workforce | `worker_get_schedule` | none | reads `WorkerShiftAssignments` with `WorkerAvailabilityPreferences` and `ShiftSwapRequests` alongside, keyed by the worker; swaps are read only for a reader who may read `shift_swap`, open only, at most 20, as before. The resolvers' team scope governs. `WorkerShiftAssignment` and `WorkerAvailabilityPreference` now declare `@kind(resource: "worker_schedule")` and `ShiftSwapRequest` `@kind(resource: "shift_swap")`, all Internal by default; each assignment's `shiftTemplate` shows the shift template reference view (code, name, colour, start, length) to a reader who cannot read shift templates, and a swap's workers the worker reference view |
| Workforce | `worker_list_dot_random_draws` | none | reads `DotRandomDraws` (one pool's or every pool's, the server's page of 50) with the active pools alongside (`DotRandomPools`); every field is Internal |
| Workforce | `worker_list_shift_templates` | none | reads `ShiftTemplates`, the active patterns unless the call asks for the retired ones, 25 unless it asks for more, at most 100; every field is Internal. The spec's `shape` spells out the days, start and length |
| Formula | `formula_list_templates` | none | reads `DeskFormulaTemplates`, added for the Desk: the row the repository tool returned (name, description, status, version, test count), not the imports and usage `FormulaTemplateTable` selects for the editor's table. Every `formula_template` field is Internal. The repository list was not gated, so at a role ceiling below Internal it now withholds every field but `id` |
| Formula | `formula_get_template` | none | reads `DeskFormulaTemplateCard`, added for the Desk: the template whole, as the repository record was (Tariff source, inputs, imports, review state, version), with `usageCount` (one batched loader read) beside it. Every field is Internal, as the repository tool's redactor gated it |
| Formula | `formula_list_libraries` | none | reads `DeskFormulaLibraries`, added for the Desk: path, description, status, version and test count, as the repository row carried. Every `formula_library` field is Internal; the repository list was not gated, so below Internal it now withholds every field but `id`. The tool came with Tariff after the inventory was taken |
| Formula | `formula_get_library` | none | reads `DeskFormulaLibraryCard`, added for the Desk (the screen's `FormulaLibrary` reads three roots, its versions and reviews, which a get cannot bind): the library whole. Every field is Internal, as the repository tool's redactor gated it |
| Fuel | `fuel_list_cards` | none | reads `DeskFuelCards`, added for the Desk: what the repository row carried (provider, last four, label, status, expiry, who carries it), with `unassigned` and `discoveredByFeed` kept by the spec's `shape`. Every `fuel_card` field but `externalCardId` (Confidential, never selected, never a parameter) is Internal; the repository list was not gated, so below Internal it now withholds every field but `id`. The tractor and driver show their reference views to a reader who cannot read them |
| Fuel | `fuel_list_purchases` | none | reads `DeskFuelPurchases`, added for the Desk: the repository row's fields, with the tractor, driver and jurisdiction as linked records. Every `fuel_purchase` field is Internal, so the amount and card digits read as before at Internal; below Internal everything but `id` is withheld, where the repository list was ungated |
| Fuel | `fuel_list_purchase_imports` | none | reads `DeskFuelPurchaseImports`, added for the Desk (the table's selection carries the column mapping and per-jurisdiction breakdowns). `FuelPurchaseImportBatch` had no kind, so the guard did not gate it; it now declares `@kind(resource: "fuel_purchase_import")`, and `fuel_purchase_import` gains a reference view (file name, origin, provider, row count) so the fuel card and purchase cards that carry a batch still name it to a reader of theirs. Every field is Internal |
| Fuel | `fuel_list_ifta_returns` | none | reads `DeskIftaReturns`, added for the Desk: the repository row's figures and each return's problems, with `blockingProblems` counted by the domain's `ProblemCode.Blocks` in the spec's `shape`. Every `ifta_return` field is Internal |
| Fuel | `fuel_list_ifta_mileage_entries` | none | reads `DeskIftaMileageEntries`, added for the Desk; every `ifta_jurisdiction_mileage` field is Internal |
| Fuel | `fuel_list_ifta_jurisdictions` | none | reads `DeskIftaJurisdictions`; `iftaJurisdictions` gains `countryCode` and `activeOnly`, which the resolver passes to `ListJurisdictions`, and the spec reads the active ones by default as the repository tool always did. The jurisdictions are global reference data under the root's `fuel_purchase` check and stay ungated, as the repository tool read them: declaring a kind on `IFTAJurisdiction` would withhold a mileage entry's or purchase's jurisdiction code from a reader without fuel purchase read |
| Settlements | `settlement_list_driver` | none | reads `DriverSettlementTable`, named because `mySettlements` is the driver portal's own read of the record. The pay figures (gross, reimbursements, deductions, carry-forwards, net, miles) are Restricted and withheld at Internal, as the repository row gated them; each settlement's `worker` (now declared Internal on `driver_settlement`) shows the worker reference view. `query` matches the settlement number and pay profile name, no longer the driver's name: a driver is named by `workerId` from `worker_list`. `open: true` becomes `status` with Draft, PendingApproval and Approved, which its note says, and `hasExceptions: false` now narrows to settlements without exceptions |
| Settlements | `settlement_list_pay_events` | none | reads `DriverPayEventTable`. `DriverPayEvent` had no kind, so the guard did not gate it; it now declares `@kind(resource: "driver_settlement")`, and its gross, miles, hold and void reasons follow `driver_settlement`'s Restricted fields, as the repository row gated the gross and hold reason (it did not return the void reason) |
| Settlements | `settlement_list_carrier` | none | reads `CarrierSettlementTable`, named because `DeskCarrierSettlements` reads one carrier's settlements for its card and names no carrier. The gross, adjustments and net payable stay Restricted, as the repository row gated them; the payment method and reference, which the repository row did not return, are Restricted on `carrier_settlement`. `carrier` and `lines` are now declared Internal so the carrier shows its reference view |
| Settlements | `settlement_list_driver_expenses` | `amount` | reads `DriverExpenseTable`. `driver_expense` declared no field sensitivities, so under its Restricted default every field but `id` would be withheld at Internal; who submitted it, its status, dates, receipt, review and links are now Internal, and the amount, the driver's description and the review note stay Restricted. The repository row returned the amount ungated, so it is newly withheld at Internal and the description says so. Marked: a row is marked as read from outside only where the description is shown |
| Settlements | `settlement_list_pay_profiles` | none | reads `PayProfileTable`. `PayProfile` had no kind; it now declares `@kind(resource: "driver_pay_profile")`. `driver_pay_profile` declared no field sensitivities, so every field but `id` would be withheld at Internal; a profile's name, classification, status, currency, description and its components' kind, method and description are now Internal, and the guaranteed minimum, per-diem rate and cap, each component's rate, bands and amount bounds stay Restricted |
| Settlements | `settlement_list_escrow_accounts` | none | reads `EscrowAccountTable`; the balance, target and interest rate stay Restricted, as the repository row gated them. `query` no longer resolves a driver's name to an id: a driver is named by `workerId` from `worker_list` |
| Settlements | `settlement_list_advances` | none | reads `PayAdvanceTable`; the amount, recovered, written off and outstanding figures, notes and write-off reason stay Restricted, as the repository row gated the amounts. The repository tool's "no advance in this status" note becomes the generic near-miss probe |
| Settlements | `settlement_list_recurring_deductions` | none | reads `RecurringDeductionTable`; the amount, cap and amount taken stay Restricted, as the repository row gated them. `payCodeId` and `escrowAccountId` become typed references |
| Settlements | `settlement_list_recurring_earnings` | none | reads `RecurringEarningTable`; the amount, cap and amount paid stay Restricted, as the repository row gated them. `payCodeId` becomes a typed reference |
| Settlements | `settlement_list_pay_assignments` | `notes` | reads `WorkerPayAssignments`, an unpaged list keyed by the worker. `WorkerPayAssignment` had no kind; it now declares `@kind(resource: "driver_pay_profile")`. The split percent and the assignment's notes are Restricted (the repository row returned the notes ungated); each row's `payProfile` names the profile. `inForce` is the spec's `shape`, from the call's clock |
| Settlements | `settlement_list_pay_codes` | none | reads `DeskPayCodeOptions`, written for it (no operation read `payCodeOptions`): the active codes, narrowed by `direction`. Every field it selects is Internal |
| Settlements | `settlement_get_driver` | none | reads `DeskDriverSettlementStatement`, written for it: the Desk's card sums the lines by how each was paid, and the adjustment tools take a line's id. The disputes on it come with their status (all of them, not only the open ones), and only for a reader who may read `settlement_dispute`, as the resolver gives them. The settlement's `exceptions` (code, severity, message) and `lines`, `disputes` and `worker` are now declared Internal; each line's quantity, rate and amount stay Restricted |
| Settlements | `settlement_get_driver_earnings` | none | reads `WorkerEarningsSummary`, a get keyed by the worker. `WorkerEarningsSummary` had no kind, so the guard did not gate it on the settlement workspace either; it now declares `@kind(resource: "driver_settlement")`, the accrued count is Internal and the accrued gross, outstanding advances and escrow balance are Restricted, as the repository tool gated them. `outstandingAdvances` holds minor units under an unmarked name, so the spec names it in `minorUnits` |
| Settlements | `settlement_get_dispute` | none | reads `SettlementDisputeDetail`. `settlement_dispute` declared no field sensitivities; its status, category, links, who submitted and resolved it and when are now Internal, and the driver's description and the resolution note stay Restricted, as the repository tool gated them. Marked only where the description is shown |
| Settlements | `settlement_get_carrier` | none | reads `CarrierSettlementDetail`, named because the Desk's card sums the cost lines by kind and the tool returns each line by id. `lineNumber`, `costEventId` and `glAccountId` are now declared Internal on `carrier_settlement`; the payment method, which the repository tool gated as Restricted only when set, is Restricted |
| Billing and receivables | `invoice_list` | none | reads `DeskInvoices`, written for it: the register row the repository tool returned (number, statuses, bill-to, PRO, total, dates). The total stays Restricted and withheld at Internal, as the repository tool's `fetchGated` withheld it. The catalog's `id` filter is gone; `invoice_get` reads several invoices by id. Under an "own" read scope both paths read the invoices of the person's shipments (`TestConvertedLists_InvoiceListIsNarrowedToOwnShipmentsOnBothPaths`) |
| Billing and receivables | `invoice_get` | none | reads `DeskInvoiceDetail`, named in `reads.yml` because the Desk's card leaves the lines to the invoice's page and an adjustment names a line by its id. Up to 50 ids, one read each, at most 10 at once. `Invoice` gains `memo`, `remittanceInstructions`, `lastSendError`, `supersedesInvoiceId` and `supersededByInvoiceId`, which the repository tool read. `scope`, `shipmentCount`, `voidedAt`, `ediSendStatus`, `lines` (the list itself: undeclared, the guard withheld every line) and a line's `chargeCode` and `accessorialChargeId`, which it returned ungated, are now declared Internal; the amounts, the memo, the remittance instructions, the void reason and the send and EDI errors stay Restricted, as it gated them |
| Billing and receivables | `invoice_get_adjustment` | `reason`, `policyReason`, `rejectionReason`, `creditQuantity`, `rebillQuantity` | reads `DeskInvoiceAdjustmentCard`, written for it. `InvoiceAdjustment` had no kind (it is not named for a resource), so the guard did not gate it; it now declares `@kind(resource: "invoice")`, as the repository tool gated its amounts by the invoice's sensitivities, and the invoice's declarations for an adjustment's reasons (Restricted) now apply to it on every read of the type. The Go type is not added to `responseguard.namedKinds`: `invoice` is database-scoped, so the REST adjustment routes would then return invoices outside the invoice's handler. `executionError` and the new `supportingDocumentIds` are declared Internal. The description says the reasons can be withheld |
| Billing and receivables | `invoice_list_disputes` | `notes`, `resolutionNotes` | reads `DeskInvoiceDisputes`, a new operation. `invoiceDisputes` takes `invoiceIds` (at most 100) beside `invoiceId`, which becomes optional; the tool sends the list as a `paramReferences` parameter. `invoice_dispute` declared no field sensitivities, so under its Restricted default every field but `id` would be withheld at Internal; the case's invoice, customer, status, reason, dates and resolution are now Internal, and the disputed amount (gated by the repository tool too) and the two notes stay Restricted |
| Billing and receivables | `receivable_list_customer_payments` | `amountMinor`, `appliedAmountMinor`, `unappliedAmountMinor`, `memo`, `applications.appliedAmountMinor`, `applications.shortPayAmountMinor` | reads `DeskCustomerPayments`, written for it. The repository tool returned every column ungated; `customer_payment` already declares the amounts and the memo Restricted, so they are withheld at Internal now, and the description says so; `applications`, the list itself, was undeclared and is declared Internal, so the invoices a payment settled still show. Amounts are in minor units, as the screen reads them, and the summary says how to read them. Posted payments are read unless the call names a status, as before |

| Factoring | `factoring_list_readiness` | none | reads `DeskFactoringInvoiceStates` (written for it, the queue's input spread from the table input) with the queue's totals alongside (`FactoringSummary`, read on every call). `FactoringInvoiceState` and `FactoringSummary` had no kind, so the guard did not gate them; both now declare `@kind(resource: "invoice")`, the resource their roots check, and `invoice` declares what a readiness state carries (verdict, failed checks, the checks' code, message and fix, the dates, the factor) Internal. The open balance and every total stay Restricted, as the repository tool withheld them by the invoice's amounts. Ready is the default verdict, as before; `verdict` is now a list and can name Excluded or Funded too |
| Factoring | `factoring_get_readiness` | none | reads `DeskFactoringReadiness`, which evaluates the invoice now through `factoringReadiness(invoiceId:)`, as the repository tool did; `InvoiceFactoring` reads the stored state instead and was not used |
| Factoring | `factoring_list_agreements` | `advanceRateBp`, `reserveRateBp`, `minimumFeeMinor`, `feeTiers.rateBp`, `feeTiers.flatMinor` | renamed from `factoring_get_agreement` (migration `20261231010425`): it lists a factor's agreements, read whole by `factoringAgreements(factoringCompanyId:)` through `DeskFactoringAgreements`. `factoring_agreement` declared no field sensitivities, so under its Restricted default every field but `id` would be withheld at Internal; the terms that are not money (status, dates, recourse, treatment, funding days, chargeback window, submission time, accounts) are now Internal and the rates and fees stay Restricted, where the repository tool returned them ungated. The description says they can be withheld |
| Factoring | `factoring_list_noas` | none | reads `DeskFactoringNoas`. `factoring_noa` declared no field sensitivities; every field the notice list showed is now Internal and the notes stay Restricted (not selected) |
| Factoring | `factoring_list_submissions` | none | reads `DeskFactoringSubmissions`. `factoring_submission` declared no field sensitivities; where a schedule stands is Internal and its face and expected funding stay Restricted, as the repository tool withheld them by the invoice's amounts |
| Factoring | `factoring_get_submission` | none | reads `DeskFactoringSubmission`, bound by the rule once a get may read a record that carries `items`. Each item's amounts and funding follow `factoring_submission`'s Restricted money |
| Factoring | `factoring_list_remittances` | none | reads `DeskFactoringRemittances`. `factoring_remittance` declared no field sensitivities; where a remittance stands is Internal and its total cash stays Restricted |
| Factoring | `factoring_get_remittance` | `lines.cells` | reads `DeskFactoringRemittance` with the lines it asks for: `FactoringRemittance` gains `lines(matchStatuses:)` (one safe addition, its resolver reading the remittance's lines once). A line's amounts stay Restricted, and so do the cells the factor printed, which carry the same amounts; the repository tool returned an unreadable line's cells ungated |
| Factoring | `factoring_get_reserve_balance` | none | reads `FactoringReserveBalance`. `FactoringReserveBalance` had no kind, so the guard did not gate it; it now declares `@kind(resource: "factoring_reserve")`, whose balances stay Restricted and whose entry count and last movement are Internal |
| Carriers | `carrier_list_intelligence_events` | none | reads `DeskCarrierIntelEvents`, written for it and named in `reads.yml` because the tool is named for the carrier it watches: the repository row's fields. Status, severity and category are `CarrierIntelEventFilterInput`'s typed filters, whose fields now carry descriptions, so the record fields of the same names are left unoffered, as are the event's ids, rule, values and resolution note. `fromDate`/`toDate` became `detectedFrom`/`detectedTo`. Every field it selects is Internal under `carrier_intelligence` |
| Carriers | `carrier_list_capacity_postings` | none | `field-no-op`: reads `DeskCarrierCapacityPostings`, written for it. Every `carrier_capacity_posting` field is Internal; the carrier, origin location, states and equipment type are linked records and show their reference views. `openOnly` replaces telling the model to filter `availableTo` from today. The repository list spec stays in `listCatalogSpecs` for `FilterCatalog()` |
| Carriers | `carrier_list_invoice_matches` | none | reads `DeskCarrierInvoiceMatches`, an `items` list paged by its root's limit and offset, checked by `carrier_invoice_match` read as its root is; the repository tool checked `carrier_settlement` read, which still reaches it as the parent. `carrier_invoice_match` declared only the EDI invoice's fields, so under its Restricted default every match field but its reference view would be withheld at Internal; the match's links, how it was matched, its resolution and its carrier are now declared Internal, and the invoice, expected and variance totals stay Restricted, as the repository row gated them. The spec sets `amounts` and computes `source` (EDI, Document or Manual) in its `shape`; a match whose invoice a carrier sent over EDI marks the run with that EDI invoice (`marks`) |

At a person's role ceiling a factoring record's money now follows that record's own kind, as
every converted read does, where the repository tools judged it by the person's ceiling on
invoices. A role that reaches Restricted on a factoring kind and only Internal on invoices is
shown the factoring money it was granted.

The ledger's repository tools applied the ceiling to their totals, reasons and notes only, and
returned every other column at any tier; at a person's role ceiling below Internal each converted
ledger read now withholds every field but `id`. Their descriptions say the totals and amounts
show when the data access reaches them.

None of the repository-backed reference, party, equipment or operations lists was gated (only `order_list`
gated its amounts, all Internal), so at a person's role ceiling below Internal each now withholds every
field but `id`, where it used to return them all. No description names a field that is newly withheld
at Internal, so none changed. The fuel index price and fuel surcharge program reads were not gated
either and now withhold the same way below Internal; the rate agreement list and get already gated
their rates.

The EDI lists and gets were gated only where they named a field (a contact, the raw X12, a
tender's rate), so at a person's role ceiling below Internal each now withholds every field but
`id`, where the lists used to return them all. Nothing is newly withheld at Internal, so no EDI
description changed for it.

The billing and receivables module converted five of its 20 tools. The others stay on the
repository path, each with its reason in the inventory: the REST-only records (the billing
queue, invoice runs and open statements, bank receipts), the computed share candidates, and seven
reads whose root lacks what the tool takes (the adjustment lineage, the aging and the statement,
which are keyed by a customer or a date rather than an id, the open items' filters and paging,
the worklist's `asOf`, credit applications by invoice ids, and the transfer decision).

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
kind. `callTaint` reads them from the result as it reads `TaintedRecords` today. The spec names
the fields whose text was written outside the organization (`outside`: a driver's own description
of an expense or a pay dispute), and a record is marked only where one of them is shown: one the
ceiling withheld was never read, as the repository tools marked a description only when they
showed it. A list's outcome and a get's record each carry the marks (`graphQLListOutcome`,
`markedRecord`); an unpaged list's outcome carries them as a `markedRecord`, and a page read with
records alongside keeps them (`withBeside`). Where a row's outside text belongs to another record
than the row (a carrier invoice match's EDI invoice), the spec names that record per row in
`marks`, and its marks join the `outside` fields' on the same outcome.

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
- **Amounts.** The schema carries money in minor units, named `<amount>Minor` (`netPayMinor`:
  `230000`). A list or get spec that sets `amounts` shows each one as the decimal a person reads,
  under the name without the suffix (`netPay: "2300.00"`), beside the record's `currencyCode`, so
  a model never reads cents as dollars. `withheldByAccess`, `columns` and the near-miss note name
  it the same way, and where the record already carries the decimal under that name the
  minor-unit copy is dropped (`graphql_money.go`). It runs after the spec's `shape`, which still
  reads the minor units (the accounting drift lists, which do not set it, compare them to the
  tolerance). An unpaged list may set it too, unless one of its root's arguments holds minor units,
since its arguments are sent as the call gives them. An amount the schema holds in minor units under another name
  (`WorkerEarningsSummary.outstandingAdvances`) is named in the get spec's `minorUnits`. Such a
  spec's amount filters take the decimal too (Typed parameters). The settlement and pay reads set
  it; a module whose reads carry money sets it when it converts.
- **Columns.** `columns` are the node's top-level fields, in the operation's order.
- **Near-misses.** When page one is empty, `listnearmiss` probes again with the query or one
  typed parameter dropped, through the same operation; each probe is charged to the budget. Then
  it says what to loosen.
- **Gets.** A get returns the operation's record, shaped the same way, with
  `withheldByAccess`. A get for several ids (`invoice_get`) names a `batch` in its spec: its
  parameter is a list of at most that many ids, and it runs one read per id, at most 10
  concurrently (`graphQLGetsAtOnce`), under one span each. One id reads as a single get does;
  several come back as `items` in the order named, with `withheldByAccess` named once for all of
  them and the ids that are no record of the organization (an empty root or a `NOT_FOUND`) named
  in `note`. A refused or failed read fails the call. A converted get keeps its `getSpec` (name,
  summary, id parameter, policy) with no `fetch`; `graphQLGetSpecs` lists every one, registers it
  as a `graphQLGetTool` and is what the tests range over. The tool sends the id as `$id`, or as
  the one id the operation requires (`$workerId` for `worker_get_hos`), and a card whose root
  returns nothing is a missing record, not an empty one, unless the spec's `absent` says what
  nothing means (no ELD has reported the driver's hours, which is the finding). A spec's `shape`
  applies to the record what the card's resolver does not, as a list's does to each row:
  `worker_get` folds a NonCompliant credential into `canBeAssigned`, `worker_get_hos` gives the
  clocks in minutes and marks a reading over twelve hours old as stale,
  `customer_get_update_preferences` adds whether arrivals and departures are wanted, the warning
  for a customer who takes no routine notices, and the notice profile's recipients as the
  fallback, and `invoice_get` counts an invoice's lines (`lineCount`) and keeps 100 of them. A
  batch spec's `shapeEach` applies what a record read beside others needs beyond that
  (`invoice_get` keeps 25 lines of each of several, as the repository tool did). A get's node
  needs no Go entity: it filters by nothing but its id, so a card type (`JournalEntryCard`) binds
  as a record does. A card the get reads declares `@kind` of the record it draws, so the parts it
  carries that have no kind of their own (a journal's lines) are gated by that record's
  sensitivities rather than by none.

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

A converted get is held to the same on Postgres by
`TestConvertedGets_ReadOneRecordOfTheTenantUnderRowLevelSecurity`: it reads each of the tenant's
records with nothing withheld at Internal, refuses the other tenant's, and at Public withholds
what its row names. `TestConvertedGets_RefuseExactlyWhatAuthorizeRefuses` covers refusal.

Every list converted by a spec is listed once, in `graphQLListSpecs`, which registers them and
which the tests range over: `TestConvertedLists_ReadWhatTheRepositoryToolsReadUnderRowLevelSecurity`
holds each to the above, and `TestConvertedLists_RefuseExactlyWhatAuthorizeRefuses` covers refusal
for each, as a person and as an unattended agent. A module's PR adds its specs and its cases.

A repository-backed list that another tool composes (`shipment_find_records` matches a new
shipment's customer, stops and commodities through `newListCustomersTool`, `newListLocationsTool`
and `newListCommoditiesTool`) keeps that use until the composing tool converts. The Operations
module left `shipment_get`, `shipment_list`, `dispatch_get_board` and
`exception_list_watchtower_items` on the repository path; the inventory says what each needs.

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
