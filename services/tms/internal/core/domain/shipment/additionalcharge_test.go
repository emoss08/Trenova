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
