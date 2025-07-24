/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package jsonutils

import (
	"strings"
)

// ExtractJSON extracts the first JSON object or array from a string
func ExtractJSON(s string) string {
	// Find the first { or [
	start := strings.IndexAny(s, "{[")
	if start == -1 {
		return ""
	}

	// Determine if it's an object or array
	isJSONObject := s[start] == '{'

	// Find the matching closing bracket
	depth := 0
	inString := false
	escaped := false

	for i := start; i < len(s); i++ {
		ch := s[i]

		// Handle string literals
		if ch == '"' && !escaped {
			inString = !inString
		} else if ch == '\\' && !escaped {
			escaped = true
			continue
		}

		// Only count brackets outside of strings
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

	// If we're looking for an object and didn't find closing }, try to extract anyway
	if isJSONObject && depth > 0 {
		// Find the last }
		lastBrace := strings.LastIndex(s, "}")
		if lastBrace > start {
			return s[start : lastBrace+1]
		}
	}

	return ""
}
