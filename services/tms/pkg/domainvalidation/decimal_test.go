package domainvalidation

import (
	"testing"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestDecimalAtLeast_ComparesEveryDecimalShape(t *testing.T) {
	t.Parallel()

	rule := DecimalAtLeast(decimal.Zero, "cannot be negative")
	negative := decimal.NewFromInt(-5)
	positive := decimal.NewFromFloat(2.85)

	assert.NoError(t, validation.Validate(positive, rule))
	assert.NoError(t, validation.Validate(&positive, rule))
	assert.NoError(t, validation.Validate(decimal.NewNullDecimal(positive), rule))
	assert.NoError(t, validation.Validate(decimal.NullDecimal{}, rule), "an absent amount is not checked")
	assert.EqualError(t, validation.Validate(negative, rule), "cannot be negative")
	assert.EqualError(t, validation.Validate(decimal.NewNullDecimal(negative), rule), "cannot be negative")
	assert.Error(t, validation.Validate("12", rule))
}

func TestDecimalAtMost_RefusesAboveTheBound(t *testing.T) {
	t.Parallel()

	rule := DecimalAtMost(decimal.NewFromInt(1), "must be <= 1")
	assert.NoError(t, validation.Validate(decimal.NewFromFloat(0.75), rule))
	assert.EqualError(t, validation.Validate(decimal.NewFromFloat(1.5), rule), "must be <= 1")
}
