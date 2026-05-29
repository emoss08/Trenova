package floatutils

import (
	"strconv"
	"strings"
)

func Parse(value string) float64 {
	parsed, _ := strconv.ParseFloat(strings.TrimSpace(value), 64)
	return parsed
}
