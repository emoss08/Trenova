package utils

import (
	"crypto/rand"
	"strconv"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// CharRange defines the type for character ranges
type CharRange int

const (
	CharRangeNumeric CharRange = iota
	CharRangeAlphaLowerCase
	CharRangeAlphaUpperCase
)

// Title converts a string to title case
func ToTitleFormat(s string) string {
	caser := cases.Title(language.AmericanEnglish)
	title := caser.String(s)

	return title
}

func TruncateString(name string, maxLength int) string {
	if len(name) > maxLength {
		return name[:maxLength]
	}

	return name
}

func EnsureFixedLength(value string, length int) string {
	if len(value) > length {
		return value[:length]
	}
	return value + strings.Repeat("0", length-len(value))
}

func StringToInt(value string, defaultCount int) int {
	if value == "" {
		return defaultCount
	}
	count, err := strconv.Atoi(value)
	if err != nil {
		return defaultCount
	}
	return count
}

// GenerateRandomBytes returns n random bytes securely generated using the system's default CSPRNG.
//
// An error will be returned if reading from the secure random number generator fails, at which point
// the returned result should be discarded and not used any further.
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)

	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}
