package graph

import (
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
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
	Root      string
	FromCache bool

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

func (g *Graph) ingest(pkgs []*packages.Package) {
	variants := make(map[string][]*packages.Package)

	for _, p := range pkgs {
		if p == nil || p.PkgPath == "" {
			continue
		}
		if isGeneratedTestMain(p) {
			continue
		}
		if owner, isVariant := testVariantOwner(p.ID); isVariant {
			variants[owner] = append(variants[owner], p)

			continue
		}
		g.addProduction(p)
	}

	owners := make([]string, 0, len(variants))
	for owner := range variants {
		owners = append(owners, owner)
	}
	sort.Strings(owners)

	for _, owner := range owners {
		g.attachTestVariants(owner, variants[owner])
	}
}

func (g *Graph) addProduction(p *packages.Package) {
	if _, exists := g.pkgs[p.PkgPath]; exists {
		return
	}

	module := ""
	if p.Module != nil {
		module = p.Module.Path
	}

	g.pkgs[p.PkgPath] = &Package{
		ImportPath: p.PkgPath,
		Dir:        packageDir(p),
		Module:     module,
		Imports:    importPaths(p),
		HasTests:   hasBuildTaggedTestFile(p),
		Broken:     len(p.Errors) > 0,
	}
}

func (g *Graph) attachTestVariants(owner string, variants []*packages.Package) {
	pkg, ok := g.pkgs[owner]
	if !ok {
		pkg = &Package{ImportPath: owner}
		for _, variant := range variants {
			if dir := packageDir(variant); dir != "" {
				pkg.Dir = dir

				break
			}
		}
		if pkg.Dir == "" {
			return
		}
		g.pkgs[owner] = pkg
	}

	pkg.HasTests = true

	production := make(map[string]struct{}, len(pkg.Imports))
	for _, imp := range pkg.Imports {
		production[imp] = struct{}{}
	}

	seen := make(map[string]struct{})
	var testOnly []string
	for _, variant := range variants {
		if len(variant.Errors) > 0 {
			pkg.Broken = true
		}
		for _, imp := range importPaths(variant) {
			if imp == owner {
				continue
			}
			if _, isProduction := production[imp]; isProduction {
				continue
			}
			if _, dup := seen[imp]; dup {
				continue
			}
			seen[imp] = struct{}{}
			testOnly = append(testOnly, imp)
		}
	}

	sort.Strings(testOnly)
	pkg.TestImports = testOnly
}

func isGeneratedTestMain(p *packages.Package) bool {
	return p.ID == p.PkgPath && strings.HasSuffix(p.PkgPath, ".test")
}

func testVariantOwner(id string) (string, bool) {
	open := strings.LastIndex(id, " [")
	if open < 0 || !strings.HasSuffix(id, "]") {
		return "", false
	}

	binary := id[open+2 : len(id)-1]
	owner, ok := strings.CutSuffix(binary, ".test")
	if !ok || owner == "" {
		return "", false
	}

	return owner, true
}

func packageDir(p *packages.Package) string {
	for _, list := range [][]string{p.GoFiles, p.IgnoredFiles, p.OtherFiles, p.CompiledGoFiles} {
		for _, file := range list {
			if filepath.IsAbs(file) {
				return filepath.Clean(filepath.Dir(file))
			}
		}
	}

	return ""
}

func hasBuildTaggedTestFile(p *packages.Package) bool {
	for _, file := range p.IgnoredFiles {
		if strings.HasSuffix(file, "_test.go") {
			return true
		}
	}

	return false
}

func importPaths(p *packages.Package) []string {
	if len(p.Imports) == 0 {
		return nil
	}

	out := make([]string, 0, len(p.Imports))
	for path := range p.Imports {
		if path == "" || path == "C" {
			continue
		}
		out = append(out, path)
	}
	sort.Strings(out)

	return out
}

func (g *Graph) index() {
	g.dirs = make([]*Package, 0, len(g.pkgs))
	for _, pkg := range g.pkgs {
		if pkg.Dir != "" {
			g.dirs = append(g.dirs, pkg)
		}

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

func (g *Graph) BrokenPackages() []string {
	var out []string
	for path, pkg := range g.pkgs {
		if pkg.Broken {
			out = append(out, path)
		}
	}
	sort.Strings(out)

	return out
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
	if dir == "" {
		return false
	}
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

func (g *Graph) DependencyClosure(seeds []string) []string {
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

		pkg, ok := g.pkgs[current]
		if !ok {
			continue
		}
		for _, deps := range [][]string{pkg.Imports, pkg.TestImports} {
			for _, dep := range deps {
				if closure[dep] {
					continue
				}
				if _, known := g.pkgs[dep]; !known {
					continue
				}
				closure[dep] = true
				queue = append(queue, dep)
			}
		}
	}

	out := make([]string, 0, len(closure))
	for path := range closure {
		out = append(out, path)
	}
	sort.Strings(out)

	return out
}

func (g *Graph) Modules() []string {
	seen := make(map[string]struct{})
	for _, pkg := range g.pkgs {
		if pkg.Module != "" {
			seen[pkg.Module] = struct{}{}
		}
	}

	out := make([]string, 0, len(seen))
	for module := range seen {
		out = append(out, module)
	}
	sort.Strings(out)

	return out
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
