package intutils

import (
	"regexp"
	"strconv"
	"strings"
)

var leadingIntegerPattern = regexp.MustCompile(`\d[\d,]*`)

func Parse(value string) int {
	parsed, _ := strconv.Atoi(strings.TrimSpace(value))
	return parsed
}

func FirstInteger(value string) (int64, bool) {
	match := leadingIntegerPattern.FindString(value)
	if match == "" {
		return 0, false
	}
	parsed, err := strconv.ParseInt(strings.ReplaceAll(match, ",", ""), 10, 64)
	if err != nil {
		return 0, false
	}

	return parsed, true
}
