package detentionflows

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/app/detentionreview"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type settleRig struct {
	svc       services
	shipments *previewShipments
	orders    *previewOrders
	comments  *previewComments
	stored    detention.DetentionOccurrence
	sibling   *detention.DetentionOccurrence
	charge    *shipment.AdditionalCharge
}

func newSettleRig(t *testing.T) *settleRig {
	t.Helper()

	accessorialID := pulid.MustNew("acc_")
	delivery := pendingCharge()
	delivery.BillableAmount = decimal.NewFromInt(1650)
	delivery.PolicySnapshot.AccessorialChargeID = accessorialID

	pickup := pendingCharge()
	pickup.ShipmentID = delivery.ShipmentID
	pickup.OrganizationID = delivery.OrganizationID
	pickup.BusinessUnitID = delivery.BusinessUnitID
	pickup.BillableAmount = decimal.NewFromInt(1425)
	snapshot := *delivery.PolicySnapshot
	pickup.PolicySnapshot = &snapshot

	charge := &shipment.AdditionalCharge{
		ID:                  pulid.MustNew("ac_"),
		AccessorialChargeID: accessorialID,
		IsSystemGenerated:   true,
		IsDetention:         true,
		Method:              accessorialcharge.MethodFlat,
		Amount:              decimal.NewFromInt(3075),
		Unit:                1,
	}
	shipments := &previewShipments{stored: &shipment.Shipment{
		ID:                  delivery.ShipmentID,
		OrganizationID:      delivery.OrganizationID,
		BusinessUnitID:      delivery.BusinessUnitID,
		FreightChargeAmount: decimal.NewNullDecimal(decimal.RequireFromString("3468.14")),
		AdditionalCharges:   []*shipment.AdditionalCharge{charge},
	}}

	repo := &previewOccurrenceRepo{stored: delivery, sibling: &pickup}
	w := previewWiring(repo)
	w.Shipments = shipments
	orders := &previewOrders{}
	w.Orders = orders
	comments := &previewComments{}
	w.CommentRepo = comments

	return &settleRig{
		svc:       wire(w),
		shipments: shipments,
		orders:    orders,
		comments:  comments,
		stored:    delivery,
		sibling:   &pickup,
		charge:    charge,
	}
}

func TestWaive_TakesTheWaivedStopOffTheShipmentsDetentionCharge(t *testing.T) {
	t.Parallel()

	rig := newSettleRig(t)
	userID := pulid.MustNew("usr_")

	_, err := rig.svc.review.Waive(t.Context(), &detentionreview.WaiveParams{
		OccurrenceID: rig.stored.ID,
		TenantInfo:   tenantOfOccurrence(&rig.stored),
		Reason:       detention.WaiverReasonDataCorrection,
		Note:         "No signed POD on file",
		UserID:       userID,
	})
	require.NoError(t, err)

	require.Len(t, rig.shipments.saved, 1, "a waiver rewrites the shipment's detention charge")
	saved := rig.shipments.saved[0]
	require.Len(t, saved.AdditionalCharges, 1)
	assert.Equal(t, rig.charge.ID, saved.AdditionalCharges[0].ID)
	assert.Equal(t, "1425", saved.AdditionalCharges[0].Amount.String())
	assert.Equal(t, []pulid.ID{rig.sibling.ID}, saved.AdditionalCharges[0].DetentionOccurrenceIDs)
	assert.Equal(t, "1425", saved.OtherChargeAmount.Decimal.String())
	assert.Equal(t, "4893.14", saved.TotalChargeAmount.Decimal.String())

	assert.Equal(t, []pulid.ID{rig.stored.ShipmentID}, rig.orders.recomputed)
	require.Len(t, rig.comments.created, 1)
	assert.Equal(t,
		"Detention charge lowered from 3075.00 USD to 1425.00 USD\n\n"+
			"Waived as DataCorrection: No signed POD on file",
		rig.comments.created[0].Comment)
	assert.Equal(t, userID, rig.comments.created[0].UserID)
}

func TestWaive_LastBilledStopRemovesTheDetentionCharge(t *testing.T) {
	t.Parallel()

	rig := newSettleRig(t)
	rig.sibling.Status = detention.OccurrenceStatusWaived
	rig.shipments.stored.AdditionalCharges[0].Amount = decimal.NewFromInt(1650)

	_, err := rig.svc.review.Waive(t.Context(), &detentionreview.WaiveParams{
		OccurrenceID: rig.stored.ID,
		TenantInfo:   tenantOfOccurrence(&rig.stored),
		Reason:       detention.WaiverReasonDataCorrection,
		Note:         "No signed POD on file",
		UserID:       pulid.MustNew("usr_"),
	})
	require.NoError(t, err)

	require.Len(t, rig.shipments.saved, 1)
	saved := rig.shipments.saved[0]
	assert.Empty(t, saved.AdditionalCharges)
	assert.Equal(t, "3468.14", saved.TotalChargeAmount.Decimal.String())
	require.Len(t, rig.comments.created, 1)
	assert.Equal(t,
		"Detention charge removed — 1650.00 USD\n\nWaived as DataCorrection: No signed POD on file",
		rig.comments.created[0].Comment)
}

func TestDispute_HoldsTheDisputedStopOffTheShipmentsCharge(t *testing.T) {
	t.Parallel()

	rig := newSettleRig(t)

	_, err := rig.svc.review.Dispute(t.Context(), &detentionreview.DisputeParams{
		OccurrenceID: rig.stored.ID,
		TenantInfo:   tenantOfOccurrence(&rig.stored),
		Note:         "Customer says the gate log shows 4 hours",
		UserID:       pulid.MustNew("usr_"),
	})
	require.NoError(t, err)

	require.Len(t, rig.shipments.saved, 1)
	assert.Equal(t, "1425", rig.shipments.saved[0].AdditionalCharges[0].Amount.String())
}

func TestWaive_ShipmentWithoutADetentionChargeIsNotSaved(t *testing.T) {
	t.Parallel()

	rig := newSettleRig(t)
	rig.shipments.stored.AdditionalCharges = nil

	_, err := rig.svc.review.Waive(t.Context(), &detentionreview.WaiveParams{
		OccurrenceID: rig.stored.ID,
		TenantInfo:   tenantOfOccurrence(&rig.stored),
		Reason:       detention.WaiverReasonDataCorrection,
		Note:         "No signed POD on file",
		UserID:       pulid.MustNew("usr_"),
	})
	require.NoError(t, err)

	assert.Empty(t, rig.shipments.saved)
	assert.Empty(t, rig.orders.recomputed)
	assert.Empty(t, rig.comments.created)
}
