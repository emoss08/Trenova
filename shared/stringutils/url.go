package stringutils

func Truncate(value string, maxLength int) string {
	if len(value) > maxLength {
		return value[:maxLength] + "..."
	}
	return value
}
