package watch

import (
	"context"
	"fmt"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/sourcegraph/conc/pool"

	"github.com/emoss08/assay/internal/cache"
	"github.com/emoss08/assay/internal/cover"
	"github.com/emoss08/assay/internal/graph"
	"github.com/emoss08/assay/internal/index"
	"github.com/emoss08/assay/internal/runpattern"
	"github.com/emoss08/assay/internal/selection"
	"github.com/emoss08/assay/internal/vcs"
)

// Planner turns "these files were just saved" into "run exactly these tests".
//
// It reuses the whole selection pipeline — the same graph, the same narrowing,
// the same fail-safe rules — so watch mode can never skip a test that a CI run
// of `assay run` would have executed for the same dirty tree. What it adds is
// the delta: of everything the dirty tree affects, only the packages reachable
// from this save are run now.
type Planner struct {
	Root  string
	Tags  []string
	Env   []string
	Store *cache.Store
	// NoIndex disables line-level narrowing, mirroring the global --no-index
	// flag: plans stay at package precision even when an index exists.
	NoIndex bool

	indexStore *cache.Store
	loaderHead string
	loader     *index.Loader

	lastFingerprints *index.Fingerprinter

	memoGraph  *graph.Graph
	signatures map[string]string
}

// PlannedRun is one package's execution order for a cycle.
type PlannedRun struct {
	ImportPath string
	Dir        string
	// Pattern selects the tests to run; empty means the whole package.
	Pattern string
	Tests   []string
	Reason  string
	// Fingerprint keys the warm binary for this package.
	Fingerprint cache.Fingerprint
}

// Cycle is everything one save resolved to.
type Cycle struct {
	Runs     []PlannedRun
	Deferred int
	Narrowed bool
	Note     string
}

func (p *Planner) Plan(ctx context.Context, batch []string) (*Cycle, error) {
	manifest, err := cache.Scan(ctx, cache.Inputs{Root: p.Root, Tags: p.Tags, Env: p.Env})
	if err != nil {
		return nil, err
	}

	g, err := p.ensureGraph(ctx, manifest, batch)
	if err != nil {
		return nil, err
	}

	changes, err := vcs.Changes(ctx, vcs.Options{Root: p.Root, IncludeUntracked: true})
	if err != nil {
		return nil, err
	}

	loader, err := p.loaderFor(ctx, g, changes.BaseCommit)
	if err != nil {
		return nil, err
	}

	result := selection.Select(selection.Options{Graph: g, Changes: changes.Changes})
	narrowed := selection.Narrow(result, selection.NarrowOptions{
		Graph:      g,
		Loader:     loader,
		Changes:    changes.Changes,
		BaseCommit: changes.BaseCommit,
	})

	cycle := &Cycle{Narrowed: narrowed.Enabled, Note: changes.Note}
	if !narrowed.Enabled && narrowed.DisabledReason != "" {
		cycle.Note = joinNotes(cycle.Note, "narrowing off: "+narrowed.DisabledReason)
	}

	affected := p.affectedByBatch(g, batch)
	fingerprints := index.NewFingerprinter(g, manifest.Digests(), p.Tags)
	p.lastFingerprints = fingerprints

	for _, plan := range narrowed.Plans {
		if plan.FullReason == "" && !plan.Narrowed() {
			continue
		}
		if affected != nil {
			if _, hit := affected[plan.ImportPath]; !hit {
				cycle.Deferred++

				continue
			}
		}

		pkg, ok := g.Package(plan.ImportPath)
		if !ok {
			continue
		}

		run := PlannedRun{
			ImportPath:  plan.ImportPath,
			Dir:         pkg.Dir,
			Reason:      string(plan.Reason),
			Fingerprint: fingerprints.For(plan.ImportPath),
		}
		if plan.Narrowed() {
			run.Pattern = runpattern.From(plan.Tests)
			run.Tests = append([]string(nil), plan.Tests...)
		} else if refined, tests := p.refineTestEdit(g, changes.Changes, plan.ImportPath); refined {
			run.Pattern = runpattern.From(tests)
			run.Tests = tests
			run.Reason = "test-function edit"
		}

		cycle.Runs = append(cycle.Runs, run)
	}

	sort.Slice(cycle.Runs, func(i, j int) bool {
		return cycle.Runs[i].ImportPath < cycle.Runs[j].ImportPath
	})

	return cycle, nil
}

// affectedByBatch limits a cycle to the packages this save can reach. Nil means
// no filter — an empty batch (the initial run) and anything unattributable both
// run the full dirty plan, because guessing narrowly on unclear input is how a
// watch loop quietly stops being trustworthy.
func (p *Planner) affectedByBatch(g *graph.Graph, batch []string) map[string]struct{} {
	if len(batch) == 0 {
		return nil
	}

	var owners []string
	for _, path := range batch {
		if isModulePath(path) {
			return nil
		}
		pkg, ok := g.PackageForFile(path)
		if !ok {
			return nil
		}
		owners = append(owners, pkg.ImportPath)
	}

	affected := make(map[string]struct{})
	for _, candidate := range g.AffectedTestPackages(owners) {
		affected[candidate] = struct{}{}
	}

	return affected
}

// refineTestEdit narrows a full-package run down to the test functions whose
// declarations the edit touched.
//
// The conditions are deliberately strict, because this refinement bypasses the
// coverage index: every dirty file owned by the package must be a modified test
// file with line attribution, every changed line must fall inside a test
// declaration, and the package must not also be selected as a dependent of some
// other change — which is checked by re-deriving the affected set without this
// package's own edits.
func (p *Planner) refineTestEdit(
	g *graph.Graph,
	changes []vcs.Change,
	importPath string,
) (bool, []string) {
	var foreignOwners []string
	var ownedChanges []vcs.Change

	for _, change := range changes {
		pkg, ok := g.PackageForFile(change.Path)
		if !ok {
			// A stray markdown file or client-side asset anywhere in the monorepo
			// must not disable the refinement; only an unattributable Go file is a
			// reason to refuse, mirroring what package selection itself does.
			if filepath.Ext(change.Path) == ".go" {
				return false, nil
			}

			continue
		}
		if pkg.ImportPath == importPath {
			ownedChanges = append(ownedChanges, change)

			continue
		}
		foreignOwners = append(foreignOwners, pkg.ImportPath)
	}
	if len(ownedChanges) == 0 {
		return false, nil
	}

	for _, foreign := range g.AffectedTestPackages(foreignOwners) {
		if foreign == importPath {
			return false, nil
		}
	}

	names := make(map[string]struct{})
	for _, change := range ownedChanges {
		if change.WholeFile() || !strings.HasSuffix(change.Path, "_test.go") {
			return false, nil
		}

		functions, err := cover.TestFunctions(change.Path)
		if err != nil {
			return false, nil
		}

		for _, line := range change.LineNumbers() {
			name, inside := enclosingTest(functions, line)
			if !inside {
				return false, nil
			}
			// Only Test and Fuzz functions actually execute under -test.run; a
			// Benchmark needs -test.bench and an output-less Example never runs at
			// all. Narrowing to one of those would print a green cycle in which
			// zero tests executed, so the whole package runs instead.
			if !runnableByTestRun(name) {
				return false, nil
			}
			names[name] = struct{}{}
		}
	}
	if len(names) == 0 {
		return false, nil
	}

	tests := make([]string, 0, len(names))
	for name := range names {
		tests = append(tests, name)
	}
	sort.Strings(tests)

	return true, tests
}

func runnableByTestRun(name string) bool {
	for _, prefix := range [...]string{"Test", "Fuzz"} {
		if rest, ok := strings.CutPrefix(name, prefix); ok {
			if rest == "" || rest[0] < 'a' || rest[0] > 'z' {
				return true
			}
		}
	}

	return false
}

func enclosingTest(functions []cover.TestFunction, line int) (string, bool) {
	for _, fn := range functions {
		if line >= fn.Start && line <= fn.End {
			return fn.Name, true
		}
	}

	return "", false
}

// ensureGraph keeps the package graph in memory across cycles, reloading only
// when a save could have changed its shape.
//
// The graph cache is keyed by content fingerprints, so every save misses it by
// design — but a body edit does not move any import edge. A file's "import
// signature" (package clause plus import block) is what the graph is actually
// built from, so the memoised graph stays valid until a saved file's signature
// differs from the one recorded at load time. New files, deleted files,
// unparseable files and module-file edits all reload: when the shape is in any
// doubt, a fresh load is the only answer that cannot under-select.
func (p *Planner) ensureGraph(ctx context.Context, manifest *cache.Manifest, batch []string) (*graph.Graph, error) {
	if p.memoGraph != nil && p.graphShapeUnchanged(batch) {
		return p.memoGraph, nil
	}

	g, err := graph.Load(ctx, graph.LoadOptions{
		Root:     p.Root,
		Tags:     p.Tags,
		Env:      p.Env,
		Cache:    p.Store,
		Manifest: manifest,
	})
	if err != nil {
		return nil, err
	}

	p.memoGraph = g
	p.signatures = importSignatures(ctx, manifest)

	return g, nil
}

func (p *Planner) graphShapeUnchanged(batch []string) bool {
	for _, path := range batch {
		if isModulePath(path) {
			return false
		}
		if !strings.HasSuffix(path, ".go") {
			continue
		}

		previous, seen := p.signatures[path]
		if !seen {
			return false
		}
		current, err := importSignature(path)
		if err != nil || current != previous {
			return false
		}
	}

	return true
}

// importSignatures records every Go file's shape-relevant content at graph load
// time, so later saves can be compared in microseconds instead of reloading a
// workspace graph that takes over a second to build fresh.
func importSignatures(ctx context.Context, manifest *cache.Manifest) map[string]string {
	paths := make([]string, 0, len(manifest.Files))
	for _, file := range manifest.Files {
		if strings.HasSuffix(file.RelPath, ".go") {
			paths = append(paths, file.AbsPath)
		}
	}

	signatures := make([]string, len(paths))
	parsing := pool.New().WithMaxGoroutines(runtime.GOMAXPROCS(0)).WithContext(ctx)
	for i := range paths {
		parsing.Go(func(ctx context.Context) error {
			if sig, err := importSignature(paths[i]); err == nil {
				signatures[i] = sig
			}

			return nil
		})
	}
	_ = parsing.Wait()

	out := make(map[string]string, len(paths))
	for i, path := range paths {
		if signatures[i] != "" {
			out[path] = signatures[i]
		}
	}

	return out
}

// importSignature captures everything about a file that can move a graph edge:
// the package clause, the import set, and the build constraints. Constraints
// are comments to the parser but not to the graph — removing a //go:build line
// changes which files are in the package and therefore which edges exist, and
// missing that here would leave the memoised graph stale for the session.
func importSignature(path string) (string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly|parser.ParseComments)
	if err != nil {
		return "", err
	}

	parts := []string{"package " + file.Name.Name}

	imports := make([]string, 0, len(file.Imports))
	for _, imp := range file.Imports {
		imports = append(imports, imp.Path.Value)
	}
	sort.Strings(imports)
	parts = append(parts, imports...)

	for _, group := range file.Comments {
		for _, comment := range group.List {
			text := strings.TrimSpace(comment.Text)
			if strings.HasPrefix(text, "//go:build") || strings.HasPrefix(text, "// +build") {
				parts = append(parts, text)
			}
		}
	}

	return strings.Join(parts, "\n"), nil
}

// Invalidate drops every piece of memoised state. Called when filesystem events
// were lost: a graph whose inputs may have changed unseen cannot be trusted to
// stay memoised.
func (p *Planner) Invalidate() {
	p.memoGraph = nil
	p.signatures = nil
}

// lastKnown resolves a package against the most recent cycle's graph and
// working-tree fingerprints. Sticky failures use it: the fingerprint must be the
// current one, because carrying the fingerprint from the cycle that failed would
// let a stale binary satisfy the cache after the source changed.
func (p *Planner) lastKnown(importPath string) (string, cache.Fingerprint, bool) {
	if p.memoGraph == nil || p.lastFingerprints == nil {
		return "", cache.Fingerprint{}, false
	}

	pkg, ok := p.memoGraph.Package(importPath)
	if !ok {
		return "", cache.Fingerprint{}, false
	}

	return pkg.Dir, p.lastFingerprints.For(importPath), true
}

// loaderFor rebuilds the index loader only when HEAD moves: the loader's
// fingerprints are derived from HEAD's tree digests, which are stable for the
// life of a commit no matter how dirty the working tree gets.
func (p *Planner) loaderFor(ctx context.Context, g *graph.Graph, head string) (*index.Loader, error) {
	if p.NoIndex || p.Store == nil || head == "" {
		return nil, nil
	}

	if p.loader != nil && p.loaderHead == head {
		return p.loader, nil
	}

	store, err := p.Store.Namespace("index", 0)
	if err != nil {
		return nil, err
	}
	p.indexStore = store

	digests, err := vcs.TreeDigests(ctx, p.Root, head)
	if err != nil {
		return nil, fmt.Errorf("resolve tree digests: %w", err)
	}

	p.loader = index.NewLoader(store, index.NewFingerprinter(g, digests, p.Tags))
	p.loaderHead = head

	return p.loader, nil
}

func isModulePath(path string) bool {
	switch filepath.Base(path) {
	case "go.mod", "go.sum", "go.work", "go.work.sum":
		return true
	default:
		return false
	}
}

func joinNotes(a, b string) string {
	switch {
	case a == "":
		return b
	case b == "":
		return a
	default:
		return a + "; " + b
	}
}
