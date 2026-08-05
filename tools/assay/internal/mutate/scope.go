package mutate

import (
	"fmt"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/emoss08/assay/internal/cover"
	"github.com/emoss08/assay/internal/graph"
	"github.com/emoss08/assay/internal/index"
	"github.com/emoss08/assay/internal/vcs"
)

type ScopeOptions struct {
	Graph      *graph.Graph
	Loader     *index.Loader
	BaseCommit string

	// Changes restricts mutation to lines the diff touched. Nil means the whole
	// package, which is the expensive mode and must be asked for explicitly.
	Changes []vcs.Change
}

type Scope struct {
	graph      *graph.Graph
	loader     *index.Loader
	query      *index.CoverageQuery
	changed    map[string]map[int]struct{}
	diffScoped bool

	mu         sync.Mutex
	executable map[string][]cover.Block
}

func NewScope(opts ScopeOptions) (*Scope, error) {
	if opts.Graph == nil {
		return nil, fmt.Errorf("mutation scope needs a package graph")
	}
	if opts.Loader == nil {
		return nil, fmt.Errorf("mutation needs a coverage index; run assay index first")
	}
	if opts.BaseCommit == "" {
		return nil, fmt.Errorf("mutation needs a base commit to match the index against")
	}

	s := &Scope{
		graph:      opts.Graph,
		loader:     opts.Loader,
		query:      index.NewCoverageQuery(opts.Graph, opts.Loader, opts.BaseCommit),
		executable: make(map[string][]cover.Block),
	}

	if opts.Changes != nil {
		s.diffScoped = true
		s.changed = make(map[string]map[int]struct{})
		for _, change := range opts.Changes {
			if change.WholeFile() || filepath.Ext(change.Path) != ".go" {
				continue
			}
			lines := s.changed[change.Path]
			if lines == nil {
				lines = make(map[int]struct{})
				s.changed[change.Path] = lines
			}
			for _, line := range change.LineNumbers() {
				lines[line] = struct{}{}
			}
		}
	}

	return s, nil
}

// Packages lists the packages worth mutating: those owning at least one eligible
// changed file, or every testable package when not diff-scoped.
func (s *Scope) Packages() []string {
	if !s.diffScoped {
		return s.graph.TestablePackages()
	}

	seen := make(map[string]struct{})
	for path := range s.changed {
		if pkg, ok := s.graph.PackageForFile(path); ok {
			seen[pkg.ImportPath] = struct{}{}
		}
	}

	out := make([]string, 0, len(seen))
	for pkg := range seen {
		out = append(out, pkg)
	}
	sort.Strings(out)

	return out
}

// Eligible restricts mutation to executable statements, and to changed lines when
// diff-scoped. Coverage is deliberately not consulted here: an uncovered site is
// still generated so it can be reported as no-coverage, which costs nothing
// because such mutants are never built.
func (s *Scope) Eligible(absPath string, line int) bool {
	if s.diffScoped {
		lines, ok := s.changed[absPath]
		if !ok {
			return false
		}
		if _, ok := lines[line]; !ok {
			return false
		}
	}

	for _, span := range s.executableRanges(absPath) {
		if span.Contains(line) {
			return true
		}
	}

	return false
}

func (s *Scope) executableRanges(absPath string) []cover.Block {
	s.mu.Lock()
	defer s.mu.Unlock()

	ranges, cached := s.executable[absPath]
	if !cached {
		var err error
		ranges, err = cover.ExecutableRanges(absPath)
		if err != nil {
			ranges = nil
		}
		s.executable[absPath] = ranges
	}

	return ranges
}

// Tests finds the tests that execute a line, across every package whose tests can
// reach the mutated one.
//
// Always-run tests are deliberately excluded. Their attribution is unknown by
// definition, and a test that was already failing when the index was built would
// mark every mutant killed and silently inflate the score. A survivor therefore
// means "no test with known coverage of this line killed it", which is the honest
// claim.
func (s *Scope) Tests(absPath string, line int) TestPlan {
	covering, _ := s.query.TestsCovering(absPath, []int{line})

	return TestPlan(covering)
}

// Budget sums the indexed durations of the tests a mutant will run, so its timeout
// is a multiple of what those tests actually cost rather than a fixed guess.
func (s *Scope) Budget(plan TestPlan) time.Duration {
	var total time.Duration
	for pkg, tests := range plan {
		record, found := s.loader.Record(pkg)
		if !found {
			continue
		}
		total += record.DurationOf(tests)
	}

	return total
}
