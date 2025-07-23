// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

// Package sequencegen provides a unified sequence generation system for various entity types
package sequencegen

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
)

// * SequenceType represents the type of sequence to generate
type SequenceType string

const (
	// * Common sequence types
	SequenceTypeProNumber     SequenceType = "pro_number"
	SequenceTypeConsolidation SequenceType = "consolidation"
	SequenceTypeInvoice       SequenceType = "invoice"
	SequenceTypeWorkOrder     SequenceType = "work_order"
)

// * Default configurations for different sequence types
var defaultConfigs = map[SequenceType]SequenceFormat{
	SequenceTypeProNumber: {
		Prefix:              "S",
		IncludeYear:         true,
		YearDigits:          2,
		IncludeMonth:        true,
		SequenceDigits:      4,
		IncludeLocationCode: true,
		LocationCode:        "12",
		IncludeRandomDigits: true,
		RandomDigitsCount:   6,
		IncludeCheckDigit:   false,
		UseSeparators:       false,
	},
	SequenceTypeConsolidation: {
		Prefix:              "C",
		IncludeYear:         true,
		YearDigits:          2,
		IncludeMonth:        true,
		SequenceDigits:      4,
		IncludeLocationCode: true,
		LocationCode:        "12",
		IncludeRandomDigits: true,
		RandomDigitsCount:   6,
		IncludeCheckDigit:   false,
		UseSeparators:       false,
	},
	SequenceTypeInvoice: {
		Prefix:              "INV",
		IncludeYear:         true,
		YearDigits:          4,
		IncludeMonth:        true,
		SequenceDigits:      6,
		IncludeLocationCode: false,
		IncludeRandomDigits: false,
		IncludeCheckDigit:   true,
		UseSeparators:       true,
		SeparatorChar:       "-",
	},
	SequenceTypeWorkOrder: {
		Prefix:              "WO",
		IncludeYear:         true,
		YearDigits:          2,
		IncludeMonth:        false,
		SequenceDigits:      6,
		IncludeLocationCode: true,
		LocationCode:        "01",
		IncludeRandomDigits: false,
		IncludeCheckDigit:   false,
		UseSeparators:       true,
		SeparatorChar:       "-",
	},
}

// * GenerateRequest represents a request to generate a sequence number
type GenerateRequest struct {
	SequenceType SequenceType
	OrgID        pulid.ID
	BusinessUnit pulid.ID
	Count        int             // * For batch generation
	Time         time.Time       // * Optional: override current time
	Format       *SequenceFormat // * Optional: override default format
}

// * GenerateSequenceNumber generates a sequence number based on the provided format and configuration
func GenerateSequenceNumber(
	_ context.Context,
	format *SequenceFormat,
	sequenceNumber int64,
	currentTime time.Time,
) (string, error) {
	if format == nil {
		return "", eris.New("sequence format cannot be nil")
	}

	// * Validate the format
	if err := format.Validate(); err != nil {
		return "", eris.Wrap(err, "invalid sequence format")
	}

	// * Use custom format if specified
	if format.AllowCustomFormat && format.CustomFormat != "" {
		return generateCustomFormat(format, sequenceNumber, currentTime)
	}

	// * Build standard format
	return generateStandardFormat(format, sequenceNumber, currentTime)
}

// * generateStandardFormat builds a sequence number using standard component ordering
func generateStandardFormat(
	format *SequenceFormat,
	sequenceNumber int64,
	currentTime time.Time,
) (string, error) {
	var builder strings.Builder
	components := make([]string, 0, 10)

	// * Add prefix
	if format.Prefix != "" {
		components = append(components, format.Prefix)
	}

	// * Add business unit code
	if format.IncludeBusinessUnitCode && format.BusinessUnitCode != "" {
		components = append(components, format.BusinessUnitCode)
	}

	// * Add year
	if format.IncludeYear {
		year := getYearString(currentTime, format.YearDigits)
		components = append(components, year)
	}

	// * Add month or week
	if format.IncludeWeekNumber {
		_, week := currentTime.ISOWeek()
		components = append(components, fmt.Sprintf("%02d", week))
	} else if format.IncludeMonth {
		components = append(components, fmt.Sprintf("%02d", currentTime.Month()))
	}

	// * Add day
	if format.IncludeDay {
		components = append(components, fmt.Sprintf("%02d", currentTime.Day()))
	}

	// * Add location code
	if format.IncludeLocationCode && format.LocationCode != "" {
		components = append(components, format.LocationCode)
	}

	// * Add sequence number
	seqFormat := fmt.Sprintf("%%0%dd", format.SequenceDigits)
	components = append(components, fmt.Sprintf(seqFormat, sequenceNumber))

	// * Add random digits
	if format.IncludeRandomDigits && format.RandomDigitsCount > 0 {
		randomDigits := generateRandomDigits(format.RandomDigitsCount)
		components = append(components, randomDigits)
	}

	// * Join components
	separator := ""
	if format.UseSeparators && format.SeparatorChar != "" {
		separator = format.SeparatorChar
	}
	result := strings.Join(components, separator)

	// * Add check digit if needed
	if format.IncludeCheckDigit {
		checkDigit := calculateLuhnCheckDigit(result)
		if separator != "" {
			builder.WriteString(result)
			builder.WriteString(separator)
			builder.WriteString(strconv.Itoa(checkDigit))
		} else {
			builder.WriteString(result)
			builder.WriteString(strconv.Itoa(checkDigit))
		}
		return builder.String(), nil
	}

	return result, nil
}

// * generateCustomFormat builds a sequence number using a custom format string
func generateCustomFormat(
	format *SequenceFormat,
	sequenceNumber int64,
	currentTime time.Time,
) (string, error) {
	result := format.CustomFormat

	// * Replace placeholders
	replacements := map[string]string{
		"{P}": format.Prefix,
		"{Y}": getYearString(currentTime, format.YearDigits),
		"{M}": fmt.Sprintf("%02d", currentTime.Month()),
		"{W}": fmt.Sprintf("%02d", getISOWeek(currentTime)),
		"{D}": fmt.Sprintf("%02d", currentTime.Day()),
		"{L}": format.LocationCode,
		"{B}": format.BusinessUnitCode,
		"{S}": fmt.Sprintf(fmt.Sprintf("%%0%dd", format.SequenceDigits), sequenceNumber),
	}

	// * Generate random digits if needed
	if strings.Contains(result, "{R}") {
		randomDigits := generateRandomDigits(format.RandomDigitsCount)
		replacements["{R}"] = randomDigits
	}

	// * Replace all placeholders
	for placeholder, value := range replacements {
		result = strings.ReplaceAll(result, placeholder, value)
	}

	// * Calculate and add check digit if needed
	if strings.Contains(result, "{C}") {
		// * Remove {C} placeholder temporarily to calculate check digit
		tempResult := strings.ReplaceAll(result, "{C}", "")
		checkDigit := calculateLuhnCheckDigit(tempResult)
		result = strings.ReplaceAll(result, "{C}", strconv.Itoa(checkDigit))
	}

	return result, nil
}

// * getYearString returns the year as a string with the specified number of digits
func getYearString(t time.Time, digits int) string {
	year := t.Year()
	switch digits {
	case 2:
		return fmt.Sprintf("%02d", year%100)
	case 4:
		return fmt.Sprintf("%04d", year)
	default:
		yearStr := strconv.Itoa(year)
		if len(yearStr) > digits {
			return yearStr[len(yearStr)-digits:]
		}
		return yearStr
	}
}

// * getISOWeek returns the ISO week number
func getISOWeek(t time.Time) int {
	_, week := t.ISOWeek()
	return week
}

// * generateRandomDigits generates cryptographically secure random digits
func generateRandomDigits(count int) string {
	if count <= 0 {
		return ""
	}

	var result strings.Builder
	result.Grow(count)

	for i := 0; i < count; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			// * Fallback to time-based randomness if crypto fails
			return generateFallbackRandomDigits(count)
		}
		result.WriteString(n.String())
	}

	return result.String()
}

// * generateFallbackRandomDigits generates random digits using time-based randomness
func generateFallbackRandomDigits(count int) string {
	var result strings.Builder
	result.Grow(count)

	// * Use nanoseconds as a source of randomness
	nano := time.Now().UnixNano()
	for i := 0; i < count; i++ {
		digit := nano % 10
		result.WriteString(strconv.FormatInt(digit, 10))
		nano /= 10
		if nano == 0 {
			nano = time.Now().UnixNano()
		}
	}

	return result.String()
}

// * calculateLuhnCheckDigit calculates the Luhn check digit for a given string
func calculateLuhnCheckDigit(s string) int {
	// * Remove non-digit characters
	digits := regexp.MustCompile(`\D`).ReplaceAllString(s, "")

	sum := 0
	parity := len(digits) % 2

	for i, r := range digits {
		digit := int(r - '0')

		if i%2 == parity {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
	}

	return (10 - (sum % 10)) % 10
}

// * ValidateSequenceNumber validates a sequence number using its check digit
func ValidateSequenceNumber(number string, format *SequenceFormat) bool {
	if !format.IncludeCheckDigit {
		return true // * No check digit to validate
	}

	// * Remove separators if present
	cleaned := number
	if format.UseSeparators && format.SeparatorChar != "" {
		cleaned = strings.ReplaceAll(cleaned, format.SeparatorChar, "")
	}

	if cleaned == "" {
		return false
	}

	// * Extract check digit (last character)
	checkDigitStr := cleaned[len(cleaned)-1:]
	checkDigit, err := strconv.Atoi(checkDigitStr)
	if err != nil {
		return false
	}

	// * Calculate expected check digit
	numberWithoutCheck := cleaned[:len(cleaned)-1]
	expectedCheckDigit := calculateLuhnCheckDigit(numberWithoutCheck)

	return checkDigit == expectedCheckDigit
}

// * GetDefaultFormat returns the default format for a given sequence type
func GetDefaultFormat(sequenceType SequenceType) (*SequenceFormat, error) {
	format, ok := defaultConfigs[sequenceType]
	if !ok {
		return nil, eris.Errorf("unknown sequence type: %s", sequenceType)
	}

	// * Return a copy to avoid mutation
	formatCopy := format
	return &formatCopy, nil
}

// * ParseSequenceNumber attempts to parse a sequence number and extract its components
func ParseSequenceNumber(number string, format *SequenceFormat) (*SequenceComponents, error) {
	if format == nil {
		return nil, eris.New("format cannot be nil")
	}

	components := &SequenceComponents{
		Original: number,
	}

	// * Remove separators if present
	cleaned := number
	if format.UseSeparators && format.SeparatorChar != "" {
		cleaned = strings.ReplaceAll(cleaned, format.SeparatorChar, "")
	}

	// * Parse based on format configuration
	pos := 0

	// * Extract prefix
	if format.Prefix != "" {
		if strings.HasPrefix(cleaned[pos:], format.Prefix) {
			components.Prefix = format.Prefix
			pos += len(format.Prefix)
		} else {
			return nil, eris.New("sequence number does not match expected prefix")
		}
	}

	// * Extract business unit code
	if format.IncludeBusinessUnitCode && format.BusinessUnitCode != "" {
		if pos+len(format.BusinessUnitCode) <= len(cleaned) {
			components.BusinessUnitCode = cleaned[pos : pos+len(format.BusinessUnitCode)]
			pos += len(format.BusinessUnitCode)
		}
	}

	// * Extract year
	if format.IncludeYear && format.YearDigits > 0 {
		if pos+format.YearDigits <= len(cleaned) {
			components.Year = cleaned[pos : pos+format.YearDigits]
			pos += format.YearDigits
		}
	}

	// * Extract month or week
	if format.IncludeWeekNumber {
		if pos+2 <= len(cleaned) {
			components.Week = cleaned[pos : pos+2]
			pos += 2
		}
	} else if format.IncludeMonth {
		if pos+2 <= len(cleaned) {
			components.Month = cleaned[pos : pos+2]
			pos += 2
		}
	}

	// * Extract day
	if format.IncludeDay {
		if pos+2 <= len(cleaned) {
			components.Day = cleaned[pos : pos+2]
			pos += 2
		}
	}

	// * Extract location code
	if format.IncludeLocationCode && format.LocationCode != "" {
		if pos+len(format.LocationCode) <= len(cleaned) {
			components.LocationCode = cleaned[pos : pos+len(format.LocationCode)]
			pos += len(format.LocationCode)
		}
	}

	// * Extract sequence number
	if format.SequenceDigits > 0 && pos+format.SequenceDigits <= len(cleaned) {
		components.Sequence = cleaned[pos : pos+format.SequenceDigits]
		pos += format.SequenceDigits
	}

	// * Extract random digits
	if format.IncludeRandomDigits && format.RandomDigitsCount > 0 {
		if pos+format.RandomDigitsCount <= len(cleaned) {
			components.RandomDigits = cleaned[pos : pos+format.RandomDigitsCount]
			pos += format.RandomDigitsCount
		}
	}

	// * Extract check digit
	if format.IncludeCheckDigit && pos < len(cleaned) {
		components.CheckDigit = cleaned[pos:]
	}

	return components, nil
}

// * SequenceComponents represents the parsed components of a sequence number
type SequenceComponents struct {
	Original         string
	Prefix           string
	BusinessUnitCode string
	Year             string
	Month            string
	Week             string
	Day              string
	LocationCode     string
	Sequence         string
	RandomDigits     string
	CheckDigit       string
}
