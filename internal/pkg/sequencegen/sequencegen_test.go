package sequencegen_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/pkg/sequencegen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSequenceNumber(t *testing.T) {
	ctx := context.Background()
	currentTime := time.Date(2024, 12, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name           string
		format         *sequencegen.SequenceFormat
		sequenceNumber int64
		wantPrefix     string
		wantContains   []string
		wantLength     int
		wantError      bool
	}{
		{
			name: "pro number with default format",
			format: &sequencegen.SequenceFormat{
				Prefix:              "S",
				IncludeYear:         true,
				YearDigits:          2,
				IncludeMonth:        true,
				SequenceDigits:      4,
				IncludeLocationCode: true,
				LocationCode:        "12",
				IncludeRandomDigits: true,
				RandomDigitsCount:   6,
			},
			sequenceNumber: 1234,
			wantPrefix:     "S",
			wantContains:   []string{"24", "12", "1234"},
			wantLength:     17, // S(1) + 24(2) + 12(2) + 12(2) + 1234(4) + 6 random(6) = 17
		},
		{
			name: "consolidation number format",
			format: &sequencegen.SequenceFormat{
				Prefix:              "C",
				IncludeYear:         true,
				YearDigits:          2,
				IncludeMonth:        true,
				SequenceDigits:      4,
				IncludeLocationCode: true,
				LocationCode:        "12",
				IncludeRandomDigits: true,
				RandomDigitsCount:   6,
			},
			sequenceNumber: 5678,
			wantPrefix:     "C",
			wantContains:   []string{"24", "12", "5678"},
			wantLength:     17,
		},
		{
			name: "invoice with separators and check digit",
			format: &sequencegen.SequenceFormat{
				Prefix:            "INV",
				IncludeYear:       true,
				YearDigits:        4,
				IncludeMonth:      true,
				SequenceDigits:    6,
				IncludeCheckDigit: true,
				UseSeparators:     true,
				SeparatorChar:     "-",
			},
			sequenceNumber: 123,
			wantPrefix:     "INV",
			wantContains:   []string{"INV", "2024", "12", "000123"},
			wantLength:     16, // INV(3) + 2024(4) + 12(2) + 000123(6) + check digit(1) = 16 (without separators)
		},
		{
			name: "with business unit code",
			format: &sequencegen.SequenceFormat{
				Prefix:                  "WO",
				IncludeBusinessUnitCode: true,
				BusinessUnitCode:        "NYC",
				IncludeYear:             true,
				YearDigits:              2,
				SequenceDigits:          6,
			},
			sequenceNumber: 42,
			wantPrefix:     "WO",
			wantContains:   []string{"WO", "NYC", "24", "000042"},
			wantLength:     13,
		},
		{
			name: "with week number instead of month",
			format: &sequencegen.SequenceFormat{
				Prefix:            "W",
				IncludeYear:       true,
				YearDigits:        2,
				IncludeWeekNumber: true,
				SequenceDigits:    4,
			},
			sequenceNumber: 999,
			wantPrefix:     "W",
			wantContains:   []string{"24", "50", "0999"}, // Week 50 of 2024
			wantLength:     9,
		},
		{
			name: "with day included",
			format: &sequencegen.SequenceFormat{
				Prefix:         "D",
				IncludeYear:    true,
				YearDigits:     2,
				IncludeMonth:   true,
				IncludeDay:     true,
				SequenceDigits: 3,
			},
			sequenceNumber: 1,
			wantPrefix:     "D",
			wantContains:   []string{"24", "12", "15", "001"},
			wantLength:     10,
		},
		{
			name: "custom format with placeholders",
			format: &sequencegen.SequenceFormat{
				Prefix:              "ORD",
				BusinessUnitCode:    "US",
				LocationCode:        "LA",
				SequenceDigits:      5,
				RandomDigitsCount:   3,
				AllowCustomFormat:   true,
				CustomFormat:        "{P}-{B}-{Y}{M}-{L}-{S}-{R}",
				IncludeYear:         true,
				YearDigits:          2,
				IncludeMonth:        true,
				IncludeLocationCode: true,
				IncludeRandomDigits: true,
			},
			sequenceNumber: 12345,
			wantPrefix:     "ORD",
			wantContains:   []string{"ORD", "US", "24", "12", "LA", "12345"},
		},
		{
			name: "custom format with check digit",
			format: &sequencegen.SequenceFormat{
				Prefix:            "CHK",
				SequenceDigits:    4,
				AllowCustomFormat: true,
				CustomFormat:      "{P}{S}{C}",
				IncludeCheckDigit: true,
			},
			sequenceNumber: 1234,
			wantPrefix:     "CHK",
			wantLength:     8, // CHK + 1234 + check digit
		},
		{
			name: "minimal format",
			format: &sequencegen.SequenceFormat{
				Prefix:         "M",
				SequenceDigits: 6,
			},
			sequenceNumber: 999999,
			wantPrefix:     "M",
			wantContains:   []string{"999999"},
			wantLength:     7,
		},
		{
			name: "invalid format - no prefix",
			format: &sequencegen.SequenceFormat{
				Prefix:         "",
				SequenceDigits: 4,
			},
			sequenceNumber: 1,
			wantError:      true,
		},
		{
			name: "invalid format - invalid sequence digits",
			format: &sequencegen.SequenceFormat{
				Prefix:         "X",
				SequenceDigits: 0,
			},
			sequenceNumber: 1,
			wantError:      true,
		},
		{
			name: "invalid format - missing location code",
			format: &sequencegen.SequenceFormat{
				Prefix:              "L",
				SequenceDigits:      4,
				IncludeLocationCode: true,
				LocationCode:        "",
			},
			sequenceNumber: 1,
			wantError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sequencegen.GenerateSequenceNumber(
				ctx,
				tt.format,
				tt.sequenceNumber,
				currentTime,
			)

			if tt.wantError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, result)

			// * Check prefix
			if tt.wantPrefix != "" {
				assert.True(
					t,
					strings.HasPrefix(result, tt.wantPrefix),
					"Expected prefix %s, got %s",
					tt.wantPrefix,
					result,
				)
			}

			// * Check contains
			for _, want := range tt.wantContains {
				assert.Contains(t, result, want, "Expected to contain %s in %s", want, result)
			}

			// * Check length if specified
			if tt.wantLength > 0 {
				// * Account for separators
				actualLength := len(strings.ReplaceAll(result, "-", ""))
				assert.Equal(
					t,
					tt.wantLength,
					actualLength,
					"Expected length %d, got %d for %s",
					tt.wantLength,
					actualLength,
					result,
				)
			}
		})
	}
}

func TestValidateSequenceNumber(t *testing.T) {
	tests := []struct {
		name     string
		number   string
		format   *sequencegen.SequenceFormat
		expected bool
	}{
		{
			name:   "valid number with check digit",
			number: "INV-2024-12-000123-6",
			format: &sequencegen.SequenceFormat{
				Prefix:            "INV",
				IncludeYear:       true,
				YearDigits:        4,
				IncludeMonth:      true,
				SequenceDigits:    6,
				IncludeCheckDigit: true,
				UseSeparators:     true,
				SeparatorChar:     "-",
			},
			expected: true,
		},
		{
			name:   "invalid check digit",
			number: "INV-2024-12-000123-5",
			format: &sequencegen.SequenceFormat{
				Prefix:            "INV",
				IncludeYear:       true,
				YearDigits:        4,
				IncludeMonth:      true,
				SequenceDigits:    6,
				IncludeCheckDigit: true,
				UseSeparators:     true,
				SeparatorChar:     "-",
			},
			expected: false,
		},
		{
			name:   "no check digit validation",
			number: "S2412120001",
			format: &sequencegen.SequenceFormat{
				Prefix:            "S",
				IncludeCheckDigit: false,
			},
			expected: true,
		},
		{
			name:   "empty number",
			number: "",
			format: &sequencegen.SequenceFormat{
				Prefix:            "X",
				IncludeCheckDigit: true,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sequencegen.ValidateSequenceNumber(tt.number, tt.format)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseSequenceNumber(t *testing.T) {
	tests := []struct {
		name           string
		number         string
		format         *sequencegen.SequenceFormat
		wantComponents *sequencegen.SequenceComponents
		wantError      bool
	}{
		{
			name:   "parse standard pro number",
			number: "S2412120001234567",
			format: &sequencegen.SequenceFormat{
				Prefix:              "S",
				IncludeYear:         true,
				YearDigits:          2,
				IncludeMonth:        true,
				SequenceDigits:      4,
				IncludeLocationCode: true,
				LocationCode:        "12",
				IncludeRandomDigits: true,
				RandomDigitsCount:   6,
			},
			wantComponents: &sequencegen.SequenceComponents{
				Original:     "S2412120001234567",
				Prefix:       "S",
				Year:         "24",
				Month:        "12",
				LocationCode: "12",
				Sequence:     "0001",
				RandomDigits: "234567",
			},
		},
		{
			name:   "parse with separators",
			number: "INV-2024-12-000123",
			format: &sequencegen.SequenceFormat{
				Prefix:         "INV",
				IncludeYear:    true,
				YearDigits:     4,
				IncludeMonth:   true,
				SequenceDigits: 6,
				UseSeparators:  true,
				SeparatorChar:  "-",
			},
			wantComponents: &sequencegen.SequenceComponents{
				Original: "INV-2024-12-000123",
				Prefix:   "INV",
				Year:     "2024",
				Month:    "12",
				Sequence: "000123",
			},
		},
		{
			name:   "parse with business unit code",
			number: "WONYC24000042",
			format: &sequencegen.SequenceFormat{
				Prefix:                  "WO",
				IncludeBusinessUnitCode: true,
				BusinessUnitCode:        "NYC",
				IncludeYear:             true,
				YearDigits:              2,
				SequenceDigits:          6,
			},
			wantComponents: &sequencegen.SequenceComponents{
				Original:         "WONYC24000042",
				Prefix:           "WO",
				BusinessUnitCode: "NYC",
				Year:             "24",
				Sequence:         "000042",
			},
		},
		{
			name:   "parse with week number",
			number: "W24500999",
			format: &sequencegen.SequenceFormat{
				Prefix:            "W",
				IncludeYear:       true,
				YearDigits:        2,
				IncludeWeekNumber: true,
				SequenceDigits:    4,
			},
			wantComponents: &sequencegen.SequenceComponents{
				Original: "W24500999",
				Prefix:   "W",
				Year:     "24",
				Week:     "50",
				Sequence: "0999",
			},
		},
		{
			name:   "parse with check digit",
			number: "CHK12344",
			format: &sequencegen.SequenceFormat{
				Prefix:            "CHK",
				SequenceDigits:    4,
				IncludeCheckDigit: true,
			},
			wantComponents: &sequencegen.SequenceComponents{
				Original:   "CHK12344",
				Prefix:     "CHK",
				Sequence:   "1234",
				CheckDigit: "4",
			},
		},
		{
			name:   "parse invalid prefix",
			number: "X2412120001",
			format: &sequencegen.SequenceFormat{
				Prefix:         "S",
				SequenceDigits: 4,
			},
			wantError: true,
		},
		{
			name:      "parse nil format",
			number:    "S2412120001",
			format:    nil,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sequencegen.ParseSequenceNumber(tt.number, tt.format)

			if tt.wantError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantComponents.Original, result.Original)
			assert.Equal(t, tt.wantComponents.Prefix, result.Prefix)
			assert.Equal(t, tt.wantComponents.BusinessUnitCode, result.BusinessUnitCode)
			assert.Equal(t, tt.wantComponents.Year, result.Year)
			assert.Equal(t, tt.wantComponents.Month, result.Month)
			assert.Equal(t, tt.wantComponents.Week, result.Week)
			assert.Equal(t, tt.wantComponents.Day, result.Day)
			assert.Equal(t, tt.wantComponents.LocationCode, result.LocationCode)
			assert.Equal(t, tt.wantComponents.Sequence, result.Sequence)
			assert.Equal(t, tt.wantComponents.RandomDigits, result.RandomDigits)
			assert.Equal(t, tt.wantComponents.CheckDigit, result.CheckDigit)
		})
	}
}

func TestGetDefaultFormat(t *testing.T) {
	tests := []struct {
		name         string
		sequenceType sequencegen.SequenceType
		wantPrefix   string
		wantError    bool
	}{
		{
			name:         "pro number default",
			sequenceType: sequencegen.SequenceTypeProNumber,
			wantPrefix:   "S",
		},
		{
			name:         "consolidation default",
			sequenceType: sequencegen.SequenceTypeConsolidation,
			wantPrefix:   "C",
		},
		{
			name:         "invoice default",
			sequenceType: sequencegen.SequenceTypeInvoice,
			wantPrefix:   "INV",
		},
		{
			name:         "work order default",
			sequenceType: sequencegen.SequenceTypeWorkOrder,
			wantPrefix:   "WO",
		},
		{
			name:         "unknown type",
			sequenceType: sequencegen.SequenceType("unknown"),
			wantError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, err := sequencegen.GetDefaultFormat(tt.sequenceType)

			if tt.wantError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, format)
			assert.Equal(t, tt.wantPrefix, format.Prefix)
		})
	}
}

func TestSequenceFormat_Validate(t *testing.T) {
	tests := []struct {
		name      string
		format    *sequencegen.SequenceFormat
		wantError string
	}{
		{
			name: "valid format",
			format: &sequencegen.SequenceFormat{
				Prefix:         "S",
				SequenceDigits: 4,
			},
			wantError: "",
		},
		{
			name: "missing prefix",
			format: &sequencegen.SequenceFormat{
				Prefix:         "",
				SequenceDigits: 4,
			},
			wantError: "prefix is required",
		},
		{
			name: "invalid year digits",
			format: &sequencegen.SequenceFormat{
				Prefix:         "S",
				IncludeYear:    true,
				YearDigits:     5,
				SequenceDigits: 4,
			},
			wantError: "year digits must be between 2 and 4",
		},
		{
			name: "invalid sequence digits",
			format: &sequencegen.SequenceFormat{
				Prefix:         "S",
				SequenceDigits: 11,
			},
			wantError: "sequence digits must be between 1 and 10",
		},
		{
			name: "missing location code",
			format: &sequencegen.SequenceFormat{
				Prefix:              "S",
				SequenceDigits:      4,
				IncludeLocationCode: true,
				LocationCode:        "",
			},
			wantError: "location code is required when include location code is true",
		},
		{
			name: "invalid random digits count",
			format: &sequencegen.SequenceFormat{
				Prefix:              "S",
				SequenceDigits:      4,
				IncludeRandomDigits: true,
				RandomDigitsCount:   15,
			},
			wantError: "random digits count must be between 1 and 10",
		},
		{
			name: "missing business unit code",
			format: &sequencegen.SequenceFormat{
				Prefix:                  "S",
				SequenceDigits:          4,
				IncludeBusinessUnitCode: true,
				BusinessUnitCode:        "",
			},
			wantError: "business unit code is required when include business unit code is true",
		},
		{
			name: "missing separator char",
			format: &sequencegen.SequenceFormat{
				Prefix:         "S",
				SequenceDigits: 4,
				UseSeparators:  true,
				SeparatorChar:  "",
			},
			wantError: "separator character is required when use separators is true",
		},
		{
			name: "missing custom format",
			format: &sequencegen.SequenceFormat{
				Prefix:            "S",
				SequenceDigits:    4,
				AllowCustomFormat: true,
				CustomFormat:      "",
			},
			wantError: "custom format is required when allow custom format is true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.format.Validate()

			if tt.wantError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantError)
			}
		})
	}
}

// * Benchmark tests
func BenchmarkGenerateSequenceNumber(b *testing.B) {
	ctx := context.Background()
	currentTime := time.Now()
	format := &sequencegen.SequenceFormat{
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = sequencegen.GenerateSequenceNumber(ctx, format, int64(i), currentTime)
	}
}

func BenchmarkGenerateSequenceNumberWithCheckDigit(b *testing.B) {
	ctx := context.Background()
	currentTime := time.Now()
	format := &sequencegen.SequenceFormat{
		Prefix:            "INV",
		IncludeYear:       true,
		YearDigits:        4,
		IncludeMonth:      true,
		SequenceDigits:    6,
		IncludeCheckDigit: true,
		UseSeparators:     true,
		SeparatorChar:     "-",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = sequencegen.GenerateSequenceNumber(ctx, format, int64(i), currentTime)
	}
}

func BenchmarkValidateSequenceNumber(b *testing.B) {
	format := &sequencegen.SequenceFormat{
		Prefix:            "INV",
		IncludeCheckDigit: true,
		UseSeparators:     true,
		SeparatorChar:     "-",
	}
	number := "INV-202412-000123-4"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sequencegen.ValidateSequenceNumber(number, format)
	}
}
