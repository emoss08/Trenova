package vcs

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

func TestParseNameStatusZ(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "repo")

	cases := []struct {
		name    string
		payload string
		want    []Change
	}{
		{
			name:    "single modification",
			payload: "M\x00svc/svc.go\x00",
			want:    []Change{{Path: filepath.Join(root, "svc/svc.go"), Status: "M"}},
		},
		{
			name:    "addition and deletion",
			payload: "A\x00a.go\x00D\x00b.go\x00",
			want: []Change{
				{Path: filepath.Join(root, "a.go"), Status: "A"},
				{Path: filepath.Join(root, "b.go"), Status: "D"},
			},
		},
		{
			name:    "rename yields both paths",
			payload: "R100\x00old/x.go\x00new/x.go\x00",
			want: []Change{
				{Path: filepath.Join(root, "old/x.go"), Status: "R100"},
				{Path: filepath.Join(root, "new/x.go"), Status: "R100"},
			},
		},
		{
			name:    "copy yields both paths",
			payload: "C75\x00src.go\x00dst.go\x00",
			want: []Change{
				{Path: filepath.Join(root, "src.go"), Status: "C75"},
				{Path: filepath.Join(root, "dst.go"), Status: "C75"},
			},
		},
		{
			name:    "truncated rename is dropped",
			payload: "M\x00a.go\x00R100\x00only-old.go\x00",
			want:    []Change{{Path: filepath.Join(root, "a.go"), Status: "M"}},
		},
		{
			name:    "empty payload",
			payload: "",
			want:    nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseNameStatusZ(tc.payload, root)
			if !slices.Equal(got, tc.want) {
				t.Errorf("parseNameStatusZ = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDedupeChangesSortsAndDrops(t *testing.T) {
	got := dedupeChanges([]Change{
		{Path: "/b", Status: "M"},
		{Path: "/a", Status: "A"},
		{Path: "/b", Status: "D"},
	})

	want := []Change{{Path: "/a", Status: "A"}, {Path: "/b", Status: "M"}}
	if !slices.Equal(got, want) {
		t.Errorf("dedupeChanges = %v, want %v", got, want)
	}
}

func initRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=assay", "GIT_AUTHOR_EMAIL=assay@example.com",
			"GIT_COMMITTER_NAME=assay", "GIT_COMMITTER_EMAIL=assay@example.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}

	write := func(rel, content string) {
		t.Helper()
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	run("init", "--initial-branch=main")
	write("base.go", "package base\n")
	run("add", ".")
	run("commit", "-m", "base")

	run("checkout", "-b", "feature")
	write("feature.go", "package feature\n")
	run("add", ".")
	run("commit", "-m", "feature")

	write("dirty.go", "package dirty\n")
	write("untracked.txt", "hello\n")
	run("add", "dirty.go")

	return root
}

func relPaths(t *testing.T, root string, changes []Change) []string {
	t.Helper()

	out := make([]string, 0, len(changes))
	for _, c := range changes {
		rel, err := filepath.Rel(root, c.Path)
		if err != nil {
			t.Fatalf("rel: %v", err)
		}
		out = append(out, filepath.ToSlash(rel))
	}
	slices.Sort(out)

	return out
}

func TestChangesAgainstMergeBase(t *testing.T) {
	root := initRepo(t)

	got, err := Changes(context.Background(), Options{Root: root, Base: "main", IncludeUntracked: true})
	if err != nil {
		t.Fatalf("Changes: %v", err)
	}

	want := []string{"dirty.go", "feature.go", "untracked.txt"}
	if diff := relPaths(t, root, got); !slices.Equal(diff, want) {
		t.Errorf("Changes = %v, want %v", diff, want)
	}
}

func TestChangesWithoutBaseSeesWorkingTreeOnly(t *testing.T) {
	root := initRepo(t)

	got, err := Changes(context.Background(), Options{Root: root, IncludeUntracked: true})
	if err != nil {
		t.Fatalf("Changes: %v", err)
	}

	want := []string{"dirty.go", "untracked.txt"}
	if diff := relPaths(t, root, got); !slices.Equal(diff, want) {
		t.Errorf("Changes = %v, want %v", diff, want)
	}
}

func TestChangesExcludesUntrackedWhenDisabled(t *testing.T) {
	root := initRepo(t)

	got, err := Changes(context.Background(), Options{Root: root, Base: "main"})
	if err != nil {
		t.Fatalf("Changes: %v", err)
	}

	if diff := relPaths(t, root, got); slices.Contains(diff, "untracked.txt") {
		t.Errorf("Changes = %v, want no untracked files", diff)
	}
}

func TestRepoRootResolvesFromSubdirectory(t *testing.T) {
	root := initRepo(t)
	sub := filepath.Join(root, "nested", "deep")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	got, err := RepoRoot(context.Background(), sub)
	if err != nil {
		t.Fatalf("RepoRoot: %v", err)
	}

	wantResolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("eval symlinks: %v", err)
	}
	gotResolved, err := filepath.EvalSymlinks(got)
	if err != nil {
		t.Fatalf("eval symlinks: %v", err)
	}
	if gotResolved != wantResolved {
		t.Errorf("RepoRoot = %q, want %q", gotResolved, wantResolved)
	}
}
