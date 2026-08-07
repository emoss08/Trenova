package vcs

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ChurnStat is one file's recent change activity.
type ChurnStat struct {
	Commits      int   `json:"commits"`
	LinesChanged int   `json:"linesChanged"`
	LastUnix     int64 `json:"lastUnix"`
}

// churnMarker separates commits in the log output. A control byte, because
// commit subjects can contain anything printable.
const churnMarker = "\x01"

// Churn reports per-file change activity over the trailing window, keyed by
// slash-separated path relative to the repository root. Only production Go
// files count: churn in tests is not production exposure, and churn in
// non-Go files is not this tool's business.
func Churn(ctx context.Context, root string, window time.Duration) (map[string]ChurnStat, error) {
	since := fmt.Sprintf("--since=%d.seconds.ago", int64(window.Seconds()))
	out, err := runGit(ctx, root,
		"log", since, "--numstat", "--no-renames", "--format="+churnMarker+"%at", "--", ".")
	if err != nil {
		return nil, fmt.Errorf("read churn: %w", err)
	}

	churn := make(map[string]ChurnStat)
	var commitUnix int64
	seenThisCommit := make(map[string]struct{})

	for line := range strings.Lines(string(out)) {
		line = strings.TrimRight(line, "\n")
		if line == "" {
			continue
		}

		if rest, isCommit := strings.CutPrefix(line, churnMarker); isCommit {
			commitUnix, _ = strconv.ParseInt(strings.TrimSpace(rest), 10, 64)
			clear(seenThisCommit)

			continue
		}

		added, deleted, path, ok := parseNumstat(line)
		if !ok || !isProductionGoFile(path) {
			continue
		}

		stat := churn[path]
		if _, counted := seenThisCommit[path]; !counted {
			seenThisCommit[path] = struct{}{}
			stat.Commits++
		}
		stat.LinesChanged += added + deleted
		if commitUnix > stat.LastUnix {
			stat.LastUnix = commitUnix
		}
		churn[path] = stat
	}

	return churn, nil
}

// parseNumstat reads one "added<TAB>deleted<TAB>path" line. Binary files
// report "-" for both counts and still count as a change.
func parseNumstat(line string) (added, deleted int, path string, ok bool) {
	fields := strings.SplitN(line, "\t", 3)
	if len(fields) != 3 {
		return 0, 0, "", false
	}

	if fields[0] != "-" {
		added, _ = strconv.Atoi(fields[0])
	}
	if fields[1] != "-" {
		deleted, _ = strconv.Atoi(fields[1])
	}
	path = strings.TrimSpace(fields[2])
	if path == "" {
		return 0, 0, "", false
	}

	return added, deleted, path, true
}

func isProductionGoFile(path string) bool {
	return strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go")
}
