// Package risk joins what the coverage index knows with what git knows: a
// function is exposed when its code changes often and few tests execute it.
// Neither signal alone ranks anything useful — untested-but-frozen code is a
// dormant liability, and heavily-tested hot code is fine — the product of the
// two is where the next regression is most likely to land unseen.
package risk

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/emoss08/assay/internal/cover"
	"github.com/emoss08/assay/internal/gopkg"
	"github.com/emoss08/assay/internal/graph"
	"github.com/emoss08/assay/internal/index"
	"github.com/emoss08/assay/internal/vcs"
)

type Options struct {
	Root   string
	Commit string
	Graph  *graph.Graph
	Loader *index.Loader
	// Churn is keyed by slash-separated path relative to Root.
	Churn map[string]vcs.ChurnStat
	// Packages restricts analysis to these import paths; empty means every
	// package in the graph.
	Packages []string
}

// FunctionRisk is one function's exposure.
type FunctionRisk struct {
	Package         string `json:"package"`
	Function        string `json:"function"`
	File            string `json:"file"`
	StartLine       int    `json:"startLine"`
	EndLine         int    `json:"endLine"`
	ExecutableLines int    `json:"executableLines"`
	UncoveredLines  int    `json:"uncoveredLines"`
	Commits         int    `json:"commits"`
	LinesChanged    int    `json:"linesChanged"`
	// Score is commits × uncovered executable lines: no weights, no tuning,
	// both factors visible in the report so the ranking can be argued with.
	Score int `json:"score"`
}

// Report is the ranked exposure analysis.
type Report struct {
	Functions []FunctionRisk `json:"functions"`
	// CoveredChurning counts churned functions every line of which is covered
	// — change is landing where tests are watching.
	CoveredChurning int `json:"coveredChurning"`
	// UncoveredStable counts functions with uncovered lines in files no commit
	// touched inside the window. Dormant, not safe; below every churning entry
	// in priority.
	UncoveredStable int `json:"uncoveredStable"`
	// Unindexed counts packages none of whose covering test packages have a
	// usable record at this commit. Their functions are invisible to the
	// report, and pretending otherwise would make missing data look like
	// safety.
	Unindexed int `json:"unindexed"`
	// Partial counts packages analyzed with some covering records missing.
	// Coverage can only be under-counted there, so their risk may be
	// overstated — the honest direction, but worth a note.
	Partial  int `json:"partial"`
	Analyzed int `json:"analyzedPackages"`
}

// Analyze walks every package's production files, intersects each function's
// executable lines with the covered lines the index attributes, and scores the
// uncovered remainder by the owning file's churn.
//
// Coverage is unioned across every test package that can reach the analyzed
// one: with -coverpkg spanning the workspace, a package with no tests of its
// own is routinely covered by its importers, and reading only its own record
// would report tested code as exposed.
func Analyze(opts Options) (Report, error) {
	if opts.Graph == nil || opts.Loader == nil {
		return Report{}, fmt.Errorf("risk analysis needs a package graph and a coverage index")
	}

	targets := opts.Packages
	if len(targets) == 0 {
		targets = opts.Graph.Packages()
	}

	var report Report
	for _, importPath := range targets {
		pkg, ok := opts.Graph.Package(importPath)
		if !ok || pkg.Broken {
			continue
		}

		records, missing := coveringRecords(opts, importPath)
		if missing > 0 && len(records) == 0 {
			report.Unindexed++

			continue
		}
		if missing > 0 {
			report.Partial++
		}
		report.Analyzed++

		if err := analyzePackage(&report, opts, pkg, records); err != nil {
			return Report{}, err
		}
	}

	sort.SliceStable(report.Functions, func(i, j int) bool {
		a, b := report.Functions[i], report.Functions[j]
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if a.UncoveredLines != b.UncoveredLines {
			return a.UncoveredLines > b.UncoveredLines
		}
		if a.Package != b.Package {
			return a.Package < b.Package
		}

		return a.Function < b.Function
	})

	return report, nil
}

// coveringRecords resolves the usable records of every test package whose
// tests can reach importPath. A package nothing tests and nothing imports gets
// an empty set with nothing missing: all of its lines are uncovered by
// definition, which is data, not absence of data.
func coveringRecords(opts Options, importPath string) ([]*index.Record, int) {
	candidates := opts.Graph.AffectedTestPackages([]string{importPath})

	records := make([]*index.Record, 0, len(candidates))
	missing := 0
	for _, candidate := range candidates {
		record, found := opts.Loader.Record(candidate)
		if !found || !record.Usable() || record.IndexedAt != opts.Commit {
			missing++

			continue
		}
		records = append(records, record)
	}

	return records, missing
}

func analyzePackage(report *Report, opts Options, pkg *graph.Package, records []*index.Record) error {
	files, err := gopkg.SourceFiles(pkg.Dir)
	if err != nil {
		return fmt.Errorf("list %s: %w", pkg.ImportPath, err)
	}

	for _, name := range files {
		absPath := filepath.Join(pkg.Dir, name)

		relPath, relErr := filepath.Rel(opts.Root, absPath)
		if relErr != nil {
			return relErr
		}
		churn := opts.Churn[filepath.ToSlash(relPath)]

		functions, fnErr := cover.Functions(absPath)
		if fnErr != nil {
			// A file the parser rejects is a build problem, not a risk datum.
			continue
		}
		if len(functions) == 0 {
			continue
		}

		executable, execErr := cover.ExecutableRanges(absPath)
		if execErr != nil {
			continue
		}

		covered := make(map[int]int)
		for _, record := range records {
			for line, tests := range record.LineCoverage(absPath) {
				covered[line] += tests
			}
		}

		for _, fn := range functions {
			executableLines, uncovered := exposure(fn, executable, covered)
			if executableLines == 0 {
				continue
			}

			switch {
			case uncovered == 0 && churn.Commits > 0:
				report.CoveredChurning++
			case uncovered > 0 && churn.Commits == 0:
				report.UncoveredStable++
			case uncovered > 0:
				report.Functions = append(report.Functions, FunctionRisk{
					Package:         pkg.ImportPath,
					Function:        fn.Name,
					File:            filepath.ToSlash(relPath),
					StartLine:       fn.Start,
					EndLine:         fn.End,
					ExecutableLines: executableLines,
					UncoveredLines:  uncovered,
					Commits:         churn.Commits,
					LinesChanged:    churn.LinesChanged,
					Score:           churn.Commits * uncovered,
				})
			}
		}
	}

	return nil
}

// exposure counts a function's executable lines and how many of them no
// attributable test executes.
func exposure(fn cover.Function, executable []cover.Block, covered map[int]int) (total, uncovered int) {
	for _, block := range executable {
		start := max(block.StartLine, fn.Start)
		end := min(block.EndLine, fn.End)
		for line := start; line <= end; line++ {
			total++
			if covered[line] == 0 {
				uncovered++
			}
		}
	}

	return total, uncovered
}
