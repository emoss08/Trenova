# Testing

Every behaviour change ships with a test of what a caller relies on, named for that behaviour and
asserting the exact result. Write fixtures from the contract (the schema, the migration, the
spec), not from what the code returns today; ask what the contract allows that no fixture builds
yet (absent vs null vs empty, many where every fixture has one, a second tenant). Paths are under
`services/tms/`.

## Unit tests

- testify (`require` for preconditions, `assert` for checks), `t.Parallel()`, `t.Context()`.
- Mocks come from mockery (`.mockery.yaml`) into `internal/testutil/mocks/<pkg>mock`, built with
  `NewMockX(t)`. When a port changes, update the affected mocks only; never run mockery over the
  whole codebase.
- Table-driven when the cases share a shape; separate named tests when they don't.
- Non-test code never imports `testutil`, `testify/mock` or a `*test` helper package (lint).
- Production files hold no test constructors (`NewTest*`, `*ForTest`, a `testing.go`). A helper one
  package's tests need lives in that directory's `_test.go` files (`export_test.go` when the
  external test package uses it); one several packages need lives in a `*test` kit and builds
  through the real constructor (`dbtest.CapabilityProbe`), never a cut-down twin.
- Domain-specific helpers live in the module's `<module>test/` package or `internal/testutil/*test`
  (`permtest`, `validationtest`, `imagingtest`, …); look there before writing a fixture.

## Integration tests

- `//go:build integration`, an external `_test` package, `seedtest.SetupTestDB(t)`
  (testcontainers, or `TRENOVA_TEST_POSTGRES_DSN`), then `seedtest.SeedFullTestData` and the
  fixtures in `internal/testutil` (`stopeta_integration_test.go` is the model). A test that needs
  the development fixtures (the admin account, the Trenova Logistics organization, its workers,
  shipments and ledgers) uses `seededtest.SetupTestDB(t)` instead of running seeds itself; it
  is its own package because the fixtures import application code that a package under test
  may sit below.
  Each test gets a fresh database copied from a keel template (`migrate.Template`): the
  migrations plus the reference seeds for `SetupTestDB`, and the fixtures on top for
  `seededtest`. A template is keyed by the seeds' fingerprints, so editing a seed or its
  data builds a new one. The first test on a server builds each template and every later one
  reuses it until a migration or a seed changes.
- A migration that rewrites data gets a test that runs its file against rows it must change and
  rows it must leave: `migrations.File("<version>_<name>")` returns the SQL
  (`formula_template_backfill_integration_test.go`).
- A repository method, a migration, an RLS policy or a lock gets an integration test: the database
  is what is being tested. Include a second tenant and assert it sees nothing.
- keel SQL is asserted by rendering, not sqlmock: `keeltest.Render(t, q)`
  (`internal/testutil/keeltest`) renders a query under a test tenant without a database
  (`tractorrepository/tractor_test.go`); `keeltest.Snapshot(t, q)` pins a whole statement in a
  golden file (`KEEL_UPDATE=1` rewrites them). A module's integration fixtures build on `seedtest` in
  its `<module>test` package: `equipmenttest.NewDatabase(t)` is two tenants, an RLS-enforcing
  connection and scoped contexts.
- The imaging worker runs from its image through `internal/testutil/imagingtest`; unit tests use
  `imaging/imagingfake`.

## API chains

A flow that crosses resolvers, routes and modules (create, assign, deliver, bill, post) gets an
API chain: a YAML file in `internal/apichains/testdata/chains/` that runs against the whole API in
process, GraphQL and REST in one session, with probes for an unauthenticated caller, a caller with
no permissions and another tenant, compared with a reviewed golden file. A new client operation or
route on such a flow extends its chain. See [api-chains.md](../api-chains.md).

## Fuzz tests

- `//go:build fuzz`, in the package under test (`<name>_fuzz_test.go`), one `FuzzX` per
  property. Fuzz code that takes untrusted input (a file name, a scanned code, a CSV cell) or
  does arithmetic that must always balance (minor units, payments against an invoice, a
  packet's pages and turns). Assert a property every input must keep (a header that parses,
  shares that add up, pages numbered without gaps), not one output; seed the corpus with the
  cases the unit tests already name and the hostile ones they don't.
- The tag keeps them out of `task test`. `task test-fuzz` runs every target's seeds as ordinary
  tests (CI does on every change); `task fuzz` fuzzes each target for 30 seconds, one at a time,
  and `task fuzz name=FuzzX time=5m` one target for longer (CI fuzzes every target after each
  merge to master and weekly). Both cover `shared` too (`scripts/gofuzz.sh`). A failing input
  lands in the package's `testdata/fuzz/FuzzX/`; commit it with the fix, so it stays a seed.
- Parsers in Rust (the imaging worker's decoders and PDF engine, X12, Tariff, charge splits)
  have `cargo-fuzz` targets instead ([rust.md](../rust.md)).

## Shapes and budgets

- **Plan guard.** Every database an integration test opens through `seedtest`, `dbtest` or a test
  `Connection` records its distinct statement texts to the file `KEEL_SHAPES` names
  (`internal/testutil/shapetest`; one shared, append-only file per process, closed at exit, a
  write error failing the test at cleanup). A database opened another way passes
  `shapetest.Observers(t)` as its observers. `task db-plans` runs the suite with it and explains
  every shape ([data-access.md](data-access.md#plan-guard)), so a new query path is checked as soon
  as a test runs it.
- **Budgets.** Pin a path that must not grow with its input with a hard budget, which fails the
  terminal that crosses it with `keel.ErrBudgetExceeded`:

  ```go
  ctx = keel.Budget(ctx, keel.MaxRepeats(1), keel.MaxRoundTrips(3))
  _, err := svc.Get(ctx, req)
  require.NoError(t, err)
  report, _ := keel.BudgetReportFrom(ctx)
  assert.Empty(t, report.Repeats)
  ```

  `keel.MaxRoundTrips(0)` asserts that a cached path touches no database
  (`postgres/runtime_integration_test.go`).

## Running them

The Taskfile and CI set `GOFLAGS=-tags=nomsgpack,devlog`; a `-tags` on the command line replaces
it, so pass them all: `go test -tags=integration,nomsgpack,devlog -run TestName ./path/...`.
`devlog` builds in development-only code (the `pretty` log format, `pkg/prettylog`); release
builds (`deploy/Dockerfile.tms`) leave it out, so nothing behind it ships. Code that must never
reach production goes behind `//go:build devlog`, with a `!devlog` stub that refuses it. The
integration steps in CI pass `-tags=integration,nomsgpack`, so they run the `!devlog` side.

Run only the packages you changed; add `KEEL_SHAPES=$PWD/build/keel-shapes.jsonl` to record shapes,
or `task db-plans` for the whole suite and its plans. `task test` also runs the repository's lint tests (`authzlint`, `rlslint`,
`permissionlint`, `routelint`, `tenantbindlint`, `schemalint`), so name the ones your change
touches in the handoff.
