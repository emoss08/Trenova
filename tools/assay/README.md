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

**Mutation testing.** `assay mutate` changes the code in small deliberate ways and
reports whether any test objected. A surviving mutant is a gap coverage cannot
show you: the line runs under test, but nothing asserts on what it does.

**Flake intelligence.** Every run assay performs is evidence: `assay flakes`
reports tests observed both passing and failing at the same code state, hunts
them on demand, and quarantines them so they can never judge a mutant.

**Risk map.** `assay risk` joins the index with git history: functions ranked
by churn × uncovered executable lines — where the next regression is most
likely to land unseen.

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

## Watch mode

```bash
assay index && assay watch
```

`assay watch` keeps a session open: every save re-runs the tests that change can
actually affect, and nothing else. It is the same selection and narrowing
pipeline the CI commands use — the same graph, the same fail-safe rules — so
the loop can never skip a test that `assay run` would have executed for the
same dirty tree.

Three things make the loop fast enough to live in:

- **Warm binaries.** Test binaries are kept linked between saves and rebuilt
  only when a package's dependency closure actually changes, so a cycle is
  bounded by the tests themselves, not by `go test` start-up.
- **A memoised graph.** The package graph reloads only when a save changes a
  file's package clause or imports — a body edit costs microseconds of
  signature comparison instead of a workspace reload. On this 695-package
  workspace a warm cycle plans in ~170ms.
- **Test-edit refinement.** Editing the body of `TestShipmentCreate` runs
  `TestShipmentCreate`, not its whole package — guarded strictly: any edit
  outside a test declaration, or a package that is also reached by some other
  change, runs in full.

Failing tests are **sticky**: they re-run at the front of every following cycle
until they pass, whichever file the save touched. Saves of other dirty files
defer their packages' tests until a save reaches them, and the header says so.

An index built at `HEAD` makes the loop narrow to individual covering tests;
without one it still works at package precision and says why.

## Line-level narrowing

Package-level selection has a structural ceiling: a change anywhere in a package
that 200 others import must select all 200, even when it touches one function
three tests exercise. Narrowing removes that ceiling by recording which tests
execute which lines.

```bash
# Build the index. Do this on the commit you will diff against.
assay index

# Now selection narrows to individual tests
assay select --since "$BASE" -v

# Which tests cover this line?
assay explain internal/core/domain/shipment/shipment.go:412
```

`assay index` compiles one test binary per package, enumerates its tests, and
attributes coverage per test. It is incremental: only packages whose dependency
closure changed are re-indexed. Records live beside the graph cache and are
never committed.

### How collection runs

By default the whole package is collected in **one process**: a `TestMain`
injected through `-overlay` — never written to your tree — runs each test in the
same binary, snapshotting and clearing the `runtime/coverage` counters between
tests. Package init executes once instead of once per test, which is where the
time goes in packages that parse config, compile regexes, or register fixtures
at init. Durations recorded this way are measured around the test itself, not
around a process that spends most of its life initialising.

A hanging test cannot take the package with it: the harness reports progress
after each test's counters are safely on disk, a rolling per-test deadline kills
the process group when one stalls, and everything already collected is kept —
only the stalled test and the remainder re-run one process at a time, where the
per-test timeout applies.

Packages that define their own `TestMain` are collected one process per test,
since injecting a second `TestMain` would not compile. `--legacy-collection`
forces that mode everywhere; the two modes are asserted to produce identical
records, so it is a safety valve rather than a different answer.

One attribution difference is worth knowing: lazy initialisation triggered by
whichever test happens to run first — a `sync.Once`, a package-level cache — is
attributed to that test's window in single-process mode, where fresh processes
attributed it to every test. Package-level `init` and `var` initialisers are
unaffected: their window is merged into every test.

### The index is pinned to a commit

An index describes one tree. A diff has two sides, and the index's line numbers
only mean anything in the coordinates of the tree it was built from — so
narrowing engages **only when a record's commit equals the selection's base
commit**, and looks up base-side line numbers. Anything else runs in full.

In practice:

- **Locally**: `assay index` at a clean `HEAD`, then edit. `assay run` diffs
  against `HEAD`, which is what the index describes, so narrowing engages.
- **In CI**: index on the base branch and cache it, or index the merge-base
  commit in the job. `assay select --since "$BASE"` then matches.

`assay index` refuses to run on a dirty tree, because the records would claim to
describe `HEAD` while actually describing something else. `--allow-dirty` builds
them anyway and marks them unusable for narrowing.

### What narrowing refuses to do

A selector that skips a test it should have run is worse than no selector, and
coverage makes that easy to get wrong: coverage instruments **statements**, so a
changed `const`, struct field, or function signature appears in no coverage block
at all. A naive lookup would find "no test covers this line" and skip everything.

Every changed line is classified against the AST first:

| Changed line | Behaviour |
|---|---|
| Inside a function body, covered | Narrow to the covering tests |
| Outside any function body — type, const, var, import, signature | **Run the package in full** |
| Inside a body but in no coverage block | Skip: no test executes it, so no test can observe the change |
| Blank line, or a plain comment | Ignored entirely |
| A `//go:` directive | **Run in full** — directives change build behaviour |
| A one-line function (`func F() int { return 1 }`) | **Run in full** — the body shares its line with the braces |

And per package:

| Condition | Behaviour |
|---|---|
| No index record, or one from a different commit | Run in full |
| Record degraded (a coverage path could not be resolved) | Run in full |
| A test in the record but not attributable — skipped, panicked, no profile | Always selected |
| Reached by any change with no line attribution | Run in full |
| Added, deleted or renamed file; any non-Go file | Run in full |

Every fallback is reported with its reason, in text and in `--json`.

### Proving it

```bash
assay verify --since "$BASE"
```

`verify` runs the **complement** of the narrowed selection — every test that
package-level selection would have run and narrowing dropped — and asserts they
all pass. A failure there is proof the narrowing was unsound, and it says so in
those words. Intended for a nightly job; the complement is the expensive half.

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

## Mutation testing

```bash
# Only the lines this diff touched (the default; fast enough for a PR)
assay mutate --since "$BASE"

# A whole package (minutes to hours; has to be asked for)
assay mutate --whole --packages ./internal/cover

# Accept the current survivors so CI fails only on new ones
assay mutate --whole --write-baseline
```

Mutation reuses the coverage index twice over: only lines a test executes are
worth mutating, and only the tests covering a mutated line are run to judge it.
On `internal/cover` that is a handful of tests per mutant instead of 201.

Mutants are injected through `-overlay`, so the source tree is never written to
and an interrupted run leaves nothing behind.

### One binary per package, not per mutant

Compiling a test binary for every mutant is what makes mutation testing slow, so
by default every mutation site in a package is compiled into the *same* binary
behind a switch on `ASSAY_MUTANT`. Each mutant is then a run rather than a build.
`ASSAY_MUTANT=0`, and anything unset or unparseable, leaves every site behaving
exactly like the original — which is why the same binary can serve as its own
control.

The switch lives in an `assay_schemata.go` that reaches the compiler through
`-overlay` and is never written to your tree. It imports only `os`, `sync` and
`sync/atomic`, so injecting it into a low-level package is unlikely to create an
import cycle.

A site is only rewritten this way when the rewrite is provably type-correct, and
assay would rather be slow than emit something that does not compile — under a
shared binary a single bad rewrite would invalidate every mutant in the package,
not one. These fall back to a binary of their own:

| Declined | Why |
|---|---|
| a comparison whose result is a named boolean type | the helper returns `bool`, which is not assignable to it |
| `&&` / `\|\|` on named boolean types | the short-circuit thunk is spelled `func() bool` |
| anything in a `const` declaration or an array length | the language wants a constant, not a call |
| operands of a type parameter | its constraint need not imply the helper's |
| `m[k]++` | legal Go, but `&m[k]` is not, and the helper steps through a pointer |
| operands of different types | inference would fail rather than pick one |

Sharing the binary removes the compile per mutant; sharing the *process* removes
the spawn and package init per mutant. A generated `TestMain`, injected into each
covering test package the same way, runs the package's tests once per mutant and
switches the active mutant between runs — so judging N mutants costs one process
per covering test package rather than N. A mutant that crashes or hangs the
shared process is judged by that death, and the rest continue in a fresh one.

Two kinds of package decline the harness, quietly and by design. A covering test
package with its own `TestMain` cannot host the injected one, so its mutants run
in env-selected processes on the same shared binary. And a mutant whose site
executes during package *init* cannot be judged by an in-process switch at all —
init has already run, as the original, before `TestMain` gets control. The
runtime tracks exactly which sites init executed, and those mutants are judged
in processes of their own, where the environment selects the mutant before init
runs. Cheapest covering package first, in every mode: the index knows what each
package's tests cost, and a kill found by the fast package never pays for the
slow one.

The summary reports the split whenever anything fell back. `--no-schemata` forces
every mutant onto its own binary and `--no-harness` forces a process per mutant;
all three modes are asserted to reach identical verdicts, mutant for mutant, so
they are safety valves rather than different answers.

One caveat applies to both modes: rewriting expressions changes what the compiler
can inline and what escapes, so a test asserting an exact allocation count could
behave differently under mutation than it does normally.

### What the outcomes mean

| Outcome | Meaning |
|---|---|
| `killed` | A covering test noticed. This is what you want |
| `timeout` | The mutant stopped terminating — also a kill, since behaviour changed |
| `survived` | Every covering test still passed. A gap |
| `no-coverage` | No indexed test executes the line. Excluded from the score; the coverage report already told you |
| `not-built` | The mutant did not compile. An assay defect, excluded from the score rather than counted as a kill |

A survivor means *no test with known coverage of that line objected*. Always-run
tests — ones whose attribution the index could not determine — are deliberately
excluded from mutant plans, because a test that was already failing would mark
every mutant killed and inflate the score.

### Every survivor is a prescription

A survivor's location tells you where the gap is; the report also tells you
what to do about it. Survivors are grouped by function and ranked worst-first —
the function with the most survivors is the biggest assertion gap — and each
one names the covering test to extend and the input that is missing:

```
7 new surviving mutants in 3 functions — worst first:
  Weak (heavy.go) — 4 survivors:
    cbdbe541  heavy.go:40  < -> <=  (1 tests ran)
        → extend TestWeakBarely (covers 7 lines): no test pins the boundary:
          add an input where both sides of `<` are equal — the only input
          where `<` and `<=` disagree
```

The suggested test is the *most focused* covering test — same package first,
then fewest covered lines, from the index — because a test that already
executes the line with the narrowest scope is the cheapest place to add the
missing assertion. The prescription is derived from the mutation itself: every
covering test ran both versions to the same verdict, so the input that
distinguishes them is, by construction, one nobody supplies. `--json` carries
the same advice per survivor for tooling.

### The score only means something on a green suite

Before judging any mutant, `assay mutate` runs the union of the covering tests
against unmutated code. If any already fail it refuses to score and names them,
because a failing test marks every mutant it judges as killed. `--allow-failing-tests`
proceeds by dropping those tests from every plan instead.

This catches a deterministically red suite. It does **not** catch a test that only
fails under load — mutation runs many concurrent `go test -c` builds, and a
timing-sensitive test can produce a false kill that flatters the score. Run
mutation on a quiet machine, and treat a surprising kill as worth checking.

### Equivalent mutants

Some mutants cannot be killed by any test because they do not change behaviour.
`if n > limit { return limit }` mutated to `>=` returns the same value at
`n == limit`. That is not a hole in your suite, and no tool can tell the two apart
in general — the problem is undecidable.

The baseline is how they are managed. `--write-baseline` records the current
survivors in `.assay/mutation-baseline.json`, which is committed; later runs fail
only on survivors that are not in it. Annotate the entries with *why* each is
accepted — the file is curated policy, not generated output.

Mutant IDs hash the enclosing function, the mutator, and the changed text, and
deliberately exclude line numbers, so editing code above a mutation site does not
churn the baseline.

Writing the baseline checks `git check-ignore` first and warns if the path would
not be committed. A stray `coverage/` rule once silently swallowed an entire
package in this repository, so the guard is there on purpose.

## Flake intelligence

assay runs tests more often than anything else in your toolchain — every watch
cycle, every mutation preflight — and every one of those runs is evidence about
test reliability. That evidence is kept: each attributable verdict is recorded
in a machine-local journal, keyed by the package's content fingerprint. A test
observed both passing and failing **at the same fingerprint** changed its
verdict while the code did not, which is the definition of a flake. A verdict
that changed across fingerprints is just the code changing, and is never
reported.

```
assay flakes                                       # report, from accumulated evidence
assay flakes --hunt --packages ./internal/... --runs 20
assay flakes --quarantine                          # record them in .assay/flaky-tests.json
```

Watch is a natural flake detector by construction: a failing test re-runs at
the front of every following cycle, and when the package has not changed, a
pass on the re-run is a same-code conflict — recorded, no extra work. `--hunt`
generates evidence deliberately instead of waiting for it: one compile per
package, then repeated runs at the current tree state, verdicts attributed per
test. Both feed the same journal, as does every mutation preflight.

The quarantine file is committed policy, like the mutation baseline. A
quarantined test is excluded from every mutant's plan — a flaky covering test
marks mutants killed for reasons that have nothing to do with the mutation, and
every such false kill inflates the score. Watch keeps running quarantined
tests, but labels their failures as possibly not the change's fault; hiding
them entirely would let a real regression in a flaky test go unseen.

`assay flakes` exits non-zero when it detects unquarantined flakes, so it can
gate in CI the same way `--min-msi` does.

## The risk map

Coverage alone ranks nothing useful: untested-but-frozen code is a dormant
liability, and heavily-tested hot code is fine. `assay risk` ranks by the
product of the two signals that matter together — how often a file changes,
and how many of a function's executable lines no attributable test executes:

```
1 exposed functions — churn × uncovered lines, worst first (top 1):
     4  example.com/app/pricing.Weak
        pricing/weak.go:38-45 · 2 of 5 executable lines uncovered · 2 commits touched this file in 90 days
14 churning functions are fully covered — change landing where tests are watching
```

The score is `commits × uncovered lines` — no weights, no tuning, both factors
printed so the ranking can be argued with. Coverage comes from the index,
unioned across every test package that can reach the analyzed one, so a
package with no tests of its own is still credited with the coverage its
importers give it. Churn is per file (`--window-days`, default 90); a rename
reads as fresh churn at the new path, which is defensible — freshly moved code
is risk.

The footer keeps the report honest: functions churning under full coverage,
uncovered functions in untouched files (dormant, lower priority), packages
analyzed with partial records (risk possibly overstated), and packages the
index cannot see at all.

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
| `--no-index` | Skip line-level narrowing; select whole packages |
| `--cache-dir` | Cache location for graph and index (default: user cache dir, or `ASSAY_CACHE`) |

`assay index` additionally takes `--jobs`, `--timeout`, `--packages`, `--quiet`,
`--allow-dirty` and `--legacy-collection`. `assay mutate` takes `--whole`, `--packages`, `--baseline`,
`--write-baseline`, `--min-msi`, `--allow-failing-tests`, `--no-schemata`, `--no-harness`, `--jobs`
and `--json`. `assay flakes` takes `--hunt`, `--runs`, `--packages`, `--timeout`
and `--quarantine`. `assay risk` takes `--window-days`, `--top`, `--packages`
and `--json`.

## Commands

| Command | Purpose |
|---|---|
| `assay select` | Show the plan without running anything |
| `assay run` | Run the plan through `go test` |
| `assay index` | Build the line-to-test coverage index |
| `assay explain <file>:<line>` | Which tests cover this line? |
| `assay mutate` | Change the code deliberately; report which tests failed to notice |
| `assay verify` | Run what narrowing excluded, to prove it was sound |
| `assay flakes` | Report tests whose verdicts changed while the code did not |
| `assay risk` | Rank functions by churn × uncovered executable lines |

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
| `github.com/sourcegraph/conc` | Structured concurrency. Its pools propagate panics through `Wait()` instead of letting one worker take the process down with an unrelated stack, and handle context cancellation and first-error for free |
| `github.com/stretchr/testify` | Test assertions |

`assay` deliberately shells out to `git` rather than embedding `go-git`: the
porcelain commands used here are stable, and `go-git`'s diff is markedly slower
on large repositories.
