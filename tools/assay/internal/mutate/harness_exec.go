package mutate

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/sourcegraph/conc/pool"

	"github.com/emoss08/assay/internal/gopkg"
	"github.com/emoss08/assay/internal/proc"
	"github.com/emoss08/assay/internal/tailbuf"
)

// harnessShardFloor is the fewest mutants worth a harness process of their own.
// Below it, splitting for parallelism spends more on spawn and package init than
// the concurrency returns.
const harnessShardFloor = 8

// limiter bounds the child processes running at once across every batch
// orchestrator. The orchestrators themselves are cheap goroutines that spend
// their lives waiting; the machine's cores are what the semaphore protects.
type limiter chan struct{}

func (l limiter) acquire(ctx context.Context) error {
	select {
	case l <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (l limiter) release() { <-l }

// runHarnessBatches judges every batch's mutants by switching mutants inside
// shared processes: one harness process per (mutated package × covering test
// package × shard) instead of one process per (mutant × covering test package).
func runHarnessBatches(
	ctx context.Context,
	mutants []Mutant,
	opts ExecuteOptions,
	batches []*schemataBatch,
	results []Result,
	buildParallelism int,
	progress *ticker,
) error {
	if len(batches) == 0 {
		return nil
	}

	sem := make(limiter, opts.jobs(len(mutants)))

	orchestrating := pool.New().
		WithContext(ctx).
		WithFirstError().
		WithCancelOnError()

	for _, batch := range batches {
		orchestrating.Go(func(ctx context.Context) error {
			return batch.judgeAll(ctx, mutants, opts, results, sem, buildParallelism, progress)
		})
	}

	return orchestrating.Wait()
}

// mutInstr is one harness instruction: run this mutant's covering tests in the
// current test package, under its budget.
type mutInstr struct {
	index   int
	id      int32
	pattern string
	budget  time.Duration
}

// mutVerdict is a judged instruction.
type mutVerdict struct {
	instr    mutInstr
	verdict  string
	detail   string
	duration time.Duration
}

// judgeAll walks the batch's covering test packages in order, judging every
// still-alive mutant against each. A mutant killed by one package never reaches
// the next — the same kill-fast ordering the per-mutant path uses — and a
// mutant alive after the last package survived.
func (b *schemataBatch) judgeAll(
	ctx context.Context,
	mutants []Mutant,
	opts ExecuteOptions,
	results []Result,
	sem limiter,
	buildParallelism int,
	progress *ticker,
) error {
	acc := make(map[int]*Result, len(b.indices))
	alive := make(map[int]bool, len(b.indices))
	for _, index := range b.indices {
		m := mutants[index]
		acc[index] = &Result{Mutant: m, Tests: m.tests.count(), Mode: ModeHarness}
		alive[index] = true
	}

	finalize := func(index int) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		results[index] = *acc[index]
		delete(alive, index)
		progress.tick()

		return nil
	}

	for _, testPkg := range b.orderedTestPkgs(mutants, opts) {
		candidates := make([]int, 0, len(b.indices))
		for _, index := range b.indices {
			if alive[index] && len(mutants[index].tests[testPkg]) > 0 {
				candidates = append(candidates, index)
			}
		}
		if len(candidates) == 0 {
			continue
		}

		verdicts, err := b.judgePackage(ctx, judgePackageRequest{
			mutants:          mutants,
			opts:             opts,
			testPkg:          testPkg,
			candidates:       candidates,
			sem:              sem,
			buildParallelism: buildParallelism,
		})
		if err != nil {
			return err
		}

		for _, index := range candidates {
			v, ok := verdicts[index]
			if !ok {
				return fmt.Errorf("no verdict for mutant %s in %s", mutants[index].ID, testPkg)
			}

			out := acc[index]
			out.Duration += v.duration
			if v.fallback != "" && out.Fallback == "" {
				out.Fallback = v.fallback
			}
			if v.mode != "" {
				out.Mode = v.mode
			}

			switch {
			case v.evaluated:
				*out = v.result
				if finalizeErr := finalize(index); finalizeErr != nil {
					return finalizeErr
				}
			case v.outcome.Killed():
				out.Outcome = v.outcome
				out.KilledBy = testPkg
				out.Detail = v.detail
				if finalizeErr := finalize(index); finalizeErr != nil {
					return finalizeErr
				}
			}
		}
	}

	for _, index := range b.indices {
		if !alive[index] {
			continue
		}
		acc[index].Outcome = OutcomeSurvived
		if err := finalize(index); err != nil {
			return err
		}
	}

	return nil
}

// orderedTestPkgs resolves the batch-level judging order from the union of its
// mutants' plans, so the batch walks its covering packages in the same
// cost-informed order the per-mutant paths use.
func (b *schemataBatch) orderedTestPkgs(mutants []Mutant, opts ExecuteOptions) []string {
	if opts.PackageOrder == nil {
		return b.testPkgs
	}

	sets := make(map[string]map[string]struct{}, len(b.testPkgs))
	for _, index := range b.indices {
		for pkg, tests := range mutants[index].tests {
			if len(tests) == 0 {
				continue
			}
			set := sets[pkg]
			if set == nil {
				set = make(map[string]struct{})
				sets[pkg] = set
			}
			for _, test := range tests {
				set[test] = struct{}{}
			}
		}
	}

	merged := make(TestPlan, len(sets))
	for pkg, set := range sets {
		merged[pkg] = setToSorted(set)
	}

	return opts.orderedPackages(merged)
}

// tpVerdict is one mutant's judgement against one test package.
type tpVerdict struct {
	outcome   Outcome
	detail    string
	duration  time.Duration
	mode      Mode
	fallback  string
	evaluated bool
	result    Result
}

type judgePackageRequest struct {
	mutants          []Mutant
	opts             ExecuteOptions
	testPkg          string
	candidates       []int
	sem              limiter
	buildParallelism int
}

// judgePackage judges the candidates against one test package. The harness path
// is tried first; mutants it cannot judge — an init-executed site, a test
// package with its own TestMain, a harness that will not build or run — fall
// back to an env-selected process of their own on the same shared binary, and
// only a binary that cannot be built at all costs a full per-mutant evaluate.
func (b *schemataBatch) judgePackage(
	ctx context.Context,
	req judgePackageRequest,
) (map[int]tpVerdict, error) {
	opts := req.opts
	dir := opts.dirFor(req.testPkg)
	verdicts := make(map[int]tpVerdict, len(req.candidates))

	binary, harnessed, reason, err := b.binaryForPackage(ctx, opts, req.testPkg, dir, req.buildParallelism, req.sem)
	if err != nil {
		return nil, err
	}
	if binary == "" {
		// Neither flavour of the shared binary exists; every candidate pays for a
		// binary of its own, exactly like the pre-harness fallback path.
		return b.evaluateAll(ctx, req, reason)
	}

	fallback := req.candidates
	if harnessed {
		var judged map[int]tpVerdict
		judged, fallback, err = b.judgeViaHarness(ctx, req, binary, dir)
		if err != nil {
			return nil, err
		}
		for index, v := range judged {
			verdicts[index] = v
		}
	}

	if len(fallback) == 0 {
		return verdicts, nil
	}

	judging := pool.New().WithContext(ctx).WithFirstError().WithCancelOnError()
	envVerdicts := make([]tpVerdict, len(fallback))
	for i, index := range fallback {
		judging.Go(func(ctx context.Context) error {
			v, envErr := b.envJudge(ctx, req, binary, dir, index)
			if envErr != nil {
				return envErr
			}
			envVerdicts[i] = v

			return nil
		})
	}
	if waitErr := judging.Wait(); waitErr != nil {
		return nil, waitErr
	}
	for i, index := range fallback {
		v := envVerdicts[i]
		// A structural decline — a package with its own TestMain, a site that
		// executes during init — is by-design routing and stays quiet; only a
		// harness that failed carries a reason worth a warning.
		if prior, ok := verdicts[index]; ok && prior.fallback != "" {
			v.fallback = prior.fallback
		} else {
			v.fallback = reason
		}
		verdicts[index] = v
	}

	return verdicts, nil
}

// binaryForPackage picks the shared binary flavour this test package can host.
// A package that can take the harness gets the harness build — which behaves
// exactly like a plain test binary when the plan variable is unset, so the same
// binary also serves every env-selected fallback run. A package that cannot
// host it gets the plain shared build.
func (b *schemataBatch) binaryForPackage(
	ctx context.Context,
	opts ExecuteOptions,
	testPkg, dir string,
	buildParallelism int,
	sem limiter,
) (binary string, harnessed bool, reason string, err error) {
	capable, reason := b.harnessCapability(testPkg, dir)

	if err := sem.acquire(ctx); err != nil {
		return "", false, "", err
	}
	defer sem.release()

	if capable {
		binary, buildErr := b.harnessBinaryFor(ctx, opts, testPkg, dir, buildParallelism)
		if buildErr == nil {
			return binary, true, "", nil
		}
		if ctx.Err() != nil {
			return "", false, "", ctx.Err()
		}
		reason = "mutant harness build failed: " + buildErr.Error()
	}

	binary, plainErr := b.binaryFor(ctx, opts, testPkg, buildParallelism)
	if plainErr != nil {
		if ctx.Err() != nil {
			return "", false, "", ctx.Err()
		}

		return "", false, reason + "; shared binary unavailable: " + plainErr.Error(), nil
	}

	return binary, false, reason, nil
}

// harnessCapability reports whether the harness TestMain can be injected into
// this test package. The reason is empty for structural declines — a package
// with its own TestMain is a fact, not a failure — and non-empty only for
// problems worth surfacing.
func (b *schemataBatch) harnessCapability(testPkg, dir string) (bool, string) {
	hasTestMain, err := gopkg.DefinesTestMain(dir)
	if err != nil {
		return false, "cannot inspect " + testPkg + ": " + err.Error()
	}
	if hasTestMain {
		return false, ""
	}
	if _, statErr := os.Stat(filepath.Join(dir, MutHarnessFileName)); statErr == nil {
		return false, ""
	}
	if _, nameErr := gopkg.Name(dir); nameErr != nil {
		return false, "cannot resolve the package name for " + testPkg + ": " + nameErr.Error()
	}

	return true, ""
}

// judgeViaHarness shards the candidates and runs each shard through harness
// processes. It returns the verdicts it reached and the candidates it could
// not judge, which the caller sends down the env-selected path.
func (b *schemataBatch) judgeViaHarness(
	ctx context.Context,
	req judgePackageRequest,
	binary, dir string,
) (map[int]tpVerdict, []int, error) {
	instrs := make([]mutInstr, 0, len(req.candidates))
	for _, index := range req.candidates {
		m := req.mutants[index]
		instrs = append(instrs, mutInstr{
			index:   index,
			id:      int32(b.ids[index]),
			pattern: RunPattern(m.tests[req.testPkg]),
			budget:  req.opts.PackageBudget(m.tests, req.testPkg),
		})
	}

	shards := shardInstructions(instrs, cap(req.sem))

	type shardOutcome struct {
		verdicts []mutVerdict
		fallback []mutInstr
		reason   string
	}
	outcomes := make([]shardOutcome, len(shards))

	running := pool.New().WithContext(ctx).WithFirstError().WithCancelOnError()
	for i, shard := range shards {
		running.Go(func(ctx context.Context) error {
			verdicts, unjudged, reason, shardErr := b.judgeShard(ctx, req.opts, binary, dir, shard, req.sem)
			if shardErr != nil {
				return shardErr
			}
			outcomes[i] = shardOutcome{verdicts: verdicts, fallback: unjudged, reason: reason}

			return nil
		})
	}
	if err := running.Wait(); err != nil {
		return nil, nil, err
	}

	judged := make(map[int]tpVerdict, len(instrs))
	var fallback []int
	for _, outcome := range outcomes {
		for _, v := range outcome.verdicts {
			switch v.verdict {
			case verdictKilled:
				judged[v.instr.index] = tpVerdict{
					outcome:  OutcomeKilled,
					detail:   v.detail,
					duration: v.duration,
					mode:     ModeHarness,
				}
			case verdictTimeout:
				judged[v.instr.index] = tpVerdict{
					outcome:  OutcomeTimeout,
					detail:   v.detail,
					duration: v.duration,
					mode:     ModeHarness,
				}
			case verdictSurvived:
				judged[v.instr.index] = tpVerdict{
					outcome:  OutcomeSurvived,
					duration: v.duration,
					mode:     ModeHarness,
				}
			case verdictTainted:
				judged[v.instr.index] = tpVerdict{}
				fallback = append(fallback, v.instr.index)
			}
		}
		for _, instr := range outcome.fallback {
			judged[instr.index] = tpVerdict{fallback: outcome.reason}
			fallback = append(fallback, instr.index)
		}
	}

	return judged, fallback, nil
}

// verdictTimeout is executor-internal: the harness never prints it, the rolling
// deadline infers it.
const verdictTimeout = "timeout"

// judgeShard runs one shard's instructions, restarting the harness process as
// mutants kill it. A mutant that crashes or hangs the process is judged by that
// death; the rest continue in a fresh process. A fresh process that dies before
// announcing its first instruction is a broken harness, and the remaining
// instructions are handed back for the env-selected path rather than blamed on
// a mutant.
func (b *schemataBatch) judgeShard(
	ctx context.Context,
	opts ExecuteOptions,
	binary, dir string,
	instrs []mutInstr,
	sem limiter,
) ([]mutVerdict, []mutInstr, string, error) {
	verdicts := make([]mutVerdict, 0, len(instrs))
	remaining := instrs

	for len(remaining) > 0 {
		if err := sem.acquire(ctx); err != nil {
			return nil, nil, "", err
		}
		run := b.runHarnessOnce(ctx, opts, binary, dir, remaining)
		sem.release()

		if err := ctx.Err(); err != nil {
			return nil, nil, "", err
		}

		verdicts = append(verdicts, run.verdicts...)
		remaining = remaining[len(run.verdicts):]
		if len(remaining) == 0 {
			break
		}

		inflight := remaining[0]
		switch {
		case run.timedOut && run.began:
			verdicts = append(verdicts, mutVerdict{
				instr:   inflight,
				verdict: verdictTimeout,
				detail:  fmt.Sprintf("exceeded %s", inflight.budget),
			})
			remaining = remaining[1:]
		case run.began:
			detail := run.detail
			if detail == "" {
				detail = "test binary reported failure"
			}
			verdicts = append(verdicts, mutVerdict{
				instr:   inflight,
				verdict: verdictKilled,
				detail:  detail,
			})
			remaining = remaining[1:]
		default:
			reason := "mutant harness stopped before its first verdict"
			if run.detail != "" {
				reason += ": " + run.detail
			}

			return verdicts, remaining, reason, nil
		}
	}

	return verdicts, nil, "", nil
}

type harnessRun struct {
	verdicts []mutVerdict
	began    bool
	timedOut bool
	detail   string
}

// runHarnessOnce spawns one harness process over the instructions and follows
// its progress. The deadline is rolling and per-instruction: each begin or
// verdict line resets it to the instruction now in flight, so one hanging
// mutant cannot borrow the budget of the mutants behind it.
func (b *schemataBatch) runHarnessOnce(
	ctx context.Context,
	opts ExecuteOptions,
	binary, dir string,
	instrs []mutInstr,
) harnessRun {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	planPath, err := writeMutPlan(b.workdir, instrs)
	if err != nil {
		return harnessRun{detail: err.Error()}
	}
	defer os.Remove(planPath)

	env := opts.Env
	if len(env) == 0 {
		env = os.Environ()
	}
	cmd := exec.CommandContext(ctx, binary)
	cmd.Dir = dir
	cmd.Env = append(append([]string(nil), env...),
		SchemataEnvVar+"=",
		PlanEnvVar+"="+planPath,
		TrackEnvVar+"=1",
	)
	proc.Isolate(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return harnessRun{detail: "open harness stdout: " + err.Error()}
	}
	stderr := tailbuf.New(32 << 10)
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		return harnessRun{detail: "start harness: " + err.Error()}
	}

	var timedOut atomic.Bool
	deadline := time.AfterFunc(instrs[0].budget, func() {
		timedOut.Store(true)
		cancel()
	})
	// The timer must outlive Wait: if the reader dies first, the child can fill
	// the pipe and block forever, and this deadline is the only killer left.
	defer deadline.Stop()

	run := harnessRun{}
	next := 0
	firstFail := ""

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64<<10), 1<<20)
	for scanner.Scan() {
		line := scanner.Text()
		if next >= len(instrs) {
			continue
		}
		if firstFail == "" {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "--- FAIL:") || strings.HasPrefix(trimmed, "panic:") {
				firstFail = trimmed
			}
		}

		event, ok := parseMutHarnessLine(line, instrs[next].id)
		if !ok {
			continue
		}
		switch event.kind {
		case mutHarnessBegin:
			run.began = true
			deadline.Reset(instrs[next].budget)
		case mutHarnessVerdict:
			detail := ""
			if event.verdict == verdictKilled {
				detail = firstFail
				if detail == "" {
					detail = "test binary reported failure"
				}
			}
			run.verdicts = append(run.verdicts, mutVerdict{
				instr:    instrs[next],
				verdict:  event.verdict,
				detail:   detail,
				duration: event.duration,
			})
			next++
			run.began = false
			firstFail = ""
			if next < len(instrs) {
				deadline.Reset(instrs[next].budget)
			}
		}
	}

	scanErr := scanner.Err()
	if scanErr != nil {
		cancel()
		_, _ = io.Copy(io.Discard, stdout)
	}

	waitErr := cmd.Wait()

	run.timedOut = timedOut.Load()
	if next < len(instrs) {
		run.detail = firstFail
		if run.detail == "" {
			run.detail = strings.TrimSpace(stderr.String())
		}
		if run.detail == "" && scanErr != nil {
			run.detail = "harness output unreadable: " + scanErr.Error()
		}
		if run.detail == "" && waitErr != nil {
			run.detail = waitErr.Error()
		}
	}

	return run
}

// envJudge runs one mutant against one test package in a process of its own,
// selected by the environment — full per-process semantics, including init
// running under the mutant, on a binary that is already built.
func (b *schemataBatch) envJudge(
	ctx context.Context,
	req judgePackageRequest,
	binary, dir string,
	index int,
) (tpVerdict, error) {
	if err := req.sem.acquire(ctx); err != nil {
		return tpVerdict{}, err
	}
	defer req.sem.release()

	m := req.mutants[index]
	started := time.Now()
	outcome, detail := runTests(ctx, testRun{
		binary: binary,
		dir:    dir,
		env:    mutantEnv(req.opts.Env, b.ids[index]),
		budget: req.opts.PackageBudget(m.tests, req.testPkg),
		tests:  m.tests[req.testPkg],
	})
	if err := ctx.Err(); err != nil {
		return tpVerdict{}, err
	}

	return tpVerdict{
		outcome:  outcome,
		detail:   detail,
		duration: time.Since(started),
		mode:     ModeSchemata,
	}, nil
}

// evaluateAll is the last resort for one test package: no shared binary exists,
// so every candidate is judged by evaluate, which compiles a binary of its own
// and re-runs the mutant's whole plan.
func (b *schemataBatch) evaluateAll(
	ctx context.Context,
	req judgePackageRequest,
	reason string,
) (map[int]tpVerdict, error) {
	verdicts := make(map[int]tpVerdict, len(req.candidates))
	evaluated := make([]Result, len(req.candidates))

	judging := pool.New().WithContext(ctx).WithFirstError().WithCancelOnError()
	for i, index := range req.candidates {
		judging.Go(func(ctx context.Context) error {
			if err := req.sem.acquire(ctx); err != nil {
				return err
			}
			defer req.sem.release()

			result := evaluate(ctx, req.mutants[index], req.opts, req.buildParallelism)
			if err := ctx.Err(); err != nil {
				return err
			}
			result.Fallback = reason
			evaluated[i] = result

			return nil
		})
	}
	if err := judging.Wait(); err != nil {
		return nil, err
	}

	for i, index := range req.candidates {
		verdicts[index] = tpVerdict{evaluated: true, result: evaluated[i]}
	}

	return verdicts, nil
}

func shardInstructions(instrs []mutInstr, maxShards int) [][]mutInstr {
	shards := min(maxShards, (len(instrs)+harnessShardFloor-1)/harnessShardFloor)
	if shards < 1 {
		shards = 1
	}

	out := make([][]mutInstr, 0, shards)
	size := (len(instrs) + shards - 1) / shards
	for start := 0; start < len(instrs); start += size {
		out = append(out, instrs[start:min(start+size, len(instrs))])
	}

	return out
}

func writeMutPlan(workdir string, instrs []mutInstr) (string, error) {
	var sb strings.Builder
	for _, instr := range instrs {
		sb.WriteString(strconv.Itoa(int(instr.id)))
		sb.WriteByte('\t')
		sb.WriteString(instr.pattern)
		sb.WriteByte('\n')
	}

	file, err := os.CreateTemp(workdir, "plan-*.txt")
	if err != nil {
		return "", fmt.Errorf("write harness plan: %w", err)
	}
	if _, err := file.WriteString(sb.String()); err != nil {
		file.Close()
		os.Remove(file.Name())

		return "", fmt.Errorf("write harness plan: %w", err)
	}
	if err := file.Close(); err != nil {
		os.Remove(file.Name())

		return "", fmt.Errorf("write harness plan: %w", err)
	}

	return file.Name(), nil
}
