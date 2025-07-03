package sequencegen

import (
	"github.com/rotisserie/eris"
)

// SequenceFormat represents the configuration for generating sequence numbers
type SequenceFormat struct {
	// * Prefix is the letter prefix for pro numbers (e.g., "S")
	Prefix string

	// * IncludeYear determines whether to include year digits in the pro number
	IncludeYear bool

	// YearDigits is the number of digits to use for the year (e.g., 2 for "23" representing 2023)
	YearDigits int

	// * IncludeMonth determines whether to include month digits in the pro number
	IncludeMonth bool

	// * SequenceDigits is the number of digits to use for the sequence number (will be zero-padded)
	SequenceDigits int

	// * IncludeLocationCode determines whether to include location code in the pro number
	IncludeLocationCode bool

	// * LocationCode is a code representing the location/region (e.g., "12" for a specific terminal)
	LocationCode string

	// * IncludeRandomDigits determines whether to include random digits for additional uniqueness
	IncludeRandomDigits bool

	// * RandomDigitsCount is the number of random digits to include
	RandomDigitsCount int

	// * IncludeCheckDigit adds a check digit for validation (Luhn algorithm)
	IncludeCheckDigit bool

	// * IncludeBusinessUnitCode adds the business unit code to the pro number
	IncludeBusinessUnitCode bool

	// * BusinessUnitCode is the code representing the business unit
	BusinessUnitCode string

	// * UseSeparators determines whether to use separators in the pro number
	UseSeparators bool

	// * SeparatorChar is the character to use as a separator (e.g., "-")
	SeparatorChar string

	// * IncludeWeekNumber determines whether to include the week number instead of month
	IncludeWeekNumber bool

	// * IncludeDay determines whether to include the day of month
	IncludeDay bool

	// * AllowCustomFormat allows for a completely custom format string with placeholders
	AllowCustomFormat bool

	// * CustomFormat is a string with placeholders for dynamic values
	// Example: "{P}-{Y}{M}-{S}-{C}" where:
	// {P} = Prefix, {Y} = Year, {M} = Month, {S} = Sequence, {R} = Random, {C} = Checksum
	// {B} = Business unit, {L} = Location, {W} = Week, {D} = Day
	CustomFormat string
}

// * Validate validates the sequence format configuration
func (f *SequenceFormat) Validate() error {
	if f.Prefix == "" {
		return eris.New("prefix is required")
	}

	if f.IncludeYear && (f.YearDigits < 2 || f.YearDigits > 4) {
		return eris.New("year digits must be between 2 and 4")
	}

	if f.SequenceDigits < 1 || f.SequenceDigits > 10 {
		return eris.New("sequence digits must be between 1 and 10")
	}

	if f.IncludeLocationCode && f.LocationCode == "" {
		return eris.New("location code is required when include location code is true")
	}

	if f.IncludeRandomDigits && (f.RandomDigitsCount < 1 || f.RandomDigitsCount > 10) {
		return eris.New("random digits count must be between 1 and 10")
	}

	if f.IncludeBusinessUnitCode && f.BusinessUnitCode == "" {
		return eris.New("business unit code is required when include business unit code is true")
	}

	if f.UseSeparators && f.SeparatorChar == "" {
		return eris.New("separator character is required when use separators is true")
	}

	if f.AllowCustomFormat && f.CustomFormat == "" {
		return eris.New("custom format is required when allow custom format is true")
	}

	return nil
}
