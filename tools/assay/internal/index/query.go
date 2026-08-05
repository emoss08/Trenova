package index

import (
	"sync"

	"github.com/emoss08/assay/internal/graph"
)

// CoverageQuery answers the question the index exists to answer — which tests
// execute this line — with one implementation and one staleness rule.
//
// It exists because two callers grew their own copies and diverged: mutation's
// scope checked that a record was indexed at the base commit, `assay explain`
// did not, and so explain would happily report coverage measured against source
// that is not the source being asked about. Whatever consults the index goes
// through here, and the guard applies to everyone.
//
// The memoization is not optional at mutation scale: resolving a file to its
// package walks the directory table and finding the affected test packages is a
// graph traversal, and both were being redone for every mutation site in a file.
type CoverageQuery struct {
	graph  *graph.Graph
	loader *Loader
	commit string

	mu       sync.Mutex
	owners   map[string]string
	affected map[string][]string
}

func NewCoverageQuery(g *graph.Graph, loader *Loader, commit string) *CoverageQuery {
	return &CoverageQuery{
		graph:    g,
		loader:   loader,
		commit:   commit,
		owners:   make(map[string]string),
		affected: make(map[string][]string),
	}
}

// TestsCovering returns the tests with known coverage of any of the lines,
// grouped by the test package that owns them. The second result reports whether
// any candidate record was usable at this query's commit — it separates "the
// index says nothing covers this" from "there is no index to ask".
func (q *CoverageQuery) TestsCovering(absPath string, lines []int) (map[string][]string, bool) {
	owner, ok := q.ownerOf(absPath)
	if !ok {
		return nil, false
	}

	usable := false
	covering := make(map[string][]string)
	for _, candidate := range q.affectedBy(owner) {
		record, found := q.loader.Record(candidate)
		if !found || record.IndexedAt != q.commit {
			continue
		}
		usable = true
		if !record.Knows(absPath) {
			continue
		}
		if tests := record.TestsCovering(absPath, lines); len(tests) > 0 {
			covering[candidate] = tests
		}
	}

	if len(covering) == 0 {
		return nil, usable
	}

	return covering, usable
}

func (q *CoverageQuery) ownerOf(absPath string) (string, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if owner, cached := q.owners[absPath]; cached {
		return owner, owner != ""
	}

	pkg, ok := q.graph.PackageForFile(absPath)
	if !ok {
		q.owners[absPath] = ""

		return "", false
	}
	q.owners[absPath] = pkg.ImportPath

	return pkg.ImportPath, true
}

func (q *CoverageQuery) affectedBy(owner string) []string {
	q.mu.Lock()
	defer q.mu.Unlock()

	if cached, ok := q.affected[owner]; ok {
		return cached
	}

	candidates := q.graph.AffectedTestPackages([]string{owner})
	q.affected[owner] = candidates

	return candidates
}
