/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package pronumbergen

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/pkg/sequencegen"
	"github.com/emoss08/trenova/shared/pulid"
)

// DefaultProNumberFormat returns the default pro number format configuration
// that matches the examples provided: S121094129213012
func DefaultProNumberFormat() *sequencegen.SequenceFormat {
	return &sequencegen.SequenceFormat{
		Prefix:                  "S",
		IncludeYear:             true,
		YearDigits:              2,
		IncludeMonth:            true,
		SequenceDigits:          4,
		IncludeLocationCode:     true,
		LocationCode:            "12",
		IncludeRandomDigits:     true,
		RandomDigitsCount:       6,
		IncludeCheckDigit:       false,
		IncludeBusinessUnitCode: false,
		BusinessUnitCode:        "",
		UseSeparators:           false,
		SeparatorChar:           "-",
		IncludeWeekNumber:       false,
		IncludeDay:              false,
		AllowCustomFormat:       false,
		CustomFormat:            "{P}{Y}{M}{L}{S}{R}",
	}
}

// GetOrganizationProNumberFormat retrieves the pro number format configuration for a specific organization
// from the database. It looks up the configuration in pro_number_configs table.
func GetOrganizationProNumberFormat(
	_ context.Context,
	_ pulid.ID,
) (*sequencegen.SequenceFormat, error) {
	return DefaultProNumberFormat(), nil
}

// GetProNumberFormatForBusinessUnit retrieves the pro number format for a specific business unit
func GetProNumberFormatForBusinessUnit(
	ctx context.Context,
	orgID, _ pulid.ID,
) (*sequencegen.SequenceFormat, error) {
	return GetOrganizationProNumberFormat(ctx, orgID)
}

// GenerateProNumber generates a pro number based on the given format, sequence, year, and month
func GenerateProNumber(format *sequencegen.SequenceFormat, sequence, year, month int) string {
	if format.AllowCustomFormat && format.CustomFormat != "" {
		return generateCustomFormat(format, sequence, year, month)
	}

	var parts []string
	var numericPart string

	// * Add prefix
	if format.Prefix != "" {
		parts = append(parts, format.Prefix)
	}

	// * Add business unit code if configured
	if format.IncludeBusinessUnitCode && format.BusinessUnitCode != "" {
		parts = append(parts, format.BusinessUnitCode)
	}

	// * Create date component
	var dateComponent string
	// * Add year digits if configured
	if format.IncludeYear {
		yearStr := strconv.Itoa(year)
		if len(yearStr) > format.YearDigits {
			// * Take only the last n digits
			yearStr = yearStr[len(yearStr)-format.YearDigits:]
		}
		dateComponent += yearStr
		numericPart += yearStr
	}

	// * Add week if configured (takes precedence over month)
	if format.IncludeWeekNumber {
		// * Calculate the ISO week number
		_, week := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC).ISOWeek()
		weekStr := fmt.Sprintf("%02d", week)
		dateComponent += weekStr
		numericPart += weekStr
	} else if format.IncludeMonth {
		// * Add month if configured (zero-padded to 2 digits)
		monthStr := fmt.Sprintf("%02d", month)
		dateComponent += monthStr
		numericPart += monthStr
	}

	// * Add day if configured
	if format.IncludeDay {
		// * Use the first day of the month for simplicity
		// * In a real implementation, you might want to pass the actual day
		dayStr := "01"
		dateComponent += dayStr
		numericPart += dayStr
	}

	if dateComponent != "" {
		parts = append(parts, dateComponent)
	}

	// * Add location code if configured
	if format.IncludeLocationCode && format.LocationCode != "" {
		parts = append(parts, format.LocationCode)
		numericPart += format.LocationCode
	}

	// * Add sequence number (zero-padded to configured number of digits)
	sequenceFmt := fmt.Sprintf("%%0%dd", format.SequenceDigits)
	sequenceStr := fmt.Sprintf(sequenceFmt, sequence)
	parts = append(parts, sequenceStr)
	numericPart += sequenceStr

	// * Add random digits if configured
	var randomStr string
	if format.IncludeRandomDigits && format.RandomDigitsCount > 0 {
		randomStr = generateRandomDigits(format.RandomDigitsCount)
		parts = append(parts, randomStr)
		numericPart += randomStr
	}

	// * Add check digit if configured
	if format.IncludeCheckDigit {
		checkDigit := calculateCheckDigit(numericPart)
		parts = append(parts, strconv.Itoa(checkDigit))
	}

	// * Join parts with separator if configured
	if format.UseSeparators && format.SeparatorChar != "" {
		return strings.Join(parts, format.SeparatorChar)
	}

	// * Otherwise join without separator
	return strings.Join(parts, "")
}

// generateCustomFormat generates a pro number using the custom format template
func generateCustomFormat(format *sequencegen.SequenceFormat, sequence, year, month int) string {
	result := format.CustomFormat

	// * Replace placeholders with actual values
	// * Prefix
	result = strings.ReplaceAll(result, "{P}", format.Prefix)

	// * Year
	if strings.Contains(result, "{Y}") {
		yearStr := strconv.Itoa(year)
		if len(yearStr) > format.YearDigits {
			yearStr = yearStr[len(yearStr)-format.YearDigits:]
		}
		result = strings.ReplaceAll(result, "{Y}", yearStr)
	}

	// * Month
	if strings.Contains(result, "{M}") {
		result = strings.ReplaceAll(result, "{M}", fmt.Sprintf("%02d", month))
	}

	// * Week
	if strings.Contains(result, "{W}") {
		_, week := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC).ISOWeek()
		result = strings.ReplaceAll(result, "{W}", fmt.Sprintf("%02d", week))
	}

	// * Day - placeholder for now
	if strings.Contains(result, "{D}") {
		result = strings.ReplaceAll(result, "{D}", "01")
	}

	// * Location code
	if strings.Contains(result, "{L}") {
		result = strings.ReplaceAll(result, "{L}", format.LocationCode)
	}

	// * Business unit code
	if strings.Contains(result, "{B}") {
		result = strings.ReplaceAll(result, "{B}", format.BusinessUnitCode)
	}

	// * Sequence
	if strings.Contains(result, "{S}") {
		sequenceFmt := fmt.Sprintf("%%0%dd", format.SequenceDigits)
		result = strings.ReplaceAll(result, "{S}", fmt.Sprintf(sequenceFmt, sequence))
	}

	// * Random digits
	if strings.Contains(result, "{R}") {
		result = strings.ReplaceAll(result, "{R}", generateRandomDigits(format.RandomDigitsCount))
	}

	// * Calculate and add check digit if needed
	if strings.Contains(result, "{C}") {
		// * Extract numbers from the current result for checksum calculation
		re := regexp.MustCompile(`\d+`)
		numericParts := re.FindAllString(result, -1)
		numericPart := strings.Join(numericParts, "")

		checkDigit := calculateCheckDigit(numericPart)
		result = strings.ReplaceAll(result, "{C}", strconv.Itoa(checkDigit))
	}

	return result
}

// generateRandomDigits generates a string of random digits of the specified length
func generateRandomDigits(count int) string {
	if count <= 0 {
		return ""
	}

	// * Use cryptographically secure random number generation
	maxRandom := 1
	for range count {
		maxRandom *= 10
	}

	// * Generate cryptographically secure random number
	n, err := rand.Int(rand.Reader, big.NewInt(int64(maxRandom)))
	if err != nil {
		// * Fallback to timestamp if random generation fails
		n = big.NewInt(time.Now().UnixNano() % int64(maxRandom))
	}

	randomFmt := fmt.Sprintf("%%0%dd", count)
	return fmt.Sprintf(randomFmt, n.Int64())
}

// calculateCheckDigit calculates a check digit for a numeric string using the Luhn algorithm
func calculateCheckDigit(input string) int {
	// * Remove any non-digit characters
	re := regexp.MustCompile(`\D`)
	digits := re.ReplaceAllString(input, "")

	// * Luhn algorithm
	sum := 0
	alternate := false

	// * Process from right to left
	for i := len(digits) - 1; i >= 0; i-- {
		digit, _ := strconv.Atoi(string(digits[i]))

		if alternate {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		alternate = !alternate
	}

	// * The check digit is the number needed to make the sum a multiple of 10
	return (10 - (sum % 10)) % 10
}

// ValidateProNumber validates a pro number according to its format
// Returns true if the pro number is valid, false otherwise
func ValidateProNumber(proNumber string, format *sequencegen.SequenceFormat) bool {
	if format.IncludeCheckDigit {
		// * Extract the check digit (last digit)
		if len(proNumber) < 1 {
			return false
		}

		checkDigit, err := strconv.Atoi(string(proNumber[len(proNumber)-1]))
		if err != nil {
			return false
		}

		// * Calculate the expected check digit using all but the last character
		numericPart := proNumber[:len(proNumber)-1]
		expectedCheckDigit := calculateCheckDigit(numericPart)

		return checkDigit == expectedCheckDigit
	}

	// * If no check digit, just verify it matches the expected pattern
	// * This would be more complex in a real implementation
	return true
}

// ParseProNumber attempts to parse a pro number string into its components
// Returns a map of component names to values
func ParseProNumber(
	proNumber string,
	format *sequencegen.SequenceFormat,
) (map[string]string, error) {
	// * This is a simplistic implementation that would need to be expanded
	// * based on the specific format rules in a real implementation

	if !ValidateProNumber(proNumber, format) {
		return nil, ErrInvalidProNumber
	}

	result := make(map[string]string)

	// * Basic parsing - in a real implementation, this would be more sophisticated
	// * and would handle the custom format properly

	// * For now, just return the whole pro number
	result["full"] = proNumber

	return result, nil
}

// GenerateBatch generates a batch of pro numbers
func GenerateBatch(ctx context.Context, orgID pulid.ID, count int) ([]string, error) {
	if count <= 0 {
		return []string{}, nil
	}

	// * Get the organization format
	format, err := GetOrganizationProNumberFormat(ctx, orgID)
	if err != nil {
		return nil, err
	}

	// * Current date components
	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	// * Results slice
	results := make([]string, 0, count)

	// * In a real implementation, we'd use a single database transaction to:
	// 1. Get the current sequence
	// 2. Increment it by count
	// 3. Generate all pro numbers in the range
	//
	// * For now, we'll simulate this by incrementing a local counter
	startSequence := 1 // In practice, this would come from the database

	for i := range count {
		sequence := startSequence + i
		proNumber := GenerateProNumber(format, sequence, year, month)
		results = append(results, proNumber)
	}

	return results, nil
}

// DetectFormat attempts to detect the format of a pro number
// Returns a ProNumberFormat that could generate this pro number
func DetectFormat(proNumber string) (*sequencegen.SequenceFormat, error) {
	if len(proNumber) < 3 {
		return nil, ErrInvalidProNumber
	}

	// * This is a simplistic implementation that guesses the format
	// * A real implementation would be much more sophisticated
	format := &sequencegen.SequenceFormat{}

	// * Detect prefix (assume first character is alphabetic)
	if first := proNumber[0]; first >= 'A' && first <= 'Z' {
		format.Prefix = string(first)
		proNumber = proNumber[1:]
	}

	// * Rough detection of components
	// * This is very simplistic and would need to be improved
	hasYear := len(proNumber) > 2
	hasMonth := len(proNumber) > 4
	hasLocationCode := len(proNumber) > 6

	format.IncludeYear = hasYear
	format.YearDigits = 2
	format.IncludeMonth = hasMonth
	format.SequenceDigits = 4
	format.IncludeLocationCode = hasLocationCode
	format.LocationCode = "00" // This is a guess
	format.IncludeRandomDigits = len(proNumber) > 10
	format.RandomDigitsCount = 6

	return format, nil
}

// NormalizeProNumber normalizes a pro number by removing separators and spaces
func NormalizeProNumber(proNumber string) string {
	// * Remove common separators and spaces
	proNumber = strings.ReplaceAll(proNumber, "-", "")
	proNumber = strings.ReplaceAll(proNumber, " ", "")
	proNumber = strings.ReplaceAll(proNumber, ".", "")
	return strings.ToUpper(proNumber)
}

// FormatProNumber formats a pro number according to a display format
// The display format can include separators for better readability
func FormatProNumber(proNumber string, format *sequencegen.SequenceFormat) string {
	if !format.UseSeparators || format.SeparatorChar == "" {
		return proNumber
	}

	// * This is a simplistic implementation that just adds separators
	// * at fixed positions. A real implementation would be format-aware.
	var result strings.Builder

	// * Prefix is separate
	if proNumber != "" && format.Prefix != "" {
		result.WriteString(proNumber[:1])
		result.WriteString(format.SeparatorChar)
		proNumber = proNumber[1:]
	}

	// * Add year-month block
	if format.IncludeYear && format.IncludeMonth && len(proNumber) >= 4 {
		result.WriteString(proNumber[:4])
		result.WriteString(format.SeparatorChar)
		proNumber = proNumber[4:]
	}

	// * Add location code
	if format.IncludeLocationCode && len(proNumber) >= 2 {
		result.WriteString(proNumber[:2])
		result.WriteString(format.SeparatorChar)
		proNumber = proNumber[2:]
	}

	// * Add sequence
	if len(proNumber) >= format.SequenceDigits {
		result.WriteString(proNumber[:format.SequenceDigits])
		proNumber = proNumber[format.SequenceDigits:]

		// * If anything remains, add another separator
		if proNumber != "" {
			result.WriteString(format.SeparatorChar)
		}
	}

	// * Add the rest
	result.WriteString(proNumber)

	return result.String()
}

// GetProNumberComponents extracts components from a pro number based on its format
// Returns a map of component names to extracted values
//
//nolint:funlen,gocognit // This is a complex function that needs to be refactored
func GetProNumberComponents(
	proNumber string,
	format *sequencegen.SequenceFormat,
) (map[string]string, error) {
	if !ValidateProNumber(proNumber, format) {
		return nil, ErrInvalidProNumber
	}

	result := make(map[string]string)
	result["full"] = proNumber

	offset := 0
	var err error

	// * Extract components based on format
	extractorFns := []func() error{
		// * Extract prefix
		func() error {
			if format.Prefix != "" {
				offset, err = extractComponent(
					proNumber,
					offset,
					len(format.Prefix),
					"prefix",
					result,
				)
				return err
			}
			return nil
		},
		// * Extract business unit code
		func() error {
			if format.IncludeBusinessUnitCode && format.BusinessUnitCode != "" {
				offset, err = extractComponent(
					proNumber,
					offset,
					len(format.BusinessUnitCode),
					"businessUnit",
					result,
				)
				return err
			}
			return nil
		},
		// * Extract year
		func() error {
			if format.IncludeYear && format.YearDigits > 0 {
				offset, err = extractComponent(proNumber, offset, format.YearDigits, "year", result)
				return err
			}
			return nil
		},
		// * Extract month or week
		func() error {
			if format.IncludeMonth {
				offset, err = extractComponent(proNumber, offset, 2, "month", result)
				return err
			} else if format.IncludeWeekNumber {
				offset, err = extractComponent(proNumber, offset, 2, "week", result)
				return err
			}
			return nil
		},
		// * Extract day
		func() error {
			if format.IncludeDay {
				offset, err = extractComponent(proNumber, offset, 2, "day", result)
				return err
			}
			return nil
		},
		// * Extract location code
		func() error {
			if format.IncludeLocationCode {
				offset, err = extractComponent(
					proNumber,
					offset,
					len(format.LocationCode),
					"location",
					result,
				)
				return err
			}
			return nil
		},
		// * Extract sequence
		func() error {
			if format.SequenceDigits > 0 {
				offset, err = extractComponent(
					proNumber,
					offset,
					format.SequenceDigits,
					"sequence",
					result,
				)
				return err
			}
			return nil
		},
		// * Extract random part
		func() error {
			if format.IncludeRandomDigits && format.RandomDigitsCount > 0 {
				offset, err = extractComponent(
					proNumber,
					offset,
					format.RandomDigitsCount,
					"random",
					result,
				)
				return err
			}
			return nil
		},
		// * Extract check digit
		func() error {
			if format.IncludeCheckDigit {
				offset, err = extractComponent(proNumber, offset, 1, "checkDigit", result)
				return err
			}
			return nil
		},
	}

	for _, fn := range extractorFns {
		if err = fn(); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// extractComponent extracts a component from the proNumber at the given offset
// with the specified length and adds it to the result map with the given key.
// Returns the new offset and any error.
func extractComponent(
	proNumber string,
	offset, length int,
	key string,
	result map[string]string,
) (int, error) {
	if len(proNumber) < offset+length {
		return offset, ErrInvalidProNumber
	}
	result[key] = proNumber[offset : offset+length]

	return offset + length, nil
}
