package cli

import "fmt"

// summariseReasons dedupes a stream of reason strings, keeps the first few in
// arrival order, and folds the rest into a count. Empty strings mean "no
// reason" and are skipped. Both mutate and index report fallbacks this way: a
// fallback multiplies that unit's cost by a full compile, so hiding the reason
// turns a diagnosable failure into a mystery slowdown — but a hundred packages
// failing identically deserve one line, not a hundred.
func summariseReasons(reasons []string, limit int) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, reason := range reasons {
		if reason == "" {
			continue
		}
		if _, dup := seen[reason]; dup {
			continue
		}
		seen[reason] = struct{}{}
		if len(out) < limit {
			out = append(out, reason)
		}
	}
	if extra := len(seen) - len(out); extra > 0 {
		out = append(out, fmt.Sprintf("… and %d more distinct reasons", extra))
	}

	return out
}
