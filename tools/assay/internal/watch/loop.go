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
	Batches  <-chan []string
	Out      io.Writer
	UseColor bool
	Jobs     int
	Timeout  time.Duration
}

const defaultRunTimeout = 2 * time.Minute

// Loop is the watch session: one cycle per batch, until the channel closes or
// the context ends.
//
// Failures are sticky. A test that failed keeps running at the front of every
// following cycle until it passes, whichever file the save touched — the loop's
// job is to get the developer back to green, and the fastest route is telling
// them immediately whether the thing that was broken is still broken.
func Loop(ctx context.Context, opts LoopOptions) error {
	failing := make(map[string][]string)

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

func runCycle(ctx context.Context, opts LoopOptions, batch []string, failing map[string][]string) error {
	started := time.Now()
	lines := report.NewLines(opts.Out, opts.UseColor)

	cycle, err := opts.Planner.Plan(ctx, batch)
	if err != nil {
		return err
	}

	runs := withStickyFailures(cycle.Runs, failing, opts.Planner, batch)
	if len(runs) == 0 {
		lines.Muted("%s  nothing to run for this change", time.Now().Format("15:04:05"))

		return nil
	}

	header := fmt.Sprintf("%s  %d packages", time.Now().Format("15:04:05"), len(runs))
	if len(batch) > 0 {
		header = fmt.Sprintf("%s  %s → %d packages",
			time.Now().Format("15:04:05"), summariseBatch(batch), len(runs))
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

	results := executeRuns(ctx, opts, runs)
	if ctx.Err() != nil {
		return ctx.Err()
	}

	passed, failed := 0, 0
	testCount := 0
	for _, result := range results {
		testCount += len(result.Tests)
		switch {
		case result.Err != nil:
			failed++
			lines.Warn("  %s  error: %v", shorten(result.ImportPath), result.Err)
			delete(failing, result.ImportPath)
		case result.Passed:
			passed++
			lines.Good("  %s  %s  %s%s", shorten(result.ImportPath), describeTests(result.Tests),
				result.Duration.Round(time.Millisecond), reusedTag(result.Reused))
			delete(failing, result.ImportPath)
		default:
			failed++
			lines.Warn("  %s  FAIL  %s", shorten(result.ImportPath), result.Duration.Round(time.Millisecond))
			printFailure(opts.Out, result.Output)
			failing[result.ImportPath] = failedTestNames(result.Output)
		}
	}

	total := time.Since(started)
	if failed == 0 {
		lines.Good("✓ %d packages · %s tests · %s",
			passed, countOrAll(testCount, results), total.Round(time.Millisecond))
	} else {
		lines.Warn("✗ %d of %d packages failed · %s", failed, len(results), total.Round(time.Millisecond))
	}
	lines.Line("")

	return nil
}

// withStickyFailures prepends still-failing packages that this cycle would not
// otherwise run, narrowed to the tests that were failing.
func withStickyFailures(
	runs []PlannedRun,
	failing map[string][]string,
	planner *Planner,
	batch []string,
) []PlannedRun {
	planned := make(map[string]struct{}, len(runs))
	for _, run := range runs {
		planned[run.ImportPath] = struct{}{}
	}

	var sticky []PlannedRun
	for importPath, tests := range failing {
		if _, already := planned[importPath]; already {
			continue
		}
		run, ok := planner.runForFailing(importPath, tests)
		if !ok {
			continue
		}
		sticky = append(sticky, run)
	}
	sort.Slice(sticky, func(i, j int) bool { return sticky[i].ImportPath < sticky[j].ImportPath })

	// The batch's own runs still come first: the save the developer just made is
	// the thing they are waiting on.
	if len(batch) == 0 {
		return append(sticky, runs...)
	}

	return append(runs, sticky...)
}

func executeRuns(ctx context.Context, opts LoopOptions, runs []PlannedRun) []RunResult {
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
				result.Passed = true
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
		return nil
	}

	return results
}

// runForFailing rebuilds a minimal run for a package whose tests failed in an
// earlier cycle.
func (p *Planner) runForFailing(importPath string, tests []string) (PlannedRun, bool) {
	// The planner's graph state lives cycle to cycle through the cache, so the
	// cheap route to a directory is the last loaded graph via the loader's
	// fingerprinter — but correctness only needs the directory, which the run
	// carries from the cycle that recorded the failure. Failing that, skip: the
	// package will resurface the next time a save reaches it.
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
	if len(tests) > 0 {
		run.Pattern = runpattern.From(tests)
		run.Tests = append([]string(nil), tests...)
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
		if len(result.Tests) == 0 {
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
