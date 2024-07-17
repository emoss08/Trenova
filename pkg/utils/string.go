// Copyright (c) 2024 Trenova Technologies, LLC
//
// Licensed under the Business Source License 1.1 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://trenova.app/pricing/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// Key Terms:
// - Non-production use only
// - Change Date: 2026-11-16
// - Change License: GNU General Public License v2 or later
//
// For full license text, see the LICENSE file in the root directory.

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
