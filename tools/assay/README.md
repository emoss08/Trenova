# assay

Test intelligence for Go. Runs the tests a change can actually affect, instead of
all of them.

## Status

**M0 — package-graph test impact analysis.** `assay` builds the workspace package
graph, attributes changed files to packages, and walks reverse dependency edges to
find every test package a diff can reach.

Later milestones (per-test coverage index, line-level selection, mutation testing)
build on this foundation.

On the Trenova workspace (687 packages, 349 with tests), M0 gives a median 32%
reduction in test packages run — 98.6% on commits touching five files or fewer,
21.8% on commits touching more than twenty. See [BENCHMARKS.md](BENCHMARKS.md)
for the method and for why the large-commit case is the argument for the
line-level index.

## Install

```bash
go install github.com/emoss08/assay/cmd/assay@latest
```

## Use

```bash
# What would run?
assay select --since origin/main -v

# Run it
assay run --since origin/main -- -count=1 -race

# Integration suites gated behind a build tag
assay run --tags integration --since origin/main

# Skip git entirely when CI already knows the changed paths
git diff --name-only origin/main | assay select --files - --json
```

With no `--since`, `assay` diffs the working tree against `HEAD` — useful locally
while you edit. In CI, pass the base branch.

## How selection works

1. **Collect changes.** `git diff --name-status --merge-base <ref>`, plus untracked
   files. Renames contribute both their old and new paths.
2. **Attribute files to packages** by directory containment, longest prefix wins.
   This deliberately covers more than `go list` reports — `testdata/`, `.sql`
   fixtures, embedded assets and golden files all belong to the package whose
   directory holds them.
3. **Walk reverse production edges** transitively from every changed package.
4. **Select test packages**: any package in that closure that has tests, plus any
   package whose *test* imports reach into the closure.

Test-import edges are deliberately terminal. If `svc`'s tests import `fixtures`,
a change to `fixtures` selects `svc` — but not `app`, which only reaches
`fixtures` through `svc`'s test code. Propagating through test edges would select
most of the repo for any fixture change.

## Failing safe

A test selector that skips a test it should have run is worse than no selector at
all, so `assay` falls back to running everything when it cannot prove a narrower
set is sound:

| Trigger | Why |
|---|---|
| `go.mod`, `go.sum`, `go.work`, `go.work.sum` changed | Dependency resolution can shift under any package |
| A changed `.go` file belongs to no known package | Build-tag-excluded or otherwise invisible to `go list` |

Both cases print a `select-all:` line naming the reason. Anything else that maps
to no package — docs, CI config, editor settings — is reported as ignored and does
not select tests.

Packages whose only test files are hidden behind a build tag still count as
testable, so `--tags integration` runs never silently skip them.

## Flags

| Flag | Meaning |
|---|---|
| `--since ref` | Diff against `ref`'s merge-base with `HEAD` |
| `--files path` | Read newline-separated changed paths from a file, or `-` for stdin |
| `--root dir` | Workspace root (default: git repository root) |
| `--tags list` | Comma-separated build tags |
| `--all` | Skip selection; use every testable package |
| `--json` | Machine-readable output (`select` only) |
| `-v`, `--verbose` | List every selected package and ignored file |
| `--no-color` | Disable colored output (also honours `NO_COLOR`) |

Arguments after `--` pass through to `go test`.

## Workspaces

`assay` reads `go list -m -json` to find every main module, then loads each
module's packages with `golang.org/x/tools/go/packages`. Multi-module `go.work`
setups work without configuration; reverse dependency edges cross module
boundaries.

Packages load with `Tests: true`, so `assay` sees the real test variants the Go
toolchain builds — the in-package variant `pkg [pkg.test]` and the external test
package `pkg_test [pkg.test]` — and derives each package's test-only imports by
subtracting its production imports. That is why a package consisting solely of
`_test.go` files is still reachable, and why external test packages attach to
their subject instead of appearing as phantom `_test` packages.

## Dependencies

| Module | Why |
|---|---|
| `golang.org/x/tools/go/packages` | Package loading. Handles workspaces, build tags, test variants and `GOPACKAGESDRIVER`, and its `Overlay` support is what the mutation milestones will build on |
| `github.com/spf13/cobra` | Subcommands, flag handling, generated help and completions |
| `github.com/fatih/color` | Terminal color with `NO_COLOR` and TTY detection |
| `github.com/stretchr/testify` | Test assertions |

`assay` deliberately shells out to `git` rather than embedding `go-git`: the
porcelain commands used here are stable, and `go-git`'s diff is markedly slower
on large repositories.
