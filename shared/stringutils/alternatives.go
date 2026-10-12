package stringutils

import "strings"

// JoinAlternatives lists choices the way a sentence offers them: "a", "a or
// b", "a, b or c". A refusal naming the kinds a value may be reads as one
// sentence rather than as a list.
func JoinAlternatives(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	default:
		return strings.Join(items[:len(items)-1], ", ") + " or " + items[len(items)-1]
	}
}
