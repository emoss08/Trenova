package selection_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/emoss08/assay/internal/graph"
	"github.com/emoss08/assay/internal/selection"
	"github.com/emoss08/assay/internal/vcs"
)

const fixtureModule = "example.com/fixture"

var fixtureFiles = map[string]string{
	"go.mod": "module " + fixtureModule + "\n\ngo 1.26\n",

	"repo/repo.go":      "package repo\n\nfunc Get() int { return 1 }\n",
	"repo/repo_test.go": "package repo\n\nimport \"testing\"\n\nfunc TestGet(t *testing.T) { _ = Get() }\n",

	"svc/svc.go": "package svc\n\nimport \"" + fixtureModule + "/repo\"\n\nfunc Do() int { return repo.Get() }\n",
	"svc/svc_test.go": "package svc\n\nimport (\n\t\"testing\"\n\n\t\"" + fixtureModule +
		"/fixtures\"\n)\n\nfunc TestDo(t *testing.T) { _ = Do(); _ = fixtures.Seed() }\n",
	"svc/testdata/seed.sql": "select 1;\n",

	"app/app.go":      "package app\n\nimport \"" + fixtureModule + "/svc\"\n\nfunc Run() int { return svc.Do() }\n",
	"app/app_test.go": "package app\n\nimport \"testing\"\n\nfunc TestRun(t *testing.T) { _ = Run() }\n",

	"tool/tool.go": "package tool\n\nimport \"" + fixtureModule + "/svc\"\n\nfunc Main() int { return svc.Do() }\n",

	"fixtures/fixtures.go": "package fixtures\n\nfunc Seed() int { return 7 }\n",

	"docs/design.md": "# design\n",
}

func loadFixture(t *testing.T) (*graph.Graph, string) {
	t.Helper()

	root := t.TempDir()
	for rel, content := range fixtureFiles {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", full, err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", full, err)
		}
	}

	g, err := graph.Load(t.Context(), graph.LoadOptions{
		Root: root,
		Env:  append(os.Environ(), "GOWORK=off"),
	})
	if err != nil {
		t.Fatalf("graph.Load: %v", err)
	}

	return g, root
}

func changed(root string, rels ...string) []vcs.Change {
	out := make([]vcs.Change, 0, len(rels))
	for _, rel := range rels {
		out = append(out, vcs.Change{Path: filepath.Join(root, filepath.FromSlash(rel)), Status: "M"})
	}

	return out
}

func TestSelectPropagatesToDependents(t *testing.T) {
	g, root := loadFixture(t)

	res := selection.Select(selection.Options{Graph: g, Changes: changed(root, "repo/repo.go")})

	want := []string{fixtureModule + "/app", fixtureModule + "/repo", fixtureModule + "/svc"}
	if !slices.Equal(res.Packages, want) {
		t.Fatalf("Packages = %v, want %v", res.Packages, want)
	}
	if res.SelectAll {
		t.Error("SelectAll should be false for an ordinary source change")
	}
	if res.Reasons[fixtureModule+"/repo"] != selection.ReasonDirect {
		t.Errorf("repo reason = %q, want %q", res.Reasons[fixtureModule+"/repo"], selection.ReasonDirect)
	}
	if res.Reasons[fixtureModule+"/app"] != selection.ReasonDependent {
		t.Errorf("app reason = %q, want %q", res.Reasons[fixtureModule+"/app"], selection.ReasonDependent)
	}
}

func TestSelectStopsAtTestOnlyEdges(t *testing.T) {
	g, root := loadFixture(t)

	res := selection.Select(selection.Options{Graph: g, Changes: changed(root, "fixtures/fixtures.go")})

	want := []string{fixtureModule + "/svc"}
	if !slices.Equal(res.Packages, want) {
		t.Fatalf("Packages = %v, want %v", res.Packages, want)
	}
}

func TestSelectAttributesNonGoAssetsToOwningPackage(t *testing.T) {
	g, root := loadFixture(t)

	res := selection.Select(selection.Options{Graph: g, Changes: changed(root, "svc/testdata/seed.sql")})

	want := []string{fixtureModule + "/app", fixtureModule + "/svc"}
	if !slices.Equal(res.Packages, want) {
		t.Fatalf("Packages = %v, want %v", res.Packages, want)
	}
	if len(res.Unattributed) != 0 {
		t.Errorf("Unattributed = %v, want empty", res.Unattributed)
	}
}

func TestSelectIgnoresFilesOutsideAnyPackage(t *testing.T) {
	g, root := loadFixture(t)

	res := selection.Select(selection.Options{Graph: g, Changes: changed(root, "docs/design.md")})

	if len(res.Packages) != 0 {
		t.Errorf("Packages = %v, want empty", res.Packages)
	}
	if len(res.Unattributed) != 1 {
		t.Fatalf("Unattributed = %v, want exactly one entry", res.Unattributed)
	}
}

func TestSelectFallsBackToAllOnModuleChange(t *testing.T) {
	g, root := loadFixture(t)

	for _, trigger := range []string{"go.mod", "go.sum", "go.work", "go.work.sum"} {
		t.Run(trigger, func(t *testing.T) {
			res := selection.Select(selection.Options{Graph: g, Changes: changed(root, trigger)})

			if !res.SelectAll {
				t.Fatalf("SelectAll = false for %s, want true", trigger)
			}
			if !slices.Equal(res.Packages, g.TestablePackages()) {
				t.Errorf("Packages = %v, want every testable package", res.Packages)
			}
			if res.SelectAllReason == "" {
				t.Error("SelectAllReason must explain the fallback")
			}
		})
	}
}

func TestSelectFallsBackToAllOnUnattributedGoFile(t *testing.T) {
	g, root := loadFixture(t)

	res := selection.Select(selection.Options{Graph: g, Changes: changed(root, "stray/orphan.go")})

	if !res.SelectAll {
		t.Fatal("a Go file outside every known package must trigger the select-all fallback")
	}
}

func TestSelectWithNoChangesSelectsNothing(t *testing.T) {
	g, _ := loadFixture(t)

	res := selection.Select(selection.Options{Graph: g})

	if len(res.Packages) != 0 {
		t.Errorf("Packages = %v, want empty", res.Packages)
	}
	if res.SelectAll {
		t.Error("SelectAll should be false when nothing changed")
	}
}

func TestAllSelectsEveryTestablePackage(t *testing.T) {
	g, _ := loadFixture(t)

	res := selection.All(g, "because")

	if !res.SelectAll || res.SelectAllReason != "because" {
		t.Fatalf("All() = %+v, want SelectAll with the given reason", res)
	}
	if !slices.Equal(res.Packages, g.TestablePackages()) {
		t.Errorf("Packages = %v, want every testable package", res.Packages)
	}
	for _, pkg := range res.Packages {
		if res.Reasons[pkg] != selection.ReasonFallback {
			t.Errorf("%s reason = %q, want %q", pkg, res.Reasons[pkg], selection.ReasonFallback)
		}
	}
}
