package pronumbergen

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/pkg/types/pulid"
	"golang.org/x/exp/rand"
)

// Errors
var (
	ErrSequenceUpdateConflict = errors.New("sequence update conflict")
	ErrInvalidYear            = errors.New("year out of range for int16")
	ErrInvalidMonth           = errors.New("month out of range for int16")
)

// ProNumberFormat represents the configuration for generating pro numbers
type ProNumberFormat struct {
	// Prefix is the letter prefix for pro numbers (e.g., "S")
	Prefix string

	// IncludeYear determines whether to include year digits in the pro number
	IncludeYear bool

	// YearDigits is the number of digits to use for the year (e.g., 2 for "23" representing 2023)
	YearDigits int

	// IncludeMonth determines whether to include month digits in the pro number
	IncludeMonth bool

	// SequenceDigits is the number of digits to use for the sequence number (will be zero-padded)
	SequenceDigits int

	// IncludeLocationCode determines whether to include location code in the pro number
	IncludeLocationCode bool

	// LocationCode is a code representing the location/region (e.g., "12" for a specific terminal)
	LocationCode string

	// IncludeRandomDigits determines whether to include random digits for additional uniqueness
	IncludeRandomDigits bool

	// RandomDigitsCount is the number of random digits to include
	RandomDigitsCount int
}

// DefaultProNumberFormat returns the default pro number format configuration
// that matches the examples provided: S121094129213012
func DefaultProNumberFormat() *ProNumberFormat {
	return &ProNumberFormat{
		Prefix:              "S",
		IncludeYear:         true,
		YearDigits:          2,
		IncludeMonth:        true,
		SequenceDigits:      4,
		IncludeLocationCode: true,
		LocationCode:        "12",
		IncludeRandomDigits: true,
		RandomDigitsCount:   6,
	}
}

// GetOrganizationProNumberFormat should retrieve the pro number format configuration for a specific organization
// from the database or configuration system. For now, it returns the default format.
func GetOrganizationProNumberFormat(ctx context.Context, orgID pulid.ID) (*ProNumberFormat, error) {
	// In a real implementation, you'd fetch the organization-specific format from the database
	// For now, return the default format
	return DefaultProNumberFormat(), nil
}

// GenerateProNumber generates a pro number based on the given format, sequence, year, and month
func GenerateProNumber(format *ProNumberFormat, sequence int, year int, month int) string {
	var result string

	// Add prefix
	result += format.Prefix

	// Add year digits if configured
	if format.IncludeYear {
		yearStr := fmt.Sprintf("%d", year)
		if len(yearStr) > format.YearDigits {
			// Take only the last n digits
			yearStr = yearStr[len(yearStr)-format.YearDigits:]
		}
		result += yearStr
	}

	// Add month if configured (zero-padded to 2 digits)
	if format.IncludeMonth {
		result += fmt.Sprintf("%02d", month)
	}

	// Add location code if configured
	if format.IncludeLocationCode {
		result += format.LocationCode
	}

	// Add sequence number (zero-padded to configured number of digits)
	sequenceFmt := fmt.Sprintf("%%0%dd", format.SequenceDigits)
	result += fmt.Sprintf(sequenceFmt, sequence)

	// Add random digits if configured
	if format.IncludeRandomDigits && format.RandomDigitsCount > 0 {
		r := rand.New(rand.NewSource(uint64(time.Now().UnixNano())))
		maxRandom := 1
		for i := 0; i < format.RandomDigitsCount; i++ {
			maxRandom *= 10
		}
		randomFmt := fmt.Sprintf("%%0%dd", format.RandomDigitsCount)
		result += fmt.Sprintf(randomFmt, r.Intn(maxRandom))
	}

	return result
}
