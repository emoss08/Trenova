package assignmentservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/rateengine"
	"github.com/emoss08/trenova/internal/core/services/shipmentcommercial"
	"github.com/emoss08/trenova/internal/core/services/shipmentservice"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// previewUnassignService reads what unassigning reads and holds no writer:
// the mocks fail the test on an Unassign or an Update, which is how a preview
// that wrote would show itself.
func previewUnassignService(
	t *testing.T,
	tenantInfo pagination.TenantInfo,
	original *shipment.Shipment,
) *service {
	t.Helper()

	moveID := original.Moves[0].ID
	repo := mocks.NewMockAssignmentRepository(t)
	repo.EXPECT().
		GetMoveByID(mock.Anything, tenantInfo, moveID).
		Return(&shipment.ShipmentMove{
			ID:             moveID,
			ShipmentID:     original.ID,
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			Status:         shipment.MoveStatusAssigned,
			CoverageType:   shipment.MoveCoverageTypeDriver,
		}, nil).
		Once()
	repo.EXPECT().
		GetByMoveID(mock.Anything, tenantInfo, moveID).
		Return(original.Moves[0].Assignment, nil).
		Once()

	shipmentRepo := mocks.NewMockShipmentRepository(t)
	shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*repositories.GetShipmentByIDRequest")).
		RunAndReturn(func(
			context.Context,
			*repositories.GetShipmentByIDRequest,
		) (*shipment.Shipment, error) {
			return shipment.CloneForUpdate(original), nil
		}).
		Once()
	controlRepo := mocks.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, repositories.GetShipmentControlRequest{TenantInfo: tenantInfo}).
		Return(&tenant.ShipmentControl{AutoDelayShipmentsThreshold: new(int16(30))}, nil).
		Once()
	formula := mocks.NewMockFormulaCalculator(t)
	formula.EXPECT().
		Calculate(mock.Anything, mock.AnythingOfType("*formulatemplatetypes.CalculateRequest")).
		Return(&formulatemplatetypes.CalculateResponse{Amount: decimal.Zero}, nil).
		Once()

	return &service{
		l:            zap.NewNop(),
		db:           dbtest.NopConnection{},
		repo:         repo,
		shipmentRepo: shipmentRepo,
		holdRepo:     mocks.NewMockShipmentHoldRepository(t),
		controlRepo:  controlRepo,
		commercial: shipmentcommercial.New(shipmentcommercial.Params{
			Logger:          zap.NewNop(),
			RateEngine:      rateengine.NewFallbackEngine(t, formula),
			AccessorialRepo: mocks.NewMockAccessorialChargeRepository(t),
		}),
		shipmentValidator: shipmentservice.NewTestValidator(t),
		eventService:      noopShipmentEventService{},
		coordinator:       shipmentstate.NewCoordinatorWithClock(func() int64 { return 10 }),
	}
}

func assignedShipment(
	shipmentID, moveID pulid.ID,
	tenantInfo pagination.TenantInfo,
) *shipment.Shipment {
	original := validShipment(shipmentID, moveID, tenantInfo)
	original.Status = shipment.StatusAssigned
	original.Moves[0].Status = shipment.MoveStatusAssigned
	original.Moves[0].CoverageType = shipment.MoveCoverageTypeDriver
	original.Moves[0].Assignment = &shipment.Assignment{
		ID:              pulid.MustNew("asn_"),
		OrganizationID:  tenantInfo.OrgID,
		BusinessUnitID:  tenantInfo.BuID,
		ShipmentMoveID:  moveID,
		PrimaryWorkerID: pulid.Must("wrk_"),
		TractorID:       pulid.Must("trc_"),
		Status:          shipment.AssignmentStatusNew,
	}

	return original
}

func TestPreviewUnassign_ProjectsTheMoveWithoutItsDriver(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	original := assignedShipment(pulid.MustNew("shp_"), moveID, tenantInfo)

	svc := previewUnassignService(t, tenantInfo, original)

	plan, err := svc.PreviewUnassign(t.Context(), &repositories.UnassignShipmentMoveRequest{
		TenantInfo:     tenantInfo,
		ShipmentMoveID: moveID,
	})
	require.NoError(t, err)

	require.NotNil(t, plan.Assignment)
	assert.Equal(t, original.Moves[0].Assignment.ID, plan.Assignment.ID)
	assert.Equal(t, shipment.StatusAssigned, plan.ShipmentBefore.Status)
	assert.Equal(t, shipment.MoveStatusAssigned, plan.ShipmentBefore.Moves[0].Status)
	assert.Equal(t, shipment.StatusNew, plan.ShipmentAfter.Status)
	assert.Equal(t, shipment.MoveStatusNew, plan.ShipmentAfter.Moves[0].Status)
	assert.Equal(t, shipment.MoveCoverageTypeUnassigned, plan.ShipmentAfter.Moves[0].CoverageType)
	assert.Nil(t, plan.ShipmentAfter.Moves[0].Assignment)
}

// The preview refuses what the write refuses, from the same check: a move a
// driver has started is not taken off them.
func TestPreviewUnassign_RefusesAMoveAlreadyUnderway(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	repo := mocks.NewMockAssignmentRepository(t)
	repo.EXPECT().
		GetMoveByID(mock.Anything, tenantInfo, moveID).
		Return(&shipment.ShipmentMove{ID: moveID, Status: shipment.MoveStatusInTransit}, nil).
		Twice()

	svc := &service{
		l:            zap.NewNop(),
		db:           dbtest.NopConnection{},
		repo:         repo,
		shipmentRepo: mocks.NewMockShipmentRepository(t),
	}
	req := &repositories.UnassignShipmentMoveRequest{TenantInfo: tenantInfo, ShipmentMoveID: moveID}

	_, previewErr := svc.PreviewUnassign(t.Context(), req)
	writeErr := svc.Unassign(t.Context(), req)

	require.Error(t, previewErr)
	assert.True(t, errortypes.IsBusinessError(previewErr))
	assert.Equal(t, writeErr.Error(), previewErr.Error())
}
