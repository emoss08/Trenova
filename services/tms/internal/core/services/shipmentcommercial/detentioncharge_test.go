package shipmentcommercial

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func chargeableOccurrence(amount string) *detention.DetentionOccurrence {
	return &detention.DetentionOccurrence{
		ID:             pulid.MustNew("dto_"),
		Status:         detention.OccurrenceStatusPending,
		BillableAmount: decimal.RequireFromString(amount),
		PolicySnapshot: &detention.PolicySnapshot{
			AccessorialChargeID: pulid.MustNew("acc_"),
		},
	}
}

func TestReconcileDetentionOccurrenceCharges_KeepsOneChargePerOccurrence(t *testing.T) {
	t.Parallel()

	entity := validShipment()
	manual := &shipment.AdditionalCharge{
		ID:                  pulid.MustNew("ac_"),
		AccessorialChargeID: pulid.MustNew("acc_"),
		Amount:              decimal.NewFromInt(25),
	}
	entity.AdditionalCharges = []*shipment.AdditionalCharge{manual}

	occurrence := chargeableOccurrence("150")

	reconcileDetentionOccurrenceCharges(
		entity,
		[]*detention.DetentionOccurrence{occurrence},
	)
	require.Len(t, entity.AdditionalCharges, 2)
	require.Same(t, manual, entity.AdditionalCharges[0])

	generated := entity.AdditionalCharges[1]
	require.True(t, generated.IsSystemGenerated)
	require.Equal(t, occurrence.ID, *generated.DetentionOccurrenceID)
	require.True(t, generated.Amount.Equal(decimal.NewFromInt(150)))

	generated.ID = pulid.MustNew("ac_")
	generated.Version = 3
	persistedID := generated.ID

	occurrence.BillableAmount = decimal.NewFromInt(180)

	reconcileDetentionOccurrenceCharges(
		entity,
		[]*detention.DetentionOccurrence{occurrence},
	)
	require.Len(t, entity.AdditionalCharges, 2, "a re-sync must not duplicate the charge")

	resynced := entity.AdditionalCharges[1]
	require.Equal(t, persistedID, resynced.ID, "the row keeps its identity across saves")
	require.Equal(t, int64(3), resynced.Version)
	require.True(t, resynced.Amount.Equal(decimal.NewFromInt(180)))
}

func TestReconcileDetentionOccurrenceCharges_DropsChargesThatStoppedBilling(t *testing.T) {
	t.Parallel()

	entity := validShipment()
	occurrence := chargeableOccurrence("150")
	reconcileDetentionOccurrenceCharges(entity, []*detention.DetentionOccurrence{occurrence})
	require.Len(t, entity.AdditionalCharges, 1)

	waived := chargeableOccurrence("0")
	waived.ID = occurrence.ID
	waived.Status = detention.OccurrenceStatusWaived

	reconcileDetentionOccurrenceCharges(entity, []*detention.DetentionOccurrence{waived})
	require.Empty(t, entity.AdditionalCharges)
}

func TestReconcileDetentionOccurrenceCharges_IgnoresOccurrencesWithoutASnapshot(t *testing.T) {
	t.Parallel()

	entity := validShipment()
	orphan := chargeableOccurrence("150")
	orphan.PolicySnapshot = nil

	reconcileDetentionOccurrenceCharges(entity, []*detention.DetentionOccurrence{orphan})
	require.Empty(t, entity.AdditionalCharges)
}
