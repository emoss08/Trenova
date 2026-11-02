package main

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"slices"
	"strings"
)

const (
	todoReason  = "TODO reason"
	allowFields = 3
	allowHeader = "# genericlint allowlist: exported API that may name an instantiated generic type\n" +
		"# whose methods every importer compiles again. One entry per line:\n" +
		"# <package path> <identifier> <origin type> # reason\n" +
		"# The package path and identifier may be *, to accept an origin type everywhere.\n" +
		"# See docs/engineering/module-architecture.md#generics-and-build-time.\n"
)

type allowEntry struct {
	line   int
	key    string
	reason string
	used   bool
}

type allowlist struct {
	path     string
	entries  []*allowEntry
	byKey    map[string]*allowEntry
	problems []string
}

func readAllow(path string) (*allowlist, error) {
	a := &allowlist{path: path, byKey: make(map[string]*allowEntry)}
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return a, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	for i, line := range strings.Split(string(raw), "\n") {
		a.parse(i+1, strings.TrimSpace(line))
	}

	return a, nil
}

func (a *allowlist) parse(lineNo int, line string) {
	if line == "" || strings.HasPrefix(line, "#") {
		return
	}
	spec, reason, _ := strings.Cut(line, "#")
	fields := strings.Fields(spec)
	if len(fields) != allowFields {
		a.problems = append(a.problems, fmt.Sprintf(
			"%s:%d: want `<package path> <identifier> <origin type> # reason`",
			a.path,
			lineNo,
		))
		return
	}
	key := strings.Join(fields, " ")
	reason = strings.TrimSpace(reason)
	if reason == "" {
		a.problems = append(a.problems, fmt.Sprintf("%s:%d: %s has no reason", a.path, lineNo, key))
	}
	if prev, ok := a.byKey[key]; ok {
		a.problems = append(a.problems, fmt.Sprintf(
			"%s:%d: %s repeats line %d",
			a.path,
			lineNo,
			key,
			prev.line,
		))
		return
	}
	entry := &allowEntry{line: lineNo, key: key, reason: reason}
	a.entries = append(a.entries, entry)
	a.byKey[key] = entry
}

func (a *allowlist) match(key string) *allowEntry {
	fields := strings.Fields(key)
	if len(fields) != allowFields {
		return a.byKey[key]
	}
	for _, k := range []string{
		key,
		fields[0] + " * " + fields[2],
		"* * " + fields[2],
	} {
		if entry, ok := a.byKey[k]; ok {
			return entry
		}
	}
	return nil
}

func (a *allowlist) filter(findings []finding) []string {
	problems := slices.Clone(a.problems)
	for _, f := range findings {
		if entry := a.match(f.key()); entry != nil {
			entry.used = true
			continue
		}
		problems = append(problems, f.String())
	}
	for _, entry := range a.entries {
		if !entry.used {
			problems = append(problems, fmt.Sprintf(
				"%s:%d: %s matches nothing; remove it (or rewrite the file with -write-allow)",
				a.path,
				entry.line,
				entry.key,
			))
		}
	}

	return problems
}

func (a *allowlist) write(findings []finding) (int, error) {
	reasons := make(map[string]string, len(findings))
	for _, f := range findings {
		key, reason := f.key(), todoReason
		if entry := a.match(key); entry != nil {
			key = entry.key
			if entry.reason != "" {
				reason = entry.reason
			}
		}
		reasons[key] = reason
	}

	var b strings.Builder
	b.WriteString(allowHeader)
	for _, key := range slices.Sorted(maps.Keys(reasons)) {
		fmt.Fprintf(&b, "%s # %s\n", key, reasons[key])
	}
	if err := os.WriteFile(a.path, []byte(b.String()), 0o600); err != nil {
		return 0, fmt.Errorf("write %s: %w", a.path, err)
	}

	return len(reasons), nil
}
