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

func ModelKeyModel(modelKey string) string {
	model := modelKey
	if at := strings.LastIndexByte(model, '@'); at >= 0 {
		model = model[:at]
	}
	if slash := strings.LastIndexByte(model, '/'); slash >= 0 {
		model = model[slash+1:]
	}
	if colon := strings.IndexByte(model, ':'); colon >= 0 {
		model = model[:colon]
	}

	return strings.ToLower(strings.TrimSpace(model))
}
