package detentioncharge_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentioncharge"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func detentionCharge(accessorialID pulid.ID, amount string) *shipment.AdditionalCharge {
	return &shipment.AdditionalCharge{
		ID:                  pulid.MustNew("ac_"),
		AccessorialChargeID: accessorialID,
		IsSystemGenerated:   true,
		IsDetention:         true,
		Method:              accessorialcharge.MethodFlat,
		Amount:              decimal.RequireFromString(amount),
		Unit:                1,
	}
}

func waived(occurrence *detention.DetentionOccurrence) *detention.DetentionOccurrence {
	occurrence.WaivedAmount = occurrence.BillableAmount
	occurrence.Status = detention.OccurrenceStatusWaived
	return occurrence
}

func TestSettle_WaivingOneStopLowersTheChargeToTheStopsStillBilled(t *testing.T) {
	t.Parallel()

	accessorialID := pulid.MustNew("acc_")
	pickup := chargeableOccurrenceFor(accessorialID, "1425")
	delivery := waived(chargeableOccurrenceFor(accessorialID, "1650"))
	charge := detentionCharge(accessorialID, "3075")
	entity := newShipment()
	entity.AdditionalCharges = []*shipment.AdditionalCharge{charge}

	changes := detentioncharge.Settle(entity, []*detention.DetentionOccurrence{pickup, delivery})

	require.Equal(t, []detentioncharge.Change{{
		AccessorialChargeID: accessorialID,
		Before:              decimal.RequireFromString("3075"),
		After:               decimal.RequireFromString("1425"),
	}}, changes)
	require.Len(t, entity.AdditionalCharges, 1)
	require.Same(t, charge, entity.AdditionalCharges[0])
	require.Equal(t, "1425", entity.AdditionalCharges[0].Amount.String())
	require.Equal(t, []pulid.ID{pickup.ID}, entity.AdditionalCharges[0].DetentionOccurrenceIDs)
}

func TestSettle_WaivingEveryStopRemovesTheCharge(t *testing.T) {
	t.Parallel()

	accessorialID := pulid.MustNew("acc_")
	manual := &shipment.AdditionalCharge{
		ID:                  pulid.MustNew("ac_"),
		AccessorialChargeID: pulid.MustNew("acc_"),
		Method:              accessorialcharge.MethodFlat,
		Amount:              decimal.NewFromInt(40),
	}
	entity := newShipment()
	entity.AdditionalCharges = []*shipment.AdditionalCharge{
		manual,
		detentionCharge(accessorialID, "1650"),
	}

	changes := detentioncharge.Settle(entity, []*detention.DetentionOccurrence{
		waived(chargeableOccurrenceFor(accessorialID, "1425")),
		waived(chargeableOccurrenceFor(accessorialID, "1650")),
	})

	require.Len(t, changes, 1)
	require.True(t, changes[0].Removed())
	require.Equal(t, "1650", changes[0].Before.String())
	require.Equal(t, []*shipment.AdditionalCharge{manual}, entity.AdditionalCharges)
}

func TestSettle_DisputedStopsLeaveTheCharge(t *testing.T) {
	t.Parallel()

	accessorialID := pulid.MustNew("acc_")
	disputed := chargeableOccurrenceFor(accessorialID, "900")
	disputed.Status = detention.OccurrenceStatusDisputed
	entity := newShipment()
	entity.AdditionalCharges = []*shipment.AdditionalCharge{detentionCharge(accessorialID, "900")}

	changes := detentioncharge.Settle(entity, []*detention.DetentionOccurrence{disputed})

	require.Len(t, changes, 1)
	require.Empty(t, entity.AdditionalCharges)
}

func TestSettle_ApprovalThatMovesNoMoneyChangesNothing(t *testing.T) {
	t.Parallel()

	accessorialID := pulid.MustNew("acc_")
	approved := chargeableOccurrenceFor(accessorialID, "1425")
	approved.Status = detention.OccurrenceStatusApproved
	charge := detentionCharge(accessorialID, "1425")
	charge.DetentionOccurrenceIDs = []pulid.ID{approved.ID}
	entity := newShipment()
	entity.AdditionalCharges = []*shipment.AdditionalCharge{charge}

	changes := detentioncharge.Settle(entity, []*detention.DetentionOccurrence{approved})

	require.Empty(t, changes)
	require.Equal(t, []*shipment.AdditionalCharge{charge}, entity.AdditionalCharges)
}

func TestSettle_NeverAddsAChargeTheShipmentDoesNotCarry(t *testing.T) {
	t.Parallel()

	entity := newShipment()

	changes := detentioncharge.Settle(entity, []*detention.DetentionOccurrence{
		chargeableOccurrence("500"),
	})

	require.Empty(t, changes)
	require.Empty(t, entity.AdditionalCharges)
}

func TestSettle_LeavesDetentionChargesNoOccurrenceGoverns(t *testing.T) {
	t.Parallel()

	legacy := detentionCharge(pulid.MustNew("acc_"), "300")
	legacy.Method = accessorialcharge.MethodPerUnit
	legacy.Amount = decimal.NewFromInt(75)
	legacy.Unit = 4
	entity := newShipment()
	entity.AdditionalCharges = []*shipment.AdditionalCharge{legacy}

	changes := detentioncharge.Settle(entity, []*detention.DetentionOccurrence{
		waived(chargeableOccurrence("500")),
	})

	require.Empty(t, changes)
	require.Equal(t, []*shipment.AdditionalCharge{legacy}, entity.AdditionalCharges)
}

func TestSettle_FoldsADuplicateRowIntoTheFirst(t *testing.T) {
	t.Parallel()

	accessorialID := pulid.MustNew("acc_")
	first := detentionCharge(accessorialID, "1425")
	duplicate := detentionCharge(accessorialID, "1650")
	occurrence := chargeableOccurrenceFor(accessorialID, "1425")
	entity := newShipment()
	entity.AdditionalCharges = []*shipment.AdditionalCharge{first, duplicate}

	changes := detentioncharge.Settle(entity, []*detention.DetentionOccurrence{occurrence})

	require.Equal(t, []*shipment.AdditionalCharge{first}, entity.AdditionalCharges)
	require.Len(t, changes, 1)
	require.True(t, changes[0].Removed())
}
