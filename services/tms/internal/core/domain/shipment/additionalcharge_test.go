package shipment

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

func TestRestoreSystemOwnedCharges_CallersCannotClaimSystemOwnership(t *testing.T) {
	t.Parallel()

	occurrenceID := pulid.MustNew("dto_")
	detentionChargeID := pulid.MustNew("ac_")
	manualChargeID := pulid.MustNew("ac_")

	original := []*AdditionalCharge{
		{
			ID:                    detentionChargeID,
			IsSystemGenerated:     true,
			DetentionOccurrenceID: &occurrenceID,
		},
		{ID: manualChargeID},
	}

	forgedOccurrenceID := pulid.MustNew("dto_")
	updated := []*AdditionalCharge{
		{ID: detentionChargeID},
		{ID: manualChargeID, IsSystemGenerated: true, DetentionOccurrenceID: &forgedOccurrenceID},
		{IsSystemGenerated: true, DetentionOccurrenceID: &forgedOccurrenceID},
		nil,
	}

	RestoreSystemOwnedCharges(original, updated)

	require.True(t, updated[0].IsSystemGenerated, "a dropped system flag is restored")
	require.Equal(t, occurrenceID, *updated[0].DetentionOccurrenceID,
		"a dropped detention link is restored")

	require.False(t, updated[1].IsSystemGenerated, "a manual charge cannot promote itself")
	require.Nil(t, updated[1].DetentionOccurrenceID, "a manual charge cannot claim an occurrence")

	require.False(t, updated[2].IsSystemGenerated, "a new charge is never system generated")
	require.Nil(t, updated[2].DetentionOccurrenceID, "a new charge cannot claim an occurrence")
}

func TestRestoreSystemOwnedCharges_UnknownChargeIsTreatedAsManual(t *testing.T) {
	t.Parallel()

	occurrenceID := pulid.MustNew("dto_")
	updated := []*AdditionalCharge{
		{
			ID:                    pulid.MustNew("ac_"),
			IsSystemGenerated:     true,
			DetentionOccurrenceID: &occurrenceID,
		},
	}

	RestoreSystemOwnedCharges(nil, updated)

	require.False(t, updated[0].IsSystemGenerated)
	require.Nil(t, updated[0].DetentionOccurrenceID)
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

	original := []*AdditionalCharge{
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
	updated := []*AdditionalCharge{
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

func TestAdditionalCharge_OwnerNamesExactlyOneEngine(t *testing.T) {
	t.Parallel()

	occurrenceID := pulid.MustNew("dto_")
	programID := pulid.MustNew("fsp_")
	accessorialID := pulid.MustNew("raga_")

	require.Equal(t, SystemOwnerNone, (&AdditionalCharge{}).Owner(),
		"a charge nobody generated has no owner")
	require.Equal(t, SystemOwnerNone,
		(&AdditionalCharge{DetentionOccurrenceID: &occurrenceID}).Owner(),
		"an owner column on a manual charge does not make it system owned")

	require.Equal(t, SystemOwnerFuel, (&AdditionalCharge{
		IsSystemGenerated:      true,
		FuelSurchargeProgramID: &programID,
	}).Owner())
	require.Equal(t, SystemOwnerDetention, (&AdditionalCharge{
		IsSystemGenerated:     true,
		DetentionOccurrenceID: &occurrenceID,
	}).Owner())
	require.Equal(t, SystemOwnerAgreement, (&AdditionalCharge{
		IsSystemGenerated:          true,
		RateAgreementAccessorialID: &accessorialID,
	}).Owner())
}
