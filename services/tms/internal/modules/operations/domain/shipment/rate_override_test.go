package shipment

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func rateOverrideErrors(amount decimal.NullDecimal) []string {
	multiErr := errortypes.NewMultiError()
	(&Shipment{RateOverrideAmount: amount}).Validate(multiErr)

	messages := make([]string, 0, 1)
	for _, err := range multiErr.Errors {
		if strings.Contains(err.Field, "rateOverrideAmount") {
			messages = append(messages, err.Message)
		}
	}

	return messages
}

func TestValidate_AcceptsAPositiveRateOverride(t *testing.T) {
	t.Parallel()

	assert.Empty(t, rateOverrideErrors(decimal.NewNullDecimal(decimal.NewFromFloat(5959.35))))
	assert.Empty(t, rateOverrideErrors(decimal.NullDecimal{}))
}

func TestValidate_RefusesANegativeRateOverride(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"An overridden rate cannot be negative"},
		rateOverrideErrors(decimal.NewNullDecimal(decimal.NewFromInt(-1))))
}
