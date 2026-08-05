package mutate

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/sourcegraph/conc/pool"

	"github.com/emoss08/assay/internal/proc"
)

type Outcome string

const (
	// OutcomeKilled means a covering test noticed the change, which is the result
	// the suite is supposed to produce.
	OutcomeKilled Outcome = "killed"
	// OutcomeSurvived means every covering test still passed. A survivor is a gap.
	OutcomeSurvived Outcome = "survived"
	// OutcomeTimeout counts as killed: the mutant changed observable behaviour.
	OutcomeTimeout Outcome = "timeout"
	// OutcomeNotBuilt is a defect in assay, not in the suite under test, so it is
	// excluded from the score and reported separately.
	OutcomeNotBuilt Outcome = "not-built"
	// OutcomeNoCoverage means no indexed test executes the line, so there is
	// nothing to learn. Excluded from the score.
	OutcomeNoCoverage Outcome = "no-coverage"
)

func (o Outcome) Killed() bool {
	return o == OutcomeKilled || o == OutcomeTimeout
}

func (o Outcome) Scored() bool {
	return o.Killed() || o == OutcomeSurvived
}

// Mode records how a mutant was judged. Both modes must reach the same verdict;
// schemata is simply the one that does not pay for a compile per mutant.
type Mode string

const (
	ModeSchemata Mode = "schemata"
	ModeOverlay  Mode = "overlay"
)

type Result struct {
	Mutant   Mutant        `json:"mutant"`
	Outcome  Outcome       `json:"outcome"`
	Mode     Mode          `json:"mode,omitempty"`
	KilledBy string        `json:"killedBy,omitempty"`
	Detail   string        `json:"detail,omitempty"`
	Tests    int           `json:"tests"`
	Duration time.Duration `json:"durationNanos"`
}

type ExecuteOptions struct {
	Root       string
	Tags       []string
	Env        []string
	Jobs       int
	Budget     func(plan TestPlan) time.Duration
	MinTimeout time.Duration
	Progress   func(phase string, done, total int)

	// PackageDir resolves a test package's directory. Test binaries have to run
	// there, the way `go test` does, or a test that opens testdata/ fails for
	// reasons that have nothing to do with the mutant.
	PackageDir func(importPath string) string

	// NoSchemata forces every mutant to compile its own binary. Slow, and the only
	// way to cross-check the shared-binary path.
	NoSchemata bool
}

const (
	defaultMinTimeout  = 2 * time.Second
	timeoutMultiplier  = 3
	fallbackBudgetPer  = 5 * time.Second
	maxMutantBudgetCap = 5 * time.Minute
)

// Execute judges every mutant, sharing one compile across all the mutants of a
// package wherever the schemata rewrite is provably type-correct and falling back
// to a compile per mutant where it is not.
func Execute(ctx context.Context, mutants []Mutant, opts ExecuteOptions) ([]Result, error) {
	results := make([]Result, len(mutants))
	if len(mutants) == 0 {
		return results, nil
	}

	cores := runtime.GOMAXPROCS(0)
	jobs := opts.jobs(len(mutants))

	// Same trap as indexing: each `go test -c` fans out its own compile actions, so
	// unbounded inner parallelism multiplies with jobs and buries the machine.
	buildParallelism := max(1, cores/jobs)

	var batches []*schemataBatch
	direct := make([]int, 0, len(mutants))

	if opts.NoSchemata {
		for i := range mutants {
			direct = append(direct, i)
		}
	} else {
		batches, direct = planBatches(mutants)
	}

	ready, declined := prepareBatches(batches)
	defer func() {
		for _, batch := range ready {
			batch.cleanup()
		}
	}()

	direct = append(direct, declined...)
	sort.Ints(direct)

	progress := newTicker(opts.Progress, "mutants", len(mutants))

	judging := pool.New().
		WithMaxGoroutines(jobs).
		WithContext(ctx).
		WithFirstError().
		WithCancelOnError()

	for _, batch := range ready {
		for _, index := range batch.indices {
			judging.Go(func(ctx context.Context) error {
				if err := ctx.Err(); err != nil {
					return err
				}

				result, judged := batch.judge(ctx, index, mutants[index], opts, buildParallelism)
				if !judged {
					result = evaluate(ctx, mutants[index], opts, buildParallelism)
				}
				results[index] = result
				progress.tick()

				return nil
			})
		}
	}

	for _, index := range direct {
		judging.Go(func(ctx context.Context) error {
			if err := ctx.Err(); err != nil {
				return err
			}

			results[index] = evaluate(ctx, mutants[index], opts, buildParallelism)
			progress.tick()

			return nil
		})
	}

	if err := judging.Wait(); err != nil {
		return nil, err
	}

	return results, nil
}

// Modes counts how each mutant was judged, so a run can report how much of it took
// the fast path.
func Modes(results []Result) map[Mode]int {
	out := make(map[Mode]int, 2)
	for _, r := range results {
		if r.Mode != "" {
			out[r.Mode]++
		}
	}

	return out
}

func (o ExecuteOptions) jobs(work int) int {
	jobs := o.Jobs
	if jobs <= 0 {
		jobs = max(1, runtime.GOMAXPROCS(0)/2)
	}

	return min(jobs, max(1, work))
}

func (o ExecuteOptions) dirFor(importPath string) string {
	if o.PackageDir != nil {
		if dir := o.PackageDir(importPath); dir != "" {
			return dir
		}
	}

	return o.Root
}

// evaluate judges one mutant by compiling a binary that contains only that
// mutation. This is the path for candidates the schemata rewrite declined, and the
// reference implementation the shared-binary path is checked against.
func evaluate(ctx context.Context, m Mutant, opts ExecuteOptions, buildParallelism int) Result {
	started := time.Now()
	result := Result{Mutant: m, Tests: m.tests.count()}

	if m.tests.empty() {
		result.Outcome = OutcomeNoCoverage
		result.Detail = "no indexed test executes this line"
		result.Duration = time.Since(started)

		return result
	}

	result.Mode = ModeOverlay

	workdir, err := os.MkdirTemp("", "assay-mutant-")
	if err != nil {
		result.Outcome = OutcomeNotBuilt
		result.Detail = err.Error()
		result.Duration = time.Since(started)

		return result
	}
	defer os.RemoveAll(workdir)

	overlay, err := writeOverlay(workdir, m)
	if err != nil {
		result.Outcome = OutcomeNotBuilt
		result.Detail = err.Error()
		result.Duration = time.Since(started)

		return result
	}

	budget := opts.budgetFor(m.tests)

	for _, testPackage := range sortedKeys(m.tests) {
		tests := m.tests[testPackage]
		if len(tests) == 0 {
			continue
		}

		binary := filepath.Join(workdir, "bin", sanitisePath(testPackage)+".test")
		if mkErr := os.MkdirAll(filepath.Dir(binary), 0o755); mkErr != nil {
			result.Outcome = OutcomeNotBuilt
			result.Detail = mkErr.Error()
			result.Duration = time.Since(started)

			return result
		}

		if buildErr := buildTestBinary(ctx, buildRequest{
			root:             opts.Root,
			env:              opts.Env,
			tags:             opts.Tags,
			overlay:          overlay,
			testPackage:      testPackage,
			output:           binary,
			buildParallelism: buildParallelism,
		}); buildErr != nil {
			result.Outcome = OutcomeNotBuilt
			result.Detail = buildErr.Error()
			result.Duration = time.Since(started)

			return result
		}

		outcome, detail := runTests(ctx, testRun{
			binary: binary,
			dir:    opts.dirFor(testPackage),
			env:    opts.Env,
			budget: budget,
			tests:  tests,
		})
		if outcome.Killed() {
			result.Outcome = outcome
			result.KilledBy = testPackage
			result.Detail = detail
			result.Duration = time.Since(started)

			return result
		}
	}

	result.Outcome = OutcomeSurvived
	result.Duration = time.Since(started)

	return result
}

func (o ExecuteOptions) budgetFor(plan TestPlan) time.Duration {
	floor := o.MinTimeout
	if floor <= 0 {
		floor = defaultMinTimeout
	}

	var budget time.Duration
	if o.Budget != nil {
		budget = o.Budget(plan) * timeoutMultiplier
	} else {
		budget = time.Duration(plan.count()) * fallbackBudgetPer
	}

	return min(max(budget, floor), maxMutantBudgetCap)
}

func writeOverlay(workdir string, m Mutant) (string, error) {
	replacement, err := writeUnderBasename(workdir, "mutant", m.File, m.Source())
	if err != nil {
		return "", err
	}

	return writeOverlayFile(workdir, map[string]string{m.File: replacement})
}

type buildRequest struct {
	root             string
	env              []string
	tags             []string
	overlay          string
	testPackage      string
	output           string
	buildParallelism int
}

// buildTestBinary compiles a test binary with the overlay applied.
//
// It uses `go test -c` and then executes the binary rather than running `go test`
// directly. Overlay support for build operations is unambiguous, and this is the
// same path the indexer uses; `go test`'s own docs still claim overlays do not
// reach tests, which is stale but not worth betting on.
func buildTestBinary(ctx context.Context, req buildRequest) error {
	args := []string{
		"test", "-c",
		"-overlay=" + req.overlay,
		"-p", strconv.Itoa(req.buildParallelism),
		"-o", req.output,
	}
	if len(req.tags) > 0 {
		args = append(args, "-tags", strings.Join(req.tags, ","))
	}
	args = append(args, req.testPackage)

	if _, _, err := run(ctx, req.root, req.env, 0, "go", args...); err != nil {
		return fmt.Errorf("compile %s: %w", req.testPackage, err)
	}
	if _, err := os.Stat(req.output); err != nil {
		return fmt.Errorf("compiling %s produced no test binary", req.testPackage)
	}

	return nil
}

type testRun struct {
	binary string
	dir    string
	env    []string
	budget time.Duration
	tests  []string
}

func runTests(ctx context.Context, req testRun) (Outcome, string) {
	out, timedOut, err := run(ctx, req.dir, req.env, req.budget,
		req.binary, "-test.run", RunPattern(req.tests), "-test.count=1")

	switch {
	case timedOut:
		return OutcomeTimeout, fmt.Sprintf("exceeded %s", req.budget)
	case err != nil:
		return OutcomeKilled, firstFailure(out)
	default:
		return OutcomeSurvived, ""
	}
}

// mutantEnv selects a mutant inside a shared binary. The variable has to be added
// to a full environment rather than passed alone, because a test binary with a
// one-entry environment loses PATH, HOME and the whole toolchain configuration.
func mutantEnv(base []string, id int) []string {
	env := base
	if len(env) == 0 {
		env = os.Environ()
	}

	out := make([]string, 0, len(env)+1)
	out = append(out, env...)

	return append(out, SchemataEnvVar+"="+strconv.Itoa(id))
}

type ticker struct {
	report func(phase string, done, total int)
	phase  string
	total  int
	done   atomic.Int64
}

func newTicker(report func(phase string, done, total int), phase string, total int) *ticker {
	return &ticker{report: report, phase: phase, total: total}
}

func (t *ticker) tick() {
	if t == nil || t.report == nil {
		return
	}

	t.report(t.phase, int(t.done.Add(1)), t.total)
}

func firstFailure(out []byte) string {
	for line := range strings.SplitSeq(string(out), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--- FAIL:") {
			return trimmed
		}
		if strings.HasPrefix(trimmed, "panic:") {
			return trimmed
		}
	}

	return "test binary reported failure"
}

func RunPattern(tests []string) string {
	quoted := make([]string, 0, len(tests))
	for _, test := range tests {
		quoted = append(quoted, quoteMeta(test))
	}
	sort.Strings(quoted)

	return "^(" + strings.Join(quoted, "|") + ")$"
}

func quoteMeta(name string) string {
	var b strings.Builder
	for _, r := range name {
		if strings.ContainsRune(`\.+*?()|[]{}^$`, r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}

	return b.String()
}

func sortedKeys(plan TestPlan) []string {
	out := make([]string, 0, len(plan))
	for key := range plan {
		out = append(out, key)
	}
	sort.Strings(out)

	return out
}

func run(
	ctx context.Context,
	dir string,
	env []string,
	timeout time.Duration,
	name string,
	args ...string,
) ([]byte, bool, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	if len(env) > 0 {
		cmd.Env = env
	}
	proc.Isolate(cmd)

	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined

	err := cmd.Run()
	timedOut := errors.Is(ctx.Err(), context.DeadlineExceeded)

	if err != nil && !timedOut {
		return combined.Bytes(), false, fmt.Errorf("%w: %s", err, strings.TrimSpace(combined.String()))
	}

	return combined.Bytes(), timedOut, err
}
