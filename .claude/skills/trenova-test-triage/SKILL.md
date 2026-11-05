---
name: trenova-test-triage
description: How to work through failing Go tests in Trenova, separating stale tests from real bugs. Use when a test, a package or a CI job fails, when many tests fail at once, after a refactor or a new dependency breaks test rigs, or when asked to get a suite green. Covers grouping failures by cause, classifying each one, what may and may not be changed, generated fixtures and running heavy suites safely.
---

# Triaging failing tests in Trenova

Getting a suite green and finding out what it is telling you are different jobs. Do the second one.
[testing.md](../../../docs/engineering/backend/testing.md) has the patterns and the commands.

## 1. Group before you fix

Many failures usually have few causes. Before opening any test, count the messages:

```bash
go test ./... > "$TMP/test.txt" 2>&1
grep -E '^FAIL\s|^--- FAIL|^panic:' "$TMP/test.txt"
grep -A1 'Received unexpected error' "$TMP/test.txt" | sort | uniq -c | sort -rn | head
```

A panic ends its package, so a package that fails on one test may hide more behind it. Twenty-two
integration failures were once one seed error that stopped the test database being built.

## 2. Classify each one

| What you see | What it usually is | What to do |
|---|---|---|
| nil pointer in a production frame, called from a test's hand-built `Params` or struct | a dependency the rig doesn't supply | supply it in the test |
| `mock: I don't know what to return` or `called over 1 times` | the code now makes another call | read why, then widen the mock if that call is right |
| a test of behaviour when a dependency is missing | a branch `fx` no longer allows | delete the test, or the part about the missing dependency |
| a snapshot, golden or catalog is stale | a generated file behind the source | regenerate it, and read the diff ([generated-artifacts.md](../../../docs/engineering/generated-artifacts.md)) |
| a task, permission or status the test assumes has moved | the product changed | check the change was intended, then update the test from the contract |
| an assertion about a value, a permission, a tenant or money | **possibly a real bug** | read the production path before touching the test |
| only sometimes, around a second boundary | a clock read per item, or a race | fix the code: read the clock once |

Treat a test named for a security rule as possibly a real bug, whatever the error looks like. Read
it, then say what it actually proved. A crash in setup proves nothing either way, and neither a pass
nor a failure counts until it reaches its assertion.

Regenerating is not triage. A `gqlext` failure read like stale output and was the field guard
checking a protected field, so read a guard's message before refreshing anything.

## 3. What you may change

- Supply dependencies in tests. Never add a nil guard or an optional dependency to production.
- Never weaken an assertion, skip a test or rewrite a test to match a bug. If production looks wrong,
  stop and report it.
- Delete a test only when the behaviour it covers no longer exists, and say which ones you deleted.
- A failing test can be the rule that rejects your change. Find out before you update it.
- Fix a real bug at its cause, and keep the test that found it.

## 4. Run it safely

- One heavy command at a time, one package at a time while repairing. Don't start a second run
  while one is going, and don't edit code under a run that's still compiling.
- Run from `services/tms`, where relative package paths resolve. The operator is outside `go.work`:
  `GOWORK=off go test ./...` from `services/operator`.
- `-tags` on the command line replaces `GOFLAGS`, so pass them all:
  `-tags=integration,nomsgpack,devlog`.
- Integration tests need `TRENOVA_TEST_POSTGRES_DSN` and `APP_ENV=test`. `task test-integration`
  sets both and uses its own Postgres on port 55432, never the development database.
- Compile first: `go vet -tags=integration,nomsgpack,devlog ./...` finds every test that no longer
  builds, without waiting for a run.
- Stop a run by its PID, after listing it. `pkill -f` matches your own shell's command line, and
  killing every `go test` on the machine takes other sessions' work with it.

## 5. Report

Say how many passed, what you repaired, what you deleted and why, and every failure that may be a
real bug. "Green" isn't the report: a test you never ran isn't passing, so name the suites you didn't
run.
