package inflect

import (
	"strings"
	"unicode"

	"github.com/emoss08/trenova/shared/textcase"
)

var irregularPlurals = map[string]string{
	"person":    "people",
	"child":     "children",
	"criterion": "criteria",
	"matrix":    "matrices",
	"vertex":    "vertices",
	"appendix":  "appendices",
	"analysis":  "analyses",
	"basis":     "bases",
	"datum":     "data",
}

var irregularSingulars = func() map[string]string {
	out := make(map[string]string, len(irregularPlurals))
	for singular, plural := range irregularPlurals {
		out[plural] = singular
	}

	return out
}()

func PluralNoun(phrase string) string {
	head, last := splitLastWord(phrase)
	if last == "" {
		return phrase
	}

	return head + pluralWord(last)
}

func SingularNoun(phrase string) string {
	head, last := splitLastWord(phrase)
	if last == "" {
		return phrase
	}

	return head + singularWord(last)
}

func splitLastWord(phrase string) (string, string) {
	cut := strings.LastIndexFunc(phrase, func(r rune) bool {
		return unicode.IsSpace(r) || r == '_' || r == '-'
	})

	return phrase[:cut+1], phrase[cut+1:]
}

func pluralWord(word string) string {
	if isInitialism(word) {
		return word + "s"
	}
	lower := strings.ToLower(word)
	if plural, ok := irregularPlurals[lower]; ok {
		return matchLeadingCase(word, plural)
	}
	if _, ok := irregularSingulars[lower]; ok {
		return word
	}

	switch {
	case hasConsonantY(lower):
		return word[:len(word)-1] + "ies"
	case strings.HasSuffix(lower, "s"),
		strings.HasSuffix(lower, "x"),
		strings.HasSuffix(lower, "z"),
		strings.HasSuffix(lower, "ch"),
		strings.HasSuffix(lower, "sh"):
		return word + "es"
	default:
		return word + "s"
	}
}

func singularWord(word string) string {
	if isInitialism(strings.TrimSuffix(word, "s")) {
		return strings.TrimSuffix(word, "s")
	}
	lower := strings.ToLower(word)
	if singular, ok := irregularSingulars[lower]; ok {
		return matchLeadingCase(word, singular)
	}
	if _, ok := irregularPlurals[lower]; ok {
		return word
	}

	switch {
	case strings.HasSuffix(lower, "ies") && len(lower) > 3:
		return word[:len(word)-3] + "y"
	case strings.HasSuffix(lower, "sses"),
		strings.HasSuffix(lower, "xes"),
		strings.HasSuffix(lower, "zzes"),
		strings.HasSuffix(lower, "ches"),
		strings.HasSuffix(lower, "shes"):
		return word[:len(word)-2]
	case strings.HasSuffix(lower, "ss"),
		strings.HasSuffix(lower, "us"),
		strings.HasSuffix(lower, "is"):
		return word
	case strings.HasSuffix(lower, "s"):
		return word[:len(word)-1]
	default:
		return word
	}
}

func hasConsonantY(lower string) bool {
	if len(lower) < 2 || !strings.HasSuffix(lower, "y") {
		return false
	}

	return !strings.ContainsRune("aeiou", rune(lower[len(lower)-2]))
}

func isInitialism(word string) bool {
	letters := 0
	for _, r := range word {
		if unicode.IsLower(r) {
			return false
		}
		if unicode.IsLetter(r) {
			letters++
		}
	}

	return letters > 1
}

func matchLeadingCase(original, replacement string) string {
	if original != "" && unicode.IsUpper([]rune(original)[0]) {
		return textcase.CapitalizeFirst(replacement)
	}

	return replacement
}
