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
  (`tractorrepository/tractor_test.go`). A module's integration fixtures build on `seedtest` in
  its `<module>test` package: `equipmenttest.NewDatabase(t)` is two tenants, an RLS-enforcing
  connection and scoped contexts.
- The imaging worker runs from its image through `internal/testutil/imagingtest`; unit tests use
  `imaging/imagingfake`.

## Running them

The Taskfile and CI set `GOFLAGS=-tags=nomsgpack`; a `-tags` on the command line replaces it, so
pass both: `go test -tags=integration,nomsgpack -run TestName ./path/...`. Run only the packages you
changed. `task test` also runs the repository's lint tests (`authzlint`, `rlslint`,
`permissionlint`, `routelint`, `tenantbindlint`, `schemalint`), so name the ones your change
touches in the handoff.
