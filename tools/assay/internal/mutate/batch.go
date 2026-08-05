package mutate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"
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

	binaries map[string]*batchBinary
}

// batchBinary is one test package's shared binary. The Once means concurrent
// mutants needing the same binary compile it exactly once and the rest wait for it.
type batchBinary struct {
	once sync.Once
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
	for _, testPkg := range b.testPkgs {
		b.binaries[testPkg] = &batchBinary{}
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

		path, writeErr := writeUnderBasename(workdir, strconv.Itoa(n), file, rendered)
		if writeErr != nil {
			return writeErr
		}
		replace[file] = path
	}

	runtimePath, err := writeUnderBasename(workdir, "runtime", injected, SchemataSource(b.pkgName, b.helpers))
	if err != nil {
		return err
	}
	replace[injected] = runtimePath

	b.overlay, err = writeOverlayFile(workdir, replace)

	return err
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

	slot.once.Do(func() {
		path := filepath.Join(b.workdir, "bin", sanitisePath(testPkg)+".test")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			slot.err = fmt.Errorf("create binary directory: %w", err)

			return
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
			slot.err = err

			return
		}

		slot.path = path
	})

	return slot.path, slot.err
}

func (b *schemataBatch) cleanup() {
	if b.workdir != "" {
		_ = os.RemoveAll(b.workdir)
	}
}

// judge runs the shared binary once, with this mutant selected.
//
// The false return asks the caller to judge this mutant on a binary of its own: a
// package that will not compile under the shared rewrite still deserves a real
// verdict, and only the mutants that needed that package should pay for it.
func (b *schemataBatch) judge(
	ctx context.Context,
	index int,
	m Mutant,
	opts ExecuteOptions,
	buildParallelism int,
) (Result, bool) {
	started := time.Now()
	result := Result{Mutant: m, Tests: m.tests.count(), Mode: ModeSchemata}

	budget := opts.budgetFor(m.tests)
	env := mutantEnv(opts.Env, b.ids[index])

	for _, testPkg := range sortedKeys(m.tests) {
		tests := m.tests[testPkg]
		if len(tests) == 0 {
			continue
		}

		binary, err := b.binaryFor(ctx, opts, testPkg, buildParallelism)
		if err != nil {
			return Result{}, false
		}

		outcome, detail := runTests(ctx, testRun{
			binary: binary,
			dir:    opts.dirFor(testPkg),
			env:    env,
			budget: budget,
			tests:  tests,
		})
		if outcome.Killed() {
			result.Outcome = outcome
			result.KilledBy = testPkg
			result.Detail = detail
			result.Duration = time.Since(started)

			return result, true
		}
	}

	result.Outcome = OutcomeSurvived
	result.Duration = time.Since(started)

	return result, true
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

// writeUnderBasename keeps the original file name, because the go command decides
// build constraints such as _linux or _amd64 from the name it is given.
func writeUnderBasename(workdir, group, original string, content []byte) (string, error) {
	dir := filepath.Join(workdir, group)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create overlay directory: %w", err)
	}

	path := filepath.Join(dir, filepath.Base(original))
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}

	return path, nil
}

func writeOverlayFile(workdir string, replace map[string]string) (string, error) {
	payload, err := json.Marshal(map[string]map[string]string{"Replace": replace})
	if err != nil {
		return "", fmt.Errorf("encode overlay: %w", err)
	}

	path := filepath.Join(workdir, "overlay.json")
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return "", fmt.Errorf("write overlay: %w", err)
	}

	return path, nil
}

func sanitisePath(importPath string) string {
	out := make([]byte, 0, len(importPath))
	for i := 0; i < len(importPath); i++ {
		c := importPath[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '.', c == '-', c == '_':
			out = append(out, c)
		default:
			out = append(out, '_')
		}
	}

	return string(out)
}
