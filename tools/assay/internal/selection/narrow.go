package selection

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/emoss08/assay/internal/cover"
	"github.com/emoss08/assay/internal/graph"
	"github.com/emoss08/assay/internal/index"
	"github.com/emoss08/assay/internal/runpattern"
	"github.com/emoss08/assay/internal/vcs"
)

type PackagePlan struct {
	ImportPath string
	Reason     Reason
	Tests      []string
	AllTests   []string
	FullReason string
}

func (p PackagePlan) ExcludedTests() []string {
	if p.FullReason != "" {
		return nil
	}

	selected := make(map[string]struct{}, len(p.Tests))
	for _, test := range p.Tests {
		selected[test] = struct{}{}
	}

	var out []string
	for _, test := range p.AllTests {
		if _, ok := selected[test]; ok {
			continue
		}
		out = append(out, test)
	}
	sort.Strings(out)

	return out
}

func (p PackagePlan) Narrowed() bool {
	return len(p.Tests) > 0
}

type Narrowed struct {
	Result
	Plans            []PackagePlan
	NarrowedPackages int
	DroppedPackages  int
	SelectedTests    int
	Blocked          []string
	Enabled          bool
	DisabledReason   string
}

type NarrowOptions struct {
	Graph      *graph.Graph
	Loader     *index.Loader
	Changes    []vcs.Change
	BaseCommit string
}

func Narrow(res Result, opts NarrowOptions) Narrowed {
	out := Narrowed{Result: res}

	if opts.Loader == nil {
		return disabled(out, res, "no index available")
	}
	if opts.BaseCommit == "" {
		return disabled(out, res, "base commit unknown")
	}
	if res.SelectAll {
		return disabled(out, res, "package selection already fell back to everything")
	}

	attributable, blocking := partitionChanges(opts.Changes)

	blockedPackages := make(map[string]bool)
	if len(blocking) > 0 {
		seeds := owningPackages(opts.Graph, blocking)
		for _, pkg := range opts.Graph.AffectedTestPackages(seeds) {
			blockedPackages[pkg] = true
		}
		out.Blocked = blocking
	}

	plans := make([]PackagePlan, 0, len(res.Packages))
	for _, importPath := range res.Packages {
		plans = append(plans, planFor(planInput{
			importPath:   importPath,
			reason:       res.Reasons[importPath],
			blocked:      blockedPackages[importPath],
			attributable: attributable,
			loader:       opts.Loader,
			baseCommit:   opts.BaseCommit,
		}))
	}

	out.Plans = plans
	out.Enabled = true
	for _, plan := range plans {
		switch {
		case plan.FullReason != "":
		case plan.Narrowed():
			out.NarrowedPackages++
			out.SelectedTests += len(plan.Tests)
		default:
			out.DroppedPackages++
		}
	}

	return out
}

func disabled(out Narrowed, res Result, reason string) Narrowed {
	out.DisabledReason = reason
	out.Plans = make([]PackagePlan, 0, len(res.Packages))
	for _, importPath := range res.Packages {
		out.Plans = append(out.Plans, PackagePlan{
			ImportPath: importPath,
			Reason:     res.Reasons[importPath],
			FullReason: reason,
		})
	}

	return out
}

type fileChange struct {
	path      string
	baseLines []int
}

func partitionChanges(changes []vcs.Change) ([]fileChange, []string) {
	var attributable []fileChange
	var blocking []string

	for _, change := range changes {
		if change.WholeFile() {
			blocking = append(blocking, change.Path)

			continue
		}
		if filepath.Ext(change.Path) != ".go" {
			blocking = append(blocking, change.Path)

			continue
		}

		classification, err := cover.ClassifyFile(change.Path, change.LineNumbers())
		if err != nil || !classification.Narrowable() {
			blocking = append(blocking, change.Path)

			continue
		}
		if len(classification.Executable) == 0 {
			continue
		}

		baseLines := change.BaseLineNumbers()
		if len(baseLines) == 0 {
			blocking = append(blocking, change.Path)

			continue
		}
		attributable = append(attributable, fileChange{path: change.Path, baseLines: baseLines})
	}

	sort.Strings(blocking)

	return attributable, blocking
}

func owningPackages(g *graph.Graph, paths []string) []string {
	seen := make(map[string]struct{})
	for _, path := range paths {
		if pkg, ok := g.PackageForFile(path); ok {
			seen[pkg.ImportPath] = struct{}{}
		}
	}

	out := make([]string, 0, len(seen))
	for path := range seen {
		out = append(out, path)
	}
	sort.Strings(out)

	return out
}

type planInput struct {
	importPath   string
	reason       Reason
	blocked      bool
	attributable []fileChange
	loader       *index.Loader
	baseCommit   string
}

func planFor(in planInput) PackagePlan {
	plan := PackagePlan{ImportPath: in.importPath, Reason: in.reason}

	if in.blocked {
		plan.FullReason = "reached by a change with no line-level attribution"

		return plan
	}

	record, ok := in.loader.Record(in.importPath)
	if !ok {
		plan.FullReason = "no index record for this package"

		return plan
	}
	if record.Degraded != "" {
		plan.FullReason = "index record is degraded: " + record.Degraded

		return plan
	}
	if record.IndexedAt != in.baseCommit {
		plan.FullReason = fmt.Sprintf("index was built at %s, base is %s",
			vcs.ShortSHA(record.IndexedAt), vcs.ShortSHA(in.baseCommit))

		return plan
	}
	if len(record.Tests) == 0 {
		plan.FullReason = "index record lists no tests"

		return plan
	}

	plan.AllTests = record.TestNames()

	selected := make(map[string]struct{})
	for _, test := range record.AlwaysRunTests() {
		selected[test] = struct{}{}
	}

	for _, change := range in.attributable {
		if !record.Knows(change.path) {
			continue
		}
		for _, test := range record.TestsCovering(change.path, change.baseLines) {
			selected[test] = struct{}{}
		}
	}

	plan.Tests = make([]string, 0, len(selected))
	for test := range selected {
		plan.Tests = append(plan.Tests, test)
	}
	sort.Strings(plan.Tests)

	return plan
}

func RunPattern(tests []string) string {
	return runpattern.From(tests)
}
