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

### Switching mutants inside one process

The lever named above is now built: the schemata switch became an in-process
setter, and a generated `TestMain` injected into each covering test package runs
its tests once per mutant, switching between runs — one process per (mutated
package × covering test package × shard) instead of one per (mutant × covering
test package). Mutants whose sites execute during package init are detected
dynamically and judged in env-selected processes of their own, because init runs
before any in-process switch can reach it; test packages with their own
`TestMain` decline the harness the same way. All three modes — harness,
process-per-mutant (`--no-harness`), binary-per-mutant (`--no-schemata`) — are
asserted to reach identical verdicts.

What the harness removes is a spawn and a package init per (mutant × test
package), so the win scales with init cost. Measured both ways, `--jobs 2`,
warm build caches, identical verdicts in every row:

| Corpus | process per mutant | harness | |
|---|---|---|---|
| synthetic: 25 mutants, init sleeps 500ms | 7.96s | **2.37s** | **3.4×** |
| `internal/cover`, whole package, 98 mutants | 347–362s | 372–375s | ~wash (0.95×) |

The synthetic is the honest upper bound: its init is pure cost and its tests are
microseconds, so nearly everything the harness can remove is there to remove.
`internal/cover` is close to the worst case, for the same reason it was in the
schemata measurement: it is survivor-bound, and its survivors' covering tests
are this repository's heavyweight integration suites — the harness removes
overhead, not test execution.

The wash took two scheduling fixes to reach, and the honest telling is that the
first harness executor *lost* to the path it replaced, 459s to 337s. Charged
test time said the harness was doing 19% less work per survivor — the init and
spawn savings are real — while wall clock said the walkers were barely
overlapping. Two causes, fixed in sequence: a batch-level barrier that
synchronised every walker on each test package, and static sharding, which no
cost estimate can balance because a survivor costs its whole covering plan, a
kill costs almost nothing, and which is which is exactly what the run exists to
find out. Walkers now pull guided chunks from a shared queue, so a walker
anchored on an expensive survivor anchors only itself. The residual few percent
is the process restarts that crashing kills force, plus the tail of the last
chunk.

Two structural facts keep the harness honest rather than fast-but-wrong. A test
package with its own TestMain declines it and takes env-selected processes on
the same shared binary. And a mutant whose site executes during package init
cannot be judged by an in-process switch at all — init has already run, as the
original, before TestMain gets control — so the runtime records which sites
init executed and those mutants are judged in processes of their own, where the
environment selects the mutant first. Both paths are pinned by fixtures whose
verdicts would silently flip without them.

### Cheapest covering package first

Covering packages were judged in lexicographic order, so a kill the cheap
package finds in microseconds could first pay for an expensive integration
suite by accident of naming. Every judging path now orders a plan's packages by
their indexed durations, cheapest first. On `internal/cover` the effect is not
a headline and was not expected to be — kills already averaged ~0.3s in both
modes because the cheap package usually *is* first alphabetically here — the
point is that the order is now chosen by data rather than by names, which is
pinned by a test whose kill attribution flips with the ordering hook.

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

## Watch mode

The watch loop's promise is a bounded save-to-verdict cycle, and its two costs
were measured separately at `--jobs 2`:

| Cost | Measured |
|---|---|
| Warm cycle end to end, two-package fixture (save → verdict) | **~420ms** |
| Plan phase alone, full 695-package workspace, graph memoised | **~172ms** |
| Plan phase, same workspace, graph reload forced | 1.9s |

The third row is why the memoisation exists: the graph cache is keyed by content
fingerprints, so every save misses it by design — but a body edit moves no
import edge. The planner records each file's package clause and import block at
load time and reloads the graph only when a save changes that signature; a body
edit costs a microsecond comparison. New, deleted, unparseable and module files
all force the reload, because a graph of uncertain shape cannot be allowed to
under-select.

The end-to-end warm cycle includes an unavoidable relink: the saved package's
test binary must be rebuilt because its source changed. Binaries for packages
the save did not touch stay warm for the whole session.

## Index cost

| Scope | Packages | Tests | Wall clock |
|---|---|---|---|
| `internal/...` of assay | 9 | 166 | 24s |
| `shared` module | 32 | 279 | 36s |

Roughly 8 tests/second on 4 cores — one process per test plus a coverage profile
parse. Incremental rebuilds re-index only packages whose dependency closure moved,
so the steady-state cost after a one-file edit is a handful of packages.

### Single-process collection

Collection now defaults to one harness process per package — a `TestMain`
injected through `-overlay` runs every test in the same process, snapshotting
`runtime/coverage` counters between tests — with the per-test-process path kept
under `--legacy-collection`. Measured at `9093af2`, `--jobs 2`, fresh caches,
identical records both ways (an agreement test enforces that in CI):

| Workload | Single-process | Per-process | |
|---|---|---|---|
| assay `internal/...`, 12 packages / 256 tests | 65s | 66s | wash |
| synthetic: 160 trivial tests, trivial init | 1.03s | 0.76s | 0.73× |
| synthetic: 40 tests behind an 80ms package init | 1.71s | 4.05s | **2.4×** |

The three rows are the honest shape of this change, and the middle one needs
saying out loud: **as implemented, the mechanism trades one process spawn for
another.** Each test's counter window still round-trips through
`go tool covdata textfmt`, so a package of trivial tests pays a covdata spawn
where it used to pay a test-binary spawn — a bounded loss of a couple of
milliseconds per test, which is why the real suite is a wash.

What the harness actually removes is **re-execution of package init**, which
per-process collection paid once per test. The 80ms-init row is 40 × 80ms of
redundant init collapsing into one, and that is the profile of the packages this
tool is aimed at — config parsing, regex compilation, fixture and model
registration all live in init in enterprise code. The loss is bounded and tiny;
the win scales with init cost times test count.

Durations also become honest in this mode: they are measured inside the harness
around `m.Run`, not around a process that spends most of its life initialising.

The follow-up that would turn the wash into a win everywhere is parsing the
binary covcounters format directly instead of shelling out to covdata per
window. It is deliberately not attempted here: the format is internal to the Go
toolchain and version-sensitive, and a maintenance trap should be walked into
deliberately or not at all.

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
