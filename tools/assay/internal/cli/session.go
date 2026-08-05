package cli

import (
	"context"
	"fmt"

	"github.com/emoss08/assay/internal/cache"
	"github.com/emoss08/assay/internal/graph"
	"github.com/emoss08/assay/internal/index"
	"github.com/emoss08/assay/internal/selection"
	"github.com/emoss08/assay/internal/vcs"
)

type session struct {
	root     string
	manifest *cache.Manifest
	graph    *graph.Graph
	store    *cache.Store
}

func (o *options) open(ctx context.Context) (*session, error) {
	root, err := resolveRoot(ctx, o.root)
	if err != nil {
		return nil, err
	}

	manifest, err := cache.Scan(ctx, cache.Inputs{Root: root, Tags: o.tags})
	if err != nil {
		return nil, err
	}

	store := o.store()

	g, err := graph.Load(ctx, graph.LoadOptions{
		Root:     root,
		Tags:     o.tags,
		Cache:    store,
		Manifest: manifest,
	})
	if err != nil {
		return nil, err
	}

	return &session{root: root, manifest: manifest, graph: g, store: store}, nil
}

func (s *session) indexStore() (*cache.Store, error) {
	if s.store == nil {
		return nil, nil
	}

	return s.store.Namespace("index", 0)
}

func (s *session) loaderFor(ctx context.Context, commit string, tags []string) (*index.Loader, error) {
	store, err := s.indexStore()
	if err != nil || store == nil || commit == "" {
		return nil, err
	}

	digests, err := vcs.TreeDigests(ctx, s.root, commit)
	if err != nil {
		return nil, err
	}

	return index.NewLoader(store, index.NewFingerprinter(s.graph, digests, tags)), nil
}

type plan struct {
	summary  reportSummary
	narrowed selection.Narrowed
}

type reportSummary struct {
	totalPackages    int
	testablePackages int
	graphFromCache   bool
	note             string
	baseCommit       string
	baseMode         vcs.BaseMode
}

func (o *options) buildPlan(ctx context.Context, s *session) (plan, error) {
	if err := o.validate(); err != nil {
		return plan{}, err
	}

	meta := reportSummary{
		totalPackages:    s.graph.Len(),
		testablePackages: len(s.graph.TestablePackages()),
		graphFromCache:   s.graph.FromCache,
	}

	if o.all {
		return plan{
			summary:  meta,
			narrowed: selection.Narrow(selection.All(s.graph, "--all requested"), selection.NarrowOptions{}),
		}, nil
	}

	changes, err := o.changeSet(ctx, s.root)
	if err != nil {
		return plan{}, err
	}
	meta.note = changes.Note
	meta.baseCommit = changes.BaseCommit
	meta.baseMode = changes.BaseMode

	result := selection.Select(selection.Options{Graph: s.graph, Changes: changes.Changes})

	var loader *index.Loader
	if !o.noIndex {
		loader, err = s.loaderFor(ctx, changes.BaseCommit, o.tags)
		if err != nil {
			return plan{}, err
		}
	}

	narrowed := selection.Narrow(result, selection.NarrowOptions{
		Graph:      s.graph,
		Loader:     loader,
		Changes:    changes.Changes,
		BaseCommit: changes.BaseCommit,
	})

	return plan{summary: meta, narrowed: narrowed}, nil
}

func (p plan) groups() ([]string, map[string]string) {
	var full []string
	narrowed := make(map[string]string)

	for _, pkg := range p.narrowed.Plans {
		switch {
		case pkg.FullReason != "":
			full = append(full, pkg.ImportPath)
		case pkg.Narrowed():
			narrowed[pkg.ImportPath] = selection.RunPattern(pkg.Tests)
		}
	}

	return full, narrowed
}

func (s *session) requireIndexCommit(ctx context.Context) (string, error) {
	commit, err := vcs.HeadCommit(ctx, s.root)
	if err != nil {
		return "", fmt.Errorf("assay index needs a git commit to pin the index to: %w", err)
	}

	return commit, nil
}
