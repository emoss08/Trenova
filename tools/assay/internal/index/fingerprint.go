package index

import (
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/zeebo/blake3"

	"github.com/emoss08/assay/internal/cache"
	"github.com/emoss08/assay/internal/graph"
)

const indexSchema = "assay-index-v3"

type Fingerprinter struct {
	graph     *graph.Graph
	byPkg     map[string][]string
	unowned   []string
	toolchain string
}

func NewFingerprinter(g *graph.Graph, treeDigests map[string]string, tags []string) *Fingerprinter {
	sorted := append([]string(nil), tags...)
	sort.Strings(sorted)

	f := &Fingerprinter{
		graph:     g,
		byPkg:     make(map[string][]string),
		toolchain: cache.BuildIdentity() + "|" + runtime.Version() + "|" + strings.Join(sorted, ","),
	}

	paths := make([]string, 0, len(treeDigests))
	for path := range treeDigests {
		if !isRelevantPath(path) {
			continue
		}
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		entry := path + " " + treeDigests[path]
		abs := filepath.Join(g.Root, filepath.FromSlash(path))

		if isModuleFile(path) {
			f.unowned = append(f.unowned, entry)

			continue
		}
		pkg, ok := g.PackageForFile(abs)
		if !ok {
			f.unowned = append(f.unowned, entry)

			continue
		}
		f.byPkg[pkg.ImportPath] = append(f.byPkg[pkg.ImportPath], entry)
	}

	return f
}

func isRelevantPath(path string) bool {
	return strings.HasSuffix(path, ".go") || isModuleFile(path)
}

func isModuleFile(path string) bool {
	switch filepath.Base(path) {
	case "go.mod", "go.sum", "go.work", "go.work.sum":
		return true
	default:
		return false
	}
}

func (f *Fingerprinter) For(importPath string) cache.Fingerprint {
	h := blake3.New()
	writeField(h, indexSchema)
	writeField(h, f.toolchain)
	writeField(h, importPath)

	for _, entry := range f.unowned {
		writeField(h, entry)
	}

	for _, dep := range f.graph.DependencyClosure([]string{importPath}) {
		writeField(h, dep)
		for _, entry := range f.byPkg[dep] {
			writeField(h, entry)
		}
	}

	var out cache.Fingerprint
	copy(out[:], h.Sum(nil))

	return out
}

func writeField(h *blake3.Hasher, value string) {
	h.Write([]byte(value))
	h.Write([]byte{0})
}
