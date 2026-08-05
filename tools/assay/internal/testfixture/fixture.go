package testfixture

import (
	"os"
	"path/filepath"
	"testing"
)

const Module = "example.com/fixture"

var files = map[string]string{
	"go.mod": "module " + Module + "\n\ngo 1.26\n",

	"repo/repo.go":      "package repo\n\nfunc Get() int { return 1 }\n",
	"repo/repo_test.go": "package repo\n\nimport \"testing\"\n\nfunc TestGet(t *testing.T) { _ = Get() }\n",

	"svc/svc.go": "package svc\n\nimport \"" + Module + "/repo\"\n\nfunc Do() int { return repo.Get() }\n",
	"svc/svc_test.go": "package svc\n\nimport (\n\t\"testing\"\n\n\t\"" + Module +
		"/fixtures\"\n)\n\nfunc TestDo(t *testing.T) { _ = Do(); _ = fixtures.Seed() }\n",
	"svc/testdata/seed.sql": "select 1;\n",

	"app/app.go": "package app\n\nimport \"" + Module + "/svc\"\n\nfunc Run() int { return svc.Do() }\n",
	"app/app_x_test.go": "package app_test\n\nimport (\n\t\"testing\"\n\n\t\"" + Module +
		"/app\"\n)\n\nfunc TestRun(t *testing.T) { _ = app.Run() }\n",

	"tool/tool.go": "package tool\n\nimport \"" + Module + "/svc\"\n\nfunc Main() int { return svc.Do() }\n",

	"fixtures/fixtures.go": "package fixtures\n\nfunc Seed() int { return 7 }\n",

	"tagged/tagged.go": "package tagged\n\nimport \"" + Module + "/repo\"\n\nfunc Use() int { return repo.Get() }\n",
	"tagged/tagged_integration_test.go": "//go:build integration\n\npackage tagged\n\nimport \"testing\"" +
		"\n\nfunc TestUse(t *testing.T) { _ = Use() }\n",

	"docs/design.md": "# design\n",
}

func Write(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	for rel, content := range files {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", full, err)
		}
	}

	return root
}

func Env() []string {
	return append(os.Environ(), "GOWORK=off")
}
