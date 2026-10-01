package llmtokens

import "unicode/utf8"

const RunesPerToken = 4

func FromRunes(runes int) int {
	if runes <= 0 {
		return 0
	}

	return (runes + RunesPerToken - 1) / RunesPerToken
}

func Estimate(text string) int {
	return FromRunes(utf8.RuneCountInString(text))
}

func EstimateAll(texts []string) int {
	total := 0
	for _, text := range texts {
		total += Estimate(text)
	}

	return total
}

func EstimateValue(value any) int {
	return FromRunes(valueRunes(value))
}

func valueRunes(value any) int {
	switch typed := value.(type) {
	case nil:
		return 4
	case string:
		return utf8.RuneCountInString(typed) + 2
	case map[string]any:
		runes := 2
		for key, item := range typed {
			runes += utf8.RuneCountInString(key) + 4 + valueRunes(item)
		}

		return runes
	case []any:
		runes := 2
		for _, item := range typed {
			runes += valueRunes(item) + 1
		}

		return runes
	case []string:
		runes := 2
		for _, item := range typed {
			runes += utf8.RuneCountInString(item) + 3
		}

		return runes
	case bool:
		return 5
	default:
		return 12
	}
}
