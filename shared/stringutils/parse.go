package stringutils

import (
	"strconv"
	"strings"
)

func ParseBool(value string) (result, ok bool) {
	switch strings.ToLower(value) {
	case "true", "1", "yes", "on":
		return true, true
	case "false", "0", "no", "off":
		return false, true
	default:
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed, true
		}
		return false, false
	}
}
