package selection

import (
	"path/filepath"
	"sort"

	"github.com/emoss08/assay/internal/graph"
	"github.com/emoss08/assay/internal/vcs"
)

type Reason string

const (
	ReasonDirect    Reason = "direct"
	ReasonDependent Reason = "dependent"
	ReasonFallback  Reason = "fallback"
)

type Options struct {
	Graph   *graph.Graph
	Changes []vcs.Change
}

type Result struct {
	Packages        []string
	Reasons         map[string]Reason
	ChangedPackages []string
	Unattributed    []string
	SelectAll       bool
	SelectAllReason string
}

var globalTriggerFiles = map[string]struct{}{
	"go.mod":      {},
	"go.sum":      {},
	"go.work":     {},
	"go.work.sum": {},
}

func Select(opts Options) Result {
	g := opts.Graph

	changedPkgs := make(map[string]struct{})
	var unattributed []string

	for _, change := range opts.Changes {
		base := filepath.Base(change.Path)
		if _, isTrigger := globalTriggerFiles[base]; isTrigger {
			return All(g, "module definition changed: "+base)
		}

		pkg, ok := g.PackageForFile(change.Path)
		if ok {
			changedPkgs[pkg.ImportPath] = struct{}{}

			continue
		}

		if filepath.Ext(change.Path) == ".go" {
			return All(g, "changed Go file outside every known package: "+change.Path)
		}

		unattributed = append(unattributed, change.Path)
	}

	seeds := make([]string, 0, len(changedPkgs))
	for path := range changedPkgs {
		seeds = append(seeds, path)
	}
	sort.Strings(seeds)

	packages := g.AffectedTestPackages(seeds)

	reasons := make(map[string]Reason, len(packages))
	for _, path := range packages {
		if _, direct := changedPkgs[path]; direct {
			reasons[path] = ReasonDirect
		} else {
			reasons[path] = ReasonDependent
		}
	}

	sort.Strings(unattributed)

	return Result{
		Packages:        packages,
		Reasons:         reasons,
		ChangedPackages: seeds,
		Unattributed:    unattributed,
	}
}

func All(g *graph.Graph, reason string) Result {
	packages := g.TestablePackages()
	reasons := make(map[string]Reason, len(packages))
	for _, path := range packages {
		reasons[path] = ReasonFallback
	}

	return Result{
		Packages:        packages,
		Reasons:         reasons,
		SelectAll:       true,
		SelectAllReason: reason,
	}
}
