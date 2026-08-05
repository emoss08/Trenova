package graph

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
)

func testRoot() string {
	return filepath.Join(string(filepath.Separator), "w")
}

func imports(paths ...string) map[string]*packages.Package {
	out := make(map[string]*packages.Package, len(paths))
	for _, p := range paths {
		out[p] = nil
	}

	return out
}

func files(dir string, names ...string) []string {
	out := make([]string, 0, len(names))
	for _, n := range names {
		out = append(out, filepath.Join(dir, n))
	}

	return out
}

func buildTestGraph(t *testing.T) *Graph {
	t.Helper()

	root := testRoot()
	dir := func(segments ...string) string {
		return filepath.Join(append([]string{root}, segments...)...)
	}

	g := newGraph(root)
	g.ingest([]*packages.Package{
		{
			ID:      "example.com/repo",
			PkgPath: "example.com/repo",
			GoFiles: files(dir("repo"), "repo.go"),
		},
		{
			ID:      "example.com/repo [example.com/repo.test]",
			PkgPath: "example.com/repo",
			GoFiles: files(dir("repo"), "repo.go", "repo_test.go"),
			Imports: imports("testing"),
		},
		{
			ID:      "example.com/repo.test",
			PkgPath: "example.com/repo.test",
			Name:    "main",
		},
		{
			ID:      "example.com/svc",
			PkgPath: "example.com/svc",
			GoFiles: files(dir("svc"), "svc.go"),
			Imports: imports("example.com/repo", "fmt"),
		},
		{
			ID:      "example.com/svc [example.com/svc.test]",
			PkgPath: "example.com/svc",
			GoFiles: files(dir("svc"), "svc.go", "svc_test.go"),
			Imports: imports("example.com/repo", "example.com/fixtures", "fmt", "testing"),
		},
		{
			ID:      "example.com/svc/internal/util",
			PkgPath: "example.com/svc/internal/util",
			GoFiles: files(dir("svc", "internal", "util"), "util.go"),
		},
		{
			ID:      "example.com/app",
			PkgPath: "example.com/app",
			GoFiles: files(dir("app"), "app.go"),
			Imports: imports("example.com/svc"),
		},
		{
			ID:      "example.com/app_test [example.com/app.test]",
			PkgPath: "example.com/app_test",
			GoFiles: files(dir("app"), "app_x_test.go"),
			Imports: imports("example.com/app", "testing"),
		},
		{
			ID:      "example.com/tool",
			PkgPath: "example.com/tool",
			GoFiles: files(dir("tool"), "main.go"),
			Imports: imports("example.com/svc"),
		},
		{
			ID:      "example.com/fixtures",
			PkgPath: "example.com/fixtures",
			GoFiles: files(dir("fixtures"), "fixtures.go"),
		},
		{
			ID:           "example.com/integrationonly",
			PkgPath:      "example.com/integrationonly",
			GoFiles:      files(dir("integrationonly"), "api.go"),
			IgnoredFiles: files(dir("integrationonly"), "api_integration_test.go"),
			Imports:      imports("example.com/repo"),
		},
	})
	g.index()

	return g
}

func TestTestVariantOwner(t *testing.T) {
	cases := []struct {
		id        string
		wantOwner string
		wantOK    bool
	}{
		{"example.com/svc [example.com/svc.test]", "example.com/svc", true},
		{"example.com/app_test [example.com/app.test]", "example.com/app", true},
		{"example.com/svc", "", false},
		{"example.com/svc.test", "", false},
		{"weird [not-a-binary]", "", false},
		{"[.test]", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			owner, ok := testVariantOwner(tc.id)
			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.wantOwner, owner)
		})
	}
}

func TestGeneratedTestMainIsSkipped(t *testing.T) {
	g := buildTestGraph(t)

	_, ok := g.Package("example.com/repo.test")
	assert.False(t, ok, "the generated test main must not enter the graph")
}

func TestTestVariantImportsExcludeProductionImports(t *testing.T) {
	g := buildTestGraph(t)

	pkg, ok := g.Package("example.com/svc")
	require.True(t, ok)
	assert.Equal(t, []string{"example.com/repo", "fmt"}, pkg.Imports)
	assert.Equal(t, []string{"example.com/fixtures", "testing"}, pkg.TestImports,
		"imports shared with production code must not be duplicated onto the test edge")
}

func TestExternalTestPackageAttachesToItsSubject(t *testing.T) {
	g := buildTestGraph(t)

	pkg, ok := g.Package("example.com/app")
	require.True(t, ok)
	assert.True(t, pkg.HasTests)
	assert.NotContains(t, pkg.TestImports, "example.com/app",
		"an external test package must not create a self edge")

	_, listed := g.Package("example.com/app_test")
	assert.False(t, listed, "the synthetic _test package must not enter the graph")
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
			require.True(t, ok, "expected %s to resolve to %s", tc.path, tc.want)
			assert.Equal(t, tc.want, pkg.ImportPath)
		})
	}
}

func TestPackageForFileRejectsSiblingPrefix(t *testing.T) {
	g := buildTestGraph(t)

	_, ok := g.PackageForFile(filepath.Join(testRoot(), "svcother", "x.go"))
	assert.False(t, ok, "a sibling directory sharing a name prefix must not match")

	_, ok = g.PackageForFile(filepath.Join(testRoot(), "unrelated", "x.go"))
	assert.False(t, ok)
}

func TestProductionClosureIsTransitive(t *testing.T) {
	g := buildTestGraph(t)

	closure := g.ProductionClosure([]string{"example.com/repo"})

	for _, want := range []string{"example.com/repo", "example.com/svc", "example.com/app", "example.com/tool"} {
		assert.True(t, closure[want], "closure missing %s", want)
	}
	assert.False(t, closure["example.com/fixtures"],
		"nothing imports fixtures from production code")
}

func TestAffectedTestPackagesSkipsPackagesWithoutTests(t *testing.T) {
	g := buildTestGraph(t)

	got := g.AffectedTestPackages([]string{"example.com/repo"})

	assert.Equal(t, []string{
		"example.com/app",
		"example.com/integrationonly",
		"example.com/repo",
		"example.com/svc",
	}, got)
	assert.NotContains(t, got, "example.com/tool", "tool has no test files")
}

func TestTestImportEdgesDoNotPropagate(t *testing.T) {
	g := buildTestGraph(t)

	got := g.AffectedTestPackages([]string{"example.com/fixtures"})

	assert.Equal(t, []string{"example.com/svc"}, got)
	assert.NotContains(t, got, "example.com/app",
		"app reaches fixtures only through svc's test code")
}

func TestBuildTagOnlyTestsCountAsTestable(t *testing.T) {
	g := buildTestGraph(t)

	pkg, ok := g.Package("example.com/integrationonly")
	require.True(t, ok)
	assert.True(t, pkg.HasTests,
		"a package whose only test file is behind a build tag is still testable")
}

func TestTestablePackagesIsSorted(t *testing.T) {
	g := buildTestGraph(t)

	got := g.TestablePackages()

	assert.IsIncreasing(t, got)
	assert.NotContains(t, got, "example.com/fixtures")
}

func TestTestOnlyPackageIsSynthesisedFromItsVariant(t *testing.T) {
	root := testRoot()
	g := newGraph(root)
	g.ingest([]*packages.Package{
		{
			ID:      "example.com/testonly [example.com/testonly.test]",
			PkgPath: "example.com/testonly",
			GoFiles: files(filepath.Join(root, "testonly"), "only_test.go"),
			Imports: imports("testing"),
		},
	})
	g.index()

	pkg, ok := g.Package("example.com/testonly")
	require.True(t, ok, "a package with only test files must still be reachable")
	assert.True(t, pkg.HasTests)
	assert.Equal(t, filepath.Join(root, "testonly"), pkg.Dir)
}

func TestSelfImportIsIgnored(t *testing.T) {
	root := testRoot()
	g := newGraph(root)
	g.ingest([]*packages.Package{
		{
			ID:      "example.com/solo",
			PkgPath: "example.com/solo",
			GoFiles: files(filepath.Join(root, "solo"), "solo.go"),
			Imports: imports("example.com/solo"),
		},
	})
	g.index()

	assert.Empty(t, g.prodImporters["example.com/solo"])
}

func TestPackageDirFallsBackToIgnoredFiles(t *testing.T) {
	dir := filepath.Join(testRoot(), "tagged")

	got := packageDir(&packages.Package{IgnoredFiles: files(dir, "only_tagged.go")})

	assert.Equal(t, dir, got)
}

func TestImportPathsDropsCgoPseudoImport(t *testing.T) {
	got := importPaths(&packages.Package{Imports: imports("b", "a", "C")})

	assert.Equal(t, []string{"a", "b"}, got)
}
