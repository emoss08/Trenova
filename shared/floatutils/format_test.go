package floatutils_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/floatutils"
	"github.com/stretchr/testify/assert"
)

func TestFormatGrouped(t *testing.T) {
	t.Parallel()

	cases := []struct {
		value    float64
		decimals int
		want     string
	}{
		{value: 0, decimals: 2, want: "0"},
		{value: 101844, decimals: 2, want: "101,844"},
		{value: 3279390000, decimals: 0, want: "3,279,390,000"},
		{value: 3.13, decimals: 4, want: "3.13"},
		{value: 1234.5678, decimals: 2, want: "1,234.57"},
		{value: -1500.25, decimals: 2, want: "-1,500.25"},
		{value: -0.001, decimals: 2, want: "0"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, floatutils.FormatGrouped(tc.value, tc.decimals))
	}
}

func TestFormatPercent(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "22.26%", floatutils.FormatPercent(22.26, 2))
	assert.Equal(t, "18.3%", floatutils.FormatFractionAsPercent(0.183, 2))
	assert.Equal(t, "44.33%", floatutils.FormatFractionAsPercent(0.44334646087254104, 2))
	assert.Equal(t, "0%", floatutils.FormatFractionAsPercent(0, 2))
}
