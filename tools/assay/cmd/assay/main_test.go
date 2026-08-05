package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestReadFileListResolvesRelativePaths(t *testing.T) {
	root := t.TempDir()
	list := filepath.Join(root, "changed.txt")
	absolute := filepath.Join(root, "already", "absolute.go")

	content := "svc/svc.go\n\n  repo/repo.go  \n" + absolute + "\n"
	if err := os.WriteFile(list, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := readFileList(list, root)
	if err != nil {
		t.Fatalf("readFileList: %v", err)
	}

	want := []string{
		filepath.Join(root, "svc", "svc.go"),
		filepath.Join(root, "repo", "repo.go"),
		absolute,
	}
	paths := make([]string, 0, len(got))
	for _, c := range got {
		paths = append(paths, c.Path)
	}
	if !slices.Equal(paths, want) {
		t.Errorf("paths = %v, want %v", paths, want)
	}
}

func TestReadFileListRejectsMissingFile(t *testing.T) {
	if _, err := readFileList(filepath.Join(t.TempDir(), "nope.txt"), t.TempDir()); err == nil {
		t.Fatal("readFileList should fail when the list does not exist")
	}
}

func TestParseFlagsSplitsGoTestArguments(t *testing.T) {
	cfg, err := parseFlags("run", []string{"--since", "origin/main", "-v", "--", "-count=1", "-race"})
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}

	if cfg.since != "origin/main" {
		t.Errorf("since = %q, want origin/main", cfg.since)
	}
	if !cfg.verbose {
		t.Error("verbose = false, want true")
	}
	if !slices.Equal(cfg.testArgs, []string{"-count=1", "-race"}) {
		t.Errorf("testArgs = %v, want [-count=1 -race]", cfg.testArgs)
	}
}

func TestParseFlagsSplitsTags(t *testing.T) {
	cfg, err := parseFlags("run", []string{"--tags", "integration, slow ,"})
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}

	if !slices.Equal(cfg.tags, []string{"integration", "slow"}) {
		t.Errorf("tags = %v, want [integration slow]", cfg.tags)
	}
}

func TestParseFlagsRejectsStrayArguments(t *testing.T) {
	if _, err := parseFlags("run", []string{"./..."}); err == nil {
		t.Fatal("parseFlags should reject positional arguments without --")
	}
}

func TestParseFlagsAcceptsNoArguments(t *testing.T) {
	cfg, err := parseFlags("select", nil)
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}

	if cfg.since != "" || cfg.all || cfg.verbose || len(cfg.testArgs) != 0 {
		t.Errorf("cfg = %+v, want zero values", cfg)
	}
}
