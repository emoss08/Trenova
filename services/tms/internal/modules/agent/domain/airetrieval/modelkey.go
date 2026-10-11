package airetrieval

import (
	"strconv"
	"strings"
)

func ModelKeyDimensions(modelKey string) (int, bool) {
	at := strings.LastIndexByte(modelKey, '@')
	if at <= 0 || at == len(modelKey)-1 {
		return 0, false
	}

	dimensions, err := strconv.Atoi(modelKey[at+1:])
	if err != nil || !IsAllowedDimension(dimensions) {
		return 0, false
	}

	return dimensions, true
}
