package carrierassignmentservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/rateconfirmation"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/shipmentservice"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// readOnlyRateCons answers the one read a preview makes; any write reaches
// the nil repository it embeds and panics.
type readOnlyRateCons struct {
	repositories.RateConfirmationRepository

	active map[pulid.ID]*rateconfirmation.RateConfirmation
}

func (r *readOnlyRateCons) GetActiveByAssignmentID(
	_ context.Context,
	_ pagination.TenantInfo,
	assignmentID pulid.ID,
) (*rateconfirmation.RateConfirmation, error) {
	return r.active[assignmentID], nil
}

// planFixture is a service wired only with readers. The mocks refuse any
// call they were not told to expect, so a preview that created, updated or
// voided anything fails the test that runs it.
type planFixture struct {
	tenantInfo  pagination.TenantInfo
	moveID      pulid.ID
	original    *shipment.Shipment
	assignments *mocks.MockAssignmentRepository
	repo        *mocks.MockCarrierAssignmentRepository
	carriers    *mocks.MockCarrierRepository
	shipments   *mocks.MockShipmentRepository
	holds       *mocks.MockShipmentHoldRepository
	controls    *mocks.MockShipmentControlRepository
	rateCons    *readOnlyRateCons
	svc         *Service
}

func newPlanFixture(t *testing.T, moveStatus shipment.MoveStatus) *planFixture {
	t.Helper()

	f := &planFixture{
		tenantInfo: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
		moveID:      pulid.MustNew("sm_"),
		assignments: mocks.NewMockAssignmentRepository(t),
		repo:        mocks.NewMockCarrierAssignmentRepository(t),
		carriers:    mocks.NewMockCarrierRepository(t),
		shipments:   mocks.NewMockShipmentRepository(t),
		holds:       mocks.NewMockShipmentHoldRepository(t),
		controls:    mocks.NewMockShipmentControlRepository(t),
		rateCons:    &readOnlyRateCons{active: map[pulid.ID]*rateconfirmation.RateConfirmation{}},
	}
	f.original = coverageShipment(pulid.MustNew("shp_"), f.moveID, f.tenantInfo)
	f.original.Moves[0].Status = moveStatus

	f.assignments.EXPECT().
		GetMoveByID(mock.Anything, f.tenantInfo, f.moveID).
		Return(&shipment.ShipmentMove{
			ID:         f.moveID,
			ShipmentID: f.original.ID,
			Status:     moveStatus,
		}, nil).
		Maybe()

	f.svc = &Service{
		l:                 zap.NewNop(),
		db:                dbtest.NopConnection{},
		repo:              f.repo,
		assignmentRepo:    f.assignments,
		carrierRepo:       f.carriers,
		shipmentRepo:      f.shipments,
		holdRepo:          f.holds,
		controlRepo:       f.controls,
		rateConRepo:       f.rateCons,
		shipmentValidator: shipmentservice.NewTestValidator(t),
		coordinator:       shipmentstate.NewCoordinatorWithClock(func() int64 { return 10 }),
	}

	return f
}

func (f *planFixture) expectShipmentRead() {
	f.shipments.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*repositories.GetShipmentByIDRequest")).
		RunAndReturn(func(
			context.Context,
			*repositories.GetShipmentByIDRequest,
		) (*shipment.Shipment, error) {
			return shipment.CloneForUpdate(f.original), nil
		}).
		Once()
	f.controls.EXPECT().
		Get(mock.Anything, repositories.GetShipmentControlRequest{TenantInfo: f.tenantInfo}).
		Return(&tenant.ShipmentControl{}, nil).
		Once()
}

func (f *planFixture) expectCarrier(entity *carrier.Carrier) {
	f.carriers.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("repositories.GetCarrierByIDRequest")).
		Return(entity, nil).
		Once()
}

func (f *planFixture) assignRequest(carrierID pulid.ID) *repositories.AssignMoveToCarrierRequest {
	return &repositories.AssignMoveToCarrierRequest{
		TenantInfo:     f.tenantInfo,
		ShipmentMoveID: f.moveID,
		CarrierID:      carrierID,
		RateMethod:     shipment.CarrierRateMethodFlat,
		BaseRate:       decimal.NewFromInt(1800),
		FuelSurcharge:  decimal.NewFromInt(120),
	}
}

func TestPreviewAssignToMove_ProjectsTheCarrierWithoutSavingIt(t *testing.T) {
	t.Parallel()

	f := newPlanFixture(t, shipment.MoveStatusNew)
	entity := qualifiedCarrier(timeutils.NowUnix())
	entity.ID = pulid.MustNew("car_")
	entity.Name = "Ridgeline Freight"
	f.original.Moves[0].CoverageType = shipment.MoveCoverageTypeUnassigned

	f.holds.EXPECT().HasActiveDispatchHold(mock.Anything, mock.Anything).Return(false, nil).Once()
	f.assignments.EXPECT().GetByMoveID(mock.Anything, f.tenantInfo, f.moveID).Return(nil, nil).Once()
	f.repo.On("GetActiveByMoveID", mock.Anything, f.tenantInfo, f.moveID).Return(nil, nil).Once()
	f.expectCarrier(entity)
	f.expectShipmentRead()

	plan, err := f.svc.PreviewAssignToMove(t.Context(), f.assignRequest(entity.ID))
	require.NoError(t, err)

	require.NotNil(t, plan.Created)
	assert.Equal(t, entity.ID, plan.Created.CarrierID)
	assert.Equal(t, shipment.CarrierAssignmentStatusPending, plan.Created.Status)
	assert.True(t, decimal.NewFromInt(1920).Equal(plan.Created.TotalCost))
	assert.Equal(t, "Ridgeline Freight", plan.Carrier.Name)
	assert.Nil(t, plan.CanceledBefore)
	assert.Equal(t, shipment.MoveCoverageTypeUnassigned, plan.ShipmentBefore.Moves[0].CoverageType)
	assert.Equal(t, shipment.MoveCoverageTypeCarrier, plan.ShipmentAfter.Moves[0].CoverageType)
}

// Replacing a carrier cancels the one on the move and voids its standing rate
// confirmation; the preview shows both, from the same methods the write uses.
func TestPreviewAssignToMove_ShowsTheCarrierItReplaces(t *testing.T) {
	t.Parallel()

	f := newPlanFixture(t, shipment.MoveStatusAssigned)
	entity := qualifiedCarrier(timeutils.NowUnix())
	entity.ID = pulid.MustNew("car_")
	existing := &shipment.CarrierAssignment{
		ID:             pulid.MustNew("casn_"),
		ShipmentMoveID: f.moveID,
		CarrierID:      pulid.MustNew("car_"),
		Status:         shipment.CarrierAssignmentStatusConfirmed,
	}
	active := &rateconfirmation.RateConfirmation{
		ID:                  pulid.MustNew("rc_"),
		CarrierAssignmentID: existing.ID,
		Status:              rateconfirmation.StatusSent,
	}

	f.holds.EXPECT().HasActiveDispatchHold(mock.Anything, mock.Anything).Return(false, nil).Once()
	f.assignments.EXPECT().GetByMoveID(mock.Anything, f.tenantInfo, f.moveID).Return(nil, nil).Once()
	f.repo.On("GetActiveByMoveID", mock.Anything, f.tenantInfo, f.moveID).
		Return(existing, nil).
		Once()
	f.rateCons.active[existing.ID] = active
	f.expectCarrier(entity)
	f.expectShipmentRead()

	req := f.assignRequest(entity.ID)
	req.Replace = true
	plan, err := f.svc.PreviewAssignToMove(t.Context(), req)
	require.NoError(t, err)

	require.NotNil(t, plan.CanceledAfter)
	assert.Equal(t, shipment.CarrierAssignmentStatusConfirmed, plan.CanceledBefore.Status)
	assert.Equal(t, shipment.CarrierAssignmentStatusCanceled, plan.CanceledAfter.Status)
	assert.Equal(t, replacedAssignmentReason, plan.CanceledAfter.CancellationReason)
	require.NotNil(t, plan.VoidedAfter)
	assert.Equal(t, rateconfirmation.StatusSent, plan.VoidedBefore.Status)
	assert.Equal(t, rateconfirmation.StatusVoided, plan.VoidedAfter.Status)
	assert.Equal(t, shipment.CarrierAssignmentStatusConfirmed, existing.Status,
		"the stored assignment is left as it was")
}

func TestPreviewAssignToMove_RefusesAMoveThatAlreadyHasACarrier(t *testing.T) {
	t.Parallel()

	f := newPlanFixture(t, shipment.MoveStatusAssigned)
	f.holds.EXPECT().HasActiveDispatchHold(mock.Anything, mock.Anything).Return(false, nil).Once()
	f.assignments.EXPECT().GetByMoveID(mock.Anything, f.tenantInfo, f.moveID).Return(nil, nil).Once()
	f.repo.On("GetActiveByMoveID", mock.Anything, f.tenantInfo, f.moveID).
		Return(&shipment.CarrierAssignment{ID: pulid.MustNew("casn_")}, nil).
		Once()

	_, err := f.svc.PreviewAssignToMove(t.Context(), f.assignRequest(pulid.MustNew("car_")))
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
}

// A contract-priced assignment writes a rate quote when it is priced, so it
// cannot be previewed without writing; the preview says so rather than
// showing a rate of zero.
func TestPreviewAssignToMove_RefusesToPriceFromTheContract(t *testing.T) {
	t.Parallel()

	f := newPlanFixture(t, shipment.MoveStatusNew)
	req := f.assignRequest(pulid.MustNew("car_"))
	req.BaseRate = decimal.Zero
	req.AutoRate = true

	_, err := f.svc.PreviewAssignToMove(t.Context(), req)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
}

func TestPreviewCancel_ProjectsTheMoveUncoveredAndTheRateConfirmationVoided(t *testing.T) {
	t.Parallel()

	f := newPlanFixture(t, shipment.MoveStatusAssigned)
	f.original.Moves[0].CoverageType = shipment.MoveCoverageTypeCarrier
	existing := &shipment.CarrierAssignment{
		ID:             pulid.MustNew("casn_"),
		ShipmentMoveID: f.moveID,
		CarrierID:      pulid.MustNew("car_"),
		Status:         shipment.CarrierAssignmentStatusPending,
		Carrier:        &carrier.Carrier{Name: "Ridgeline Freight"},
	}
	f.original.Moves[0].CarrierAssignment = existing
	active := &rateconfirmation.RateConfirmation{
		ID:                  pulid.MustNew("rc_"),
		CarrierAssignmentID: existing.ID,
		Status:              rateconfirmation.StatusGenerated,
	}

	f.repo.On("GetActiveByMoveID", mock.Anything, f.tenantInfo, f.moveID).
		Return(existing, nil).
		Once()
	f.rateCons.active[existing.ID] = active
	f.expectShipmentRead()

	plan, err := f.svc.PreviewCancel(t.Context(), &repositories.CancelCarrierAssignmentRequest{
		TenantInfo:     f.tenantInfo,
		ShipmentMoveID: f.moveID,
		Reason:         "Carrier fell off",
	})
	require.NoError(t, err)

	assert.Equal(t, shipment.CarrierAssignmentStatusPending, plan.CanceledBefore.Status)
	assert.Equal(t, shipment.CarrierAssignmentStatusCanceled, plan.CanceledAfter.Status)
	assert.Equal(t, "Carrier fell off", plan.CanceledAfter.CancellationReason)
	assert.Equal(t, rateconfirmation.StatusVoided, plan.VoidedAfter.Status)
	assert.Equal(t, "Ridgeline Freight", plan.Carrier.Name)
	assert.Equal(t, shipment.MoveCoverageTypeUnassigned, plan.ShipmentAfter.Moves[0].CoverageType)
	assert.Equal(t, shipment.CarrierAssignmentStatusPending, existing.Status)
}

// Cancel and its preview refuse a move the carrier has started from one
// check, so a proposal that previews cleanly is one that would run.
func TestPreviewCancel_RefusesWhatCancelRefuses(t *testing.T) {
	t.Parallel()

	f := newPlanFixture(t, shipment.MoveStatusInTransit)
	f.repo.On("GetActiveByMoveID", mock.Anything, f.tenantInfo, f.moveID).
		Return(&shipment.CarrierAssignment{ID: pulid.MustNew("casn_")}, nil).
		Twice()
	req := &repositories.CancelCarrierAssignmentRequest{
		TenantInfo:     f.tenantInfo,
		ShipmentMoveID: f.moveID,
		Reason:         "Carrier fell off",
	}

	_, previewErr := f.svc.PreviewCancel(t.Context(), req)
	writeErr := f.svc.Cancel(t.Context(), req)

	require.Error(t, previewErr)
	assert.True(t, errortypes.IsBusinessError(previewErr))
	require.Error(t, writeErr)
	assert.Equal(t, writeErr.Error(), previewErr.Error())
}
