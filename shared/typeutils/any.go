package typeutils

import "strings"

// StringOf reads a string out of a decoded JSON value, reporting "" for
// anything that is not a string. Numbers and booleans that came out of a
// schema-validated document are not strings, so they read as absent rather
// than being reformatted into one.
func StringOf(value any) string {
	text, _ := value.(string)

	return text
}

// StringOfTrimmed is StringOf with surrounding whitespace removed, for a
// value that goes straight into a field or a comparison.
func StringOfTrimmed(value any) string {
	return strings.TrimSpace(StringOf(value))
}

// BoolOf reads a bool out of a decoded JSON value, reporting false for
// anything that is not a bool.
func BoolOf(value any) bool {
	result, _ := value.(bool)

	return result
}
