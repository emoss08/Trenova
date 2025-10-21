package jsonutils

import (
	"strings"
)

// ExtractJSON extracts the first JSON object or array from a string
func ExtractJSON(s string) string {
	start := strings.IndexAny(s, "{[")
	if start == -1 {
		return ""
	}

	isJSONObject := s[start] == '{'

	depth := 0
	inString := false
	escaped := false

	for i := start; i < len(s); i++ {
		ch := s[i]
		if ch == '"' && !escaped {
			inString = !inString
		} else if ch == '\\' && !escaped {
			escaped = true
			continue
		}

		if !inString {
			switch ch {
			case '{', '[':
				depth++
			case '}', ']':
				depth--
				if depth == 0 {
					return s[start : i+1]
				}
			}
		}

		escaped = false
	}

	if isJSONObject && depth > 0 {
		lastBrace := strings.LastIndex(s, "}")
		if lastBrace > start {
			return s[start : lastBrace+1]
		}
	}

	return ""
}
