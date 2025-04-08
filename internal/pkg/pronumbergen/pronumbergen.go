package pronumbergen

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/emoss08/trenova/pkg/types/pulid"
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
func GetOrganizationProNumberFormat(_ context.Context, _ pulid.ID) (*ProNumberFormat, error) {
	// TODO: Implement
	return DefaultProNumberFormat(), nil
}

// GenerateProNumber generates a pro number based on the given format, sequence, year, and month
func GenerateProNumber(format *ProNumberFormat, sequence int, year int, month int) string {
	var result string

	// Add prefix
	result += format.Prefix

	// Add year digits if configured
	if format.IncludeYear {
		yearStr := strconv.Itoa(year)
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
		// Use cryptographically secure random number generation
		maxRandom := 1
		for i := 0; i < format.RandomDigitsCount; i++ {
			maxRandom *= 10
		}

		// Generate cryptographically secure random number
		n, err := rand.Int(rand.Reader, big.NewInt(int64(maxRandom)))
		if err != nil {
			// Fallback to timestamp if random generation fails
			n = big.NewInt(time.Now().UnixNano() % int64(maxRandom))
		}

		randomFmt := fmt.Sprintf("%%0%dd", format.RandomDigitsCount)
		result += fmt.Sprintf(randomFmt, n.Int64())
	}

	return result
}
