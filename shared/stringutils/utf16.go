package stringutils

import "unicode/utf8"

// ByteOffsetOfUTF16 is the byte offset in s of the position units UTF-16 code
// units in, the unit editors and the rules module count text positions in. A
// position inside a character (past its first code unit) moves to the
// character's start, so slicing s at the result never splits a character; a
// negative position is 0 and one past the end is len(s).
func ByteOffsetOfUTF16(s string, units int) int {
	if units <= 0 {
		return 0
	}
	counted := 0
	for at, r := range s {
		width := 1
		if r >= 0x10000 && r != utf8.RuneError {
			width = 2
		}
		if counted+width > units {
			return at
		}
		counted += width
		if counted == units {
			return at + utf8.RuneLen(r)
		}
	}
	return len(s)
}
