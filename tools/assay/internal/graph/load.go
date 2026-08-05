package graph

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"golang.org/x/tools/go/packages"

	"github.com/emoss08/assay/internal/cache"
)

const loadMode = packages.NeedName |
	packages.NeedFiles |
	packages.NeedImports |
	packages.NeedModule

type LoadOptions struct {
	Root  string
	Tags  []string
	Env   []string
	Cache *cache.Store
}

type mainModule struct {
	Path string
	Dir  string
}

func Load(ctx context.Context, opts LoadOptions) (*Graph, error) {
	root, err := filepath.Abs(opts.Root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}

	key, cacheUsable := cacheKey(ctx, root, opts)
	if cacheUsable {
		if payload, hit := opts.Cache.Get(key); hit {
			if cached, decodeErr := UnmarshalGraph(payload); decodeErr == nil {
				cached.FromCache = true

				return cached, nil
			}
		}
	}

	g, err := loadFresh(ctx, root, opts)
	if err != nil {
		return nil, err
	}

	if cacheUsable {
		if payload, encodeErr := g.MarshalBinary(); encodeErr == nil {
			_ = opts.Cache.Put(key, payload)
		}
	}

	return g, nil
}

func cacheKey(ctx context.Context, root string, opts LoadOptions) (cache.Fingerprint, bool) {
	if opts.Cache == nil {
		return cache.Fingerprint{}, false
	}

	key, err := cache.Compute(ctx, cache.Inputs{Root: root, Tags: opts.Tags, Env: opts.Env})
	if err != nil {
		return cache.Fingerprint{}, false
	}

	return key, true
}

func loadFresh(ctx context.Context, root string, opts LoadOptions) (*Graph, error) {
	modules, err := mainModules(ctx, root, opts.Env)
	if err != nil {
		return nil, err
	}

	type result struct {
		pkgs []*packages.Package
		err  error
	}

	results := make([]result, len(modules))
	var wg sync.WaitGroup
	for i, mod := range modules {
		wg.Add(1)
		go func(i int, dir string) {
			defer wg.Done()
			pkgs, loadErr := loadModule(ctx, dir, opts)
			results[i] = result{pkgs: pkgs, err: loadErr}
		}(i, mod.Dir)
	}
	wg.Wait()

	g := newGraph(root)
	for i, res := range results {
		if res.err != nil {
			return nil, fmt.Errorf("load packages in %s: %w", modules[i].Dir, res.err)
		}
		g.ingest(res.pkgs)
	}
	g.index()

	return g, nil
}

func loadModule(ctx context.Context, dir string, opts LoadOptions) ([]*packages.Package, error) {
	cfg := &packages.Config{
		Mode:    loadMode,
		Context: ctx,
		Dir:     dir,
		Tests:   true,
		Env:     opts.Env,
	}
	if len(opts.Tags) > 0 {
		cfg.BuildFlags = []string{"-tags", strings.Join(opts.Tags, ",")}
	}

	return packages.Load(cfg, "./...")
}

func mainModules(ctx context.Context, root string, env []string) ([]mainModule, error) {
	cmd := exec.CommandContext(ctx, "go", "list", "-m", "-json")
	cmd.Dir = root
	if len(env) > 0 {
		cmd.Env = env
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go list -m -json: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	var modules []mainModule
	dec := json.NewDecoder(bytes.NewReader(stdout.Bytes()))
	for {
		var m mainModule
		if decErr := dec.Decode(&m); decErr != nil {
			if decErr == io.EOF {
				break
			}

			return nil, fmt.Errorf("decode module list: %w", decErr)
		}
		if m.Dir != "" {
			modules = append(modules, m)
		}
	}

	if len(modules) == 0 {
		return []mainModule{{Dir: root}}, nil
	}

	sort.Slice(modules, func(i, j int) bool { return modules[i].Dir < modules[j].Dir })

	return modules, nil
}
