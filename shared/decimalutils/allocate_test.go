package decimalutils

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func dec(s string) decimal.Decimal {
	return decimal.RequireFromString(s)
}

func decs(values ...string) []decimal.Decimal {
	out := make([]decimal.Decimal, 0, len(values))
	for _, v := range values {
		out = append(out, dec(v))
	}
	return out
}

func strs(values []decimal.Decimal) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		out = append(out, v.StringFixed(2))
	}
	return out
}

func TestAllocatePercent(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		total    string
		percents []string
		expected []string
		err      error
	}{
		{
			name:     "even thirds fold the remainder into the last share",
			total:    "100.00",
			percents: []string{"33.333333", "33.333333", "33.333334"},
			expected: []string{"33.33", "33.33", "33.34"},
		},
		{
			name:     "sixty forty splits exactly",
			total:    "1250.50",
			percents: []string{"60", "40"},
			expected: []string{"750.30", "500.20"},
		},
		{
			name:     "single cent goes entirely to the last share",
			total:    "0.01",
			percents: []string{"33.333333", "33.333333", "33.333334"},
			expected: []string{"0.00", "0.00", "0.01"},
		},
		{
			name:     "negative totals keep their sign",
			total:    "-100.00",
			percents: []string{"50", "50"},
			expected: []string{"-50.00", "-50.00"},
		},
		{
			name:     "single full share",
			total:    "42.42",
			percents: []string{"100"},
			expected: []string{"42.42"},
		},
		{
			name:     "empty percents",
			total:    "10",
			percents: nil,
			err:      ErrNoPercents,
		},
		{
			name:     "zero percent",
			total:    "10",
			percents: []string{"0", "100"},
			err:      ErrPercentOutOfRange,
		},
		{
			name:     "over one hundred percent",
			total:    "10",
			percents: []string{"101"},
			err:      ErrPercentOutOfRange,
		},
		{
			name:     "percents that do not total one hundred",
			total:    "10",
			percents: []string{"60", "30"},
			err:      ErrPercentsDoNotTotal,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			shares, err := AllocatePercent(dec(tc.total), decs(tc.percents...), 2)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, strs(shares))
			assert.True(t, SumEquals(shares, dec(tc.total), 2))
		})
	}
}

func TestSumEquals(t *testing.T) {
	t.Parallel()

	assert.True(t, SumEquals(decs("33.33", "33.33", "33.34"), dec("100"), 2))
	assert.False(t, SumEquals(decs("33.33", "33.33", "33.33"), dec("100"), 2))
	assert.True(t, SumEquals(nil, decimal.Zero, 2))
	assert.True(t, SumEquals(decs("0.004", "0.004"), dec("0.01"), 2))
}
