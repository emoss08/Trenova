package mutate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

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

type Result struct {
	Mutant   Mutant        `json:"mutant"`
	Outcome  Outcome       `json:"outcome"`
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
	Budget     func(plan testPlan) time.Duration
	MinTimeout time.Duration
	Progress   func(done, total int)
}

const (
	defaultMinTimeout  = 2 * time.Second
	timeoutMultiplier  = 3
	fallbackBudgetPer  = 5 * time.Second
	maxMutantBudgetCap = 5 * time.Minute
)

func Execute(ctx context.Context, mutants []Mutant, opts ExecuteOptions) ([]Result, error) {
	cores := runtime.GOMAXPROCS(0)

	jobs := opts.Jobs
	if jobs <= 0 {
		jobs = max(1, cores/2)
	}
	jobs = min(jobs, max(1, len(mutants)))

	// Same trap as indexing: each `go test -c` fans out its own compile actions, so
	// unbounded inner parallelism multiplies with jobs and buries the machine.
	buildParallelism := max(1, cores/jobs)

	results := make([]Result, len(mutants))
	next := make(chan int)

	var mu sync.Mutex
	var done int

	var wg sync.WaitGroup
	for range jobs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range next {
				results[i] = evaluate(ctx, mutants[i], opts, buildParallelism)

				mu.Lock()
				done++
				current := done
				mu.Unlock()

				if opts.Progress != nil {
					opts.Progress(current, len(mutants))
				}
			}
		}()
	}

	for i := range mutants {
		select {
		case <-ctx.Done():
			close(next)
			wg.Wait()

			return nil, ctx.Err()
		case next <- i:
		}
	}
	close(next)
	wg.Wait()

	return results, nil
}

func evaluate(ctx context.Context, m Mutant, opts ExecuteOptions, buildParallelism int) Result {
	started := time.Now()
	result := Result{Mutant: m, Tests: m.tests.count()}

	if m.tests.empty() {
		result.Outcome = OutcomeNoCoverage
		result.Detail = "no indexed test executes this line"
		result.Duration = time.Since(started)

		return result
	}

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

		outcome, detail := runAgainst(ctx, runRequest{
			root:             opts.Root,
			env:              opts.Env,
			tags:             opts.Tags,
			workdir:          workdir,
			overlay:          overlay,
			testPackage:      testPackage,
			tests:            tests,
			budget:           budget,
			buildParallelism: buildParallelism,
		})

		switch outcome {
		case OutcomeKilled, OutcomeTimeout:
			result.Outcome = outcome
			result.KilledBy = testPackage
			result.Detail = detail
			result.Duration = time.Since(started)

			return result
		case OutcomeNotBuilt:
			result.Outcome = OutcomeNotBuilt
			result.Detail = detail
			result.Duration = time.Since(started)

			return result
		}
	}

	result.Outcome = OutcomeSurvived
	result.Duration = time.Since(started)

	return result
}

func (o ExecuteOptions) budgetFor(plan testPlan) time.Duration {
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
	replacement := filepath.Join(workdir, "mutant.go")
	if err := os.WriteFile(replacement, m.Source(), 0o644); err != nil {
		return "", fmt.Errorf("write mutant source: %w", err)
	}

	payload, err := json.Marshal(map[string]map[string]string{
		"Replace": {m.File: replacement},
	})
	if err != nil {
		return "", fmt.Errorf("encode overlay: %w", err)
	}

	overlay := filepath.Join(workdir, "overlay.json")
	if err := os.WriteFile(overlay, payload, 0o644); err != nil {
		return "", fmt.Errorf("write overlay: %w", err)
	}

	return overlay, nil
}

type runRequest struct {
	root             string
	env              []string
	tags             []string
	workdir          string
	overlay          string
	testPackage      string
	tests            []string
	budget           time.Duration
	buildParallelism int
}

// runAgainst builds the mutated test binary and runs only the covering tests.
//
// It compiles with `go test -c` and executes the binary rather than using
// `go test` directly. Overlay support for build operations is unambiguous, and
// this is the same path the indexer uses; `go test`'s own docs still claim
// overlays do not reach tests, which is stale but not worth betting on.
func runAgainst(ctx context.Context, req runRequest) (Outcome, string) {
	binary := filepath.Join(req.workdir, "mutant.test")

	buildArgs := []string{
		"test", "-c",
		"-overlay=" + req.overlay,
		"-p", strconv.Itoa(req.buildParallelism),
		"-o", binary,
	}
	if len(req.tags) > 0 {
		buildArgs = append(buildArgs, "-tags", strings.Join(req.tags, ","))
	}
	buildArgs = append(buildArgs, req.testPackage)

	if _, _, err := run(ctx, req.root, req.env, 0, "go", buildArgs...); err != nil {
		return OutcomeNotBuilt, "compile mutant: " + err.Error()
	}
	if _, err := os.Stat(binary); err != nil {
		return OutcomeNotBuilt, "mutant produced no test binary"
	}

	out, timedOut, err := run(ctx, req.root, req.env, req.budget,
		binary, "-test.run", RunPattern(req.tests), "-test.count=1")

	switch {
	case timedOut:
		return OutcomeTimeout, fmt.Sprintf("exceeded %s", req.budget)
	case err != nil:
		return OutcomeKilled, firstFailure(out)
	default:
		return OutcomeSurvived, ""
	}
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

func sortedKeys(plan testPlan) []string {
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
