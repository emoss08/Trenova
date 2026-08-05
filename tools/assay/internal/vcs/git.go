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
	Path      string
	Status    string
	Lines     []LineRange
	BaseLines []LineRange
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

func (c Change) LineNumbers() []int {
	return expand(c.Lines)
}

func (c Change) BaseLineNumbers() []int {
	return expand(c.BaseLines)
}

func expand(ranges []LineRange) []int {
	var out []int
	for _, r := range ranges {
		for line := r.Start; line <= r.End; line++ {
			out = append(out, line)
		}
	}

	return out
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
	Changes    []Change
	BaseMode   BaseMode
	BaseCommit string
	Note       string
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

	base, err := resolveBase(ctx, root, opts.Base)
	if err != nil {
		return Result{}, err
	}

	statusOut, err := runGit(ctx, root, append([]string{"diff", "--name-status", "-z"}, base.spec...)...)
	if err != nil {
		return Result{}, fmt.Errorf("git diff: %w", err)
	}
	changes := parseNameStatusZ(string(statusOut), root)

	applyHunks(ctx, root, base.spec, changes)

	if opts.IncludeUntracked {
		untracked, utErr := runGit(ctx, root, "ls-files", "--others", "--exclude-standard", "-z")
		if utErr != nil {
			return Result{}, fmt.Errorf("git ls-files: %w", utErr)
		}
		for _, rel := range splitZ(string(untracked)) {
			changes = append(changes, Change{Path: filepath.Join(root, rel), Status: "?"})
		}
	}

	return Result{
		Changes:    dedupeChanges(changes),
		BaseMode:   base.mode,
		BaseCommit: base.commit,
		Note:       base.note,
	}, nil
}

type baseSpec struct {
	spec   []string
	mode   BaseMode
	commit string
	note   string
}

func resolveBase(ctx context.Context, root, base string) (baseSpec, error) {
	if base == "" {
		head, err := runGit(ctx, root, "rev-parse", "HEAD")
		if err != nil {
			return baseSpec{}, fmt.Errorf("resolve HEAD: %w", err)
		}

		return baseSpec{
			spec:   []string{"HEAD"},
			mode:   BaseWorkingTree,
			commit: strings.TrimSpace(string(head)),
		}, nil
	}

	resolved, err := runGit(ctx, root, "rev-parse", "--verify", "--quiet", base+"^{commit}")
	if err != nil {
		hint := "check the ref name"
		if IsShallow(ctx, root) {
			hint = "this is a shallow clone; fetch the ref (git fetch --depth=... origin <branch>) " +
				"or pass --files with a precomputed change list"
		}

		return baseSpec{}, fmt.Errorf("cannot resolve base ref %q: %s", base, hint)
	}

	if mergeBase, mbErr := runGit(ctx, root, "merge-base", base, "HEAD"); mbErr == nil {
		return baseSpec{
			spec:   []string{"--merge-base", base},
			mode:   BaseMergeBase,
			commit: strings.TrimSpace(string(mergeBase)),
		}, nil
	}

	note := fmt.Sprintf("no merge base with %q; comparing directly, which over-selects", base)
	if IsShallow(ctx, root) {
		note = fmt.Sprintf("no merge base with %q in this shallow clone; comparing directly, "+
			"which over-selects (git fetch --deepen to narrow)", base)
	}

	return baseSpec{
		spec:   []string{base},
		mode:   BaseTwoDot,
		commit: strings.TrimSpace(string(resolved)),
		note:   note,
	}, nil
}

// IsIgnored reports whether git would ignore path. Worth checking before writing
// anything meant to be committed: a stray rule like `coverage/` in a root
// .gitignore can silently swallow a whole directory, and the failure is invisible
// until someone checks out the result.
func IsIgnored(ctx context.Context, root, path string) bool {
	_, err := runGit(ctx, root, "check-ignore", "-q", path)

	return err == nil
}

func TreeDigests(ctx context.Context, root, commit string) (map[string]string, error) {
	out, err := runGit(ctx, root,
		"-c", "core.quotePath=false", "ls-tree", "-r", "-z",
		"--format=%(objectname) %(path)", commit)
	if err != nil {
		return nil, fmt.Errorf("list tree for %s: %w", commit, err)
	}

	digests := make(map[string]string)
	for entry := range strings.SplitSeq(string(out), "\x00") {
		if entry == "" {
			continue
		}
		space := strings.IndexByte(entry, ' ')
		if space <= 0 || space == len(entry)-1 {
			continue
		}
		digests[entry[space+1:]] = entry[:space]
	}

	return digests, nil
}

func DirtyPaths(ctx context.Context, root string) ([]string, error) {
	out, err := runGit(ctx, root, "status", "--porcelain=v1", "-z", "--untracked-files=normal")
	if err != nil {
		return nil, fmt.Errorf("git status: %w", err)
	}

	var dirty []string
	for entry := range strings.SplitSeq(string(out), "\x00") {
		if len(entry) < 4 {
			continue
		}
		path := entry[3:]
		if strings.HasSuffix(path, ".go") || isModuleFileName(filepath.Base(path)) {
			dirty = append(dirty, path)
		}
	}
	sort.Strings(dirty)

	return dirty, nil
}

func isModuleFileName(name string) bool {
	switch name {
	case "go.mod", "go.sum", "go.work", "go.work.sum":
		return true
	default:
		return false
	}
}

func HeadCommit(ctx context.Context, root string) (string, error) {
	out, err := runGit(ctx, root, "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("resolve HEAD: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
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
			changes[i].Lines = found.new
			changes[i].BaseLines = found.base
		}
	}
}

type hunkSides struct {
	base []LineRange
	new  []LineRange
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

func parseUnifiedHunks(payload, root string) map[string]hunkSides {
	out := make(map[string]hunkSides)

	var current string
	for line := range strings.SplitSeq(payload, "\n") {
		switch {
		case strings.HasPrefix(line, "+++ "):
			current = newFilePath(line, root)
		case strings.HasPrefix(line, "@@ ") && current != "":
			base, added, ok := parseHunkSides(line)
			if !ok {
				continue
			}
			sides := out[current]
			sides.base = append(sides.base, base)
			sides.new = append(sides.new, added)
			out[current] = sides
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
	_, added, ok := parseHunkSides(line)

	return added, ok
}

func parseHunkSides(line string) (base, added LineRange, ok bool) {
	rest, found := strings.CutPrefix(line, "@@ ")
	if !found {
		return LineRange{}, LineRange{}, false
	}
	end := strings.Index(rest, " @@")
	if end < 0 {
		return LineRange{}, LineRange{}, false
	}

	var oldSpec, newSpec string
	for field := range strings.SplitSeq(rest[:end], " ") {
		switch {
		case strings.HasPrefix(field, "-") && oldSpec == "":
			oldSpec = strings.TrimPrefix(field, "-")
		case strings.HasPrefix(field, "+") && newSpec == "":
			newSpec = strings.TrimPrefix(field, "+")
		}
	}
	if oldSpec == "" || newSpec == "" {
		return LineRange{}, LineRange{}, false
	}

	base, ok = parseSpec(oldSpec)
	if !ok {
		return LineRange{}, LineRange{}, false
	}
	added, ok = parseSpec(newSpec)
	if !ok {
		return LineRange{}, LineRange{}, false
	}

	return base, added, true
}

func parseSpec(spec string) (LineRange, bool) {
	start, count := spec, "1"
	if comma := strings.IndexByte(spec, ','); comma >= 0 {
		start, count = spec[:comma], spec[comma+1:]
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
