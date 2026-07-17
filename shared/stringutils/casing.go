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
