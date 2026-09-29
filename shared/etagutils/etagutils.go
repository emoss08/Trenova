package etagutils

import "strings"

// Strong quotes a validator as a strong entity tag.
func Strong(value string) string {
	return `"` + value + `"`
}

// Matches reports whether an If-None-Match header names the entity tag, by
// the weak comparison RFC 9110 prescribes for that header.
func Matches(ifNoneMatch, etag string) bool {
	if ifNoneMatch == "" || etag == "" {
		return false
	}
	target := strings.TrimPrefix(etag, "W/")
	for candidate := range strings.SplitSeq(ifNoneMatch, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" || strings.TrimPrefix(candidate, "W/") == target {
			return true
		}
	}

	return false
}
