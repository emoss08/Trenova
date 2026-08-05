package vcs

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type LineRange struct {
	Start int
	End   int
}

func (r LineRange) Contains(line int) bool {
	return line >= r.Start && line <= r.End
}

type Change struct {
	Path   string
	Status string
	Lines  []LineRange
}

func (c Change) WholeFile() bool {
	return len(c.Lines) == 0
}

func (c Change) Touches(start, end int) bool {
	for _, r := range c.Lines {
		if start <= r.End && end >= r.Start {
			return true
		}
	}

	return false
}

type BaseMode string

const (
	BaseWorkingTree BaseMode = "working-tree"
	BaseMergeBase   BaseMode = "merge-base"
	BaseTwoDot      BaseMode = "two-dot"
)

type Options struct {
	Root             string
	Base             string
	IncludeUntracked bool
}

type Result struct {
	Changes  []Change
	BaseMode BaseMode
	Note     string
}

func RepoRoot(ctx context.Context, dir string) (string, error) {
	out, err := runGit(ctx, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

func IsShallow(ctx context.Context, root string) bool {
	out, err := runGit(ctx, root, "rev-parse", "--is-shallow-repository")
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(out)) == "true"
}

func Changes(ctx context.Context, opts Options) (Result, error) {
	root, err := filepath.Abs(opts.Root)
	if err != nil {
		return Result{}, fmt.Errorf("resolve root: %w", err)
	}

	spec, mode, note, err := resolveBase(ctx, root, opts.Base)
	if err != nil {
		return Result{}, err
	}

	statusOut, err := runGit(ctx, root, append([]string{"diff", "--name-status", "-z"}, spec...)...)
	if err != nil {
		return Result{}, fmt.Errorf("git diff: %w", err)
	}
	changes := parseNameStatusZ(string(statusOut), root)

	applyHunks(ctx, root, spec, changes)

	if opts.IncludeUntracked {
		untracked, utErr := runGit(ctx, root, "ls-files", "--others", "--exclude-standard", "-z")
		if utErr != nil {
			return Result{}, fmt.Errorf("git ls-files: %w", utErr)
		}
		for _, rel := range splitZ(string(untracked)) {
			changes = append(changes, Change{Path: filepath.Join(root, rel), Status: "?"})
		}
	}

	return Result{Changes: dedupeChanges(changes), BaseMode: mode, Note: note}, nil
}

func resolveBase(ctx context.Context, root, base string) ([]string, BaseMode, string, error) {
	if base == "" {
		return []string{"HEAD"}, BaseWorkingTree, "", nil
	}

	if _, err := runGit(ctx, root, "rev-parse", "--verify", "--quiet", base+"^{commit}"); err != nil {
		hint := "check the ref name"
		if IsShallow(ctx, root) {
			hint = "this is a shallow clone; fetch the ref (git fetch --depth=... origin <branch>) " +
				"or pass --files with a precomputed change list"
		}

		return nil, "", "", fmt.Errorf("cannot resolve base ref %q: %s", base, hint)
	}

	if _, err := runGit(ctx, root, "merge-base", base, "HEAD"); err == nil {
		return []string{"--merge-base", base}, BaseMergeBase, "", nil
	}

	note := fmt.Sprintf("no merge base with %q; comparing directly, which over-selects", base)
	if IsShallow(ctx, root) {
		note = fmt.Sprintf("no merge base with %q in this shallow clone; comparing directly, "+
			"which over-selects (git fetch --deepen to narrow)", base)
	}

	return []string{base}, BaseTwoDot, note, nil
}

func applyHunks(ctx context.Context, root string, spec []string, changes []Change) {
	args := []string{"-c", "core.quotePath=false", "diff", "-U0", "--no-color", "--no-renames"}
	out, err := runGit(ctx, root, append(args, spec...)...)
	if err != nil {
		return
	}

	ranges := parseUnifiedHunks(string(out), root)
	for i := range changes {
		if isWholeFileStatus(changes[i].Status) {
			continue
		}
		if found, ok := ranges[changes[i].Path]; ok {
			changes[i].Lines = found
		}
	}
}

func isWholeFileStatus(status string) bool {
	if status == "" {
		return true
	}
	switch status[0] {
	case 'A', 'D', 'R', 'C', '?':
		return true
	default:
		return false
	}
}

func parseUnifiedHunks(payload, root string) map[string][]LineRange {
	out := make(map[string][]LineRange)

	var current string
	for line := range strings.SplitSeq(payload, "\n") {
		switch {
		case strings.HasPrefix(line, "+++ "):
			current = newFilePath(line, root)
		case strings.HasPrefix(line, "@@ ") && current != "":
			if r, ok := parseHunkHeader(line); ok {
				out[current] = append(out[current], r)
			}
		case strings.HasPrefix(line, "diff --git "):
			current = ""
		}
	}

	return out
}

func newFilePath(line, root string) string {
	target := strings.TrimPrefix(line, "+++ ")
	if target == "/dev/null" {
		return ""
	}
	target = strings.TrimPrefix(target, "b/")
	if target == "" {
		return ""
	}

	return filepath.Join(root, filepath.FromSlash(target))
}

func parseHunkHeader(line string) (LineRange, bool) {
	rest, ok := strings.CutPrefix(line, "@@ ")
	if !ok {
		return LineRange{}, false
	}
	end := strings.Index(rest, " @@")
	if end < 0 {
		return LineRange{}, false
	}

	var added string
	for field := range strings.SplitSeq(rest[:end], " ") {
		if strings.HasPrefix(field, "+") {
			added = strings.TrimPrefix(field, "+")

			break
		}
	}
	if added == "" {
		return LineRange{}, false
	}

	start, count := added, "1"
	if comma := strings.IndexByte(added, ','); comma >= 0 {
		start, count = added[:comma], added[comma+1:]
	}

	startLine, err := strconv.Atoi(start)
	if err != nil {
		return LineRange{}, false
	}
	length, err := strconv.Atoi(count)
	if err != nil {
		return LineRange{}, false
	}

	if length == 0 {
		return LineRange{Start: max(1, startLine), End: startLine + 1}, true
	}

	return LineRange{Start: startLine, End: startLine + length - 1}, true
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
