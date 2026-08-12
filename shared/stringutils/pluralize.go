package stringutils

// Pluralize returns singular when count is exactly one and plural otherwise.
func Pluralize(singular, plural string, count int) string {
	if count == 1 {
		return singular
	}

	return plural
}
