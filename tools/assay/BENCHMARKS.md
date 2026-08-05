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

## Mutation testing

Measured on `internal/cover` (2 files, ~340 lines, 201 indexed tests in the
module), whole-package scope:

| Run | Mutants | Killed | Survived | MSI | Wall clock |
|---|---|---|---|---|---|
| Initial | 72 | 59 | 12 | **83.1%** | 5m22s |
| After acting on the survivors | 72 | 64 | 7 | **90.1%** | 3m41s |

Diff scope, a one-line edit inside a function: **1 mutant, 9.8s**. That is the
difference the default makes — mutating a whole package is a nightly job,
mutating a diff is a PR check.

### Sharing one binary across a package's mutants

Compiling a test binary per mutant is what made the runs above take minutes, so
the next step compiled every mutation site in a package into one binary behind a
runtime switch. Matched A/B at `5614746`, same index, `--jobs 2`, four cores:

| Mode | Mutants | Wall | CPU | Result |
|---|---|---|---|---|
| a binary per mutant (`--no-schemata`) | 76 | 3m49s | 698s | 68 killed / 7 survived / 1 no-coverage, MSI 90.7% |
| one binary per package | 76 | **3m07s** | **582s** | identical, mutant for mutant |

**1.22× wall, 1.20× CPU.** The literature reports 4.1×–14× for mutant schemata, so
this needs explaining rather than rounding up. Splitting by outcome says where it
went:

| Per mutant | a binary each | shared binary | |
|---|---|---|---|
| killed | 1.18s | **0.26s** | 4.5× |
| survived | 34.9s | 31.4s | 1.11× |

A kill stops at the first test that fails. A survivor has to exhaust every covering
test in every covering package before it can be called a survivor at all. So **7
survivors account for 94% of the run** and 68 kills account for 6% — and survivor
cost is test execution, which sharing a binary cannot touch. The 4.5× on the kill
path is the mechanism working exactly as advertised; the headline is diluted
because this particular package is survivor-bound.

`internal/cover` is also close to the worst case for this measurement. It is a leaf
package, so every package that imports it — `cli`, `index`, `mutate`, `selection` —
contributes covering tests, and those are the heavyweight integration suites. Each
survivor drags all of them in.

The 72-mutant rows above are **not** comparable: mutation eligibility now spans
statements rather than function-body braces, which admitted one-line function
bodies and took the population from 72 to 76.

Diff scope, the same one-line edit, 4 mutants: **46.3s shared against 44.0s
separate**. A wash. Four mutants is not enough to amortise a shared build, and the
first mutant has to wait for it while separate builds start immediately. Getting
even to a wash took a fix: the first implementation compiled a binary for every
test package a batch might reach, which cost 53.4s against 45.7s — the fast path
was slower than the path it replaced. Binaries are now compiled on first use, and
a mutant that dies to the first test package never pays for the rest.

**What would actually move the whole-package number.** Not compiling less —
survivors dominate, and they are bound by running the same covering tests once per
mutant. The lever is running each covering test once across all mutants, which
needs the runtime switch to be settable per test rather than read once per process.
That is a larger change than this one and is not attempted here.

### What it actually found

The twelve initial survivors were real gaps in this repository's own tests, in the
code that decides which lines narrowing may trust:

- `functionBodyRanges` computes a body span as `[open+1, end-1]`. Mutating `end-1`
  to `end+1` extends the span one line past the closing brace, so a **declaration
  would be classified as executable and narrowing would trust it**. Nothing caught
  that. Every one of those lines was already covered — just not asserted on.
- `add()` skipped a nil body, and no test passed a bodyless declaration.
- `classifyLine`'s `line <= len(text)` bound was never exercised on a file's final
  line.
- `ExecutableRanges`, added in the previous slice, had no error-path test at all.
- Forcing the `len(blocks) == 0` guard false sends an empty slice into a function
  that indexes `blocks[:1]`, which panics. No test passed a file of pure
  declarations.

Acting on those took MSI from 83.1% to 90.1%.

### The seven that remain are equivalent mutants

They are recorded in `.assay/mutation-baseline.json` with a reason each. Six do not
change behaviour at all — inverted empty ranges whose `Contains` is false either
way, a sort comparator whose tiebreak the following merge makes irrelevant, a `>=`
that widens an assignment to a no-op. The other two mask a read error behind the
parse error that follows, which no assertion short of matching error text can
distinguish.

Equivalent mutants are undecidable in general and are the main reason mutation
testing has poor industry adoption. The baseline is the management strategy, not a
solution, and the file says so per entry.

### A false kill, and what it cost

While measuring the diff-scoped path, a boundary mutation on a *discarded*
expression came back killed — by a test in another package that had nothing to do
with it. That test passes on its own. It had failed for its own reasons while the
mutant ran, and a covering test that fails for unrelated reasons marks its mutant
killed, so every such false kill **inflates** the score.

`assay mutate` now runs the covering tests against unmutated code first and refuses
to score if any already fail. That catches a deterministically red suite. It does
not catch the actual culprit here, which was a load-sensitive test of ours that
spawns nested `go test -c` runs — so the caveat is documented rather than closed.

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
