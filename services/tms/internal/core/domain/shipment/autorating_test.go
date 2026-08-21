package shipment

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func autoRatedPair(t *testing.T) (*Shipment, *Shipment) {
	t.Helper()

	templateID := pulid.MustNew("fmt_")
	agreementID := pulid.MustNew("ragr_")

	build := func() *Shipment {
		return &Shipment{
			FormulaTemplateID: templateID,
			BaseRate:          decimal.NewNullDecimal(decimal.NewFromFloat(2.15)),
			RateAgreementID:   &agreementID,
			AutoRated:         true,
		}
	}

	return build(), build()
}

// Editing the rating method is the rater saying the contract's answer is not
// the one being billed.
func TestRatedFieldsEdited_RatingMethod(t *testing.T) {
	t.Parallel()

	original, updated := autoRatedPair(t)
	updated.FormulaTemplateID = pulid.MustNew("fmt_")

	assert.True(t, RatedFieldsEdited(original, updated))
}

func TestRatedFieldsEdited_BaseRate(t *testing.T) {
	t.Parallel()

	original, updated := autoRatedPair(t)
	updated.BaseRate = decimal.NewNullDecimal(decimal.NewFromFloat(1.95))

	assert.True(t, RatedFieldsEdited(original, updated))
}

// A decimal carries its scale, so two spellings of the same money are not ==
// equal. Treating 2.15 and 2.1500 as a renegotiation would mark half the book
// hand-priced the first time anything round-tripped through a form.
func TestRatedFieldsEdited_IgnoresDecimalScale(t *testing.T) {
	t.Parallel()

	original, updated := autoRatedPair(t)
	updated.BaseRate = decimal.NewNullDecimal(decimal.RequireFromString("2.150000"))

	assert.False(t, RatedFieldsEdited(original, updated))
}

func TestRatedFieldsEdited_ContractAccessorialRepriced(t *testing.T) {
	t.Parallel()

	original, updated := autoRatedPair(t)
	chargeID := pulid.MustNew("ac_")
	accessorialID := pulid.MustNew("raga_")

	contractCharge := func(amount int64) *AdditionalCharge {
		return &AdditionalCharge{
			ID:                         chargeID,
			AccessorialChargeID:        pulid.MustNew("acc_"),
			IsSystemGenerated:          true,
			Method:                     accessorialcharge.MethodFlat,
			Amount:                     decimal.NewFromInt(amount),
			Unit:                       1,
			RateAgreementAccessorialID: &accessorialID,
		}
	}

	original.AdditionalCharges = []*AdditionalCharge{contractCharge(75)}
	updated.AdditionalCharges = []*AdditionalCharge{contractCharge(25)}

	assert.True(t, RatedFieldsEdited(original, updated))
}

func TestRatedFieldsEdited_ContractAccessorialRemoved(t *testing.T) {
	t.Parallel()

	original, updated := autoRatedPair(t)
	accessorialID := pulid.MustNew("raga_")

	original.AdditionalCharges = []*AdditionalCharge{
		{
			ID:                         pulid.MustNew("ac_"),
			IsSystemGenerated:          true,
			Method:                     accessorialcharge.MethodFlat,
			Amount:                     decimal.NewFromInt(75),
			Unit:                       1,
			RateAgreementAccessorialID: &accessorialID,
		},
	}
	updated.AdditionalCharges = nil

	assert.True(t, RatedFieldsEdited(original, updated))
}

// A clerk billing a lumper fee has not renegotiated the linehaul. Counting it
// as a departure would empty the contract usage report of exactly the shipments
// that prove the contracts work.
func TestRatedFieldsEdited_IgnoresAChargeAddedByHand(t *testing.T) {
	t.Parallel()

	original, updated := autoRatedPair(t)
	updated.AdditionalCharges = []*AdditionalCharge{
		{
			ID:                  pulid.MustNew("ac_"),
			AccessorialChargeID: pulid.MustNew("acc_"),
			Method:              accessorialcharge.MethodFlat,
			Amount:              decimal.NewFromInt(50),
			Unit:                1,
		},
	}

	assert.False(t, RatedFieldsEdited(original, updated))
}

// A fuel surcharge is repriced by its own engine on every save, and a detention
// charge by the detention engine. Neither is the contract's rate.
func TestRatedFieldsEdited_IgnoresOtherEnginesCharges(t *testing.T) {
	t.Parallel()

	original, updated := autoRatedPair(t)
	programID := pulid.MustNew("fsp_")
	chargeID := pulid.MustNew("ac_")

	fuelCharge := func(amount int64) *AdditionalCharge {
		return &AdditionalCharge{
			ID:                     chargeID,
			IsSystemGenerated:      true,
			Method:                 accessorialcharge.MethodFlat,
			Amount:                 decimal.NewFromInt(amount),
			Unit:                   1,
			FuelSurchargeProgramID: &programID,
		}
	}

	original.AdditionalCharges = []*AdditionalCharge{fuelCharge(120)}
	updated.AdditionalCharges = []*AdditionalCharge{fuelCharge(140)}

	assert.False(t, RatedFieldsEdited(original, updated))
}

func TestRatedFieldsEdited_UntouchedShipment(t *testing.T) {
	t.Parallel()

	original, updated := autoRatedPair(t)

	assert.False(t, RatedFieldsEdited(original, updated))
	assert.False(t, RatedFieldsEdited(nil, updated))
	assert.False(t, RatedFieldsEdited(original, nil))
}

// The rating fields are system-owned on every ordinary save, so a client that
// round-trips a shipment without them cannot unlock an invoiced shipment or
// claim a contract priced one it did not.
func TestRestoreRateOwnedFields_ReSeatsTheAutoRatedMarker(t *testing.T) {
	t.Parallel()

	ratedAt := int64(7300)
	original := &Shipment{AutoRated: true, AutoRatedAt: &ratedAt, RateLocked: true}
	updated := &Shipment{}

	RestoreRateOwnedFields(original, updated)

	assert.True(t, updated.AutoRated)
	assert.Equal(t, &ratedAt, updated.AutoRatedAt)
	assert.True(t, updated.RateLocked)
}

func TestClearAutoRating(t *testing.T) {
	t.Parallel()

	ratedAt := int64(7300)
	entity := &Shipment{AutoRated: true, AutoRatedAt: &ratedAt}

	entity.ClearAutoRating()

	assert.False(t, entity.AutoRated)
	assert.Nil(t, entity.AutoRatedAt)
}

func TestMarkAutoRated(t *testing.T) {
	t.Parallel()

	entity := &Shipment{}
	entity.MarkAutoRated(7300)

	assert.True(t, entity.AutoRated)
	assert.Equal(t, int64(7300), *entity.AutoRatedAt)
}
