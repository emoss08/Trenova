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

func previewAssignmentService(
	t *testing.T,
	tenantInfo pagination.TenantInfo,
	original *shipment.Shipment,
	existing *shipment.Assignment,
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
			Status:         shipment.MoveStatusNew,
			CoverageType:   shipment.MoveCoverageTypeUnassigned,
		}, nil).
		Once()
	repo.EXPECT().GetByMoveID(mock.Anything, tenantInfo, moveID).Return(existing, nil).Once()

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
	holdRepo := mocks.NewMockShipmentHoldRepository(t)
	holdRepo.EXPECT().HasActiveDispatchHold(mock.Anything, mock.Anything).Return(false, nil).Once()

	svc := &service{
		orgRepo:      assetOperationsOrgRepo(t, tenantInfo, true),
		l:            zap.NewNop(),
		db:           dbtest.NopConnection{},
		repo:         repo,
		shipmentRepo: shipmentRepo,
		holdRepo:     holdRepo,
		eventService: noopShipmentEventService{},
		coordinator:  shipmentstate.NewCoordinatorWithClock(func() int64 { return 10 }),
	}
	if existing != nil {
		return svc
	}

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
	svc.controlRepo = controlRepo
	svc.commercial = shipmentcommercial.New(shipmentcommercial.Params{
		Logger:          zap.NewNop(),
		RateEngine:      rateengine.NewFallbackEngine(t, formula),
		AccessorialRepo: mocks.NewMockAccessorialChargeRepository(t),
	})
	svc.shipmentValidator = shipmentservice.NewTestValidator(t)

	return svc
}

func TestPreviewAssignToMove_ProjectsTheAssignmentWithoutSavingIt(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	shipmentID := pulid.MustNew("shp_")
	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	original := validShipment(shipmentID, moveID, tenantInfo)
	original.Moves[0].CoverageType = shipment.MoveCoverageTypeUnassigned
	workerID := pulid.MustNew("wrk_")

	svc := previewAssignmentService(t, tenantInfo, original, nil)

	plan, err := svc.PreviewAssignToMove(t.Context(), &repositories.AssignShipmentMoveRequest{
		TenantInfo:      tenantInfo,
		ShipmentMoveID:  moveID,
		PrimaryWorkerID: workerID,
		TractorID:       pulid.MustNew("trac_"),
	})
	require.NoError(t, err)

	require.NotNil(t, plan.Assignment.PrimaryWorkerID)
	assert.Equal(t, workerID, *plan.Assignment.PrimaryWorkerID)
	assert.Equal(t, shipment.AssignmentStatusNew, plan.Assignment.Status)
	assert.NotEqual(t, shipment.StatusAssigned, plan.ShipmentBefore.Status)
	assert.Equal(t, shipment.StatusAssigned, plan.ShipmentAfter.Status)
	assert.Equal(t, shipment.MoveStatusAssigned, plan.ShipmentAfter.Moves[0].Status)
	assert.Equal(t, shipment.MoveCoverageTypeDriver, plan.ShipmentAfter.Moves[0].CoverageType)
}

func TestPreviewAssignToMove_RefusesAMoveAlreadyAssigned(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	original := validShipment(pulid.MustNew("shp_"), moveID, tenantInfo)

	svc := previewAssignmentService(t, tenantInfo, original, &shipment.Assignment{
		ID: pulid.MustNew("asn_"),
	})

	_, err := svc.PreviewAssignToMove(t.Context(), &repositories.AssignShipmentMoveRequest{
		TenantInfo:      tenantInfo,
		ShipmentMoveID:  moveID,
		PrimaryWorkerID: pulid.MustNew("wrk_"),
		TractorID:       pulid.MustNew("trac_"),
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
}
