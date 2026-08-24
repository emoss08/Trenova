package shipmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A create payload carries the charges the billing panel showed the rater,
// including the ones the contract preview named. None of the engines that own
// those have run yet — they run later in the same save, from the agreement and
// the controls — so anything arriving flagged as theirs is a copy, and keeping
// it bills the customer for the same accessorial twice.
func TestDropSystemGeneratedAdditionalChargesForCreate(t *testing.T) {
	t.Parallel()

	accessorialID := pulid.MustNew("racc_")
	programID := pulid.MustNew("fsp_")
	occurrenceID := pulid.MustNew("dtno_")
	tarpID := pulid.MustNew("acc_")
	lumperID := pulid.MustNew("acc_")

	entity := &shipment.Shipment{
		AdditionalCharges: []*shipment.AdditionalCharge{
			{
				AccessorialChargeID:        tarpID,
				IsSystemGenerated:          true,
				Amount:                     decimal.NewFromInt(75),
				Unit:                       1,
				RateAgreementAccessorialID: &accessorialID,
			},
			{
				AccessorialChargeID:    pulid.MustNew("acc_"),
				IsSystemGenerated:      true,
				Amount:                 decimal.NewFromInt(120),
				Unit:                   1,
				FuelSurchargeProgramID: &programID,
			},
			{
				AccessorialChargeID:   pulid.MustNew("acc_"),
				IsSystemGenerated:     true,
				Amount:                decimal.NewFromInt(50),
				Unit:                  2,
				DetentionOccurrenceID: &occurrenceID,
			},
			nil,
			{
				AccessorialChargeID: lumperID,
				IsSystemGenerated:   false,
				Amount:              decimal.NewFromInt(40),
				Unit:                1,
			},
		},
	}

	(&service{}).dropSystemGeneratedAdditionalChargesForCreate(entity)

	require.Len(t, entity.AdditionalCharges, 1)
	assert.Equal(t, lumperID, entity.AdditionalCharges[0].AccessorialChargeID)
	assert.False(t, entity.AdditionalCharges[0].IsSystemGenerated)
}

// A charge the rater typed must not be able to claim an engine's row by naming
// one: the owner columns are what the reconciliation passes tell their own rows
// apart by, and a forged one would survive the pass that should have replaced it.
func TestDropSystemGeneratedAdditionalChargesForCreate_ClearsForgedOwnership(t *testing.T) {
	t.Parallel()

	accessorialID := pulid.MustNew("racc_")
	programID := pulid.MustNew("fsp_")
	occurrenceID := pulid.MustNew("dtno_")

	entity := &shipment.Shipment{
		AdditionalCharges: []*shipment.AdditionalCharge{
			{
				AccessorialChargeID:        pulid.MustNew("acc_"),
				IsSystemGenerated:          false,
				Amount:                     decimal.NewFromInt(40),
				Unit:                       1,
				RateAgreementAccessorialID: &accessorialID,
				FuelSurchargeProgramID:     &programID,
				DetentionOccurrenceID:      &occurrenceID,
			},
		},
	}

	(&service{}).dropSystemGeneratedAdditionalChargesForCreate(entity)

	require.Len(t, entity.AdditionalCharges, 1)
	charge := entity.AdditionalCharges[0]
	assert.Nil(t, charge.RateAgreementAccessorialID)
	assert.Nil(t, charge.FuelSurchargeProgramID)
	assert.Nil(t, charge.DetentionOccurrenceID)
	assert.Equal(t, shipment.SystemOwnerNone, charge.Owner())
}

func TestDropSystemGeneratedAdditionalChargesForCreate_NilShipment(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, func() {
		(&service{}).dropSystemGeneratedAdditionalChargesForCreate(nil)
	})
}
