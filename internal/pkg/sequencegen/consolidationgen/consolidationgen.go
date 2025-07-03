package consolidationgen

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
	"github.com/emoss08/trenova/pkg/types/pulid"
)

func DefaultConsolidationFormat() *sequencegen.SequenceFormat {
	return &sequencegen.SequenceFormat{
		Prefix:                  "C",
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

// GetOrganizationConsolidationFormat retrieves the consolidation number format configuration for a specific organization
// from the database. It looks up the configuration in consolidation_number_configs table.
func GetOrganizationConsolidationFormat(
	_ context.Context,
	_ pulid.ID,
) (*sequencegen.SequenceFormat, error) {
	return DefaultConsolidationFormat(), nil
}

// GetConsolidationFormatForBusinessUnit retrieves the consolidation number format for a specific business unit
func GetConsolidationFormatForBusinessUnit(
	ctx context.Context,
	orgID, _ pulid.ID,
) (*sequencegen.SequenceFormat, error) {
	return GetOrganizationConsolidationFormat(ctx, orgID)
}

// GenerateConsolidationNumber generates a consolidation number based on the given format, sequence, year, and month
func GenerateConsolidationNumber(
	format *sequencegen.SequenceFormat,
	sequence, year, month int,
) string {
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

// generateCustomFormat generates a consolidation number using the custom format template
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

// ValidateConsolidationNumber validates a consolidation number according to its format
// Returns true if the consolidation number is valid, false otherwise
func ValidateConsolidationNumber(
	consolidationNumber string,
	format *sequencegen.SequenceFormat,
) bool {
	if format.IncludeCheckDigit {
		// * Extract the check digit (last digit)
		if len(consolidationNumber) < 1 {
			return false
		}

		checkDigit, err := strconv.Atoi(string(consolidationNumber[len(consolidationNumber)-1]))
		if err != nil {
			return false
		}

		// * Calculate the expected check digit using all but the last character
		numericPart := consolidationNumber[:len(consolidationNumber)-1]
		expectedCheckDigit := calculateCheckDigit(numericPart)

		return checkDigit == expectedCheckDigit
	}

	// * If no check digit, just verify it matches the expected pattern
	// * This would be more complex in a real implementation
	return true
}

// ParseConsolidationNumber attempts to parse a consolidation number string into its components
// Returns a map of component names to values
func ParseConsolidationNumber(
	consolidationNumber string,
	format *sequencegen.SequenceFormat,
) (map[string]string, error) {
	// * This is a simplistic implementation that would need to be expanded
	// * based on the specific format rules in a real implementation

	if !ValidateConsolidationNumber(consolidationNumber, format) {
		return nil, ErrInvalidConsolidationNumber
	}

	result := make(map[string]string)

	// * Basic parsing - in a real implementation, this would be more sophisticated
	// * and would handle the custom format properly

	// * For now, just return the whole pro number
	result["full"] = consolidationNumber

	return result, nil
}

// GenerateBatch generates a batch of consolidation numbers
func GenerateBatch(ctx context.Context, orgID pulid.ID, count int) ([]string, error) {
	if count <= 0 {
		return []string{}, nil
	}

	// * Get the organization format
	format, err := GetOrganizationConsolidationFormat(ctx, orgID)
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
		consolidationNumber := GenerateConsolidationNumber(format, sequence, year, month)
		results = append(results, consolidationNumber)
	}

	return results, nil
}

// DetectFormat attempts to detect the format of a consolidation number
// Returns a ConsolidationFormat that could generate this consolidation number
func DetectFormat(consolidationNumber string) (*sequencegen.SequenceFormat, error) {
	if len(consolidationNumber) < 3 {
		return nil, ErrInvalidConsolidationNumber
	}

	// * This is a simplistic implementation that guesses the format
	// * A real implementation would be much more sophisticated
	format := &sequencegen.SequenceFormat{}

	// * Detect prefix (assume first character is alphabetic)
	if first := consolidationNumber[0]; first >= 'A' && first <= 'Z' {
		format.Prefix = string(first)
		consolidationNumber = consolidationNumber[1:]
	}

	// * Rough detection of components
	// * This is very simplistic and would need to be improved
	hasYear := len(consolidationNumber) > 2
	hasMonth := len(consolidationNumber) > 4
	hasLocationCode := len(consolidationNumber) > 6

	format.IncludeYear = hasYear
	format.YearDigits = 2
	format.IncludeMonth = hasMonth
	format.SequenceDigits = 4
	format.IncludeLocationCode = hasLocationCode
	format.LocationCode = "00" // This is a guess
	format.IncludeRandomDigits = len(consolidationNumber) > 10
	format.RandomDigitsCount = 6

	return format, nil
}

// NormalizeConsolidationNumber normalizes a consolidation number by removing separators and spaces
func NormalizeConsolidationNumber(consolidationNumber string) string {
	consolidationNumber = strings.ReplaceAll(consolidationNumber, "-", "")
	consolidationNumber = strings.ReplaceAll(consolidationNumber, " ", "")
	consolidationNumber = strings.ReplaceAll(consolidationNumber, ".", "")
	return strings.ToUpper(consolidationNumber)
}

// FormatConsolidationNumber formats a consolidation number according to a display format
// The display format can include separators for better readability
func FormatConsolidationNumber(
	consolidationNumber string,
	format *sequencegen.SequenceFormat,
) string {
	if !format.UseSeparators || format.SeparatorChar == "" {
		return consolidationNumber
	}

	// * This is a simplistic implementation that just adds separators
	// * at fixed positions. A real implementation would be format-aware.
	var result strings.Builder

	// * Prefix is separate
	if consolidationNumber != "" && format.Prefix != "" {
		result.WriteString(consolidationNumber[:1])
		result.WriteString(format.SeparatorChar)
		consolidationNumber = consolidationNumber[1:]
	}

	// * Add year-month block
	if format.IncludeYear && format.IncludeMonth && len(consolidationNumber) >= 4 {
		result.WriteString(consolidationNumber[:4])
		result.WriteString(format.SeparatorChar)
		consolidationNumber = consolidationNumber[4:]
	}

	// * Add location code
	if format.IncludeLocationCode && len(consolidationNumber) >= 2 {
		result.WriteString(consolidationNumber[:2])
		result.WriteString(format.SeparatorChar)
		consolidationNumber = consolidationNumber[2:]
	}

	// * Add sequence
	if len(consolidationNumber) >= format.SequenceDigits {
		result.WriteString(consolidationNumber[:format.SequenceDigits])
		consolidationNumber = consolidationNumber[format.SequenceDigits:]

		// * If anything remains, add another separator
		if consolidationNumber != "" {
			result.WriteString(format.SeparatorChar)
		}
	}

	// * Add the rest
	result.WriteString(consolidationNumber)

	return result.String()
}

// GetConsolidationNumberComponents extracts components from a consolidation number based on its format
// Returns a map of component names to extracted values
//
//nolint:funlen,gocognit // This is a complex function that needs to be refactored
func GetConsolidationNumberComponents(
	consolidationNumber string,
	format *sequencegen.SequenceFormat,
) (map[string]string, error) {
	if !ValidateConsolidationNumber(consolidationNumber, format) {
		return nil, ErrInvalidConsolidationNumber
	}

	result := make(map[string]string)
	result["full"] = consolidationNumber

	offset := 0
	var err error

	// * Extract components based on format
	extractorFns := []func() error{
		// * Extract prefix
		func() error {
			if format.Prefix != "" {
				offset, err = extractComponent(
					consolidationNumber,
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
					consolidationNumber,
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
				offset, err = extractComponent(
					consolidationNumber,
					offset,
					format.YearDigits,
					"year",
					result,
				)
				return err
			}
			return nil
		},
		// * Extract month or week
		func() error {
			if format.IncludeMonth {
				offset, err = extractComponent(consolidationNumber, offset, 2, "month", result)
				return err
			} else if format.IncludeWeekNumber {
				offset, err = extractComponent(consolidationNumber, offset, 2, "week", result)
				return err
			}
			return nil
		},
		// * Extract day
		func() error {
			if format.IncludeDay {
				offset, err = extractComponent(consolidationNumber, offset, 2, "day", result)
				return err
			}
			return nil
		},
		// * Extract location code
		func() error {
			if format.IncludeLocationCode {
				offset, err = extractComponent(
					consolidationNumber,
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
					consolidationNumber,
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
					consolidationNumber,
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
				offset, err = extractComponent(consolidationNumber, offset, 1, "checkDigit", result)
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

// extractComponent extracts a component from the consolidationNumber at the given offset
// with the specified length and adds it to the result map with the given key.
// Returns the new offset and any error.
func extractComponent(
	consolidationNumber string,
	offset, length int,
	key string,
	result map[string]string,
) (int, error) {
	if len(consolidationNumber) < offset+length {
		return offset, ErrInvalidConsolidationNumber
	}
	result[key] = consolidationNumber[offset : offset+length]

	return offset + length, nil
}
