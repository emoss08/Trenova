package mutate

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/sourcegraph/conc/pool"

	"github.com/emoss08/assay/internal/gopkg"
	"github.com/emoss08/assay/internal/proc"
	"github.com/emoss08/assay/internal/tailbuf"
)

// harnessShardFloor is the fewest mutants worth a harness walker of their own.
// Below it, splitting for parallelism spends more on spawn and package init
// than the concurrency returns.
const harnessShardFloor = 8

// limiter bounds the child processes running at once across every shard
// walker. The walkers themselves are cheap goroutines that spend their lives
// waiting; the machine's cores are what the semaphore protects.
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
//
// Each batch is split into stable shards that walk the covering test packages
// independently — there is deliberately no barrier between shards or between
// packages. A whole-batch walk that synchronised on each test package idled a
// core whenever one shard's mutants were slower than another's, and measured
// slower than the per-mutant path it replaced; independent walkers keep every
// worker busy end to end, the same shape the per-mutant pool has.
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

	walking := pool.New().
		WithContext(ctx).
		WithFirstError().
		WithCancelOnError()

	for _, batch := range batches {
		ordered := batch.orderedTestPkgs(mutants, opts)
		for _, shard := range shardIndices(batch.indices, cap(sem), func(index int) time.Duration {
			return opts.budgetFor(mutants[index].tests)
		}) {
			walking.Go(func(ctx context.Context) error {
				return batch.walkShard(ctx, shardWalk{
					mutants:          mutants,
					opts:             opts,
					results:          results,
					indices:          shard,
					orderedPkgs:      ordered,
					sem:              sem,
					buildParallelism: buildParallelism,
					progress:         progress,
				})
			})
		}
	}

	return walking.Wait()
}

// shardIndices splits a batch's mutants into at most maxShards stable groups,
// balanced by estimated cost: heaviest mutant first, each into the lightest
// shard so far. Blind splits lost the parallelism the shards exist for — a
// survivor's cost is its whole covering plan, the top few survivors carry most
// of a package's judging time, and whichever walker drew them ran long after
// the others went idle. The estimate is the plan-derived budget: exactly right
// for survivors, and irrelevant for kills, which are cheap in any shard.
func shardIndices(indices []int, maxShards int, cost func(index int) time.Duration) [][]int {
	shards := min(maxShards, max(1, (len(indices)+harnessShardFloor-1)/harnessShardFloor))

	byCost := make([]int, len(indices))
	copy(byCost, indices)
	sort.SliceStable(byCost, func(i, j int) bool {
		return cost(byCost[i]) > cost(byCost[j])
	})

	out := make([][]int, shards)
	loads := make([]time.Duration, shards)
	for _, index := range byCost {
		lightest := 0
		for shard := 1; shard < shards; shard++ {
			if loads[shard] < loads[lightest] {
				lightest = shard
			}
		}
		out[lightest] = append(out[lightest], index)
		loads[lightest] += cost(index)
	}

	// Instruction order inside a shard follows source order, like every other
	// judging path; the cost ordering above only decides membership.
	for _, shard := range out {
		sort.Ints(shard)
	}

	return out
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

// verdictTimeout is executor-internal: the harness never prints it, the rolling
// deadline infers it.
const verdictTimeout = "timeout"

type shardWalk struct {
	mutants          []Mutant
	opts             ExecuteOptions
	results          []Result
	indices          []int
	orderedPkgs      []string
	sem              limiter
	buildParallelism int
	progress         *ticker
}

// walkShard judges this shard's mutants against every covering test package in
// order. A mutant killed by one package never reaches the next — the same
// kill-fast ordering the per-mutant path uses — and a mutant alive after the
// last package survived. The walker runs one child process at a time, so
// overall concurrency is the number of walkers, bounded by the semaphore.
func (b *schemataBatch) walkShard(ctx context.Context, w shardWalk) error {
	acc := make(map[int]*Result, len(w.indices))
	alive := make(map[int]bool, len(w.indices))
	for _, index := range w.indices {
		m := w.mutants[index]
		acc[index] = &Result{Mutant: m, Tests: m.tests.count(), Mode: ModeHarness}
		alive[index] = true
	}

	finalize := func(index int) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		w.results[index] = *acc[index]
		delete(alive, index)
		w.progress.tick()

		return nil
	}

	for _, testPkg := range w.orderedPkgs {
		candidates := make([]int, 0, len(w.indices))
		for _, index := range w.indices {
			if alive[index] && len(w.mutants[index].tests[testPkg]) > 0 {
				candidates = append(candidates, index)
			}
		}
		if len(candidates) == 0 {
			continue
		}

		dir := w.opts.dirFor(testPkg)

		binary, harnessed, reason, err := b.binaryForPackage(
			ctx, w.opts, testPkg, dir, w.buildParallelism, w.sem)
		if err != nil {
			return err
		}

		if binary == "" {
			// Neither flavour of the shared binary exists; every candidate pays
			// for a binary of its own, exactly like the pre-harness fallback path.
			for _, index := range candidates {
				if semErr := w.sem.acquire(ctx); semErr != nil {
					return semErr
				}
				result := evaluate(ctx, w.mutants[index], w.opts, w.buildParallelism)
				w.sem.release()
				if ctxErr := ctx.Err(); ctxErr != nil {
					return ctxErr
				}
				result.Fallback = reason
				*acc[index] = result
				if finErr := finalize(index); finErr != nil {
					return finErr
				}
			}

			continue
		}

		verdicts := make(map[int]tpVerdict, len(candidates))
		fallback := candidates
		if harnessed {
			var judged map[int]tpVerdict
			var judgeErr error
			judged, fallback, judgeErr = b.judgeViaHarness(ctx, w, testPkg, binary, dir, candidates)
			if judgeErr != nil {
				return judgeErr
			}
			for index, v := range judged {
				verdicts[index] = v
			}
		}

		for _, index := range fallback {
			v, envErr := b.envJudge(ctx, w, testPkg, binary, dir, index)
			if envErr != nil {
				return envErr
			}
			// A structural decline — a package with its own TestMain, a site
			// that executes during init — is by-design routing and stays quiet;
			// only a harness that failed carries a reason worth a warning.
			if prior, ok := verdicts[index]; ok && prior.fallback != "" {
				v.fallback = prior.fallback
			} else {
				v.fallback = reason
			}
			verdicts[index] = v
		}

		for _, index := range candidates {
			v, ok := verdicts[index]
			if !ok {
				return fmt.Errorf("no verdict for mutant %s in %s", w.mutants[index].ID, testPkg)
			}

			out := acc[index]
			out.Duration += v.duration
			if v.fallback != "" && out.Fallback == "" {
				out.Fallback = v.fallback
			}
			if v.mode != "" {
				out.Mode = v.mode
			}

			if v.outcome.Killed() {
				out.Outcome = v.outcome
				out.KilledBy = testPkg
				out.Detail = v.detail
				if finErr := finalize(index); finErr != nil {
					return finErr
				}
			}
		}
	}

	for _, index := range w.indices {
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

// tpVerdict is one mutant's judgement against one test package.
type tpVerdict struct {
	outcome  Outcome
	detail   string
	duration time.Duration
	mode     Mode
	fallback string
}

// binaryForPackage picks the shared binary flavour this test package can host.
// A package that can take the harness gets the harness build — which behaves
// exactly like a plain test binary when the plan variable is unset, so the same
// binary also serves every env-selected fallback run. A package that cannot
// host it gets the plain shared build. Builds happen once per (batch, test
// package); concurrent walkers wait on the slot rather than compiling twice.
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

// judgeViaHarness runs the candidates through harness processes. It returns
// the verdicts it reached and the candidates it could not judge, which the
// caller sends down the env-selected path.
func (b *schemataBatch) judgeViaHarness(
	ctx context.Context,
	w shardWalk,
	testPkg, binary, dir string,
	candidates []int,
) (map[int]tpVerdict, []int, error) {
	instrs := make([]mutInstr, 0, len(candidates))
	for _, index := range candidates {
		m := w.mutants[index]
		instrs = append(instrs, mutInstr{
			index:   index,
			id:      int32(b.ids[index]),
			pattern: RunPattern(m.tests[testPkg]),
			budget:  w.opts.PackageBudget(m.tests, testPkg),
		})
	}

	verdicts, unjudged, reason, err := b.judgeShard(ctx, w.opts, binary, dir, instrs, w.sem)
	if err != nil {
		return nil, nil, err
	}

	judged := make(map[int]tpVerdict, len(instrs))
	var fallback []int
	for _, v := range verdicts {
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
			fallback = append(fallback, v.instr.index)
		}
	}
	for _, instr := range unjudged {
		judged[instr.index] = tpVerdict{fallback: reason}
		fallback = append(fallback, instr.index)
	}

	return judged, fallback, nil
}

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
	w shardWalk,
	testPkg, binary, dir string,
	index int,
) (tpVerdict, error) {
	if err := w.sem.acquire(ctx); err != nil {
		return tpVerdict{}, err
	}
	defer w.sem.release()

	m := w.mutants[index]
	started := time.Now()
	outcome, detail := runTests(ctx, testRun{
		binary: binary,
		dir:    dir,
		env:    mutantEnv(w.opts.Env, b.ids[index]),
		budget: w.opts.PackageBudget(m.tests, testPkg),
		tests:  m.tests[testPkg],
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

// orderedTestPkgs resolves the batch-level judging order from the union of its
// mutants' plans, so every shard walks the covering packages in the same
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
