package stringutils

import "strings"

func Truncate(value string, maxLength int) string {
	if len(value) > maxLength {
		return value[:maxLength] + "..."
	}
	return value
}

func TruncateAndTrim(value string, maxLength int) string {
	value = strings.TrimSpace(value)

	return Truncate(value, maxLength)
}

// IsSafeAppPath reports whether a path stays inside the application: rooted,
// not protocol-relative ("//evil.example" leaves despite the leading slash),
// and free of the characters that would let it break out of an attribute or
// a header.
func IsSafeAppPath(path string) bool {
	if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") {
		return false
	}

	return !strings.ContainsAny(path, "\\\r\n")
}
