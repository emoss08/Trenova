package boolutils

import "github.com/emoss08/trenova/shared/stringutils"

func Parse(value string) bool {
	parsed, _ := stringutils.ParseBool(value)
	return parsed
}

func ParseDefault(value string, fallback bool) bool {
	parsed, ok := stringutils.ParseBool(value)
	if !ok {
		return fallback
	}

	return parsed
}
