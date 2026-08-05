# Measurements

Measured on the Trenova workspace: 6 modules, 690 packages, 353 with tests,
3,169 Go files (15 MB), 5,786 test functions. Intel Xeon @ 2.10 GHz, 4 cores.

## Package-level selection

Method: for each of the 30 commits before `assay` landed that touch `.go` files,
feed `git diff --name-only <sha>~1 <sha>` into `assay select --files - --json` and
record how many test packages the selector returns.

Caveat: the package graph comes from the current checkout, not from each
historical commit, so packages that have since moved or been renamed are
attributed to their present location. The numbers are directional, not exact.

| Metric | Value |
|---|---|
| Commits measured | 30 |
| Median reduction | **32.2%** |
| Mean reduction | 41.8% |
| Best / worst | 99.7% / 0.0% |
| Select-all fallback fired | 2 commits |

| Commit size | Median reduction | n |
|---|---|---|
| ≤ 5 files changed | **98.6%** | 7 |
| > 20 files changed | 21.8% | 13 |

Package-graph selection is excellent on small commits and mediocre on large ones.
That ceiling is structural: at package granularity, *any* change to a package that
200 others import must select all 200 — even when it touches one function that
three of those tests execute.

## Line-level narrowing

The same edit, measured both ways. A one-line change inside `SplitProfileName` in
`internal/cover/profile.go`, a package five others depend on:

| Layer | Selected | Skipped |
|---|---|---|
| Package-level | 5 packages, in full | 98.6% |
| Line-level | **3 packages, 20 tests** — 2 packages dropped entirely | **99.2%** |

`internal/cover` itself narrowed to exactly **1 test**. `internal/cli` and
`internal/report` were dropped: they depend on the changed package, but no test in
them executes the changed line.

### The fallback is not theoretical

Adding one field to a struct **eleven lines away** in the same file:

| Layer | Selected |
|---|---|
| Line-level | 5 packages, **all in full** |

Reported as `no line attribution: 1 files` with a per-package reason. A struct
field appears in no coverage block, so there is nothing to narrow against, and
guessing would mean skipping tests that could catch the regression.

### It catches real regressions

Injecting an actual bug into the same function — `path.Clean(dir) + "BUG"` — and
running the narrowed selection:

```
--- FAIL: TestSplitProfileName/example.com/m/a.go
    expected: "example.com/m"
    actual  : "example.com/mBUG"
FAIL	github.com/emoss08/assay/internal/cover
exit status 1
```

The single test narrowing selected is the one that caught it.

### `assay verify` on the same change

```
verifying 73 excluded tests across 5 packages
ok  github.com/emoss08/assay/internal/selection
ok  github.com/emoss08/assay/internal/cover
ok  github.com/emoss08/assay/internal/cli
ok  github.com/emoss08/assay/internal/index
ok  github.com/emoss08/assay/internal/report
sound: every excluded test passes
```

## Index cost

| Scope | Packages | Tests | Wall clock |
|---|---|---|---|
| `internal/...` of assay | 9 | 166 | 24s |
| `shared` module | 32 | 279 | 36s |

Roughly 8 tests/second on 4 cores — one process per test plus a coverage profile
parse. Incremental rebuilds re-index only packages whose dependency closure moved,
so the steady-state cost after a one-file edit is a handful of packages.

**The full 353-package index was not measured here, and the reason is worth
recording.** The first attempt drove load average to 30 on a 4-core box and was
killed. `--jobs N` was multiplying with Go's own build parallelism: each
`go test -c` instruments every package named by `-coverpkg` — the whole workspace
— and fans out its own compile actions, so four workers became roughly four times
`GOMAXPROCS` compile processes. The collector now divides the cores between
workers (`-p` on the inner build) and defaults `--jobs` to half of `GOMAXPROCS`.

Extrapolating the measured 8 tests/second to 5,786 tests puts a cold full index
around 12 minutes on this hardware, but that is arithmetic, not a measurement, and
the compile cost does not scale linearly with test count. Treat it as an order of
magnitude only. Indexing a large workspace wants a CI machine and a restored
cache, not a laptop.

## Where the time goes

Profiling settled a question worth settling before optimizing anything: native
code would have been aimed at the wrong 0.1%.

| Phase | Time | Share |
|---|---|---|
| `go list` via `packages.Load` (subprocess) | ~1,450 ms | **>99%** |
| Ingest + index 690 packages | 1.3 ms | 0.09% |
| Reverse-closure BFS | 0.04 ms | 0.003% |
| File→package attribution (500 files) | 0.51 ms | 0.03% |

All in-process graph work totals ~1.9 ms of a ~1,500 ms run. Rewriting it in C or
assembly — even making it infinitely fast — would save under 2 ms.

## Graph cache

So the graph is cached instead, keyed by a content fingerprint of every `.go`
file plus `go.mod`/`go.sum`/`go.work`/`go.work.sum`, the build tags, the Go
version, the toolchain environment, and assay's own build identity.

| Run | Wall clock |
|---|---|
| `--no-cache` | 1,447–1,530 ms |
| Cache hit | **48–51 ms** |

**~30× faster.** Fingerprinting is content-addressed, not mtime-based, so a CI
cache restore or a `tar -p` extraction that preserves timestamps cannot serve a
stale graph.

## Where SIMD actually earned its place

The fingerprint hashes 15 MB on every run, so it is the one genuinely hot loop.
`zeebo/blake3` ships AVX2/SSE4.1 assembly:

| Hash | Throughput | 15 MB corpus |
|---|---|---|
| **blake3** (AVX2 asm) | **2,817 MB/s** | 5.6 ms |
| sha256 (SHA-NI/AVX2 asm) | 1,370 MB/s | 11.5 ms |
| fnv128 (pure Go) | 551 MB/s | 28.5 ms |

A second fix mattered as much as the hash choice. `io.CopyBuffer(hasher, file,
buf)` silently ignored the supplied buffer: `*os.File` implements `io.WriterTo`,
and `copyBuffer` prefers that path, allocating internally per file. Replacing it
with an explicit read loop:

| | Time | Allocated |
|---|---|---|
| `io.CopyBuffer` | 72.4 ms | 106.9 MB |
| Explicit read loop | **46.4 ms** | **2.7 MB** |

1.56× faster, 39× less garbage — from deleting one stdlib call.

## Scaling

Synthetic graphs, fan-in 5, half the packages carrying tests:

| Operation | 700 packages | 10,000 packages |
|---|---|---|
| Ingest + index | 1.3 ms | 28.0 ms |
| Affected-test closure | 0.04 ms | 0.72 ms |
| Attribution (500 files) | 0.51 ms | 0.57 ms |

Attribution is flat in package count because directory matching short-circuits on
the longest prefix.

## Reproducing

```bash
go test -run '^$' -bench . ./internal/graph/ ./internal/cache/

go build -o /tmp/assay ./cmd/assay
/tmp/assay index --quiet
/tmp/assay select -v          # after an edit
/tmp/assay verify
```
