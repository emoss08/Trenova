package graph

import (
	"path/filepath"
	"sort"
	"strings"
)

type Package struct {
	ImportPath  string
	Dir         string
	Module      string
	Imports     []string
	TestImports []string
	HasTests    bool
	Broken      bool
}

type Graph struct {
	Root string

	pkgs          map[string]*Package
	dirs          []*Package
	prodImporters map[string][]string
	testImporters map[string][]string
}

func newGraph(root string) *Graph {
	return &Graph{
		Root:          root,
		pkgs:          make(map[string]*Package),
		prodImporters: make(map[string][]string),
		testImporters: make(map[string][]string),
	}
}

func (g *Graph) add(lp listPackage) {
	if lp.ImportPath == "" || lp.Dir == "" {
		return
	}
	if _, exists := g.pkgs[lp.ImportPath]; exists {
		return
	}

	module := ""
	if lp.Module != nil {
		module = lp.Module.Path
	}

	pkg := &Package{
		ImportPath:  lp.ImportPath,
		Dir:         filepath.Clean(lp.Dir),
		Module:      module,
		Imports:     dedupe(lp.Imports),
		TestImports: dedupe(append(append([]string{}, lp.TestImports...), lp.XTestImports...)),
		HasTests:    hasTestFiles(lp),
		Broken:      lp.Error != nil,
	}

	g.pkgs[pkg.ImportPath] = pkg
}

func hasTestFiles(lp listPackage) bool {
	if len(lp.TestGoFiles) > 0 || len(lp.XTestGoFiles) > 0 {
		return true
	}
	for _, f := range lp.IgnoredGoFiles {
		if strings.HasSuffix(f, "_test.go") {
			return true
		}
	}

	return false
}

func (g *Graph) index() {
	g.dirs = make([]*Package, 0, len(g.pkgs))
	for _, pkg := range g.pkgs {
		g.dirs = append(g.dirs, pkg)

		for _, imp := range pkg.Imports {
			if imp == pkg.ImportPath {
				continue
			}
			if _, known := g.pkgs[imp]; known {
				g.prodImporters[imp] = append(g.prodImporters[imp], pkg.ImportPath)
			}
		}
		for _, imp := range pkg.TestImports {
			if imp == pkg.ImportPath {
				continue
			}
			if _, known := g.pkgs[imp]; known {
				g.testImporters[imp] = append(g.testImporters[imp], pkg.ImportPath)
			}
		}
	}

	sort.Slice(g.dirs, func(i, j int) bool {
		if len(g.dirs[i].Dir) != len(g.dirs[j].Dir) {
			return len(g.dirs[i].Dir) > len(g.dirs[j].Dir)
		}

		return g.dirs[i].Dir < g.dirs[j].Dir
	})
}

func (g *Graph) Len() int { return len(g.pkgs) }

func (g *Graph) Package(importPath string) (*Package, bool) {
	pkg, ok := g.pkgs[importPath]

	return pkg, ok
}

func (g *Graph) TestablePackages() []string {
	out := make([]string, 0, len(g.pkgs))
	for path, pkg := range g.pkgs {
		if pkg.HasTests {
			out = append(out, path)
		}
	}
	sort.Strings(out)

	return out
}

func (g *Graph) PackageForFile(absPath string) (*Package, bool) {
	clean := filepath.Clean(absPath)
	for _, pkg := range g.dirs {
		if underDir(pkg.Dir, clean) {
			return pkg, true
		}
	}

	return nil, false
}

func underDir(dir, path string) bool {
	if path == dir {
		return true
	}
	if !strings.HasPrefix(path, dir) {
		return false
	}

	return len(path) > len(dir) && path[len(dir)] == filepath.Separator
}

func (g *Graph) ProductionClosure(seeds []string) map[string]bool {
	closure := make(map[string]bool, len(seeds))
	queue := make([]string, 0, len(seeds))

	for _, seed := range seeds {
		if _, known := g.pkgs[seed]; !known || closure[seed] {
			continue
		}
		closure[seed] = true
		queue = append(queue, seed)
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, importer := range g.prodImporters[current] {
			if closure[importer] {
				continue
			}
			closure[importer] = true
			queue = append(queue, importer)
		}
	}

	return closure
}

func (g *Graph) AffectedTestPackages(seeds []string) []string {
	closure := g.ProductionClosure(seeds)

	affected := make(map[string]bool)
	for path := range closure {
		if pkg, ok := g.pkgs[path]; ok && pkg.HasTests {
			affected[path] = true
		}
		for _, importer := range g.testImporters[path] {
			if pkg, ok := g.pkgs[importer]; ok && pkg.HasTests {
				affected[importer] = true
			}
		}
	}

	out := make([]string, 0, len(affected))
	for path := range affected {
		out = append(out, path)
	}
	sort.Strings(out)

	return out
}

func dedupe(in []string) []string {
	if len(in) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		if v == "" || v == "C" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)

	return out
}
