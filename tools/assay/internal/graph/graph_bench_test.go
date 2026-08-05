package graph

import (
	"fmt"
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/packages"
)

const benchFanIn = 5

func synthPackages(n int) []*packages.Package {
	root := filepath.Join(string(filepath.Separator), "bench")
	out := make([]*packages.Package, 0, n*2)

	for i := range n {
		path := fmt.Sprintf("example.com/bench/group%02d/pkg%05d", i%64, i)
		dir := filepath.Join(root, fmt.Sprintf("group%02d", i%64), fmt.Sprintf("pkg%05d", i))

		deps := make(map[string]*packages.Package, benchFanIn)
		for d := 1; d <= benchFanIn; d++ {
			if j := i - d*7; j >= 0 {
				deps[fmt.Sprintf("example.com/bench/group%02d/pkg%05d", j%64, j)] = nil
			}
		}

		out = append(out, &packages.Package{
			ID:      path,
			PkgPath: path,
			GoFiles: []string{filepath.Join(dir, "impl.go")},
			Imports: deps,
		})

		if i%2 == 0 {
			testDeps := map[string]*packages.Package{"testing": nil}
			for k := range deps {
				testDeps[k] = nil
			}
			out = append(out, &packages.Package{
				ID:      path + " [" + path + ".test]",
				PkgPath: path,
				GoFiles: []string{filepath.Join(dir, "impl.go"), filepath.Join(dir, "impl_test.go")},
				Imports: testDeps,
			})
		}
	}

	return out
}

func benchGraph(n int) *Graph {
	g := newGraph(filepath.Join(string(filepath.Separator), "bench"))
	g.ingest(synthPackages(n))
	g.index()

	return g
}

func benchChangedFiles(g *Graph, count int) []string {
	out := make([]string, 0, count)
	for _, pkg := range g.dirs {
		if len(out) == count {
			break
		}
		out = append(out, filepath.Join(pkg.Dir, "impl.go"))
	}

	return out
}

func BenchmarkIngestAndIndex(b *testing.B) {
	for _, n := range []int{700, 10000} {
		pkgs := synthPackages(n)
		b.Run(fmt.Sprintf("packages=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				g := newGraph("/bench")
				g.ingest(pkgs)
				g.index()
			}
		})
	}
}

func BenchmarkAffectedTestPackages(b *testing.B) {
	for _, n := range []int{700, 10000} {
		g := benchGraph(n)
		seeds := []string{"example.com/bench/group00/pkg00000"}
		b.Run(fmt.Sprintf("packages=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				g.AffectedTestPackages(seeds)
			}
		})
	}
}

func BenchmarkPackageForFile(b *testing.B) {
	for _, n := range []int{700, 10000} {
		g := benchGraph(n)
		for _, changed := range []int{10, 500} {
			files := benchChangedFiles(g, changed)
			b.Run(fmt.Sprintf("packages=%d/files=%d", n, changed), func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					for _, f := range files {
						g.PackageForFile(f)
					}
				}
			})
		}
	}
}
