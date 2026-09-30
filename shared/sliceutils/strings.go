package sliceutils

import (
	"slices"
	"strings"
)

func DedupeStrings(items []string) []string {
	if len(items) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

// DedupeSorted returns the distinct, non-blank members of items in ascending
// order. Callers that render the result into a stable place — a rendered
// document, a warning list — need the order to be the same on every run, which
// DedupeStrings does not promise because it keeps the order it was given.
func DedupeSorted(items []string) []string {
	out := DedupeStrings(items)
	slices.Sort(out)

	return out
}

func AppendIfMissing(items []string, value string) []string {
	if slices.Contains(items, value) {
		return items
	}
	return append(items, value)
}

func MapKeys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
