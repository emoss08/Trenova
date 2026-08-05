# assay

Test intelligence for Go. Runs the tests a change can actually affect, instead of
all of them.

## Status

Two layers, both live:

**Package-graph selection.** Builds the workspace package graph, attributes
changed files to packages, and walks reverse dependency edges to find every test
package a diff can reach. Always available, no setup.

**Line-level narrowing.** With a coverage index built by `assay index`, each
affected package is narrowed to the individual tests that execute the changed
lines — and dropped entirely if none do. Only engages when it can be proven safe;
otherwise the package runs in full.

Mutation testing is the next milestone and reuses the same index.

A one-line edit inside a function in this repo selects **3 packages / 20 tests**
where package-level selection selects 5 whole packages, dropping two packages
outright. A struct-field edit one line away correctly falls back to all 5 in full.
See [BENCHMARKS.md](BENCHMARKS.md).

## Install

```bash
go install github.com/emoss08/assay/cmd/assay@latest
```

## Use

```bash
# What would run? (BASE is your default branch: origin/main, origin/master, ...)
assay select --since "$BASE" -v

# Run it
assay run --since "$BASE" -- -count=1 -race

# Integration suites gated behind a build tag
assay run --tags integration --since "$BASE"

# Skip git entirely when CI already knows the changed paths
git diff --name-only "$BASE" | assay select --files - --json
```

With no `--since`, `assay` diffs the working tree against `HEAD` — useful locally
while you edit. In CI, pass the base branch. `--since`, `--files` and `--all` are
mutually exclusive; passing more than one is an error rather than a silent
precedence rule.

### Shallow clones

`--since <ref>` needs `<ref>` to exist locally and to share history with `HEAD`.
CI checkouts are frequently shallow (`actions/checkout` defaults to `depth=1`),
which breaks both:

- **Ref missing** → `assay` fails with an error naming the shallow clone and the
  remedies. It never degrades to an empty change set, because selecting nothing
  is the one wrong answer.
- **Ref present but no shared history** → `assay` falls back to a direct
  comparison, prints a `warning:` explaining that this over-selects, and carries
  on. Over-selecting is safe; skipping is not.

Fetch enough history (`git fetch --deepen=50 origin <branch>`, or
`fetch-depth: 0`) to get exact selection, or pass `--files` with a change list
your CI already has.

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
| `--no-cache` | Reload the package graph instead of reusing a cached one |
| `--cache-dir` | Graph cache location (default: user cache dir, or `ASSAY_CACHE`) |

Arguments after `--` pass through to `go test`.

## The graph cache

Loading the package graph means running `go list` across every module, which is
over 99% of a cold run — roughly 1.5s on this workspace against ~2ms of actual
graph work. So the graph is cached and the subprocess skipped entirely: a warm
run is **~49ms, about 30× faster**.

The cache key is a BLAKE3 fingerprint over the *content* of every `.go` file and
every `go.mod`/`go.sum`/`go.work`/`go.work.sum`, plus the build tags, the Go
version, the workspace root, and the toolchain environment (`GOOS`, `GOARCH`,
`GOFLAGS`, `GOEXPERIMENT`, `CGO_ENABLED`, `GOWORK`).

Content, deliberately — not size and mtime. A CI cache restore or a `tar -p`
extraction can preserve timestamps, and an mtime-keyed cache would then serve a
stale graph and silently skip tests. Hashing 15MB costs ~46ms; a wrong answer
costs a production incident.

Entries are content-addressed, so reverting an edit reuses the original entry
rather than reloading. The cache lives outside the repository (`os.UserCacheDir`
by default), holds the 32 most recent graphs, and commits entries by rename so
concurrent runs cannot observe a partial write. Every cache failure — unreadable
entry, corrupt payload, unwritable directory — degrades to a normal load rather
than an error.

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
| `github.com/zeebo/blake3` | Cache fingerprinting. Its AVX2/SSE4.1 assembly hashes at 2.8 GB/s — 2.06× SHA-256 and 5.1× a pure-Go hash |
| `github.com/stretchr/testify` | Test assertions |

`assay` deliberately shells out to `git` rather than embedding `go-git`: the
porcelain commands used here are stable, and `go-git`'s diff is markedly slower
on large repositories.
