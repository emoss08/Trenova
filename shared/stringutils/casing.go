package stringutils

import (
	"strings"
	"unicode"
)

func CapitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}

	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])

	return string(runes)
}

var humanizeSpecialWords = map[string]string{
	"id":   "ID",
	"url":  "URL",
	"api":  "API",
	"edi":  "EDI",
	"scac": "SCAC",
	"dot":  "DOT",
	"vin":  "VIN",
	"pto":  "PTO",
	"gl":   "GL",
	"pdf":  "PDF",
	"csv":  "CSV",
	"json": "JSON",
	"bol":  "BOL",
	"cdl":  "CDL",
	"mvr":  "MVR",
	"twic": "TWIC",
	"eta":  "ETA",
	"hos":  "HOS",
	"mc":   "MC",
	"un":   "UN",
}

func HumanizeCamelCase(s string) string {
	if s == "" {
		return ""
	}

	snake := ConvertCamelToSnake(s)
	words := strings.Split(snake, "_")

	var result strings.Builder
	result.Grow(len(s) + len(words))

	for i, word := range words {
		if word == "" {
			continue
		}
		if i > 0 {
			result.WriteByte(' ')
		}
		if special, ok := humanizeSpecialWords[word]; ok {
			result.WriteString(special)
			continue
		}
		result.WriteString(CapitalizeFirst(word))
	}

	return result.String()
}

// HumanizeCamelCaseSentence is HumanizeCamelCase in sentence case: only the
// first word is capitalized, and an initialism keeps its letters. It is the
// form a label takes in the product, where "Shipment ID" is a label and
// "Shipment Id" is a typo.
func HumanizeCamelCaseSentence(s string) string {
	if s == "" {
		return ""
	}

	words := strings.Split(ConvertCamelToSnake(s), "_")

	var result strings.Builder
	result.Grow(len(s) + len(words))

	written := 0
	for _, word := range words {
		if word == "" {
			continue
		}
		if written > 0 {
			result.WriteByte(' ')
		}
		switch {
		case humanizeSpecialWords[word] != "":
			result.WriteString(humanizeSpecialWords[word])
		case written == 0:
			result.WriteString(CapitalizeFirst(word))
		default:
			result.WriteString(word)
		}
		written++
	}

	return result.String()
}

func ConvertCamelToSnake(s string) string {
	if s == "" {
		return ""
	}

	var result strings.Builder
	result.Grow(len(s) + 10)

	runes := []rune(s)
	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 {
				prevIsLower := unicode.IsLower(runes[i-1])
				nextIsLower := i < len(runes)-1 && unicode.IsLower(runes[i+1])

				if prevIsLower || nextIsLower {
					result.WriteByte('_')
				}
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// HumanizeSnakeCase reads a snake_case identifier as words: reassign_move
// becomes "reassign move". Dots and dashes split words too, so a dotted tool
// or event name reads the same way. Case is left alone; the caller decides
// whether the phrase starts a sentence.
func HumanizeSnakeCase(s string) string {
	return strings.Join(strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == '.'
	}), " ")
}
