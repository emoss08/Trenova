package stringutils

import "strconv"

// Pluralize returns singular when count is exactly one and plural otherwise.
func Pluralize(singular, plural string, count int) string {
	if count == 1 {
		return singular
	}

	return plural
}

// CountNoun renders a count with the noun that agrees with it, as in "1 load"
// and "14 loads".
func CountNoun(count int, singular, plural string) string {
	return strconv.Itoa(count) + " " + Pluralize(singular, plural, count)
}
