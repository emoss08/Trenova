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
  fixtures in `internal/testutil` (`stopeta_integration_test.go` is the model).
- A repository method, a migration, an RLS policy or a lock gets an integration test: the database
  is what is being tested. Include a second tenant and assert it sees nothing.
- The imaging worker runs from its image through `internal/testutil/imagingtest`; unit tests use
  `imaging/imagingfake`.

## Running them

The Taskfile and CI set `GOFLAGS=-tags=nomsgpack`; a `-tags` on the command line replaces it, so
pass both: `go test -tags=integration,nomsgpack -run TestName ./path/...`. Run only the packages you
changed. `task test` also runs the repository's lint tests (`authzlint`, `rlslint`,
`permissionlint`, `routelint`, `tenantbindlint`, `schemalint`), so name the ones your change
touches in the handoff.
