package rateagreement_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const selectAt = int64(1_700_000_000)

func autoRow(id pulid.ID) *rateagreement.RateAgreementAccessorial {
	return &rateagreement.RateAgreementAccessorial{
		ID:                  id,
		AccessorialChargeID: pulid.MustNew("acc_"),
		AutoApply:           true,
		Amount:              decimal.NewFromInt(75),
	}
}

func selectFacts() rateagreement.AccessorialFacts {
	return rateagreement.AccessorialFacts{
		At:             selectAt,
		ServiceTypeID:  pulid.MustNew("st_"),
		ShipmentTypeID: pulid.MustNew("sht_"),
	}
}

// The contract is what decides which of its own rows apply, so the sell side
// and the buy side cannot drift apart on the answer.
func TestAutoApplyAccessorials_TakesTheRowsMarkedAutomatic(t *testing.T) {
	t.Parallel()

	automatic := autoRow(pulid.MustNew("raga_"))

	manual := autoRow(pulid.MustNew("raga_"))
	manual.AutoApply = false

	agreement := &rateagreement.RateAgreement{
		Accessorials: []*rateagreement.RateAgreementAccessorial{automatic, manual},
	}

	got := agreement.AutoApplyAccessorials(selectFacts())

	require.Len(t, got, 1)
	assert.Equal(t, automatic.ID, got[0].ID)
}

// A waived row is a price of zero the contract promised, not a charge to apply.
// Applying it would bill the customer for something they were told they would
// not see.
func TestAutoApplyAccessorials_SkipsWaivedRows(t *testing.T) {
	t.Parallel()

	waived := autoRow(pulid.MustNew("raga_"))
	waived.Waived = true

	agreement := &rateagreement.RateAgreement{
		Accessorials: []*rateagreement.RateAgreementAccessorial{waived},
	}

	assert.Empty(t, agreement.AutoApplyAccessorials(selectFacts()))
}

func TestAutoApplyAccessorials_SkipsRowsNotYetInEffect(t *testing.T) {
	t.Parallel()

	future := autoRow(pulid.MustNew("raga_"))
	tomorrow := selectAt + 86400
	future.EffectiveFrom = &tomorrow

	agreement := &rateagreement.RateAgreement{
		Accessorials: []*rateagreement.RateAgreementAccessorial{future},
	}

	assert.Empty(t, agreement.AutoApplyAccessorials(selectFacts()))
}

// An empty set means the row does not care, which is the convention every other
// scoped record in the system uses. Reading it as "matches nothing" would make
// every unscoped row silently stop applying.
func TestAutoApplyAccessorials_EmptyServiceTypeSetMeansAnyServiceType(t *testing.T) {
	t.Parallel()

	row := autoRow(pulid.MustNew("raga_"))

	agreement := &rateagreement.RateAgreement{
		Accessorials: []*rateagreement.RateAgreementAccessorial{row},
	}

	assert.Len(t, agreement.AutoApplyAccessorials(selectFacts()), 1)
}

func TestAutoApplyAccessorials_SkipsRowsScopedToAnotherServiceType(t *testing.T) {
	t.Parallel()

	facts := selectFacts()

	row := autoRow(pulid.MustNew("raga_"))
	row.ServiceTypeIDs = []pulid.ID{pulid.MustNew("st_")}

	agreement := &rateagreement.RateAgreement{
		Accessorials: []*rateagreement.RateAgreementAccessorial{row},
	}

	assert.Empty(t, agreement.AutoApplyAccessorials(facts))
}

func TestAutoApplyAccessorials_KeepsRowsScopedToThisShipmentType(t *testing.T) {
	t.Parallel()

	facts := selectFacts()

	row := autoRow(pulid.MustNew("raga_"))
	row.ShipmentTypeIDs = []pulid.ID{pulid.MustNew("sht_"), facts.ShipmentTypeID}

	agreement := &rateagreement.RateAgreement{
		Accessorials: []*rateagreement.RateAgreementAccessorial{row},
	}

	assert.Len(t, agreement.AutoApplyAccessorials(facts), 1)
}

// A row carrying a condition is returned rather than dropped, because only the
// caller has a formula engine to answer it with. Dropping it here would silently
// disable every conditional accessorial in the product.
func TestAutoApplyAccessorials_ReturnsConditionalRowsForTheCallerToTest(t *testing.T) {
	t.Parallel()

	row := autoRow(pulid.MustNew("raga_"))
	row.ApplyCondition = "totalStops > 2"

	agreement := &rateagreement.RateAgreement{
		Accessorials: []*rateagreement.RateAgreementAccessorial{row},
	}

	got := agreement.AutoApplyAccessorials(selectFacts())

	require.Len(t, got, 1)
	assert.Equal(t, "totalStops > 2", got[0].ApplyCondition)
}

func TestAutoApplyAccessorials_NilAgreementHasNothingToApply(t *testing.T) {
	t.Parallel()

	var agreement *rateagreement.RateAgreement

	assert.Empty(t, agreement.AutoApplyAccessorials(selectFacts()))
}

func TestAutoApplyAccessorials_SkipsNilRows(t *testing.T) {
	t.Parallel()

	agreement := &rateagreement.RateAgreement{
		Accessorials: []*rateagreement.RateAgreementAccessorial{nil, autoRow(pulid.MustNew("raga_"))},
	}

	assert.Len(t, agreement.AutoApplyAccessorials(selectFacts()), 1)
}
