package selection_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/assay/internal/graph"
	"github.com/emoss08/assay/internal/selection"
	"github.com/emoss08/assay/internal/testfixture"
	"github.com/emoss08/assay/internal/vcs"
)

const mod = testfixture.Module

func loadFixture(t *testing.T, tags ...string) (*graph.Graph, string) {
	t.Helper()

	root := testfixture.Write(t)
	g, err := graph.Load(t.Context(), graph.LoadOptions{
		Root: root,
		Tags: tags,
		Env:  testfixture.Env(),
	})
	require.NoError(t, err)

	return g, root
}

func changed(root string, rels ...string) []vcs.Change {
	out := make([]vcs.Change, 0, len(rels))
	for _, rel := range rels {
		out = append(out, vcs.Change{Path: filepath.Join(root, filepath.FromSlash(rel)), Status: "M"})
	}

	return out
}

func selectFor(t *testing.T, g *graph.Graph, root string, rels ...string) selection.Result {
	t.Helper()

	return selection.Select(selection.Options{Graph: g, Changes: changed(root, rels...)})
}

func TestSelectPropagatesToDependents(t *testing.T) {
	g, root := loadFixture(t)

	res := selectFor(t, g, root, "repo/repo.go")

	assert.Equal(t, []string{mod + "/app", mod + "/repo", mod + "/svc", mod + "/tagged"}, res.Packages)
	assert.False(t, res.SelectAll)
	assert.Equal(t, selection.ReasonDirect, res.Reasons[mod+"/repo"])
	assert.Equal(t, selection.ReasonDependent, res.Reasons[mod+"/app"])
}

func TestSelectStopsAtTestOnlyEdges(t *testing.T) {
	g, root := loadFixture(t)

	res := selectFor(t, g, root, "fixtures/fixtures.go")

	assert.Equal(t, []string{mod + "/svc"}, res.Packages)
	assert.NotContains(t, res.Packages, mod+"/app",
		"app reaches fixtures only through svc's test code")
}

func TestSelectAttributesNonGoAssetsToOwningPackage(t *testing.T) {
	g, root := loadFixture(t)

	res := selectFor(t, g, root, "svc/testdata/seed.sql")

	assert.Equal(t, []string{mod + "/app", mod + "/svc"}, res.Packages)
	assert.Empty(t, res.Unattributed)
}

func TestSelectIgnoresFilesOutsideAnyPackage(t *testing.T) {
	g, root := loadFixture(t)

	res := selectFor(t, g, root, "docs/design.md")

	assert.Empty(t, res.Packages)
	assert.Len(t, res.Unattributed, 1)
	assert.False(t, res.SelectAll, "a stray document must not force a full run")
}

func TestSelectFallsBackToAllOnModuleChange(t *testing.T) {
	g, root := loadFixture(t)

	for _, trigger := range []string{"go.mod", "go.sum", "go.work", "go.work.sum"} {
		t.Run(trigger, func(t *testing.T) {
			res := selectFor(t, g, root, trigger)

			require.True(t, res.SelectAll)
			assert.Equal(t, g.TestablePackages(), res.Packages)
			assert.NotEmpty(t, res.SelectAllReason)
		})
	}
}

func TestSelectFallsBackToAllOnUnattributedGoFile(t *testing.T) {
	g, root := loadFixture(t)

	res := selectFor(t, g, root, "stray/orphan.go")

	require.True(t, res.SelectAll)
	assert.Contains(t, res.SelectAllReason, "orphan.go")
}

func TestSelectWithNoChangesSelectsNothing(t *testing.T) {
	g, _ := loadFixture(t)

	res := selection.Select(selection.Options{Graph: g})

	assert.Empty(t, res.Packages)
	assert.False(t, res.SelectAll)
}

func TestBuildTaggedTestPackageIsSelectableWithoutTheTag(t *testing.T) {
	g, _ := loadFixture(t)

	assert.Contains(t, g.TestablePackages(), mod+"/tagged",
		"a package whose only test file is behind a build tag must not vanish from selection")
}

func TestBuildTaggedTestPackageResolvesTestImportsWithTheTag(t *testing.T) {
	g, root := loadFixture(t, "integration")

	res := selectFor(t, g, root, "repo/repo.go")

	assert.Contains(t, res.Packages, mod+"/tagged")
}

func TestAllSelectsEveryTestablePackage(t *testing.T) {
	g, _ := loadFixture(t)

	res := selection.All(g, "because")

	require.True(t, res.SelectAll)
	assert.Equal(t, "because", res.SelectAllReason)
	assert.Equal(t, g.TestablePackages(), res.Packages)
	for _, pkg := range res.Packages {
		assert.Equal(t, selection.ReasonFallback, res.Reasons[pkg])
	}
}
