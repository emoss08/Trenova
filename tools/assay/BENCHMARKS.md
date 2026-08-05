# Measurements

Measured on the Trenova workspace: 6 modules, 688 packages, 351 with tests,
3,169 Go files (15 MB), 5,786 test functions. Intel Xeon @ 2.10 GHz, 4 cores.

## Selection quality

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
| Median selected | 236 of 349 packages |
| Select-all fallback fired | 2 commits |

| Commit size | Median reduction | n |
|---|---|---|
| ≤ 5 files changed | **98.6%** | 7 |
| > 20 files changed | 21.8% | 13 |

Package-graph TIA is excellent on small, focused commits and mediocre on large
ones. A single-file service change selects 7 of 349 packages. A commit touching a
widely-imported domain package selects 273.

That ceiling is structural. At package granularity, *any* change to a package
that 200 other packages import must select all 200 — even when the change touches
one function that three of those tests execute. This is the empirical case for
the line→test index in M1/M2.

## Where the time goes

Profiling settled a question worth settling before optimizing anything: native
code would have been aimed at the wrong 0.1%.

| Phase | Time | Share |
|---|---|---|
| `go list` via `packages.Load` (subprocess) | ~1,450 ms | **>99%** |
| Ingest + index 688 packages | 1.3 ms | 0.09% |
| Reverse-closure BFS | 0.04 ms | 0.003% |
| File→package attribution (500 files) | 0.51 ms | 0.03% |

All in-process graph work totals ~1.9 ms of a ~1,500 ms run. Rewriting it in C or
assembly — even making it infinitely fast — would save under 2 ms. The subprocess
was the entire problem.

## Graph cache

So the graph is cached instead, keyed by a content fingerprint of every `.go`
file plus `go.mod`/`go.sum`/`go.work`/`go.work.sum`, the build tags, the Go
version, and the toolchain environment (`GOOS`, `GOARCH`, `GOFLAGS`,
`GOEXPERIMENT`, `CGO_ENABLED`, `GOWORK`).

| Run | Wall clock |
|---|---|
| `--no-cache` | 1,447–1,530 ms |
| Cache hit | **48–51 ms** |

**~30× faster.** Fingerprinting is content-addressed, not mtime-based, so a CI
cache restore or a `tar -p` extraction that preserves timestamps cannot serve a
stale graph.

## Where SIMD actually earned its place

The fingerprint hashes 15 MB on every run, so it is the one genuinely hot loop —
and the only place where hand-written vector assembly pays. `zeebo/blake3` ships
AVX2/SSE4.1 assembly:

| Hash | Throughput | 15 MB corpus |
|---|---|---|
| **blake3** (AVX2 asm) | **2,817 MB/s** | 5.6 ms |
| sha256 (SHA-NI/AVX2 asm) | 1,370 MB/s | 11.5 ms |
| fnv128 (pure Go) | 551 MB/s | 28.5 ms |

blake3 is 2.06× faster than SHA-256 and 5.1× faster than a pure-Go hash.

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
the longest prefix. Ingest is the component that grows; at 10k packages it is
still 2% of a cold load and irrelevant against a cache hit.

## Reproducing

```bash
go test -run '^$' -bench . ./internal/graph/ ./internal/cache/

go build -o /tmp/assay ./cmd/assay
for sha in $(git log --format='%H' -30 <baseline> -- '*.go'); do
  git diff --name-only "${sha}~1" "${sha}" | /tmp/assay select --files - --json
done
```
