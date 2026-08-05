package graph

import (
	"path/filepath"
	"slices"
	"testing"
)

func testRoot() string {
	return filepath.Join(string(filepath.Separator), "w")
}

func buildTestGraph(t *testing.T) *Graph {
	t.Helper()

	root := testRoot()
	g := newGraph(root)

	g.add(listPackage{
		ImportPath:  "example.com/repo",
		Dir:         filepath.Join(root, "repo"),
		TestGoFiles: []string{"repo_test.go"},
	})
	g.add(listPackage{
		ImportPath:  "example.com/svc",
		Dir:         filepath.Join(root, "svc"),
		Imports:     []string{"example.com/repo", "fmt"},
		TestImports: []string{"example.com/fixtures"},
		TestGoFiles: []string{"svc_test.go"},
	})
	g.add(listPackage{
		ImportPath: "example.com/svc/internal/util",
		Dir:        filepath.Join(root, "svc", "internal", "util"),
		GoFiles:    []string{"util.go"},
	})
	g.add(listPackage{
		ImportPath:   "example.com/app",
		Dir:          filepath.Join(root, "app"),
		Imports:      []string{"example.com/svc"},
		XTestGoFiles: []string{"app_x_test.go"},
	})
	g.add(listPackage{
		ImportPath: "example.com/tool",
		Dir:        filepath.Join(root, "tool"),
		Imports:    []string{"example.com/svc"},
		GoFiles:    []string{"main.go"},
	})
	g.add(listPackage{
		ImportPath: "example.com/fixtures",
		Dir:        filepath.Join(root, "fixtures"),
		GoFiles:    []string{"fixtures.go"},
	})
	g.add(listPackage{
		ImportPath:     "example.com/integrationonly",
		Dir:            filepath.Join(root, "integrationonly"),
		Imports:        []string{"example.com/repo"},
		IgnoredGoFiles: []string{"api_integration_test.go"},
	})
	g.index()

	return g
}

func TestPackageForFileMatchesLongestPrefix(t *testing.T) {
	g := buildTestGraph(t)
	root := testRoot()

	cases := []struct {
		name string
		path string
		want string
	}{
		{"file in package", filepath.Join(root, "svc", "svc.go"), "example.com/svc"},
		{"file in nested package", filepath.Join(root, "svc", "internal", "util", "u.go"), "example.com/svc/internal/util"},
		{"non-go asset in package", filepath.Join(root, "svc", "testdata", "seed.sql"), "example.com/svc"},
		{"deep asset under nested package", filepath.Join(root, "svc", "internal", "util", "testdata", "a.json"), "example.com/svc/internal/util"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pkg, ok := g.PackageForFile(tc.path)
			if !ok {
				t.Fatalf("PackageForFile(%q) = not found, want %q", tc.path, tc.want)
			}
			if pkg.ImportPath != tc.want {
				t.Errorf("PackageForFile(%q) = %q, want %q", tc.path, pkg.ImportPath, tc.want)
			}
		})
	}
}

func TestPackageForFileRejectsSiblingPrefix(t *testing.T) {
	g := buildTestGraph(t)

	if pkg, ok := g.PackageForFile(filepath.Join(testRoot(), "svcother", "x.go")); ok {
		t.Fatalf("PackageForFile matched sibling directory: got %q", pkg.ImportPath)
	}
	if _, ok := g.PackageForFile(filepath.Join(testRoot(), "unrelated", "x.go")); ok {
		t.Fatal("PackageForFile matched an unrelated directory")
	}
}

func TestProductionClosureIsTransitive(t *testing.T) {
	g := buildTestGraph(t)

	closure := g.ProductionClosure([]string{"example.com/repo"})

	for _, want := range []string{"example.com/repo", "example.com/svc", "example.com/app", "example.com/tool"} {
		if !closure[want] {
			t.Errorf("closure missing %q", want)
		}
	}
	if closure["example.com/fixtures"] {
		t.Error("closure should not contain fixtures: nothing in production imports it")
	}
}

func TestAffectedTestPackagesSkipsPackagesWithoutTests(t *testing.T) {
	g := buildTestGraph(t)

	got := g.AffectedTestPackages([]string{"example.com/repo"})

	want := []string{
		"example.com/app",
		"example.com/integrationonly",
		"example.com/repo",
		"example.com/svc",
	}
	if !slices.Equal(got, want) {
		t.Errorf("AffectedTestPackages = %v, want %v", got, want)
	}
	if slices.Contains(got, "example.com/tool") {
		t.Error("tool has no test files and must not be selected")
	}
}

func TestTestImportEdgesDoNotPropagate(t *testing.T) {
	g := buildTestGraph(t)

	got := g.AffectedTestPackages([]string{"example.com/fixtures"})

	want := []string{"example.com/svc"}
	if !slices.Equal(got, want) {
		t.Fatalf("AffectedTestPackages = %v, want %v", got, want)
	}
	if slices.Contains(got, "example.com/app") {
		t.Error("app must not be selected: it reaches fixtures only through svc's test imports")
	}
}

func TestBuildTagOnlyTestsCountAsTestable(t *testing.T) {
	g := buildTestGraph(t)

	pkg, ok := g.Package("example.com/integrationonly")
	if !ok {
		t.Fatal("integrationonly package missing from graph")
	}
	if !pkg.HasTests {
		t.Error("package whose only test file is build-tag excluded must still count as testable")
	}
}

func TestTestablePackagesIsSorted(t *testing.T) {
	g := buildTestGraph(t)

	got := g.TestablePackages()

	if !slices.IsSorted(got) {
		t.Errorf("TestablePackages not sorted: %v", got)
	}
	if slices.Contains(got, "example.com/fixtures") {
		t.Error("fixtures has no tests and must not be listed as testable")
	}
}

func TestSelfImportIsIgnored(t *testing.T) {
	root := testRoot()
	g := newGraph(root)
	g.add(listPackage{
		ImportPath:  "example.com/solo",
		Dir:         filepath.Join(root, "solo"),
		Imports:     []string{"example.com/solo"},
		TestGoFiles: []string{"solo_test.go"},
	})
	g.index()

	if got := len(g.prodImporters["example.com/solo"]); got != 0 {
		t.Errorf("self-import produced %d reverse edges, want 0", got)
	}
}

func TestDedupeSortsAndDropsCgoPseudoImport(t *testing.T) {
	got := dedupe([]string{"b", "a", "C", "b", ""})

	want := []string{"a", "b"}
	if !slices.Equal(got, want) {
		t.Errorf("dedupe = %v, want %v", got, want)
	}
}
