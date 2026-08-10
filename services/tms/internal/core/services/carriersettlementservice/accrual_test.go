package carriersettlementservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
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

func TestAccrueCarrierMoveWithoutActiveAssignmentVoidsPending(t *testing.T) {
	deps := setupAccrualTest(t)
	tenantInfo := accrualTenantInfo()
	sp := newAccrualShipment(tenantInfo)
	move := newCarrierMove(sp)

	orphaned := &carriersettlement.CostEvent{
		ID:             pulid.MustNew("cacc_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		EventType:      carriersettlement.CostEventTypeLinehaulCost,
		Status:         carriersettlement.CostEventStatusPending,
		IdempotencyKey: "carrier-cost:casn_old:LinehaulCost:0",
	}
	settled := &carriersettlement.CostEvent{
		ID:             pulid.MustNew("cacc_"),
		EventType:      carriersettlement.CostEventTypeLinehaulCost,
		Status:         carriersettlement.CostEventStatusSettled,
		IdempotencyKey: "carrier-cost:casn_old:FuelSurcharge:0",
	}

	deps.assignments.On("GetActiveByMoveID", mock.Anything, tenantInfo, move.ID).
		Return(nil, nil)
	deps.costEvents.On("ListByMove", mock.Anything, tenantInfo, move.ID).
		Return([]*carriersettlement.CostEvent{orphaned, settled}, nil)
	deps.costEvents.On("Update", mock.Anything, orphaned).Return(orphaned, nil).Once()

	err := deps.svc.accrueCarrierMove(t.Context(), tenantInfo, sp, move, 1_700_000_000)
	require.NoError(t, err)

	deps.costEvents.AssertNotCalled(t, "Create")
	assert.Equal(t, carriersettlement.CostEventStatusVoided, orphaned.Status,
		"a canceled assignment must not leave payable events behind")
	assert.Equal(t, carriersettlement.CostEventStatusSettled, settled.Status)
}

func TestUpsertCostEventSkipsCurrentPendingAndRepricesVersionBump(t *testing.T) {
	deps := setupAccrualTest(t)
	tenantInfo := accrualTenantInfo()
	sp := newAccrualShipment(tenantInfo)
	move := newCarrierMove(sp)
	assignment := newFlatAssignment()
	assignment.Version = 3
	inputs := BuildAssignmentCostEventInputs(assignment)

	current := &carriersettlement.CostEvent{
		ID:                pulid.MustNew("cacc_"),
		OrganizationID:    tenantInfo.OrgID,
		BusinessUnitID:    tenantInfo.BuID,
		Status:            carriersettlement.CostEventStatusPending,
		IdempotencyKey:    inputs[0].IdempotencyKey,
		AmountMinor:       inputs[0].AmountMinor,
		Description:       inputs[0].Description,
		ProNumber:         sp.ProNumber,
		AssignmentVersion: 3,
	}
	deps.costEvents.On(
		"GetByIdempotencyKey", mock.Anything, tenantInfo, inputs[0].IdempotencyKey,
	).Return(current, nil)

	err := deps.svc.upsertCostEvent(t.Context(), &upsertCostEventParams{
		TenantInfo: tenantInfo,
		Shipment:   sp,
		Move:       move,
		Assignment: assignment,
		Input:      inputs[0],
		EventDate:  1_700_000_000,
	})
	require.NoError(t, err)
	deps.costEvents.AssertNotCalled(t, "Update")

	stale := &carriersettlement.CostEvent{
		ID:                pulid.MustNew("cacc_"),
		OrganizationID:    tenantInfo.OrgID,
		BusinessUnitID:    tenantInfo.BuID,
		Status:            carriersettlement.CostEventStatusPending,
		IdempotencyKey:    inputs[0].IdempotencyKey,
		AmountMinor:       inputs[0].AmountMinor,
		Description:       inputs[0].Description,
		ProNumber:         sp.ProNumber,
		AssignmentVersion: 2,
	}
	deps.costEvents.ExpectedCalls = nil
	deps.costEvents.On(
		"GetByIdempotencyKey", mock.Anything, tenantInfo, inputs[0].IdempotencyKey,
	).Return(stale, nil)
	deps.costEvents.On("Update", mock.Anything, stale).Return(stale, nil).Once()

	err = deps.svc.upsertCostEvent(t.Context(), &upsertCostEventParams{
		TenantInfo: tenantInfo,
		Shipment:   sp,
		Move:       move,
		Assignment: assignment,
		Input:      inputs[0],
		EventDate:  1_700_000_000,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(3), stale.AssignmentVersion,
		"a version-bumped assignment reprices its pending events")
}

type reaccrueDeps struct {
	costEvents  *mocks.MockCarrierCostEventRepository
	assignments *mocks.MockCarrierAssignmentRepository
	shipments   *mocks.MockShipmentRepository
	control     *mocks.MockCarrierSettlementControlRepository
	svc         *Service
}

func setupReaccrueTest(t *testing.T) *reaccrueDeps {
	t.Helper()
	costEvents := mocks.NewMockCarrierCostEventRepository(t)
	assignments := mocks.NewMockCarrierAssignmentRepository(t)
	shipments := mocks.NewMockShipmentRepository(t)
	control := mocks.NewMockCarrierSettlementControlRepository(t)
	svc := &Service{
		l:                 zap.NewNop(),
		costEventRepo:     costEvents,
		assignmentRepo:    assignments,
		shipmentRepo:      shipments,
		settlementControl: control,
	}
	return &reaccrueDeps{
		costEvents:  costEvents,
		assignments: assignments,
		shipments:   shipments,
		control:     control,
		svc:         svc,
	}
}

func TestReaccrueMoveCancelAfterDeliveryVoidsEvents(t *testing.T) {
	deps := setupReaccrueTest(t)
	tenantInfo := accrualTenantInfo()
	moveID := pulid.MustNew("sm_")

	orphaned := &carriersettlement.CostEvent{
		ID:             pulid.MustNew("cacc_"),
		Status:         carriersettlement.CostEventStatusPending,
		IdempotencyKey: "carrier-cost:casn_old:LinehaulCost:0",
	}

	deps.assignments.On("GetActiveByMoveID", mock.Anything, tenantInfo, moveID).
		Return(nil, nil)
	deps.costEvents.On("ListByMove", mock.Anything, tenantInfo, moveID).
		Return([]*carriersettlement.CostEvent{orphaned}, nil)
	deps.costEvents.On("Update", mock.Anything, orphaned).Return(orphaned, nil).Once()

	err := deps.svc.ReaccrueMove(t.Context(), tenantInfo, moveID)
	require.NoError(t, err)

	assert.Equal(t, carriersettlement.CostEventStatusVoided, orphaned.Status)
	assert.Equal(t, "Assignment cost line removed", orphaned.VoidReason)
}

func TestReaccrueMoveReplaceAfterDeliveryAccruesNewCarrier(t *testing.T) {
	deps := setupReaccrueTest(t)
	tenantInfo := accrualTenantInfo()
	sp := newAccrualShipment(tenantInfo)
	sp.Status = shipment.StatusCompleted
	move := newCarrierMove(sp)
	sp.Moves = []*shipment.ShipmentMove{move}

	replacement := newFlatAssignment()
	replacement.Accessorials = nil
	replacement.SyncTotals(nil)
	replacement.ShipmentMove = &shipment.ShipmentMove{ID: move.ID, ShipmentID: sp.ID}
	inputs := BuildAssignmentCostEventInputs(replacement)

	oldEvent := &carriersettlement.CostEvent{
		ID:             pulid.MustNew("cacc_"),
		Status:         carriersettlement.CostEventStatusPending,
		IdempotencyKey: "carrier-cost:casn_old:LinehaulCost:0",
	}

	deps.assignments.On("GetActiveByMoveID", mock.Anything, tenantInfo, move.ID).
		Return(replacement, nil)
	deps.shipments.On("GetByID", mock.Anything, mock.Anything).Return(sp, nil)
	deps.control.On("GetOrCreate", mock.Anything, tenantInfo).
		Return(&tenant.CarrierSettlementControl{
			PayTrigger: tenant.PayTriggerShipmentDelivered,
		}, nil)
	deps.costEvents.On("GetByIdempotencyKey", mock.Anything, tenantInfo, mock.Anything).
		Return(nil, errors.New("not found")).Times(len(inputs))

	created := make([]*carriersettlement.CostEvent, 0, len(inputs))
	deps.costEvents.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			created = append(created, args.Get(1).(*carriersettlement.CostEvent))
		}).
		Return(&carriersettlement.CostEvent{}, nil).Times(len(inputs))
	deps.costEvents.On("ListByMove", mock.Anything, tenantInfo, move.ID).
		Return([]*carriersettlement.CostEvent{oldEvent}, nil)
	deps.costEvents.On("Update", mock.Anything, oldEvent).Return(oldEvent, nil).Once()

	err := deps.svc.ReaccrueMove(t.Context(), tenantInfo, move.ID)
	require.NoError(t, err)

	require.Len(t, created, len(inputs))
	assert.Equal(t, replacement.CarrierID, created[0].CarrierID,
		"the replacement carrier accrues immediately after the trigger crossed")
	assert.Equal(t, carriersettlement.CostEventStatusVoided, oldEvent.Status,
		"the replaced assignment's events are voided")
}

func TestReaccrueMoveBeforeTriggerOnlyVoidsStale(t *testing.T) {
	deps := setupReaccrueTest(t)
	tenantInfo := accrualTenantInfo()
	sp := newAccrualShipment(tenantInfo)
	sp.Status = shipment.StatusInTransit
	move := newCarrierMove(sp)
	move.Status = shipment.MoveStatusInTransit
	sp.Moves = []*shipment.ShipmentMove{move}

	replacement := newFlatAssignment()
	replacement.ShipmentMove = &shipment.ShipmentMove{ID: move.ID, ShipmentID: sp.ID}

	deps.assignments.On("GetActiveByMoveID", mock.Anything, tenantInfo, move.ID).
		Return(replacement, nil)
	deps.shipments.On("GetByID", mock.Anything, mock.Anything).Return(sp, nil)
	deps.control.On("GetOrCreate", mock.Anything, tenantInfo).
		Return(&tenant.CarrierSettlementControl{
			PayTrigger: tenant.PayTriggerShipmentDelivered,
		}, nil)
	deps.costEvents.On("ListByMove", mock.Anything, tenantInfo, move.ID).
		Return([]*carriersettlement.CostEvent{}, nil)

	err := deps.svc.ReaccrueMove(t.Context(), tenantInfo, move.ID)
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
