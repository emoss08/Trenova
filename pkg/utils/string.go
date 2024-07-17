// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.

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
