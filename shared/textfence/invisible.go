package textfence

import (
	"strings"
	"unicode/utf8"
)

// invisibleRune reports the code points a reader never sees but a model reads:
// the Unicode tag characters (U+E0000-U+E007F), which spell out ASCII
// invisibly; zero-width spaces, joiners and the byte order mark; the
// invisible operators; and the bidirectional controls that reorder what is
// shown.
func invisibleRune(r rune) bool {
	switch {
	case r >= 0xE0000 && r <= 0xE007F:
		return true
	case r >= 0x200B && r <= 0x200F:
		return true
	case r >= 0x202A && r <= 0x202E:
		return true
	case r >= 0x2060 && r <= 0x2064:
		return true
	case r >= 0x2066 && r <= 0x2069:
		return true
	case r == 0xFEFF, r == 0x180E, r == 0x061C, r == 0x00AD:
		return true
	default:
		return false
	}
}

// StripInvisible removes every code point invisibleRune names, leaving the
// text a person would read.
func StripInvisible(text string) string {
	index := strings.IndexFunc(text, invisibleRune)
	if index < 0 {
		return text
	}

	var b strings.Builder
	b.Grow(len(text))
	b.WriteString(text[:index])
	for _, r := range text[index:] {
		if r == utf8.RuneError || !invisibleRune(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// FenceText prepares outside text to sit inside a fence that closeTag ends,
// for a model to read: invisible code points removed first, so none can hide
// inside the close tag or carry text the person never saw, then the close tag
// neutralized.
func FenceText(text, closeTag string) string {
	return NeutralizeCloseTag(StripInvisible(text), closeTag)
}
