package shipment

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func chargeAt(method accessorialcharge.Method, amount string, unit int16) *AdditionalCharge {
	return &AdditionalCharge{
		AccessorialChargeID: pulid.MustNew("acc_"),
		Method:              method,
		Amount:              decimal.RequireFromString(amount),
		Unit:                unit,
	}
}

func amountErrors(charge *AdditionalCharge) []string {
	multiErr := errortypes.NewMultiError()
	charge.Validate(multiErr)
	var messages []string
	for _, err := range multiErr.Errors {
		if err.Field == "amount" {
			messages = append(messages, err.Message)
		}
	}
	return messages
}

func TestAdditionalCharge_AcceptsARateUpToTheMostItCanBill(t *testing.T) {
	t.Parallel()

	assert.Empty(t, amountErrors(chargeAt(accessorialcharge.MethodFlat, "250000.00", 1)))
	assert.Empty(t, amountErrors(chargeAt(accessorialcharge.MethodFlat, "99999999999.9999", 1)))
	assert.Empty(
		t,
		amountErrors(chargeAt(accessorialcharge.MethodPerUnit, "30000000000", 32767)),
	)
}

func TestAdditionalCharge_RefusesARatePastTheMostItCanBill(t *testing.T) {
	t.Parallel()

	require.Equal(
		t,
		[]string{"amount must be at most 99,999,999,999.9999"},
		amountErrors(chargeAt(accessorialcharge.MethodFlat, "100000000000", 1)),
	)
}

func TestAdditionalCharge_RefusesAPerUnitLineWhoseTotalOverflowsItsColumn(t *testing.T) {
	t.Parallel()

	require.Equal(
		t,
		[]string{"amount times unit must be at most 999,999,999,999,999.9999"},
		amountErrors(chargeAt(accessorialcharge.MethodPerUnit, "40000000000", 32767)),
	)
	assert.Empty(t, amountErrors(chargeAt(accessorialcharge.MethodFlat, "40000000000", 32767)))
}
