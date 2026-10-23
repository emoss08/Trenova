package rateagreement

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func perUnit(unit accessorialcharge.RateUnit, free int16) *RateAgreementAccessorial {
	return &RateAgreementAccessorial{
		RateAgreementID:     pulid.MustNew("ra_"),
		AccessorialChargeID: pulid.MustNew("acc_"),
		Method:              accessorialcharge.MethodPerUnit,
		RateUnit:            unit,
		Amount:              decimal.NewFromInt(10),
		FreeUnits:           &free,
	}
}

func measures() *AccessorialMeasures {
	return &AccessorialMeasures{
		Miles:  decimal.NewFromInt(120),
		Dwells: []decimal.Decimal{decimal.NewFromInt(3), decimal.NewFromInt(1)},
		Days:   decimal.NewFromInt(2),
		Stops:  decimal.NewFromInt(4),
	}
}

func TestBillableQuantitySubtractsFreeUnitsPerRateUnit(t *testing.T) {
	t.Parallel()

	cases := map[accessorialcharge.RateUnit]string{
		accessorialcharge.RateUnitMile: "110",
		accessorialcharge.RateUnitHour: "1",
		accessorialcharge.RateUnitDay:  "0",
		accessorialcharge.RateUnitStop: "0",
	}
	free := map[accessorialcharge.RateUnit]int16{
		accessorialcharge.RateUnitMile: 10,
		accessorialcharge.RateUnitHour: 2,
		accessorialcharge.RateUnitDay:  5,
		accessorialcharge.RateUnitStop: 4,
	}
	for unit, want := range cases {
		got := perUnit(unit, free[unit]).BillableQuantity(measures())
		assert.Truef(t, decimal.RequireFromString(want).Equal(got), "%s: want %s, got %s", unit, want, got)
	}
}

func TestBillableQuantityIsOneForAFlatCharge(t *testing.T) {
	t.Parallel()

	flat := &RateAgreementAccessorial{Method: accessorialcharge.MethodFlat}
	assert.True(t, decimal.NewFromInt(1).Equal(flat.BillableQuantity(measures())))
}

func TestCapHoldsAChargeAtItsMaximum(t *testing.T) {
	t.Parallel()

	row := perUnit(accessorialcharge.RateUnitMile, 0)
	row.MaxAmount = decimal.NewNullDecimal(decimal.NewFromInt(500))

	capped, applied := row.Cap(decimal.NewFromInt(900))
	assert.True(t, applied)
	assert.True(t, capped.Equal(decimal.NewFromInt(500)))

	kept, applied := row.Cap(decimal.NewFromInt(300))
	assert.False(t, applied)
	assert.True(t, kept.Equal(decimal.NewFromInt(300)))
}

func TestTariffRateUnitSpeaksTheBillsUnits(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "$/h", perUnit(accessorialcharge.RateUnitHour, 0).TariffRateUnit())
	assert.Empty(t, (&RateAgreementAccessorial{Method: accessorialcharge.MethodFlat}).TariffRateUnit())
}

func TestAccessorialConditionLengthIsBounded(t *testing.T) {
	t.Parallel()

	row := &RateAgreementAccessorial{
		RateAgreementID:     pulid.MustNew("ra_"),
		AccessorialChargeID: pulid.MustNew("acc_"),
		Method:              accessorialcharge.MethodFlat,
		Amount:              decimal.NewFromInt(25),
		AutoApply:           true,
		Condition:           "condition \"Long\"\n\n" + strings.Repeat("true and ", 3000) + "true\n",
	}
	multiErr := errortypes.NewMultiError()
	row.Validate(multiErr)
	assert.Contains(t, multiErr.Error(), "condition")
}
