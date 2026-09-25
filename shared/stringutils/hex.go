package stringutils

func IsLowerHex(value string) bool {
	if value == "" {
		return false
	}

	for i := range len(value) {
		c := value[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}

	return true
}

func IsLowerHexOfLength(value string, length int) bool {
	return len(value) == length && IsLowerHex(value)
}

func IsAllZeros(value string) bool {
	if value == "" {
		return false
	}

	for i := range len(value) {
		if value[i] != '0' {
			return false
		}
	}

	return true
}
