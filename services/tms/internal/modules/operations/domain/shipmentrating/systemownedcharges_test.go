package shipmentrating

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

func TestRestoreSystemOwnedCharges_CallersCannotClaimSystemOwnership(t *testing.T) {
	t.Parallel()

	detentionChargeID := pulid.MustNew("ac_")
	manualChargeID := pulid.MustNew("ac_")

	original := []*shipment.AdditionalCharge{
		{
			ID:                detentionChargeID,
			IsSystemGenerated: true,
			IsDetention:       true,
		},
		{ID: manualChargeID},
	}

	updated := []*shipment.AdditionalCharge{
		{ID: detentionChargeID},
		{ID: manualChargeID, IsSystemGenerated: true, IsDetention: true},
		{IsSystemGenerated: true, IsDetention: true},
		nil,
	}

	RestoreSystemOwnedCharges(original, updated)

	require.True(t, updated[0].IsSystemGenerated, "a dropped system flag is restored")
	require.True(t, updated[0].IsDetention, "a dropped detention marker is restored")

	require.False(t, updated[1].IsSystemGenerated, "a manual charge cannot promote itself")
	require.False(t, updated[1].IsDetention, "a manual charge cannot claim detention")

	require.False(t, updated[2].IsSystemGenerated, "a new charge is never system generated")
	require.False(t, updated[2].IsDetention, "a new charge cannot claim detention")
}

func TestRestoreSystemOwnedCharges_UnknownChargeIsTreatedAsManual(t *testing.T) {
	t.Parallel()

	updated := []*shipment.AdditionalCharge{
		{
			ID:                pulid.MustNew("ac_"),
			IsSystemGenerated: true,
			IsDetention:       true,
		},
	}

	RestoreSystemOwnedCharges(nil, updated)

	require.False(t, updated[0].IsSystemGenerated)
	require.False(t, updated[0].IsDetention)
}

// Three engines write system charges, and the column that names the owner is
// what keeps them from billing the same thing twice. A payload that drops one
// would orphan the charge and let the next recalculation add it again.
func TestRestoreSystemOwnedCharges_ProtectsEveryOwnerColumn(t *testing.T) {
	t.Parallel()

	accessorialID := pulid.MustNew("raga_")
	programID := pulid.MustNew("fsp_")
	agreementChargeID := pulid.MustNew("ac_")
	fuelChargeID := pulid.MustNew("ac_")

	original := []*shipment.AdditionalCharge{
		{
			ID:                         agreementChargeID,
			IsSystemGenerated:          true,
			RateAgreementAccessorialID: &accessorialID,
		},
		{
			ID:                     fuelChargeID,
			IsSystemGenerated:      true,
			FuelSurchargeProgramID: &programID,
		},
	}

	forgedAccessorialID := pulid.MustNew("raga_")
	updated := []*shipment.AdditionalCharge{
		{ID: agreementChargeID},
		{ID: fuelChargeID},
		{IsSystemGenerated: true, RateAgreementAccessorialID: &forgedAccessorialID},
	}

	RestoreSystemOwnedCharges(original, updated)

	require.True(t, updated[0].IsSystemGenerated)
	require.Equal(t, accessorialID, *updated[0].RateAgreementAccessorialID,
		"a dropped contract link is restored")

	require.True(t, updated[1].IsSystemGenerated)
	require.Equal(t, programID, *updated[1].FuelSurchargeProgramID,
		"a dropped fuel program link is restored")

	require.False(t, updated[2].IsSystemGenerated)
	require.Nil(t, updated[2].RateAgreementAccessorialID,
		"a new charge cannot claim a contract accessorial")
}
