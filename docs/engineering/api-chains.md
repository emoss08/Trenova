# API chains

An API chain is a flow through the public API, written as a YAML file, run against the whole TMS
in process and compared with an approved golden file. It catches what unit and repository tests
cannot see: a resolver that stops returning a field, a write that no longer moves a total, a
mutation that forgets its workflow or audit entry, a read that picks up an N+1, a route another
tenant can reach. Chains live in `services/tms/internal/apichains/testdata/chains/`; the harness
is `internal/testutil/apichain` (the chain model), `chainrun` (the runner) and `chainsnap` (the
golden file).

## Running

```bash
task test-chains                       # from services/tms; Docker for Postgres and Redis
CHAIN_UPDATE=1 task test-chains        # rewrite every golden, then review the diff like code
task test-chains -- -run 'TestChains/shipment-to-invoice'
```

They also run in CI's integration job, which picks up every `go:build integration` package.
`TRENOVA_TEST_POSTGRES_DSN` and `TRENOVA_TEST_REDIS_ADDR` point the run at servers you already
have, as for every integration test. The API's log for a chain is
`build/apichains/<chain>.log`; a failure names it.

## How a chain runs

Each chain gets a fresh copy of the fixtures database (`seededtest`) and its own application:
`bootstrap.Options()` and `bootstrap.APIOptions()`, the graph the API runs, with row-level
security enforced (`dbtest.RLSDatabaseConfig`), served through `httptest`. Nothing is mocked
except what leaves the process:

- **Workflows are recorded, never started.** `WorkflowStarter`, `WorkflowSignalStarter` and
  `WorkflowResults` are replaced with a recorder, so no Temporal server is needed and every
  workflow a step starts (type, queue, ID, arguments) lands in the golden. A caller that waits
  for a result gets `chainrun.ErrWorkflowRecorded`.
- **Object storage is in memory** (`objectstoretest`), and the Temporal payload store refuses.
- **Realtime events and audit entries are recorded** through decorators on
  `RealtimePublisher` and `AuditService`, then passed on.

The runner logs in through `POST /api/v1/auth/login` as the chain's actor (the fixtures password),
fetches the CSRF token and keeps one cookie jar, so GraphQL and REST steps go through the same
session, router and middleware production runs. A GraphQL step names an operation from
`internal/api/graphql/persisted-documents.json` and is sent by its hash, exactly as the web client
sends it. A REST step is checked against the router's registered routes before anything boots.

## Writing a chain

```yaml
name: shipment-to-invoice          # the file name, without .yaml
description: one sentence a reviewer reads first
actor: admin@trenova.app           # a fixture user
fixtures:                          # one value each; $1 is the actor's organization, $2 its business unit
  customer: SELECT id FROM customers WHERE code = 'RNGE' AND organization_id = $1 AND business_unit_id = $2
  driver: SELECT id FROM workers WHERE email = 'carlos.rivera@example.com' AND organization_id = $1 AND business_unit_id = $2
steps:
  - id: create
    graphql: CreateShipment        # a persisted operation, sent by hash
    vars:
      input:
        customerId: ${fixture.customer}
        bol: CHAIN-${run.unique}
        freightChargeAmount: "1500.00"
        moves: [{ stops: [{ scheduledWindowStart: "${run.now-21600}", ... }, ...] }]
    export: { shipment: data.createShipment.id, move: data.createShipment.moves.0.id }
  - id: shipment
    graphql: ShipmentDetail
    vars: { id: "${create.shipment}" }
  - id: assign
    rest: POST /api/v1/shipment-moves/${create.move}/assignment/
    body: { primaryWorkerId: "${fixture.driver}", tractorId: "${fixture.tractor}" }
    effects:
      - { read: shipment, path: data.shipment.moves.0.assignment.tractorId, equals: "${fixture.tractor}" }
  - id: post
    rest: POST /api/v1/billing-queue/${transfer.item}/post/
    export: { invoice: invoiceId }
    effects:
      - { read: shipment, path: data.shipment.status, equals: Invoiced }
  - id: invoice
    graphql: DeskInvoiceDetail
    vars: { id: "${post.invoice}" }
    expect:
      data.invoice.status: Posted
      data.invoice.totalAmount: ${create.request.variables.input.freightChargeAmount}
generated:
  proNumber: ${create.response.data.createShipment.proNumber}
probes: [unauthenticated, no-permission, other-tenant]
```

`shipment-to-invoice.yaml` is the whole of it: create, assign, four stop actuals, the bulk
transfer to billing, approval, posting over REST and the invoice read back.

- **References.** `${step.export}`, `${step.response.<path>}`, `${step.request.<path>}` (the
  request a step sent: `variables.…` or `path`/`body.…`), `${fixture.name}`, and `${run.unique}`,
  `${run.now}` (Unix seconds; `${run.now-21600}` offsets it), `${run.organization}`,
  `${run.businessUnit}`, `${run.user}`. A reference that is the
  whole value keeps its type; one inside text is interpolated. A step may refer only to steps
  before it. Paths are dot-separated, with list indexes as numbers.
- **`expect`** pins the facts that must hold whatever the golden says, so a careless
  `CHAIN_UPDATE=1` cannot approve a wrong status. Numbers compare by value (`"1250.00"` equals
  `1250`).
- **`effects`** say what a write does to a read. The runner runs the named read (a GraphQL query or
  a GET in the same chain) before the write when the rule needs it and again after: `equals`,
  `increase`, `decrease` (decimals) or `unchanged`. The before and after values land in the golden.
- **`expectError: <code>`** makes a step pass only when it is refused with that code: the
  GraphQL `extensions.code`, or the last segment of a REST problem `type`.
- **`document:`** sends an inline document for an operation no client sends (an API-only caller).
  The runner refuses it for an operation that is in the manifest, and refuses a `graphql:` name
  that is in neither.
- **`generated:`** names values the server makes up on purpose (a PRO number with random digits),
  each one reference: once it resolves, the golden shows `<name>` wherever the value appears.
- **`volatile:`** lists paths (`*` matches one segment) whose values the golden blanks, for a field
  that is honestly different every run. Prefer fixing the nondeterminism.

## Probes

After the chain, each probe replays the requests the steps sent:

| Probe | Who | Passes when |
|---|---|---|
| `unauthenticated` | no session | every step is refused |
| `no-permission` | a user the harness adds to the actor's organization with no role | every step is refused |
| `other-tenant` | the actor, switched to their other organization | every step whose request names one of the chain's records is refused, or answers with none of the chain's records the request did not already name; and the actor's reads come back identical after the replay |

The chain's records are its fixtures, its exports and every `id` a write answered with. A step
that is open on purpose (the caller's own profile) lists the probe under `openTo:`. Each probe's
status and code per step land in the golden, so a refusal that changes from 403 to 404 is a diff.

## The golden file

`<chain>.golden.json` beside the chain holds, per step, the call, the request and response,
status and code, effects, side effects (workflows, signals, realtime, audit) and `roundTrips`, the
database round trips the request made (an outer `keel.Budget` around the handler). Values that
differ every run are rewritten before comparing:

- every ID becomes its prefix and the order it first appeared (`$shp_1`), or its name when the
  chain knows one (`$fixture.customer`, `$run.user`), so `$shp_1` in step six is the shipment step
  one created;
- a time under a time-named key (`createdAt`, `due_date`, `deadline`), and any whole number within
  about a year of the run's clock, becomes `<time>`; a date under a date-named key becomes `<date>`;
- the run's unique token becomes `<run.unique>`, and each `generated` value `<name>`.

A failure names the step, the path, the old and new values, and the suspect writes: the earlier
writes whose request or response shares a record with the drifted value.

```
shipment-to-invoice drifted from shipment-to-invoice.golden.json:
  step post (POST /api/v1/billing-queue/:itemID/post/): sideEffects.audit.1 want={"operation":"update","resource":"billing_queue","resourceId":"$bqi_1"} got=<absent>
    suspect writes: post (POST /api/v1/billing-queue/:itemID/post/)
```

The golden is approved by reviewing its diff in the pull request. Rewrite it with
`CHAIN_UPDATE=1` only when the change is meant, and read every line of the diff: a new
`roundTrips` number is a query you added to the request path.

## The rules that bite

- Chains run one at a time: each flushes the Redis database it leased.
- A chain creates what it reads (a fresh shipment, not a fixture's), so it holds on any seed.
- A fixture query names a stable code or email, never a name; fixtures should be the actor's
  tenant's records, since the other-tenant probe treats them as private.
- Anything that leaves the process is recorded, not run. A flow whose next step needs a
  workflow's result cannot be a chain until that state is reachable through the API.
