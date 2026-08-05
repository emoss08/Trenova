package vcs

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type Change struct {
	Path   string
	Status string
}

type Options struct {
	Root             string
	Base             string
	IncludeUntracked bool
}

func RepoRoot(ctx context.Context, dir string) (string, error) {
	out, err := runGit(ctx, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

func Changes(ctx context.Context, opts Options) ([]Change, error) {
	root, err := filepath.Abs(opts.Root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}

	args := []string{"diff", "--name-status", "-z"}
	if opts.Base != "" {
		args = append(args, "--merge-base", opts.Base)
	} else {
		args = append(args, "HEAD")
	}

	out, err := runGit(ctx, root, args...)
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}

	changes := parseNameStatusZ(string(out), root)

	if opts.IncludeUntracked {
		untracked, utErr := runGit(ctx, root, "ls-files", "--others", "--exclude-standard", "-z")
		if utErr != nil {
			return nil, fmt.Errorf("git ls-files: %w", utErr)
		}
		for _, rel := range splitZ(string(untracked)) {
			changes = append(changes, Change{Path: filepath.Join(root, rel), Status: "?"})
		}
	}

	return dedupeChanges(changes), nil
}

func parseNameStatusZ(payload, root string) []Change {
	fields := splitZ(payload)

	var changes []Change
	for i := 0; i < len(fields); {
		status := fields[i]
		if status == "" {
			i++

			continue
		}

		switch status[0] {
		case 'R', 'C':
			if i+2 >= len(fields) {
				return changes
			}
			changes = append(changes,
				Change{Path: filepath.Join(root, fields[i+1]), Status: status},
				Change{Path: filepath.Join(root, fields[i+2]), Status: status},
			)
			i += 3
		default:
			if i+1 >= len(fields) {
				return changes
			}
			changes = append(changes, Change{Path: filepath.Join(root, fields[i+1]), Status: status})
			i += 2
		}
	}

	return changes
}

func splitZ(payload string) []string {
	parts := strings.Split(payload, "\x00")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}

	return out
}

func dedupeChanges(in []Change) []Change {
	seen := make(map[string]struct{}, len(in))
	out := make([]Change, 0, len(in))
	for _, c := range in {
		if _, ok := seen[c.Path]; ok {
			continue
		}
		seen[c.Path] = struct{}{}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })

	return out
}

func runGit(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}

	return stdout.Bytes(), nil
}
