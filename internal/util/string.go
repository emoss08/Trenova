package util

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

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

// GenerateRandomBase64String returns a string with n random bytes securely generated using the system's
// default CSPRNG in base64 encoding. The resulting string might not be of length n as the encoding for
// the raw bytes generated may vary.
//
// An error will be returned if reading from the secure random number generator fails, at which point
// the returned result should be discarded and not used any further.
func GenerateRandomBase64String(n int) (string, error) {
	b, err := GenerateRandomBytes(n)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b), nil
}

// GenerateRandomHexString returns a string with n random bytes securely generated using the system's
// default CSPRNG in hexadecimal encoding. The resulting string might not be of length n as the encoding
// for the raw bytes generated may vary.
//
// An error will be returned if reading from the secure random number generator fails, at which point
// the returned result should be discarded and not used any further.
func GenerateRandomHexString(n int) (string, error) {
	b, err := GenerateRandomBytes(n)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

// CharRange defines the type for character ranges
type CharRange int

const (
	CharRangeNumeric CharRange = iota
	CharRangeAlphaLowerCase
	CharRangeAlphaUpperCase
)

func GenerateRandomString(n int, ranges []CharRange, extra string) (string, error) {
	if len(ranges) == 0 && len(extra) == 0 {
		return "", errors.New("random string can only be created if set of characters or extra string characters supplied")
	}

	validateFn := makeValidator(ranges, extra)
	return buildString(validateFn, n)
}

// makeValidator creates a function to validate if a byte is within the allowed character sets.
func makeValidator(ranges []CharRange, extra string) func(byte) bool {
	return func(c byte) bool {
		if strings.IndexByte(extra, c) >= 0 {
			return true
		}
		for _, r := range ranges {
			if isInRange(r, c) {
				return true
			}
		}
		return false
	}
}

// isInRange checks if a given byte is within a specified character range.
func isInRange(r CharRange, c byte) bool {
	switch r {
	case CharRangeNumeric:
		return c >= '0' && c <= '9'
	case CharRangeAlphaLowerCase:
		return c >= 'a' && c <= 'z'
	case CharRangeAlphaUpperCase:
		return c >= 'A' && c <= 'Z'
	default:
		return false
	}
}

// buildString builds the final string from random bytes that pass the validation function.
// It ensures the string meets the required length by repeatedly generating random bytes if necessary.
func buildString(validateFn func(byte) bool, n int) (string, error) {
	var str strings.Builder
	for str.Len() < n {
		buf, err := GenerateRandomBytes(n - str.Len()) // Generate only as many bytes as are needed
		if err != nil {
			return "", err
		}
		for _, b := range buf {
			if validateFn(b) {
				str.WriteByte(b)
			}
			if str.Len() == n {
				break
			}
		}
	}
	return str.String(), nil
}

// Lowercases a string and trims whitespace from the beginning and end of the string
func ToUsernameFormat(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

// Title converts a string to title case
func ToTitleFormat(s string) string {
	caser := cases.Title(language.AmericanEnglish)
	title := caser.String(s)

	return title
}
