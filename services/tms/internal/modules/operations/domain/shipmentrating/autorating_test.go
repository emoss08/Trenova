package shipmentrating

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func autoRatedPair(t *testing.T) (*shipment.Shipment, *shipment.Shipment) {
	t.Helper()

	templateID := pulid.MustNew("fmt_")
	agreementID := pulid.MustNew("ragr_")

	build := func() *shipment.Shipment {
		return &shipment.Shipment{
			FormulaTemplateID: templateID,
			TariffInputs: tarifftypes.Inputs{
				"rate": rules.QuantityValue(decimal.RequireFromString("2.15"), "$/mi"),
			},
			RateAgreementID: &agreementID,
			AutoRated:       true,
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

func TestRatedFieldsEdited_TariffInputs(t *testing.T) {
	t.Parallel()

	original, updated := autoRatedPair(t)
	updated.TariffInputs = tarifftypes.Inputs{
		"rate": rules.QuantityValue(decimal.RequireFromString("1.95"), "$/mi"),
	}

	assert.True(t, RatedFieldsEdited(original, updated))
}

// A decimal carries its scale, so two spellings of the same money are not ==
// equal. Treating 2.15 and 2.1500 as a renegotiation would mark half the book
// hand-priced the first time anything round-tripped through a form.
func TestRatedFieldsEdited_IgnoresDecimalScale(t *testing.T) {
	t.Parallel()

	original, updated := autoRatedPair(t)
	updated.TariffInputs = tarifftypes.Inputs{
		"rate": rules.RawJSON(`{"$q":"2.150000","u":"$/mi"}`),
	}

	assert.False(t, RatedFieldsEdited(original, updated))
}

func TestRatedFieldsEdited_ContractAccessorialRepriced(t *testing.T) {
	t.Parallel()

	original, updated := autoRatedPair(t)
	chargeID := pulid.MustNew("ac_")
	accessorialID := pulid.MustNew("raga_")

	contractCharge := func(amount int64) *shipment.AdditionalCharge {
		return &shipment.AdditionalCharge{
			ID:                         chargeID,
			AccessorialChargeID:        pulid.MustNew("acc_"),
			IsSystemGenerated:          true,
			Method:                     accessorialcharge.MethodFlat,
			Amount:                     decimal.NewFromInt(amount),
			Unit:                       1,
			RateAgreementAccessorialID: &accessorialID,
		}
	}

	original.AdditionalCharges = []*shipment.AdditionalCharge{contractCharge(75)}
	updated.AdditionalCharges = []*shipment.AdditionalCharge{contractCharge(25)}

	assert.True(t, RatedFieldsEdited(original, updated))
}

func TestRatedFieldsEdited_ContractAccessorialRemoved(t *testing.T) {
	t.Parallel()

	original, updated := autoRatedPair(t)
	accessorialID := pulid.MustNew("raga_")

	original.AdditionalCharges = []*shipment.AdditionalCharge{
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
	updated.AdditionalCharges = []*shipment.AdditionalCharge{
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

	fuelCharge := func(amount int64) *shipment.AdditionalCharge {
		return &shipment.AdditionalCharge{
			ID:                     chargeID,
			IsSystemGenerated:      true,
			Method:                 accessorialcharge.MethodFlat,
			Amount:                 decimal.NewFromInt(amount),
			Unit:                   1,
			FuelSurchargeProgramID: &programID,
		}
	}

	original.AdditionalCharges = []*shipment.AdditionalCharge{fuelCharge(120)}
	updated.AdditionalCharges = []*shipment.AdditionalCharge{fuelCharge(140)}

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
	original := &shipment.Shipment{AutoRated: true, AutoRatedAt: &ratedAt, RateLocked: true}
	updated := &shipment.Shipment{}

	RestoreRateOwnedFields(original, updated)

	assert.True(t, updated.AutoRated)
	assert.Equal(t, &ratedAt, updated.AutoRatedAt)
	assert.True(t, updated.RateLocked)
}
