package stringutils

import "strings"

// MaskTail hides every rune except the last `visible`, so an identifier can be
// shown for recognition without exposing it. Values no longer than `visible`
// are fully masked; an empty value stays empty.
func MaskTail(value string, visible int) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return ""
	}
	if visible < 0 {
		visible = 0
	}
	if len(runes) <= visible {
		return strings.Repeat("•", len(runes))
	}
	hidden := len(runes) - visible
	return strings.Repeat("•", hidden) + string(runes[hidden:])
}
