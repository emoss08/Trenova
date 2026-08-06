package watch

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/sourcegraph/conc/pool"

	"github.com/emoss08/assay/internal/report"
	"github.com/emoss08/assay/internal/runpattern"
)

type LoopOptions struct {
	Planner  *Planner
	Binaries *BinaryCache
	Batches  <-chan Batch
	Out      io.Writer
	UseColor bool
	Jobs     int
	Timeout  time.Duration
}

const defaultRunTimeout = 2 * time.Minute

// failure is what stickiness remembers about a red package. wholePackage means
// the failing set is unknown — a package-level failure, a hang, or a run whose
// output named no test — and only a full run can clear it.
type failure struct {
	tests        map[string]struct{}
	wholePackage bool
}

// Loop is the watch session: one cycle per batch, until the channel closes or
// the context ends.
//
// Failures are sticky, and the accounting is per test name, not per package: a
// cycle that runs only a subset of a package's tests can clear exactly the
// names it executed and nothing more. A run that never executed the broken
// test — because narrowing selected different tests — cannot silently declare
// the package healthy, and an infrastructure error keeps the entry rather than
// evicting it: the one failure the mechanism must never drop is the one it
// could not re-check.
func Loop(ctx context.Context, opts LoopOptions) error {
	failing := make(map[string]*failure)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case batch, open := <-opts.Batches:
			if !open {
				return nil
			}
			if err := runCycle(ctx, opts, batch, failing); err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}

				lines := report.NewLines(opts.Out, opts.UseColor)
				lines.Warn("cycle failed: %v", err)
			}
		}
	}
}

func runCycle(ctx context.Context, opts LoopOptions, batch Batch, failing map[string]*failure) error {
	started := time.Now()
	lines := report.NewLines(opts.Out, opts.UseColor)

	if batch.Resync {
		// The kernel dropped events, so both the batch and the memoised graph are
		// unreliable; replan everything from disk.
		opts.Planner.Invalidate()
		lines.Warn("%s  filesystem events were dropped; re-checking everything dirty",
			time.Now().Format("15:04:05"))
	}

	cycle, err := opts.Planner.Plan(ctx, batch.Paths)
	if err != nil {
		return err
	}

	runs, gone := withStickyFailures(cycle.Runs, failing, opts.Planner, batch.Paths)
	for _, importPath := range gone {
		lines.Warn("  %s left the workspace while failing; dropping it from the session",
			shorten(importPath))
		delete(failing, importPath)
	}

	if len(runs) == 0 {
		lines.Muted("%s  nothing to run for this change", time.Now().Format("15:04:05"))

		return nil
	}

	header := fmt.Sprintf("%s  %d packages", time.Now().Format("15:04:05"), len(runs))
	if len(batch.Paths) > 0 {
		header = fmt.Sprintf("%s  %s → %d packages",
			time.Now().Format("15:04:05"), summariseBatch(batch.Paths), len(runs))
	}
	if cycle.Deferred > 0 {
		noun := "packages"
		if cycle.Deferred == 1 {
			noun = "package"
		}
		header += fmt.Sprintf(" (%d other dirty %s deferred)", cycle.Deferred, noun)
	}
	lines.Line("%s", header)
	if cycle.Note != "" {
		lines.Warn("note: %s", cycle.Note)
	}

	results, err := executeRuns(ctx, opts, runs)
	if err != nil {
		return err
	}

	passed, failed, skipped := 0, 0, 0
	testCount := 0
	for _, result := range results {
		testCount += len(result.Tests)
		switch {
		case result.Err != nil:
			failed++
			lines.Warn("  %s  error: %v", shorten(result.ImportPath), result.Err)
			recordUnjudged(failing, result)
		case result.Skipped:
			skipped++
			lines.Muted("  %s  no test binary (build tags?)", shorten(result.ImportPath))
		case result.Passed:
			passed++
			lines.Good("  %s  %s  %s%s", shorten(result.ImportPath), describeTests(result.Tests),
				result.Duration.Round(time.Millisecond), reusedTag(result.Reused))
			recordPass(failing, result)
		default:
			failed++
			lines.Warn("  %s  FAIL  %s", shorten(result.ImportPath), result.Duration.Round(time.Millisecond))
			printFailure(opts.Out, result.Output)
			recordFailure(failing, result)
		}
	}

	total := time.Since(started)
	switch {
	case failed == 0 && passed > 0:
		summary := fmt.Sprintf("✓ %d packages · %s tests · %s",
			passed, countOrAll(testCount, results), total.Round(time.Millisecond))
		if skipped > 0 {
			summary += fmt.Sprintf(" · %d skipped", skipped)
		}
		lines.Good("%s", summary)
	case failed == 0:
		lines.Muted("nothing executed · %s", total.Round(time.Millisecond))
	default:
		lines.Warn("✗ %d of %d packages failed · %s", failed, len(results), total.Round(time.Millisecond))
	}
	lines.Line("")

	return nil
}

// recordPass clears exactly what this run proved: the names it executed. Only a
// whole-package run may clear a whole-package failure.
func recordPass(failing map[string]*failure, result RunResult) {
	entry, ok := failing[result.ImportPath]
	if !ok {
		return
	}

	if len(result.Tests) == 0 {
		delete(failing, result.ImportPath)

		return
	}
	if entry.wholePackage {
		return
	}

	for _, name := range result.Tests {
		delete(entry.tests, name)
	}
	if len(entry.tests) == 0 {
		delete(failing, result.ImportPath)
	}
}

func recordFailure(failing map[string]*failure, result RunResult) {
	names := failedTestNames(result.Output)

	entry, ok := failing[result.ImportPath]
	if !ok {
		entry = &failure{tests: make(map[string]struct{})}
		failing[result.ImportPath] = entry
	}

	if len(names) == 0 {
		// The run failed without naming a test — a panic outside any test, a
		// TestMain exit. Only a full pass can clear this.
		entry.wholePackage = true

		return
	}

	if !entry.wholePackage && len(result.Tests) > 0 {
		executed := make(map[string]struct{}, len(result.Tests))
		newlyFailed := make(map[string]struct{}, len(names))
		for _, name := range names {
			newlyFailed[name] = struct{}{}
		}
		for _, name := range result.Tests {
			executed[name] = struct{}{}
			if _, still := newlyFailed[name]; !still {
				delete(entry.tests, name)
			}
		}
	}
	if len(result.Tests) == 0 {
		entry.wholePackage = false
		entry.tests = make(map[string]struct{})
	}
	for _, name := range names {
		entry.tests[name] = struct{}{}
	}
}

// recordUnjudged holds a package the cycle could not judge — a build error, a
// hang. If it was already failing the entry survives untouched; if it was not,
// it becomes a whole-package failure so the next cycle re-checks it.
func recordUnjudged(failing map[string]*failure, result RunResult) {
	if _, ok := failing[result.ImportPath]; ok {
		return
	}
	failing[result.ImportPath] = &failure{tests: make(map[string]struct{}), wholePackage: true}
}

// withStickyFailures folds the failing set into the cycle.
//
// A failing package this cycle already plans gets the failing names unioned
// into its run — a narrowed run must not be allowed to "pass" a package while
// the broken test sits outside its pattern. A failing package the cycle does
// not plan is appended as its own run. Packages that no longer resolve are
// returned for reporting and eviction.
func withStickyFailures(
	runs []PlannedRun,
	failing map[string]*failure,
	planner *Planner,
	batch []string,
) ([]PlannedRun, []string) {
	planned := make(map[string]int, len(runs))
	for i, run := range runs {
		planned[run.ImportPath] = i
	}

	var gone []string
	var sticky []PlannedRun

	for _, importPath := range sortedFailureKeys(failing) {
		entry := failing[importPath]

		if i, already := planned[importPath]; already {
			mergeFailureIntoRun(&runs[i], entry)

			continue
		}

		run, ok := planner.runForFailing(importPath, entry)
		if !ok {
			gone = append(gone, importPath)

			continue
		}
		sticky = append(sticky, run)
	}

	if len(batch) == 0 {
		return append(sticky, runs...), gone
	}

	return append(runs, sticky...), gone
}

func mergeFailureIntoRun(run *PlannedRun, entry *failure) {
	if run.Pattern == "" {
		return
	}
	if entry.wholePackage {
		run.Pattern = ""
		run.Tests = nil

		return
	}

	union := make(map[string]struct{}, len(run.Tests)+len(entry.tests))
	for _, name := range run.Tests {
		union[name] = struct{}{}
	}
	for name := range entry.tests {
		union[name] = struct{}{}
	}

	tests := make([]string, 0, len(union))
	for name := range union {
		tests = append(tests, name)
	}
	sort.Strings(tests)

	run.Tests = tests
	run.Pattern = runpattern.From(tests)
}

func sortedFailureKeys(failing map[string]*failure) []string {
	out := make([]string, 0, len(failing))
	for key := range failing {
		out = append(out, key)
	}
	sort.Strings(out)

	return out
}

func executeRuns(ctx context.Context, opts LoopOptions, runs []PlannedRun) ([]RunResult, error) {
	cores := runtime.GOMAXPROCS(0)
	jobs := opts.Jobs
	if jobs <= 0 {
		jobs = max(1, cores/2)
	}
	jobs = min(jobs, len(runs))
	buildParallelism := max(1, cores/jobs)

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = defaultRunTimeout
	}

	results := make([]RunResult, len(runs))

	running := pool.New().
		WithMaxGoroutines(jobs).
		WithContext(ctx).
		WithFirstError().
		WithCancelOnError()

	for i := range runs {
		running.Go(func(ctx context.Context) error {
			if err := ctx.Err(); err != nil {
				return err
			}

			run := runs[i]
			result := RunResult{ImportPath: run.ImportPath, Tests: run.Tests}

			binary, reused, err := opts.Binaries.Ensure(ctx, BuildRequest{
				Root:             opts.Planner.Root,
				Env:              opts.Planner.Env,
				Tags:             opts.Planner.Tags,
				ImportPath:       run.ImportPath,
				Fingerprint:      run.Fingerprint,
				BuildParallelism: buildParallelism,
			})
			result.Reused = reused

			switch {
			case err != nil:
				result.Err = err
			case binary == "":
				result.Skipped = true
			default:
				began := time.Now()
				passed, output, runErr := RunTests(ctx, RunRequest{
					Binary:  binary,
					Dir:     run.Dir,
					Env:     opts.Planner.Env,
					Pattern: run.Pattern,
					Timeout: timeout,
				})
				result.Duration = time.Since(began)
				result.Passed = passed
				result.Output = output
				result.Err = runErr
			}

			if err := ctx.Err(); err != nil {
				return err
			}
			results[i] = result

			return nil
		})
	}

	if err := running.Wait(); err != nil {
		return nil, err
	}

	return results, nil
}

// runForFailing rebuilds a minimal run for a package whose tests failed in an
// earlier cycle, resolved against the current graph and current fingerprints so
// a stale binary can never satisfy it.
func (p *Planner) runForFailing(importPath string, entry *failure) (PlannedRun, bool) {
	dir, fp, ok := p.lastKnown(importPath)
	if !ok {
		return PlannedRun{}, false
	}

	run := PlannedRun{
		ImportPath:  importPath,
		Dir:         dir,
		Reason:      "still failing",
		Fingerprint: fp,
	}
	if !entry.wholePackage && len(entry.tests) > 0 {
		tests := make([]string, 0, len(entry.tests))
		for name := range entry.tests {
			tests = append(tests, name)
		}
		sort.Strings(tests)
		run.Pattern = runpattern.From(tests)
		run.Tests = tests
	}

	return run, true
}

func summariseBatch(batch []string) string {
	names := make([]string, 0, len(batch))
	for _, path := range batch {
		names = append(names, shortenPath(path))
	}
	sort.Strings(names)
	if len(names) > 3 {
		return fmt.Sprintf("%s +%d more", strings.Join(names[:3], ", "), len(names)-3)
	}

	return strings.Join(names, ", ")
}

func shortenPath(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) <= 2 {
		return path
	}

	return strings.Join(parts[len(parts)-2:], "/")
}

func shorten(importPath string) string {
	parts := strings.Split(importPath, "/")
	if len(parts) <= 3 {
		return importPath
	}

	return strings.Join(parts[len(parts)-3:], "/")
}

func describeTests(tests []string) string {
	switch {
	case len(tests) == 0:
		return "all tests"
	case len(tests) == 1:
		return tests[0]
	default:
		return fmt.Sprintf("%d tests", len(tests))
	}
}

func countOrAll(count int, results []RunResult) string {
	for _, result := range results {
		if len(result.Tests) == 0 && !result.Skipped {
			return "all"
		}
	}

	return fmt.Sprintf("%d", count)
}

func reusedTag(reused bool) string {
	if reused {
		return ""
	}

	return "  (built)"
}

func printFailure(out io.Writer, output []byte) {
	text := strings.TrimRight(string(output), "\n")
	for line := range strings.SplitSeq(text, "\n") {
		fmt.Fprintf(out, "    %s\n", line)
	}
}

func failedTestNames(output []byte) []string {
	seen := make(map[string]struct{})
	for line := range strings.SplitSeq(string(output), "\n") {
		trimmed := strings.TrimSpace(line)
		rest, ok := strings.CutPrefix(trimmed, "--- FAIL:")
		if !ok {
			continue
		}
		name := strings.TrimSpace(rest)
		if idx := strings.IndexAny(name, " \t("); idx > 0 {
			name = name[:idx]
		}
		if slash := strings.IndexByte(name, '/'); slash > 0 {
			name = name[:slash]
		}
		if name != "" {
			seen[name] = struct{}{}
		}
	}

	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)

	return out
}
