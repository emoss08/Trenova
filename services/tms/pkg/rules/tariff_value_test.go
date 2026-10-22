package rules_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/pkg/rules"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTariffValuesEncodeTheWireForms(t *testing.T) {
	t.Parallel()

	assert.JSONEq(t, `{"$n":"1200"}`, string(rules.IntValue(1200)))
	assert.JSONEq(t, `{"$q":"2.35","u":"$/mi"}`,
		string(rules.QuantityValue(decimal.RequireFromString("2.35"), "$/mi")))
	assert.JSONEq(t, `{"$q":"450","u":"$"}`, string(rules.MoneyValue(decimal.NewFromInt(450))))
	assert.JSONEq(t, `{"$q":"18","u":"%"}`, string(rules.PercentValue(decimal.NewFromInt(18))))
	at := time.Date(2026, 10, 22, 8, 0, 0, 0, time.FixedZone("", -5*3600))
	assert.JSONEq(t, `{"$date":"2026-10-22"}`, string(rules.DateValue(at)))
	assert.JSONEq(t, `{"$date":"2026-10-22T08:00:00-05:00"}`, string(rules.DateTimeValue(at)))
	assert.Equal(t, `"Delivery"`, string(rules.TextValue("Delivery")))
	assert.Equal(t, "null", string(rules.NoneValue()))
}

func TestTariffValuesDecodeQuantitiesAndNumbers(t *testing.T) {
	t.Parallel()

	quantity, err := rules.DecodeQuantity(rules.RawJSON(`{"$q":"18.50","u":"$/cwt"}`))
	require.NoError(t, err)
	assert.True(t, quantity.IsMoneyRate())
	assert.Equal(t, "18.5", quantity.Value.String())

	_, err = rules.DecodeQuantity(rules.RawJSON(`{"$n":"3"}`))
	require.ErrorIs(t, err, rules.ErrNotTariffQuantity)

	number, err := rules.DecodeNumber(rules.RawJSON(`{"$n":"3"}`))
	require.NoError(t, err)
	assert.Equal(t, "3", number.String())

	_, err = rules.DecodeNumber(rules.RawJSON(`"3"`))
	require.ErrorIs(t, err, rules.ErrNotTariffNumber)
}
