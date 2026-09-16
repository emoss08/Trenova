package modeladapter

import (
	"errors"
	"strings"

	"github.com/bytedance/sonic"
)

// ErrNoJSONObject reports that a reply contained nothing that could be read as a
// JSON object.
var ErrNoJSONObject = errors.New("model response contained no JSON object")

// ExtractJSON pulls a JSON object out of a model reply and decodes it into out.
//
// A provider that enforces a schema server-side returns clean JSON and the first
// decode succeeds. Everything here exists for the providers that do not: a model
// left to follow a schema by instruction alone habitually wraps its answer in a
// markdown fence, prefixes it with "Here is the JSON:", or appends a closing
// remark. Recovering from that is the difference between a self-hosted model
// being usable and being a source of intermittent failures.
func ExtractJSON(raw string, out any) error {
	candidates := jsonCandidates(raw)
	if len(candidates) == 0 {
		return ErrNoJSONObject
	}

	var lastErr error
	for _, candidate := range candidates {
		err := sonic.Unmarshal([]byte(candidate), out)
		if err == nil {
			return nil
		}
		lastErr = err

		repaired := repairJSON(candidate)
		if repaired == candidate {
			continue
		}
		if err = sonic.Unmarshal([]byte(repaired), out); err == nil {
			return nil
		}
	}

	if lastErr != nil {
		return lastErr
	}

	return ErrNoJSONObject
}

// jsonCandidates returns the substrings worth attempting, cheapest first.
func jsonCandidates(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}

	candidates := make([]string, 0, 3)
	candidates = append(candidates, trimmed)

	// The innermost form is the most likely to parse, so each step narrows the
	// text further and the narrowed result is what the next step works from.
	narrowed := trimmed
	if fenced := stripFence(narrowed); fenced != "" && fenced != narrowed {
		candidates = append(candidates, fenced)
		narrowed = fenced
	}

	// Scanning for a balanced object handles the common "prose, then JSON, then
	// prose" shape that fence-stripping alone misses.
	if balanced := firstBalancedObject(narrowed); balanced != "" && balanced != narrowed {
		candidates = append(candidates, balanced)
	}

	return candidates
}

// stripFence removes a surrounding markdown code fence, with or without a
// language tag.
func stripFence(s string) string {
	if !strings.HasPrefix(s, "```") {
		return ""
	}

	rest := s[3:]
	if newline := strings.IndexByte(rest, '\n'); newline >= 0 {
		// Drop the language tag ("json", "JSON") that follows the opening fence.
		if tag := strings.TrimSpace(rest[:newline]); !strings.ContainsAny(tag, "{[") {
			rest = rest[newline+1:]
		}
	}

	if end := strings.LastIndex(rest, "```"); end >= 0 {
		rest = rest[:end]
	}

	return strings.TrimSpace(rest)
}

// firstBalancedObject returns the first complete top-level JSON object in s,
// tracking string state so a brace inside a string value does not end the scan.
func firstBalancedObject(s string) string {
	start := strings.IndexByte(s, '{')
	if start < 0 {
		return ""
	}

	depth := 0
	inString := false
	escaped := false

	for idx := start; idx < len(s); idx++ {
		char := s[idx]

		if escaped {
			escaped = false
			continue
		}

		switch {
		case char == '\\' && inString:
			escaped = true
		case char == '"':
			inString = !inString
		case inString:
			// Braces inside a string are data, not structure.
		case char == '{':
			depth++
		case char == '}':
			depth--
			if depth == 0 {
				return s[start : idx+1]
			}
		}
	}

	return ""
}

// repairJSON fixes the malformations small models produce most often. It is
// deliberately conservative: only trailing commas before a closing brace or
// bracket are removed, because anything more aggressive risks changing what the
// model actually said.
func repairJSON(s string) string {
	var builder strings.Builder
	builder.Grow(len(s))

	inString := false
	escaped := false

	for idx := 0; idx < len(s); idx++ {
		char := s[idx]

		if escaped {
			escaped = false
			builder.WriteByte(char)
			continue
		}

		switch {
		case char == '\\' && inString:
			escaped = true
		case char == '"':
			inString = !inString
		case !inString && char == ',':
			if next := nextMeaningfulByte(s, idx+1); next == '}' || next == ']' {
				continue
			}
		}

		builder.WriteByte(char)
	}

	return builder.String()
}

func nextMeaningfulByte(s string, from int) byte {
	for idx := from; idx < len(s); idx++ {
		switch s[idx] {
		case ' ', '\t', '\n', '\r':
			continue
		default:
			return s[idx]
		}
	}

	return 0
}
