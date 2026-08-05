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
)

type LoadOptions struct {
	Root string
	Tags []string
	Env  []string
}

type listPackage struct {
	Dir            string
	ImportPath     string
	Name           string
	Module         *listModule
	GoFiles        []string
	CgoFiles       []string
	TestGoFiles    []string
	XTestGoFiles   []string
	IgnoredGoFiles []string
	Imports        []string
	TestImports    []string
	XTestImports   []string
	Error          *listError
}

type listModule struct {
	Path string
	Dir  string
	Main bool
}

type listError struct {
	Err string
}

func Load(ctx context.Context, opts LoadOptions) (*Graph, error) {
	root, err := filepath.Abs(opts.Root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}

	modules, err := mainModules(ctx, root, opts.Env)
	if err != nil {
		return nil, err
	}

	type result struct {
		pkgs []listPackage
		err  error
	}

	results := make([]result, len(modules))
	var wg sync.WaitGroup
	for i, mod := range modules {
		wg.Add(1)
		go func(i int, dir string) {
			defer wg.Done()
			pkgs, err := listPackages(ctx, dir, opts.Tags, opts.Env)
			results[i] = result{pkgs: pkgs, err: err}
		}(i, mod.Dir)
	}
	wg.Wait()

	g := newGraph(root)
	for i, res := range results {
		if res.err != nil {
			return nil, fmt.Errorf("list packages in %s: %w", modules[i].Dir, res.err)
		}
		for _, lp := range res.pkgs {
			g.add(lp)
		}
	}
	g.index()

	return g, nil
}

func mainModules(ctx context.Context, root string, env []string) ([]listModule, error) {
	out, err := runGo(ctx, root, env, "list", "-m", "-json")
	if err != nil {
		return nil, fmt.Errorf("list modules: %w", err)
	}

	var modules []listModule
	dec := json.NewDecoder(bytes.NewReader(out))
	for {
		var m listModule
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
		return []listModule{{Dir: root, Main: true}}, nil
	}

	sort.Slice(modules, func(i, j int) bool { return modules[i].Dir < modules[j].Dir })

	return modules, nil
}

func listPackages(ctx context.Context, dir string, tags, env []string) ([]listPackage, error) {
	args := []string{"list", "-json", "-e"}
	if len(tags) > 0 {
		args = append(args, "-tags", strings.Join(tags, ","))
	}
	args = append(args, "./...")

	out, err := runGo(ctx, dir, env, args...)
	if err != nil {
		return nil, err
	}

	var pkgs []listPackage
	dec := json.NewDecoder(bytes.NewReader(out))
	for {
		var lp listPackage
		if decErr := dec.Decode(&lp); decErr != nil {
			if decErr == io.EOF {
				break
			}
			return nil, fmt.Errorf("decode package list: %w", decErr)
		}
		pkgs = append(pkgs, lp)
	}

	return pkgs, nil
}

func runGo(ctx context.Context, dir string, env []string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = dir
	if len(env) > 0 {
		cmd.Env = env
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}

	return stdout.Bytes(), nil
}
