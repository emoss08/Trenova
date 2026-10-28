package assignmentservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentcommercial"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentcoverage"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/modules/operations/ports/assignmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/rating/ratingtest"
	"github.com/emoss08/trenova/internal/testutil/capabilitytest"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks/accessorialchargeportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/assignmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentholdportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
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

	return previewAssignmentServiceWith(t, tenantInfo, original, existing, nil)
}

func previewAssignmentServiceWith(
	t *testing.T,
	tenantInfo pagination.TenantInfo,
	original *shipment.Shipment,
	existing *shipment.Assignment,
	tractorConflict *shipment.Assignment,
) *service {
	t.Helper()

	moveID := original.Moves[0].ID
	repo := assignmentportsmock.NewMockAssignmentRepository(t)
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
	repo.EXPECT().
		FindInProgressByTractorID(mock.Anything, tenantInfo, mock.Anything, moveID).
		Return(tractorConflict, nil).
		Maybe()
	repo.EXPECT().
		FindInProgressByTrailerID(mock.Anything, tenantInfo, mock.Anything, moveID).
		Return(nil, nil).
		Maybe()
	if tractorConflict != nil {
		repo.EXPECT().
			GetMoveByID(mock.Anything, tenantInfo, tractorConflict.ShipmentMoveID).
			Return(nil, errors.New("not found")).
			Once()
	}

	shipmentRepo := shipmentportsmock.NewMockShipmentRepository(t)
	shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*shipmentports.GetShipmentByIDRequest")).
		RunAndReturn(func(
			context.Context,
			*shipmentports.GetShipmentByIDRequest,
		) (*shipment.Shipment, error) {
			return shipmentcoverage.CloneForUpdate(original), nil
		}).
		Once()
	holdRepo := shipmentholdportsmock.NewMockShipmentHoldRepository(t)
	holdRepo.EXPECT().HasActiveDispatchHold(mock.Anything, mock.Anything).Return(false, nil).Once()

	svc := &service{
		orgRepo: capabilitytest.OrgRepo(t, capabilitytest.OrgRepoParams{
			TenantInfo:             tenantInfo,
			Lock:                   tenantports.CapabilityLockNone,
			BrokerageEnabled:       true,
			AssetOperationsEnabled: true,
			Optional:               true,
		}),
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

	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, tenantports.GetShipmentControlRequest{TenantInfo: tenantInfo}).
		Return(&tenant.ShipmentControl{AutoDelayShipmentsThreshold: new(int16(30))}, nil).
		Once()
	svc.controlRepo = controlRepo
	svc.commercial = shipmentcommercial.New(shipmentcommercial.Params{
		Logger:          zap.NewNop(),
		RateEngine:      ratingtest.NewFlatEngine(t, decimal.Zero),
		AccessorialRepo: accessorialchargeportsmock.NewMockAccessorialChargeRepository(t),
	})
	svc.shipmentValidator = newTestShipmentValidator(t)

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

	plan, err := svc.PreviewAssignToMove(t.Context(), &assignmentports.AssignShipmentMoveRequest{
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

// A tractor still on another move in progress can be assigned ahead, but the
// move cannot start with it; the preview says so rather than leaving the
// refusal to the pickup, after the assignment was approved.
func TestPreviewAssignToMove_SaysWhenEquipmentIsOnAMoveInProgress(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	original := validShipment(pulid.MustNew("shp_"), moveID, tenantInfo)
	original.Moves[0].CoverageType = shipment.MoveCoverageTypeUnassigned
	tractorID := pulid.MustNew("trac_")
	busyMove := pulid.MustNew("sm_")

	svc := previewAssignmentServiceWith(t, tenantInfo, original, nil,
		&shipment.Assignment{ShipmentMoveID: busyMove, TractorID: &tractorID})

	plan, err := svc.PreviewAssignToMove(t.Context(), &assignmentports.AssignShipmentMoveRequest{
		TenantInfo:      tenantInfo,
		ShipmentMoveID:  moveID,
		PrimaryWorkerID: pulid.MustNew("wrk_"),
		TractorID:       tractorID,
	})
	require.NoError(t, err, "assigning equipment ahead is allowed")
	require.Len(t, plan.EquipmentInUse, 1)
	assert.Equal(t, "tractor", plan.EquipmentInUse[0].Kind)
	assert.Equal(t, busyMove, plan.EquipmentInUse[0].ShipmentMoveID)
}

func TestPreviewAssignToMove_RefusesAMoveAlreadyAssigned(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	original := validShipment(pulid.MustNew("shp_"), moveID, tenantInfo)

	svc := previewAssignmentService(t, tenantInfo, original, &shipment.Assignment{
		ID: pulid.MustNew("asn_"),
	})

	_, err := svc.PreviewAssignToMove(t.Context(), &assignmentports.AssignShipmentMoveRequest{
		TenantInfo:      tenantInfo,
		ShipmentMoveID:  moveID,
		PrimaryWorkerID: pulid.MustNew("wrk_"),
		TractorID:       pulid.MustNew("trac_"),
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
}
