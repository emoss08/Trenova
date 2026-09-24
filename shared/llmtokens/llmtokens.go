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
