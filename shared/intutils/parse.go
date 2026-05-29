package intutils

import (
	"strconv"
	"strings"
)

func Parse(value string) int {
	parsed, _ := strconv.Atoi(strings.TrimSpace(value))
	return parsed
}
