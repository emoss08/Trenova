package carriersettlementservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type accrualDeps struct {
	costEvents  *mocks.MockCarrierCostEventRepository
	assignments *mocks.MockCarrierAssignmentRepository
	svc         *Service
}

func setupAccrualTest(t *testing.T) *accrualDeps {
	t.Helper()
	costEvents := mocks.NewMockCarrierCostEventRepository(t)
	assignments := mocks.NewMockCarrierAssignmentRepository(t)
	svc := &Service{
		l:              zap.NewNop(),
		costEventRepo:  costEvents,
		assignmentRepo: assignments,
	}
	return &accrualDeps{costEvents: costEvents, assignments: assignments, svc: svc}
}

func accrualTenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
}

func newCarrierMove(sp *shipment.Shipment) *shipment.ShipmentMove {
	return &shipment.ShipmentMove{
		ID:             pulid.MustNew("sm_"),
		OrganizationID: sp.OrganizationID,
		BusinessUnitID: sp.BusinessUnitID,
		ShipmentID:     sp.ID,
		Status:         shipment.MoveStatusCompleted,
		CoverageType:   shipment.MoveCoverageTypeCarrier,
	}
}

func newAccrualShipment(tenantInfo pagination.TenantInfo) *shipment.Shipment {
	return &shipment.Shipment{
		ID:             pulid.MustNew("shp_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ProNumber:      "PRO-100",
	}
}

func TestAccrueCarrierMoveMaterializesCostEvents(t *testing.T) {
	deps := setupAccrualTest(t)
	tenantInfo := accrualTenantInfo()
	sp := newAccrualShipment(tenantInfo)
	move := newCarrierMove(sp)
	assignment := newFlatAssignment()

	deps.assignments.On("GetActiveByMoveID", mock.Anything, tenantInfo, move.ID).
		Return(assignment, nil)
	deps.costEvents.On("GetByIdempotencyKey", mock.Anything, tenantInfo, mock.Anything).
		Return(nil, errors.New("not found")).Times(4)

	created := make([]*carriersettlement.CostEvent, 0, 4)
	deps.costEvents.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			created = append(created, args.Get(1).(*carriersettlement.CostEvent))
		}).
		Return(&carriersettlement.CostEvent{}, nil).Times(4)
	deps.costEvents.On("ListByMove", mock.Anything, tenantInfo, move.ID).
		Return([]*carriersettlement.CostEvent{}, nil)

	err := deps.svc.accrueCarrierMove(t.Context(), tenantInfo, sp, move, 1_700_000_000)
	require.NoError(t, err)

	require.Len(t, created, 4)
	assert.Equal(t, carriersettlement.CostEventTypeLinehaulCost, created[0].EventType)
	assert.Equal(t, int64(150_000), created[0].AmountMinor)
	assert.Equal(t, assignment.CarrierID, created[0].CarrierID)
	assert.Equal(t, carriersettlement.CostEventStatusPending, created[0].Status)
	assert.Equal(t, carriersettlement.CostEventTypeFuelSurcharge, created[1].EventType)
	assert.Equal(t, carriersettlement.CostEventTypeAccessorial, created[2].EventType)
	assert.Equal(t, carriersettlement.CostEventTypeAccessorial, created[3].EventType)
	for _, event := range created {
		assert.Equal(t, sp.ID, *event.ShipmentID)
		assert.Equal(t, move.ID, *event.MoveID)
	}
}

func TestAccrueCarrierMoveIdempotentRefire(t *testing.T) {
	deps := setupAccrualTest(t)
	tenantInfo := accrualTenantInfo()
	sp := newAccrualShipment(tenantInfo)
	move := newCarrierMove(sp)
	assignment := newFlatAssignment()
	assignment.Accessorials = nil
	assignment.SyncTotals(nil)

	inputs := BuildAssignmentCostEventInputs(assignment)
	require.Len(t, inputs, 2)

	pendingEvent := &carriersettlement.CostEvent{
		ID:             pulid.MustNew("cacc_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		CarrierID:      assignment.CarrierID,
		Status:         carriersettlement.CostEventStatusPending,
		IdempotencyKey: inputs[0].IdempotencyKey,
		AmountMinor:    1,
	}
	settledEvent := &carriersettlement.CostEvent{
		ID:             pulid.MustNew("cacc_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		CarrierID:      assignment.CarrierID,
		Status:         carriersettlement.CostEventStatusSettled,
		IdempotencyKey: inputs[1].IdempotencyKey,
		AmountMinor:    2,
	}

	deps.assignments.On("GetActiveByMoveID", mock.Anything, tenantInfo, move.ID).
		Return(assignment, nil)
	deps.costEvents.On(
		"GetByIdempotencyKey", mock.Anything, tenantInfo, inputs[0].IdempotencyKey,
	).Return(pendingEvent, nil)
	deps.costEvents.On(
		"GetByIdempotencyKey", mock.Anything, tenantInfo, inputs[1].IdempotencyKey,
	).Return(settledEvent, nil)
	deps.costEvents.On("Update", mock.Anything, pendingEvent).
		Return(pendingEvent, nil).Once()
	deps.costEvents.On("ListByMove", mock.Anything, tenantInfo, move.ID).
		Return([]*carriersettlement.CostEvent{pendingEvent, settledEvent}, nil)

	err := deps.svc.accrueCarrierMove(t.Context(), tenantInfo, sp, move, 1_700_000_000)
	require.NoError(t, err)

	assert.Equal(t, int64(150_000), pendingEvent.AmountMinor,
		"pending events recompute in place")
	assert.Equal(t, carriersettlement.CostEventStatusPending, pendingEvent.Status)
	assert.Equal(t, int64(2), settledEvent.AmountMinor, "settled events are never touched")
	assert.Equal(t, carriersettlement.CostEventStatusSettled, settledEvent.Status)
	deps.costEvents.AssertNumberOfCalls(t, "Create", 0)
}

func TestAccrueCarrierMoveRevivesVoidedAndVoidsStale(t *testing.T) {
	deps := setupAccrualTest(t)
	tenantInfo := accrualTenantInfo()
	sp := newAccrualShipment(tenantInfo)
	move := newCarrierMove(sp)
	assignment := newFlatAssignment()
	assignment.Accessorials = nil
	assignment.SyncTotals(nil)

	inputs := BuildAssignmentCostEventInputs(assignment)
	voidedAt := int64(1_600_000_000)
	voidedEvent := &carriersettlement.CostEvent{
		ID:             pulid.MustNew("cacc_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		CarrierID:      assignment.CarrierID,
		Status:         carriersettlement.CostEventStatusVoided,
		IdempotencyKey: inputs[0].IdempotencyKey,
		VoidedAt:       &voidedAt,
		VoidReason:     "Shipment canceled",
	}
	staleAccessorial := &carriersettlement.CostEvent{
		ID:             pulid.MustNew("cacc_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		CarrierID:      assignment.CarrierID,
		EventType:      carriersettlement.CostEventTypeAccessorial,
		Status:         carriersettlement.CostEventStatusPending,
		IdempotencyKey: "carrier-cost:" + assignment.ID.String() + ":Accessorial:0",
	}

	deps.assignments.On("GetActiveByMoveID", mock.Anything, tenantInfo, move.ID).
		Return(assignment, nil)
	deps.costEvents.On(
		"GetByIdempotencyKey", mock.Anything, tenantInfo, inputs[0].IdempotencyKey,
	).Return(voidedEvent, nil)
	deps.costEvents.On(
		"GetByIdempotencyKey", mock.Anything, tenantInfo, inputs[1].IdempotencyKey,
	).Return(nil, errors.New("not found"))
	deps.costEvents.On("Create", mock.Anything, mock.Anything).
		Return(&carriersettlement.CostEvent{}, nil).Once()
	deps.costEvents.On("Update", mock.Anything, mock.Anything).
		Return(&carriersettlement.CostEvent{}, nil)
	deps.costEvents.On("ListByMove", mock.Anything, tenantInfo, move.ID).
		Return([]*carriersettlement.CostEvent{voidedEvent, staleAccessorial}, nil)

	err := deps.svc.accrueCarrierMove(t.Context(), tenantInfo, sp, move, 1_700_000_000)
	require.NoError(t, err)

	assert.Equal(t, carriersettlement.CostEventStatusPending, voidedEvent.Status,
		"voided events revive on re-fire")
	assert.Nil(t, voidedEvent.VoidedAt)
	assert.Empty(t, voidedEvent.VoidReason)
	assert.Equal(t, carriersettlement.CostEventStatusVoided, staleAccessorial.Status,
		"pending events for removed cost lines are voided")
	assert.Equal(t, "Assignment cost line removed", staleAccessorial.VoidReason)
}

func TestAccrueCarrierMoveSkipsWithoutActiveAssignment(t *testing.T) {
	deps := setupAccrualTest(t)
	tenantInfo := accrualTenantInfo()
	sp := newAccrualShipment(tenantInfo)
	move := newCarrierMove(sp)

	deps.assignments.On("GetActiveByMoveID", mock.Anything, tenantInfo, move.ID).
		Return(nil, nil)

	err := deps.svc.accrueCarrierMove(t.Context(), tenantInfo, sp, move, 1_700_000_000)
	require.NoError(t, err)
	deps.costEvents.AssertNotCalled(t, "Create")
}

func TestVoidShipmentCostEventsVoidsPendingOnly(t *testing.T) {
	deps := setupAccrualTest(t)
	tenantInfo := accrualTenantInfo()
	shipmentID := pulid.MustNew("shp_")

	pending := &carriersettlement.CostEvent{
		ID:     pulid.MustNew("cacc_"),
		Status: carriersettlement.CostEventStatusPending,
	}
	settled := &carriersettlement.CostEvent{
		ID:     pulid.MustNew("cacc_"),
		Status: carriersettlement.CostEventStatusSettled,
	}
	attached := &carriersettlement.CostEvent{
		ID:     pulid.MustNew("cacc_"),
		Status: carriersettlement.CostEventStatusAttached,
	}

	deps.costEvents.On("ListByShipment", mock.Anything, tenantInfo, shipmentID).
		Return([]*carriersettlement.CostEvent{pending, settled, attached}, nil)
	deps.costEvents.On("Update", mock.Anything, pending).Return(pending, nil).Once()

	err := deps.svc.VoidShipmentCostEvents(
		t.Context(),
		tenantInfo,
		shipmentID,
		"Shipment canceled",
	)
	require.NoError(t, err)

	assert.Equal(t, carriersettlement.CostEventStatusVoided, pending.Status)
	assert.Equal(t, "Shipment canceled", pending.VoidReason)
	require.NotNil(t, pending.VoidedAt)
	assert.Equal(t, carriersettlement.CostEventStatusSettled, settled.Status)
	assert.Equal(t, carriersettlement.CostEventStatusAttached, attached.Status)
}
