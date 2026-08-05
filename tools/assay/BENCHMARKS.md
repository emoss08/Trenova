# M0 measurements

Measured on the Trenova workspace: 6 modules, 685 packages, 348 with tests,
5,786 test functions.

Method: for each of the last 30 commits touching `.go` files, feed
`git diff --name-only <sha>~1 <sha>` into `assay select --files - --json` and
record how many test packages the selector returns.

Caveat: the package graph comes from the current checkout, not from each
historical commit, so packages that have since moved or been renamed are
attributed to their present location. The numbers are directional, not exact.

## Results

| Metric | Value |
|---|---|
| Commits measured | 30 |
| Median reduction | **32.0%** |
| Mean reduction | 41.7% |
| Best / worst | 99.7% / 0.0% |
| Median selected | 236 of 348 packages |
| Select-all fallback fired | 2 commits |

Broken down by commit size:

| Commit size | Median reduction | n |
|---|---|---|
| ≤ 5 files changed | **98.6%** | 7 |
| > 20 files changed | 21.6% | 13 |

Both select-all commits changed `services/tms/go.mod` and `go.sum`; one also
changed `go.work.sum`. The fallback fired for the right reason in both cases.

Graph load: ~13s cold, ~2.4s warm, for 685 packages across 6 modules.

## Reading these numbers honestly

Package-graph TIA is excellent on small, focused commits and mediocre on large
ones. A one-file fix selects 1–5 packages out of 348. A commit touching a
widely-imported domain package selects 273.

That ceiling is structural, not a bug. At package granularity, *any* change to a
package that 200 other packages import must select all 200 — even when the change
touches a single function that only three of those tests ever execute.

This is the empirical case for the line→test index (M1/M2). Coverage-level
attribution is what separates "this package changed" from "this *code path*
changed," and it is where the remaining reduction lives. The same index then
drives mutant test selection in M3/M4.

## Reproducing

```bash
go build -o /tmp/assay ./cmd/assay
for sha in $(git log --format='%H' -40 --skip=1 -- '*.go'); do
  git diff --name-only "${sha}~1" "${sha}" | /tmp/assay select --files - --json
done
```
