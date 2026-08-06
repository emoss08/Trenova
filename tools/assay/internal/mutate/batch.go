package mutate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/emoss08/assay/internal/gopkg"
	"github.com/emoss08/assay/internal/overlay"
)

// schemataBatch is every schema-capable mutant in one package, compiled together.
//
// All its sites live in one binary per test package, so N mutants cost one compile
// instead of N. Those binaries are built on first use rather than up front: a
// mutant stops at the first test package that kills it, so eagerly compiling every
// package a batch might reach costs more than it saves whenever kills are common —
// which, on a diff-scoped run, is nearly always.
type schemataBatch struct {
	pkg     string
	pkgName string
	dir     string

	indices  []int
	ids      map[int]int
	sites    map[string][]*schemaSite
	sources  map[string][]byte
	helpers  []string
	testPkgs []string

	workdir string
	overlay string
	replace map[string]string

	binaries map[string]*batchBinary
	harness  map[string]*batchBinary
}

// batchBinary is one test package's shared binary. The mutex means concurrent
// mutants needing the same binary compile it exactly once and the rest wait for
// it. A plain sync.Once would also do that, but it would cache a
// cancellation-time failure forever — and a slot poisoned by Ctrl-C is
// indistinguishable from a real compile failure.
type batchBinary struct {
	mu   sync.Mutex
	done bool
	path string
	err  error
}

// planBatches splits mutants into batches that can share a binary and the ones
// that cannot. A mutant with no covering test is left to the direct path, which
// reports it without building anything.
func planBatches(mutants []Mutant) ([]*schemataBatch, []int) {
	byPackage := make(map[string]*schemataBatch)
	direct := make([]int, 0, len(mutants))
	var order []string

	for i := range mutants {
		m := mutants[i]
		if m.schema == nil || m.tests.empty() {
			direct = append(direct, i)

			continue
		}

		batch := byPackage[m.Package]
		if batch == nil {
			batch = &schemataBatch{
				pkg:     m.Package,
				pkgName: m.packageName,
				dir:     filepath.Dir(m.File),
				ids:     make(map[int]int),
				sites:   make(map[string][]*schemaSite),
				sources: make(map[string][]byte),
			}
			byPackage[m.Package] = batch
			order = append(order, m.Package)
		}
		batch.indices = append(batch.indices, i)
	}
	sort.Strings(order)

	batches := make([]*schemataBatch, 0, len(order))
	for _, pkg := range order {
		batch := byPackage[pkg]
		batch.assign(mutants)
		batches = append(batches, batch)
	}

	return batches, direct
}

// assign hands every mutant in the batch the runtime id that selects it. Ids start
// at one because zero means "no mutant", which is what leaves the shared binary
// behaving exactly like the original.
func (b *schemataBatch) assign(mutants []Mutant) {
	helpers := make(map[string]struct{})
	testPkgs := make(map[string]struct{})

	for position, index := range b.indices {
		m := mutants[index]
		id := position + 1
		b.ids[index] = id

		b.sites[m.File] = append(b.sites[m.File], &schemaSite{form: m.schema, id: id})
		b.sources[m.File] = m.source

		for _, helper := range m.schema.helpers {
			helpers[helper] = struct{}{}
		}
		for testPkg, tests := range m.tests {
			if len(tests) > 0 {
				testPkgs[testPkg] = struct{}{}
			}
		}
	}

	b.helpers = setToSorted(helpers)
	b.testPkgs = setToSorted(testPkgs)

	b.binaries = make(map[string]*batchBinary, len(b.testPkgs))
	b.harness = make(map[string]*batchBinary, len(b.testPkgs))
	for _, testPkg := range b.testPkgs {
		b.binaries[testPkg] = &batchBinary{}
		b.harness[testPkg] = &batchBinary{}
	}
}

// prepare renders the batch's files and the injected runtime into an overlay.
func (b *schemataBatch) prepare() error {
	injected := filepath.Join(b.dir, SchemataFileName)
	if _, err := os.Stat(injected); err == nil {
		return fmt.Errorf("%s already exists in %s; refusing to shadow it", SchemataFileName, b.dir)
	}
	if b.pkgName == "" {
		return fmt.Errorf("package %s has no name to inject the schemata runtime under", b.pkg)
	}

	workdir, err := os.MkdirTemp("", "assay-schemata-")
	if err != nil {
		return fmt.Errorf("create schemata work directory: %w", err)
	}
	b.workdir = workdir

	replace := make(map[string]string, len(b.sites)+1)

	for n, file := range sortedSetKeys(b.sites) {
		rendered, renderErr := renderSchemata(b.sources[file], b.sites[file])
		if renderErr != nil {
			return fmt.Errorf("render schemata for %s: %w", file, renderErr)
		}

		path, writeErr := overlay.WriteUnderBasename(workdir, strconv.Itoa(n), file, rendered)
		if writeErr != nil {
			return writeErr
		}
		replace[file] = path
	}

	runtimePath, err := overlay.WriteUnderBasename(workdir, "runtime", injected, SchemataSource(b.pkgName, b.helpers))
	if err != nil {
		return err
	}
	replace[injected] = runtimePath
	b.replace = replace

	b.overlay, err = overlay.WriteFile(workdir, replace)

	return err
}

// harnessBinaryFor compiles the shared binary with the mutant harness injected
// into testPkg, the first time a shard needs it. The harness file rides the
// same overlay as the schemata rewrite, extended per test package because the
// injection path differs per package.
func (b *schemataBatch) harnessBinaryFor(
	ctx context.Context,
	opts ExecuteOptions,
	testPkg, testDir string,
	buildParallelism int,
) (string, error) {
	slot, ok := b.harness[testPkg]
	if !ok {
		return "", fmt.Errorf("no harness binary planned for %s", testPkg)
	}

	slot.mu.Lock()
	defer slot.mu.Unlock()

	if slot.done {
		return slot.path, slot.err
	}

	fail := func(err error) (string, error) {
		slot.done = true
		slot.err = err

		return "", err
	}

	testPkgName, err := gopkg.Name(testDir)
	if err != nil {
		return fail(fmt.Errorf("resolve package name for %s: %w", testPkg, err))
	}

	injected := filepath.Join(testDir, MutHarnessFileName)
	group := "harness-" + overlay.UniqueName(testPkg)
	source := MutHarnessSource(testPkgName, b.pkg, testPkg == b.pkg)
	harnessPath, err := overlay.WriteUnderBasename(b.workdir, group, injected, source)
	if err != nil {
		return fail(err)
	}

	replace := make(map[string]string, len(b.replace)+1)
	for original, path := range b.replace {
		replace[original] = path
	}
	replace[injected] = harnessPath

	overlayPath, err := overlay.WriteFile(filepath.Join(b.workdir, group), replace)
	if err != nil {
		return fail(err)
	}

	path := filepath.Join(b.workdir, "bin", "harness-"+overlay.UniqueName(testPkg)+".test")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fail(fmt.Errorf("create binary directory: %w", err))
	}

	if err := buildTestBinary(ctx, buildRequest{
		root:             opts.Root,
		env:              opts.Env,
		tags:             opts.Tags,
		overlay:          overlayPath,
		testPackage:      testPkg,
		output:           path,
		buildParallelism: buildParallelism,
	}); err != nil {
		// A build killed by cancellation says nothing about whether the package
		// compiles, so it must not be remembered as a failure.
		if ctx.Err() != nil {
			return "", ctx.Err()
		}

		return fail(err)
	}

	slot.done = true
	slot.path = path

	return path, nil
}

// binaryFor compiles this test package's shared binary the first time a mutant
// needs it, and hands every later caller the same path.
func (b *schemataBatch) binaryFor(
	ctx context.Context,
	opts ExecuteOptions,
	testPkg string,
	buildParallelism int,
) (string, error) {
	slot, ok := b.binaries[testPkg]
	if !ok {
		return "", fmt.Errorf("no schemata binary planned for %s", testPkg)
	}

	slot.mu.Lock()
	defer slot.mu.Unlock()

	if slot.done {
		return slot.path, slot.err
	}

	path := filepath.Join(b.workdir, "bin", overlay.UniqueName(testPkg)+".test")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		slot.done = true
		slot.err = fmt.Errorf("create binary directory: %w", err)

		return "", slot.err
	}

	if err := buildTestBinary(ctx, buildRequest{
		root:             opts.Root,
		env:              opts.Env,
		tags:             opts.Tags,
		overlay:          b.overlay,
		testPackage:      testPkg,
		output:           path,
		buildParallelism: buildParallelism,
	}); err != nil {
		// A build killed by cancellation says nothing about whether the package
		// compiles, so it must not be remembered as a failure.
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		slot.done = true
		slot.err = err

		return "", slot.err
	}

	slot.done = true
	slot.path = path

	return path, nil
}

func (b *schemataBatch) cleanup() {
	if b.workdir != "" {
		_ = os.RemoveAll(b.workdir)
	}
}

// judge runs the shared binary once, with this mutant selected.
//
// A non-nil error asks the caller to judge this mutant on a binary of its own: a
// package that will not compile under the shared rewrite still deserves a real
// verdict, and only the mutants that needed that package should pay for it. The
// error carries the reason, which the caller records — a silent fallback is a
// silent 100× slowdown.
//
// Duration counts only this mutant's own test runs. Waiting for a shared binary
// someone else is compiling is not this mutant's cost, and charging it here made
// the JSON output useless for finding slow mutants.
func (b *schemataBatch) judge(
	ctx context.Context,
	index int,
	m Mutant,
	opts ExecuteOptions,
	buildParallelism int,
) (Result, error) {
	result := Result{Mutant: m, Tests: m.tests.count(), Mode: ModeSchemata}

	env := mutantEnv(opts.Env, b.ids[index])

	for _, testPkg := range sortedKeys(m.tests) {
		tests := m.tests[testPkg]
		if len(tests) == 0 {
			continue
		}

		binary, err := b.binaryFor(ctx, opts, testPkg, buildParallelism)
		if err != nil {
			return Result{}, err
		}

		started := time.Now()
		outcome, detail := runTests(ctx, testRun{
			binary: binary,
			dir:    opts.dirFor(testPkg),
			env:    env,
			budget: opts.PackageBudget(m.tests, testPkg),
			tests:  tests,
		})
		result.Duration += time.Since(started)

		if outcome.Killed() {
			result.Outcome = outcome
			result.KilledBy = testPkg
			result.Detail = detail

			return result, nil
		}
	}

	result.Outcome = OutcomeSurvived

	return result, nil
}

// prepareBatches renders each batch's overlay, moving any batch that cannot be
// prepared onto the direct path.
//
// Only preparation is wholesale. Whether a rewritten package can be rendered at all
// is a property of the package, so a failure there disqualifies every mutant in it;
// a failure to *compile* is discovered later, per test package, and costs only the
// mutants that needed it.
func prepareBatches(batches []*schemataBatch) ([]*schemataBatch, []int) {
	declined := make([]int, 0, len(batches))
	ready := make([]*schemataBatch, 0, len(batches))

	for _, batch := range batches {
		if err := batch.prepare(); err != nil {
			batch.cleanup()
			declined = append(declined, batch.indices...)

			continue
		}
		ready = append(ready, batch)
	}

	return ready, declined
}
