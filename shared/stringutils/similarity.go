package stringutils

import (
	"slices"
	"strings"
	"unicode"
)

const (
	jaroWinklerPrefixScale = 0.1
	jaroWinklerMaxPrefix   = 4
)

var legalSuffixes = map[string]struct{}{
	"inc":          {},
	"incorporated": {},
	"llc":          {},
	"l l c":        {},
	"ltd":          {},
	"limited":      {},
	"co":           {},
	"corp":         {},
	"corporation":  {},
	"company":      {},
	"lp":           {},
	"llp":          {},
	"plc":          {},
	"pllc":         {},
}

func NormalizeName(value string) string {
	var b strings.Builder
	b.Grow(len(value))
	pendingSpace := false
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if pendingSpace && b.Len() > 0 {
				b.WriteByte(' ')
			}
			pendingSpace = false
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		pendingSpace = true
	}
	return b.String()
}

func NormalizeCompanyName(value string) string {
	tokens := strings.Fields(NormalizeName(value))
	for len(tokens) > 1 {
		if _, ok := legalSuffixes[tokens[len(tokens)-1]]; !ok {
			break
		}
		tokens = tokens[:len(tokens)-1]
	}
	return strings.Join(tokens, " ")
}

func NameSimilarity(a, b string) float64 {
	return nameScore(NormalizeName(a), NormalizeName(b))
}

func CompanyNameSimilarity(a, b string) float64 {
	return nameScore(NormalizeCompanyName(a), NormalizeCompanyName(b))
}

func nameScore(a, b string) float64 {
	if a == "" || b == "" {
		return 0
	}
	if a == b {
		return 1
	}
	return max(JaroWinkler(a, b), TokenSetRatio(a, b))
}

func JaroWinkler(a, b string) float64 {
	left, right := []rune(a), []rune(b)
	if len(left) == 0 && len(right) == 0 {
		return 1
	}
	if len(left) == 0 || len(right) == 0 {
		return 0
	}

	jaro := jaroSimilarity(left, right)
	prefix := 0
	for prefix < min(len(left), len(right), jaroWinklerMaxPrefix) && left[prefix] == right[prefix] {
		prefix++
	}

	return jaro + float64(prefix)*jaroWinklerPrefixScale*(1-jaro)
}

func jaroSimilarity(left, right []rune) float64 {
	window := max(max(len(left), len(right))/2-1, 0)
	leftMatched := make([]bool, len(left))
	rightMatched := make([]bool, len(right))

	matches := 0
	for i := range left {
		start := max(0, i-window)
		end := min(len(right), i+window+1)
		for j := start; j < end; j++ {
			if rightMatched[j] || left[i] != right[j] {
				continue
			}
			leftMatched[i], rightMatched[j] = true, true
			matches++
			break
		}
	}
	if matches == 0 {
		return 0
	}

	transpositions := 0
	k := 0
	for i := range left {
		if !leftMatched[i] {
			continue
		}
		for !rightMatched[k] {
			k++
		}
		if left[i] != right[k] {
			transpositions++
		}
		k++
	}

	m := float64(matches)
	return (m/float64(len(left)) + m/float64(len(right)) + (m-float64(transpositions)/2)/m) / 3
}

func TokenSetRatio(a, b string) float64 {
	left := uniqueSortedTokens(a)
	right := uniqueSortedTokens(b)
	if len(left) == 0 || len(right) == 0 {
		return 0
	}

	var shared, onlyLeft, onlyRight []string
	i, j := 0, 0
	for i < len(left) && j < len(right) {
		switch {
		case left[i] == right[j]:
			shared = append(shared, left[i])
			i++
			j++
		case left[i] < right[j]:
			onlyLeft = append(onlyLeft, left[i])
			i++
		default:
			onlyRight = append(onlyRight, right[j])
			j++
		}
	}
	onlyLeft = append(onlyLeft, left[i:]...)
	onlyRight = append(onlyRight, right[j:]...)

	base := strings.Join(shared, " ")
	withLeft := joinNonEmpty(base, strings.Join(onlyLeft, " "))
	withRight := joinNonEmpty(base, strings.Join(onlyRight, " "))

	best := levenshteinRatio(withLeft, withRight)
	if base != "" {
		best = max(best, levenshteinRatio(base, withLeft), levenshteinRatio(base, withRight))
	}
	return best
}

func uniqueSortedTokens(value string) []string {
	tokens := strings.Fields(value)
	slices.Sort(tokens)
	return slices.Compact(tokens)
}

func joinNonEmpty(a, b string) string {
	switch {
	case a == "":
		return b
	case b == "":
		return a
	default:
		return a + " " + b
	}
}

func levenshteinRatio(a, b string) float64 {
	left, right := []rune(a), []rune(b)
	total := len(left) + len(right)
	if total == 0 {
		return 1
	}
	return float64(total-levenshteinDistance(left, right)) / float64(total)
}

func levenshteinDistance(left, right []rune) int {
	if len(left) < len(right) {
		left, right = right, left
	}
	previous := make([]int, len(right)+1)
	current := make([]int, len(right)+1)
	for j := range previous {
		previous[j] = j
	}
	for i := 1; i <= len(left); i++ {
		current[0] = i
		for j := 1; j <= len(right); j++ {
			cost := 1
			if left[i-1] == right[j-1] {
				cost = 0
			}
			current[j] = min(previous[j]+1, current[j-1]+1, previous[j-1]+cost)
		}
		previous, current = current, previous
	}
	return previous[len(right)]
}
