package stringutils

import (
	"unicode"
	"unicode/utf8"
)

// degenerateWindow is how much of the end of a text is examined. A loop long
// enough to matter is caught well inside it, and the check stays cheap enough
// to run on a stream as it arrives.
const degenerateWindow = 4096

// repeatRule is how many back-to-back copies of a unit of a given length make
// a loop. Short units need more copies, because short fragments recur in
// ordinary text; a whole sentence said four times over is already a loop.
type repeatRule struct {
	maxUnit int
	copies  int
}

var repeatRules = [...]repeatRule{
	{maxUnit: 8, copies: 12},
	{maxUnit: 24, copies: 6},
	{maxUnit: 160, copies: 4},
}

// DegenerateTail reports whether a text ends in a loop — one fragment written
// over and over — and the byte offset where the loop begins.
//
// A unit must contain a letter: a table rule, a row of zeros or a line of
// equals signs repeats by design and is not a loop.
func DegenerateTail(text string) (int, bool) {
	offset := 0
	if len(text) > degenerateWindow {
		offset = len(text) - degenerateWindow
		for offset < len(text) && !utf8.RuneStart(text[offset]) {
			offset++
		}
	}
	runes := []rune(text[offset:])

	minUnit := 1
	for _, rule := range repeatRules {
		for unit := minUnit; unit <= rule.maxUnit; unit++ {
			if start, ok := repeatedTail(runes, unit, rule.copies); ok {
				return offset + len(string(runes[:start])), true
			}
		}
		minUnit = rule.maxUnit + 1
	}

	return 0, false
}

// repeatedTail finds whether the last unit runes are repeated at least copies
// times back to back, and where the run of copies starts.
func repeatedTail(runes []rune, unit, copies int) (int, bool) {
	if len(runes) < unit*copies {
		return 0, false
	}

	tail := runes[len(runes)-unit:]
	if !hasLetter(tail) {
		return 0, false
	}

	start := len(runes) - unit
	for start-unit >= 0 && equalRunes(runes[start-unit:start], tail) {
		start -= unit
	}
	if (len(runes)-start)/unit < copies {
		return 0, false
	}

	return start, true
}

func hasLetter(runes []rune) bool {
	for _, r := range runes {
		if unicode.IsLetter(r) {
			return true
		}
	}

	return false
}

func equalRunes(a, b []rune) bool {
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
