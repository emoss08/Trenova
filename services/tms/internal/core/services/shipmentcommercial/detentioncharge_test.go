package shipmentcommercial

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func chargeableOccurrence(amount string) *detention.DetentionOccurrence {
	return chargeableOccurrenceFor(pulid.MustNew("acc_"), amount)
}

func chargeableOccurrenceFor(accessorialID pulid.ID, amount string) *detention.DetentionOccurrence {
	return &detention.DetentionOccurrence{
		ID:             pulid.MustNew("dto_"),
		Status:         detention.OccurrenceStatusPending,
		BillableAmount: decimal.RequireFromString(amount),
		PolicySnapshot: &detention.PolicySnapshot{
			AccessorialChargeID: accessorialID,
		},
	}
}

func TestReconcileDetentionOccurrenceCharges_KeepsOneChargeAcrossSaves(t *testing.T) {
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
	require.True(t, generated.IsDetention)
	require.Equal(t, shipment.SystemOwnerDetention, generated.Owner())
	require.Equal(t, []pulid.ID{occurrence.ID}, generated.DetentionOccurrenceIDs)
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

func TestReconcileDetentionOccurrenceCharges_FoldsEveryStopIntoOneCharge(t *testing.T) {
	t.Parallel()

	entity := validShipment()
	accessorialID := pulid.MustNew("acc_")
	pickup := chargeableOccurrenceFor(accessorialID, "112.50")
	delivery := chargeableOccurrenceFor(accessorialID, "450")

	reconcileDetentionOccurrenceCharges(
		entity,
		[]*detention.DetentionOccurrence{pickup, delivery},
	)
	require.Len(t, entity.AdditionalCharges, 1, "two detained stops bill as one detention charge")

	generated := entity.AdditionalCharges[0]
	require.True(t, generated.Amount.Equal(decimal.RequireFromString("562.50")))
	require.Equal(t, int16(1), generated.Unit)
	require.Equal(t, []pulid.ID{pickup.ID, delivery.ID}, generated.DetentionOccurrenceIDs)

	generated.ID = pulid.MustNew("ac_")
	persistedID := generated.ID

	delivery.Status = detention.OccurrenceStatusWaived
	delivery.BillableAmount = decimal.Zero

	reconcileDetentionOccurrenceCharges(
		entity,
		[]*detention.DetentionOccurrence{pickup, delivery},
	)
	require.Len(t, entity.AdditionalCharges, 1)
	require.Equal(t, persistedID, entity.AdditionalCharges[0].ID)
	require.True(t, entity.AdditionalCharges[0].Amount.Equal(decimal.RequireFromString("112.50")),
		"a waived stop leaves the charge, the charge itself stays")
	require.Equal(t, []pulid.ID{pickup.ID}, entity.AdditionalCharges[0].DetentionOccurrenceIDs)
}

func TestReconcileDetentionOccurrenceCharges_OneChargePerAccessorial(t *testing.T) {
	t.Parallel()

	entity := validShipment()
	detentionID := pulid.MustNew("acc_")
	layoverID := pulid.MustNew("acc_")

	reconcileDetentionOccurrenceCharges(entity, []*detention.DetentionOccurrence{
		chargeableOccurrenceFor(detentionID, "100"),
		chargeableOccurrenceFor(layoverID, "300"),
		chargeableOccurrenceFor(detentionID, "50"),
	})
	require.Len(t, entity.AdditionalCharges, 2)
	require.Equal(t, detentionID, entity.AdditionalCharges[0].AccessorialChargeID)
	require.True(t, entity.AdditionalCharges[0].Amount.Equal(decimal.NewFromInt(150)))
	require.Equal(t, layoverID, entity.AdditionalCharges[1].AccessorialChargeID)
	require.True(t, entity.AdditionalCharges[1].Amount.Equal(decimal.NewFromInt(300)))
}

func TestReconcileDetentionOccurrenceCharges_AbsorbsDuplicateAndLegacyRows(t *testing.T) {
	t.Parallel()

	entity := validShipment()
	accessorialID := pulid.MustNew("acc_")

	legacy := &shipment.AdditionalCharge{
		ID:                  pulid.MustNew("ac_"),
		AccessorialChargeID: accessorialID,
		IsSystemGenerated:   true,
		Amount:              decimal.NewFromInt(75),
		Unit:                2,
		CreatedAt:           100,
	}
	later := &shipment.AdditionalCharge{
		ID:                  pulid.MustNew("ac_"),
		AccessorialChargeID: accessorialID,
		IsSystemGenerated:   true,
		IsDetention:         true,
		Amount:              decimal.NewFromInt(450),
		Unit:                1,
		CreatedAt:           200,
	}
	entity.AdditionalCharges = []*shipment.AdditionalCharge{later, legacy}

	occurrence := chargeableOccurrenceFor(accessorialID, "562.50")
	reconcileDetentionOccurrenceCharges(entity, []*detention.DetentionOccurrence{occurrence})

	require.Len(t, entity.AdditionalCharges, 1, "the legacy row and the engine row fold into one")
	kept := entity.AdditionalCharges[0]
	require.Equal(t, legacy.ID, kept.ID, "the oldest row keeps the identity")
	require.True(t, kept.IsDetention)
	require.Equal(t, int16(1), kept.Unit)
	require.True(t, kept.Amount.Equal(decimal.RequireFromString("562.50")))
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

func TestEnsureGeneratedDetentionCharge_ReusesTheOneRow(t *testing.T) {
	t.Parallel()

	entity := validShipment()
	accessorial := &accessorialcharge.AccessorialCharge{
		ID:     pulid.MustNew("acc_"),
		Method: accessorialcharge.MethodPerUnit,
		Amount: decimal.NewFromInt(75),
	}

	ensureGeneratedDetentionCharge(entity, accessorial, 2)
	require.Len(t, entity.AdditionalCharges, 1)
	require.True(t, entity.AdditionalCharges[0].IsDetention)
	require.Equal(t, int16(2), entity.AdditionalCharges[0].Unit)

	entity.AdditionalCharges[0].ID = pulid.MustNew("ac_")
	persistedID := entity.AdditionalCharges[0].ID

	ensureGeneratedDetentionCharge(entity, accessorial, 4)
	require.Len(t, entity.AdditionalCharges, 1)
	require.Equal(t, persistedID, entity.AdditionalCharges[0].ID)
	require.Equal(t, int16(4), entity.AdditionalCharges[0].Unit)

	removeGeneratedDetentionCharges(entity, accessorial.ID)
	require.Empty(t, entity.AdditionalCharges)
}
