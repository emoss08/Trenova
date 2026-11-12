package numberguard

import (
	"slices"
	"strings"
	"unicode"

	"github.com/shopspring/decimal"
)

var spelledFigures = map[string]int64{
	"two": 2, "three": 3, "four": 4, "five": 5, "six": 6, "seven": 7, "eight": 8, "nine": 9,
	"ten": 10, "eleven": 11, "twelve": 12, "thirteen": 13, "fourteen": 14, "fifteen": 15,
	"sixteen": 16, "seventeen": 17, "eighteen": 18, "nineteen": 19, "twenty": 20,
	"thirty": 30, "forty": 40, "fifty": 50, "sixty": 60, "seventy": 70, "eighty": 80,
	"ninety": 90, "hundred": 100, "thousand": 1000, "dozen": 12, "couple": 2,
	"twice": 2, "double": 2, "triple": 3,
}

// CheckSpelled reports whether every figure the prose spells out as a word is
// one the evidence holds. CheckNumbers reads digits, so "over three hours" or
// "two weeks late" passes it however the model arrived at them; a caller that
// hands the model its figures in digits runs both. "One" is left alone: it is
// a pronoun far more often than a count.
func CheckSpelled(prose string, supported []decimal.Decimal) NumberCheck {
	check := NumberCheck{OK: true}
	words := strings.FieldsFunc(strings.ToLower(prose), func(r rune) bool {
		return !unicode.IsLetter(r)
	})
	for _, word := range words {
		value, spelled := spelledFigures[word]
		if !spelled || holds(decimal.NewFromInt(value), supported) {
			continue
		}
		check.OK = false
		check.Unsupported = append(check.Unsupported, word)
	}

	return check
}

// CheckEvery is CheckNumbers with nothing let through as sentence furniture:
// every figure in the prose, however small and whether written in digits or
// as a word, is one the evidence holds. It is for prose written only from
// evidence handed to the model, where "3 hours late" is a claim a person acts
// on and the model had no business working it out.
func CheckEvery(prose string, supported []decimal.Decimal) NumberCheck {
	check := CheckSpelled(prose, supported)
	for _, cited := range extractNumbers(prose) {
		if isSupported(cited.value, supported) || cited.readIn(supported) ||
			slices.Contains(check.Unsupported, cited.text) {
			continue
		}
		check.OK = false
		check.Unsupported = append(check.Unsupported, cited.text)
	}

	return check
}

func holds(value decimal.Decimal, supported []decimal.Decimal) bool {
	for _, candidate := range supported {
		if value.Equal(candidate) {
			return true
		}
	}

	return false
}
