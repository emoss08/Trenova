package stringutils

import "strings"

func Contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || s != "" && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// FirstNonEmpty returns the first value that is not blank, unchanged, so a
// padded value keeps its padding.
func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// FirstNonEmptyTrimmed returns the first value that is not blank, trimmed.
// Use it when the value goes straight into a field, a key, or a comparison,
// where surrounding whitespace would be a defect rather than content.
func FirstNonEmptyTrimmed(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
