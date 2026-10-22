package tarifftypes_test

import (
	"testing"

	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLiteralReadsTariffScalars(t *testing.T) {
	t.Parallel()

	cases := map[string]rules.RawJSON{
		"$450":         rules.MoneyValue(decimal.NewFromInt(450)),
		"$1,250.50":    rules.MoneyValue(decimal.RequireFromString("1250.50")),
		"-$25":         rules.MoneyValue(decimal.NewFromInt(-25)),
		"$2.35/mi":     rules.QuantityValue(decimal.RequireFromString("2.35"), "$/mi"),
		"12%":          rules.PercentValue(decimal.NewFromInt(12)),
		"500 mi":       rules.QuantityValue(decimal.NewFromInt(500), "mi"),
		"3":            rules.NumberValue(decimal.NewFromInt(3)),
		"none":         rules.NoneValue(),
		`[{"$n":"1"}]`: rules.RawJSON(`[{"$n":"1"}]`),
	}
	for text, want := range cases {
		got, err := tarifftypes.ParseLiteral(text)
		require.NoError(t, err, text)
		assert.JSONEq(t, string(want), string(got), text)
	}
}

func TestParseLiteralRefusesText(t *testing.T) {
	t.Parallel()

	for _, text := range []string{"", "cheap", "$abc", "{not json"} {
		_, err := tarifftypes.ParseLiteral(text)
		require.ErrorIs(t, err, tarifftypes.ErrNotLiteral, text)
	}
}

func TestParseLiteralRoundTripsDisplay(t *testing.T) {
	t.Parallel()

	for _, text := range []string{"$450", "$2.35/mi", "12%", "500 mi", "3"} {
		raw, err := tarifftypes.ParseLiteral(text)
		require.NoError(t, err)
		assert.Equal(t, text, tarifftypes.DisplayValue(raw))
	}
}

func TestInputsEqualIgnoresDecimalScaleAndKeyOrder(t *testing.T) {
	t.Parallel()

	left := tarifftypes.Inputs{
		"rate":   rules.RawJSON(`{"$q":"2.15","u":"$/mi"}`),
		"breaks": rules.RawJSON(`[{"from":{"$q":"0","u":"lb"},"rate":{"$q":"12.5","u":"$/cwt"}}]`),
	}
	right := tarifftypes.Inputs{
		"rate":   rules.RawJSON(`{"u":"$/mi","$q":"2.150000"}`),
		"breaks": rules.RawJSON(`[{"rate":{"$q":"12.50","u":"$/cwt"},"from":{"$q":"0.0","u":"lb"}}]`),
	}
	assert.True(t, left.Equal(right))

	right["rate"] = rules.RawJSON(`{"$q":"2.16","u":"$/mi"}`)
	assert.False(t, left.Equal(right))

	right["rate"] = rules.RawJSON(`{"$q":"2.15","u":"$/km"}`)
	assert.False(t, left.Equal(right), "the unit is part of the value")
}
