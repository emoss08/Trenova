package assignmentservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/equipmentcontinuity"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/shipmentcommercial"
	"github.com/emoss08/trenova/internal/core/services/shipmentservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func TestAssignToMove_DerivesAssignedStatusesThroughShipmentCoordinator(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	shipmentID := pulid.MustNew("shp_")
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	original := validShipment(shipmentID, moveID, tenantInfo)

	repo := mocks.NewMockAssignmentRepository(t)
	repo.EXPECT().
		GetMoveByID(mock.Anything, tenantInfo, moveID).
		Return(&shipment.ShipmentMove{
			ID:             moveID,
			ShipmentID:     shipmentID,
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			Status:         shipment.MoveStatusNew,
		}, nil).
		Once()
	repo.EXPECT().GetByMoveID(mock.Anything, tenantInfo, moveID).Return(nil, nil).Once()
	repo.EXPECT().
		Create(mock.Anything, mock.AnythingOfType("*shipment.Assignment")).
		RunAndReturn(func(_ context.Context, entity *shipment.Assignment) (*shipment.Assignment, error) {
			entity.ID = pulid.MustNew("asn_")
			return entity, nil
		}).
		Once()
	repo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*repositories.GetAssignmentByIDRequest")).
		RunAndReturn(func(
			_ context.Context,
			req *repositories.GetAssignmentByIDRequest,
		) (*shipment.Assignment, error) {
			return &shipment.Assignment{
				ID:              req.AssignmentID,
				OrganizationID:  tenantInfo.OrgID,
				BusinessUnitID:  tenantInfo.BuID,
				ShipmentMoveID:  moveID,
				PrimaryWorkerID: pulid.Must("wrk_"),
				TractorID:       pulid.Must("trac_"),
				Status:          shipment.AssignmentStatusNew,
				ShipmentMove: &shipment.ShipmentMove{
					ID:     moveID,
					Status: shipment.MoveStatusAssigned,
				},
			}, nil
		}).
		Once()

	shipmentRepo := mocks.NewMockShipmentRepository(t)
	shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*repositories.GetShipmentByIDRequest")).
		RunAndReturn(func(
			_ context.Context,
			_ *repositories.GetShipmentByIDRequest,
		) (*shipment.Shipment, error) {
			return cloneShipment(original), nil
		}).
		Once()
	shipmentRepo.EXPECT().
		Update(mock.Anything, mock.AnythingOfType("*shipment.Shipment")).
		RunAndReturn(func(
			_ context.Context,
			entity *shipment.Shipment,
		) (*shipment.Shipment, error) {
			assert.Equal(t, shipment.StatusAssigned, entity.Status)
			assert.Equal(t, shipment.MoveStatusAssigned, entity.Moves[0].Status)
			require.NotNil(t, entity.Moves[0].Assignment)
			return entity, nil
		}).
		Once()
	holdRepo := mocks.NewMockShipmentHoldRepository(t)
	holdRepo.EXPECT().
		HasActiveDispatchHold(mock.Anything, mock.MatchedBy(func(req *repositories.ActiveShipmentHoldRequest) bool {
			return req.ShipmentID == shipmentID && req.TenantInfo == tenantInfo
		})).
		Return(false, nil).
		Once()
	controlRepo := mocks.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, repositories.GetShipmentControlRequest{TenantInfo: tenantInfo}).
		Return(&tenant.ShipmentControl{AutoDelayShipmentsThreshold: ptrInt16(30)}, nil).
		Once()
	formula := mocks.NewMockFormulaCalculator(t)
	formula.EXPECT().
		Calculate(mock.Anything, mock.AnythingOfType("*formulatemplatetypes.CalculateRequest")).
		Return(&formulatemplatetypes.CalculateResponse{Amount: decimal.Zero}, nil).
		Once()

	svc := &service{
		l:            zap.NewNop(),
		db:           testDBConnection{},
		repo:         repo,
		shipmentRepo: shipmentRepo,
		holdRepo:     holdRepo,
		controlRepo:  controlRepo,
		commercial: shipmentcommercial.New(shipmentcommercial.Params{
			Formula:         formula,
			AccessorialRepo: mocks.NewMockAccessorialChargeRepository(t),
		}),
		shipmentValidator: shipmentservice.NewTestValidator(t),
		coordinator:       shipmentstate.NewCoordinatorWithClock(func() int64 { return 10 }),
	}

	entity, err := svc.AssignToMove(t.Context(), &repositories.AssignShipmentMoveRequest{
		TenantInfo:      tenantInfo,
		ShipmentMoveID:  moveID,
		PrimaryWorkerID: pulid.MustNew("wrk_"),
		TractorID:       pulid.MustNew("trac_"),
	})

	require.NoError(t, err)
	require.NotNil(t, entity)
	assert.Equal(t, shipment.AssignmentStatusNew, entity.Status)
}

func TestAssignToMove_RejectsCompletedMove(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	svc := &service{
		l:  zap.NewNop(),
		db: testDBConnection{},
		repo: func() *mocks.MockAssignmentRepository {
			repo := mocks.NewMockAssignmentRepository(t)
			repo.EXPECT().
				GetMoveByID(mock.Anything, tenantInfo, moveID).
				Return(&shipment.ShipmentMove{ID: moveID, Status: shipment.MoveStatusCompleted}, nil).
				Once()
			return repo
		}(),
		shipmentRepo: mocks.NewMockShipmentRepository(t),
		holdRepo:     mocks.NewMockShipmentHoldRepository(t),
		commercial: shipmentcommercial.New(shipmentcommercial.Params{
			Formula:         mocks.NewMockFormulaCalculator(t),
			AccessorialRepo: mocks.NewMockAccessorialChargeRepository(t),
		}),
		shipmentValidator: shipmentservice.NewTestValidator(t),
		coordinator:       shipmentstate.NewCoordinator(),
	}

	entity, err := svc.AssignToMove(t.Context(), &repositories.AssignShipmentMoveRequest{
		TenantInfo:      tenantInfo,
		ShipmentMoveID:  moveID,
		PrimaryWorkerID: pulid.MustNew("wrk_"),
		TractorID:       pulid.MustNew("trac_"),
	})

	require.Nil(t, entity)
	require.Error(t, err)
	var businessErr *errortypes.BusinessError
	require.ErrorAs(t, err, &businessErr)
}

func TestAssignToMove_RejectsDispatchBlockingHold(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	shipmentID := pulid.MustNew("shp_")
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	repo := mocks.NewMockAssignmentRepository(t)
	repo.EXPECT().
		GetMoveByID(mock.Anything, tenantInfo, moveID).
		Return(&shipment.ShipmentMove{
			ID:         moveID,
			ShipmentID: shipmentID,
			Status:     shipment.MoveStatusNew,
		}, nil).
		Once()

	holdRepo := mocks.NewMockShipmentHoldRepository(t)
	holdRepo.EXPECT().
		HasActiveDispatchHold(mock.Anything, mock.MatchedBy(func(req *repositories.ActiveShipmentHoldRequest) bool {
			return req.ShipmentID == shipmentID && req.TenantInfo == tenantInfo
		})).
		Return(true, nil).
		Once()

	svc := &service{
		l:            zap.NewNop(),
		db:           testDBConnection{},
		repo:         repo,
		shipmentRepo: mocks.NewMockShipmentRepository(t),
		holdRepo:     holdRepo,
		commercial: shipmentcommercial.New(shipmentcommercial.Params{
			Formula:         mocks.NewMockFormulaCalculator(t),
			AccessorialRepo: mocks.NewMockAccessorialChargeRepository(t),
		}),
		shipmentValidator: shipmentservice.NewTestValidator(t),
		coordinator:       shipmentstate.NewCoordinator(),
	}

	entity, err := svc.AssignToMove(t.Context(), &repositories.AssignShipmentMoveRequest{
		TenantInfo:      tenantInfo,
		ShipmentMoveID:  moveID,
		PrimaryWorkerID: pulid.MustNew("wrk_"),
		TractorID:       pulid.MustNew("trac_"),
	})

	require.Nil(t, entity)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
}

func TestAssignToMove_RejectsTrailerContinuityMismatch(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	shipmentID := pulid.MustNew("shp_")
	trailerID := pulid.MustNew("tr_")
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	original := validShipment(shipmentID, moveID, tenantInfo)
	original.Moves[0].Stops[0].LocationID = pulid.MustNew("loc_pickup_")

	repo := mocks.NewMockAssignmentRepository(t)
	repo.EXPECT().
		GetMoveByID(mock.Anything, tenantInfo, moveID).
		Return(&shipment.ShipmentMove{
			ID:             moveID,
			ShipmentID:     shipmentID,
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			Status:         shipment.MoveStatusNew,
		}, nil).
		Once()
	repo.EXPECT().GetByMoveID(mock.Anything, tenantInfo, moveID).Return(nil, nil).Once()

	shipmentRepo := mocks.NewMockShipmentRepository(t)
	shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*repositories.GetShipmentByIDRequest")).
		Return(cloneShipment(original), nil).
		Once()

	holdRepo := mocks.NewMockShipmentHoldRepository(t)
	holdRepo.EXPECT().
		HasActiveDispatchHold(mock.Anything, mock.AnythingOfType("*repositories.ActiveShipmentHoldRequest")).
		Return(false, nil).
		Once()

	dispatchControlRepo := mocks.NewMockDispatchControlRepository(t)
	dispatchControlRepo.EXPECT().
		GetOrCreate(mock.Anything, tenantInfo.OrgID, tenantInfo.BuID).
		Return(&dispatchcontrol.DispatchControl{EnforceTrailerContinuity: true}, nil).
		Once()

	currentLocationID := pulid.MustNew("loc_elsewhere_")
	continuityRepo := mocks.NewMockEquipmentContinuityRepository(t)
	continuityRepo.EXPECT().
		GetEffectiveCurrent(mock.Anything, repositories.GetCurrentEquipmentContinuityRequest{
			TenantInfo:    tenantInfo,
			EquipmentType: equipmentcontinuity.EquipmentTypeTrailer,
			EquipmentID:   trailerID,
		}).
		Return(&equipmentcontinuity.EquipmentContinuity{
			ID:                   pulid.MustNew("eqc_"),
			EquipmentType:        equipmentcontinuity.EquipmentTypeTrailer,
			EquipmentID:          trailerID,
			CurrentLocationID:    currentLocationID,
			SourceShipmentMoveID: pulid.MustNew("sm_prev_"),
		}, nil).
		Once()

	trailerRepo := mocks.NewMockTrailerRepository(t)
	trailerRepo.EXPECT().
		GetByID(mock.Anything, repositories.GetTrailerByIDRequest{
			ID:         trailerID,
			TenantInfo: tenantInfo,
		}).
		Return(&trailer.Trailer{
			ID:             trailerID,
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			Code:           "TRL-100",
		}, nil).
		Once()

	locationRepo := mocks.NewMockLocationRepository(t)
	locationRepo.EXPECT().
		GetByID(mock.Anything, repositories.GetLocationByIDRequest{
			ID:         currentLocationID,
			TenantInfo: tenantInfo,
		}).
		Return(&location.Location{
			ID:             currentLocationID,
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			Name:           "Madison Yard",
		}, nil).
		Once()

	svc := &service{
		l:                   zap.NewNop(),
		db:                  testDBConnection{},
		repo:                repo,
		shipmentRepo:        shipmentRepo,
		holdRepo:            holdRepo,
		dispatchControlRepo: dispatchControlRepo,
		continuityRepo:      continuityRepo,
		trailerRepo:         trailerRepo,
		locationRepo:        locationRepo,
		commercial: shipmentcommercial.New(shipmentcommercial.Params{
			Formula:         mocks.NewMockFormulaCalculator(t),
			AccessorialRepo: mocks.NewMockAccessorialChargeRepository(t),
		}),
		shipmentValidator: shipmentservice.NewTestValidator(t),
		coordinator:       shipmentstate.NewCoordinator(),
	}

	entity, err := svc.AssignToMove(t.Context(), &repositories.AssignShipmentMoveRequest{
		TenantInfo:      tenantInfo,
		ShipmentMoveID:  moveID,
		PrimaryWorkerID: pulid.MustNew("wrk_"),
		TractorID:       pulid.MustNew("trac_"),
		TrailerID:       &trailerID,
	})

	require.Nil(t, entity)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Equal(
		t,
		"Trailer TRL-100 is currently located at Madison Yard which doesn't match this move's current pickup location. Locate the trailer before assigning or assign a different trailer",
		err.Error(),
	)
}

func TestAssignToMove_DoesNotAdvanceTrailerContinuityBeforeCompletion(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	shipmentID := pulid.MustNew("shp_")
	trailerID := pulid.MustNew("tr_")
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	original := validShipment(shipmentID, moveID, tenantInfo)

	repo := mocks.NewMockAssignmentRepository(t)
	repo.EXPECT().
		GetMoveByID(mock.Anything, tenantInfo, moveID).
		Return(&shipment.ShipmentMove{
			ID:             moveID,
			ShipmentID:     shipmentID,
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			Status:         shipment.MoveStatusNew,
		}, nil).
		Once()
	repo.EXPECT().GetByMoveID(mock.Anything, tenantInfo, moveID).Return(nil, nil).Once()
	repo.EXPECT().
		Create(mock.Anything, mock.AnythingOfType("*shipment.Assignment")).
		RunAndReturn(func(_ context.Context, entity *shipment.Assignment) (*shipment.Assignment, error) {
			entity.ID = pulid.MustNew("asn_")
			return entity, nil
		}).
		Once()
	repo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*repositories.GetAssignmentByIDRequest")).
		Return(&shipment.Assignment{
			ID:              pulid.MustNew("asn_"),
			OrganizationID:  tenantInfo.OrgID,
			BusinessUnitID:  tenantInfo.BuID,
			ShipmentMoveID:  moveID,
			PrimaryWorkerID: pulid.Must("wrk_"),
			TractorID:       pulid.Must("trac_"),
			TrailerID:       &trailerID,
			Status:          shipment.AssignmentStatusNew,
		}, nil).
		Once()

	shipmentRepo := mocks.NewMockShipmentRepository(t)
	shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*repositories.GetShipmentByIDRequest")).
		Return(cloneShipment(original), nil).
		Once()
	shipmentRepo.EXPECT().
		Update(mock.Anything, mock.AnythingOfType("*shipment.Shipment")).
		Return(cloneShipment(original), nil).
		Once()

	holdRepo := mocks.NewMockShipmentHoldRepository(t)
	holdRepo.EXPECT().
		HasActiveDispatchHold(mock.Anything, mock.AnythingOfType("*repositories.ActiveShipmentHoldRequest")).
		Return(false, nil).
		Once()

	controlRepo := mocks.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, repositories.GetShipmentControlRequest{TenantInfo: tenantInfo}).
		Return(&tenant.ShipmentControl{AutoDelayShipmentsThreshold: ptrInt16(30)}, nil).
		Once()

	dispatchControlRepo := mocks.NewMockDispatchControlRepository(t)
	dispatchControlRepo.EXPECT().
		GetOrCreate(mock.Anything, tenantInfo.OrgID, tenantInfo.BuID).
		Return(&dispatchcontrol.DispatchControl{EnforceTrailerContinuity: true}, nil).
		Once()

	continuityRepo := mocks.NewMockEquipmentContinuityRepository(t)
	continuityRepo.EXPECT().
		GetEffectiveCurrent(mock.Anything, repositories.GetCurrentEquipmentContinuityRequest{
			TenantInfo:    tenantInfo,
			EquipmentType: equipmentcontinuity.EquipmentTypeTrailer,
			EquipmentID:   trailerID,
		}).
		Return(nil, nil).
		Once()

	formula := mocks.NewMockFormulaCalculator(t)
	formula.EXPECT().
		Calculate(mock.Anything, mock.AnythingOfType("*formulatemplatetypes.CalculateRequest")).
		Return(&formulatemplatetypes.CalculateResponse{Amount: decimal.Zero}, nil).
		Once()

	svc := &service{
		l:                   zap.NewNop(),
		db:                  testDBConnection{},
		repo:                repo,
		shipmentRepo:        shipmentRepo,
		holdRepo:            holdRepo,
		controlRepo:         controlRepo,
		dispatchControlRepo: dispatchControlRepo,
		continuityRepo:      continuityRepo,
		commercial: shipmentcommercial.New(shipmentcommercial.Params{
			Formula:         formula,
			AccessorialRepo: mocks.NewMockAccessorialChargeRepository(t),
		}),
		shipmentValidator: shipmentservice.NewTestValidator(t),
		coordinator:       shipmentstate.NewCoordinatorWithClock(func() int64 { return 10 }),
	}

	entity, err := svc.AssignToMove(t.Context(), &repositories.AssignShipmentMoveRequest{
		TenantInfo:      tenantInfo,
		ShipmentMoveID:  moveID,
		PrimaryWorkerID: pulid.MustNew("wrk_"),
		TractorID:       pulid.MustNew("trac_"),
		TrailerID:       &trailerID,
	})

	require.NoError(t, err)
	require.NotNil(t, entity)
}

func TestUnassign_ClearsAssignmentAndDerivesUnassignedStatuses(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	shipmentID := pulid.MustNew("shp_")
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	assignmentID := pulid.MustNew("asn_")
	original := validShipment(shipmentID, moveID, tenantInfo)
	original.Status = shipment.StatusAssigned
	original.Moves[0].Status = shipment.MoveStatusAssigned
	original.Moves[0].Assignment = &shipment.Assignment{
		ID:              assignmentID,
		OrganizationID:  tenantInfo.OrgID,
		BusinessUnitID:  tenantInfo.BuID,
		ShipmentMoveID:  moveID,
		PrimaryWorkerID: pulid.Must("wrk_"),
		TractorID:       pulid.Must("trc_"),
		Status:          shipment.AssignmentStatusNew,
	}

	repo := mocks.NewMockAssignmentRepository(t)
	repo.EXPECT().
		GetMoveByID(mock.Anything, tenantInfo, moveID).
		Return(&shipment.ShipmentMove{
			ID:             moveID,
			ShipmentID:     shipmentID,
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			Status:         shipment.MoveStatusAssigned,
		}, nil).
		Once()
	repo.EXPECT().
		GetByMoveID(mock.Anything, tenantInfo, moveID).
		Return(&shipment.Assignment{
			ID:              assignmentID,
			OrganizationID:  tenantInfo.OrgID,
			BusinessUnitID:  tenantInfo.BuID,
			ShipmentMoveID:  moveID,
			PrimaryWorkerID: pulid.Must("wrk_"),
			TractorID:       pulid.Must("trc_"),
			Status:          shipment.AssignmentStatusNew,
			Version:         2,
		}, nil).
		Once()
	repo.EXPECT().
		Unassign(mock.Anything, mock.AnythingOfType("*shipment.Assignment")).
		RunAndReturn(func(_ context.Context, entity *shipment.Assignment) (*shipment.Assignment, error) {
			return entity, nil
		}).
		Once()

	shipmentRepo := mocks.NewMockShipmentRepository(t)
	shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*repositories.GetShipmentByIDRequest")).
		RunAndReturn(func(_ context.Context, _ *repositories.GetShipmentByIDRequest) (*shipment.Shipment, error) {
			return cloneShipment(original), nil
		}).
		Once()
	shipmentRepo.EXPECT().
		Update(mock.Anything, mock.AnythingOfType("*shipment.Shipment")).
		RunAndReturn(func(_ context.Context, entity *shipment.Shipment) (*shipment.Shipment, error) {
			assert.Equal(t, shipment.StatusNew, entity.Status)
			assert.Equal(t, shipment.MoveStatusNew, entity.Moves[0].Status)
			assert.Nil(t, entity.Moves[0].Assignment)
			return entity, nil
		}).
		Once()
	holdRepo := mocks.NewMockShipmentHoldRepository(t)
	controlRepo := mocks.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, repositories.GetShipmentControlRequest{TenantInfo: tenantInfo}).
		Return(&tenant.ShipmentControl{AutoDelayShipmentsThreshold: ptrInt16(30)}, nil).
		Once()
	formula := mocks.NewMockFormulaCalculator(t)
	formula.EXPECT().
		Calculate(mock.Anything, mock.AnythingOfType("*formulatemplatetypes.CalculateRequest")).
		Return(&formulatemplatetypes.CalculateResponse{Amount: decimal.Zero}, nil).
		Once()

	svc := &service{
		l:            zap.NewNop(),
		db:           testDBConnection{},
		repo:         repo,
		shipmentRepo: shipmentRepo,
		holdRepo:     holdRepo,
		controlRepo:  controlRepo,
		commercial: shipmentcommercial.New(shipmentcommercial.Params{
			Formula:         formula,
			AccessorialRepo: mocks.NewMockAccessorialChargeRepository(t),
		}),
		shipmentValidator: shipmentservice.NewTestValidator(t),
		coordinator:       shipmentstate.NewCoordinatorWithClock(func() int64 { return 10 }),
	}

	err := svc.Unassign(t.Context(), &repositories.UnassignShipmentMoveRequest{
		TenantInfo:     tenantInfo,
		ShipmentMoveID: moveID,
	})

	require.NoError(t, err)
}

func TestUnassign_RejectsMissingAssignment(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	repo := mocks.NewMockAssignmentRepository(t)
	repo.EXPECT().
		GetMoveByID(mock.Anything, tenantInfo, moveID).
		Return(&shipment.ShipmentMove{
			ID:     moveID,
			Status: shipment.MoveStatusAssigned,
		}, nil).
		Once()

	shipmentRepo := mocks.NewMockShipmentRepository(t)
	shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*repositories.GetShipmentByIDRequest")).
		Return(validShipment(pulid.MustNew("shp_"), moveID, tenantInfo), nil).
		Once()

	repo.EXPECT().
		GetByMoveID(mock.Anything, tenantInfo, moveID).
		Return(nil, nil).
		Once()

	svc := &service{
		l:            zap.NewNop(),
		db:           testDBConnection{},
		repo:         repo,
		shipmentRepo: shipmentRepo,
		holdRepo:     mocks.NewMockShipmentHoldRepository(t),
		commercial: shipmentcommercial.New(shipmentcommercial.Params{
			Formula:         mocks.NewMockFormulaCalculator(t),
			AccessorialRepo: mocks.NewMockAccessorialChargeRepository(t),
		}),
		shipmentValidator: shipmentservice.NewTestValidator(t),
		coordinator:       shipmentstate.NewCoordinator(),
	}

	err := svc.Unassign(t.Context(), &repositories.UnassignShipmentMoveRequest{
		TenantInfo:     tenantInfo,
		ShipmentMoveID: moveID,
	})

	require.Error(t, err)
	var notFoundErr *errortypes.NotFoundError
	require.ErrorAs(t, err, &notFoundErr)
}

func TestUnassign_RejectsProgressedMove(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	repo := mocks.NewMockAssignmentRepository(t)
	repo.EXPECT().
		GetMoveByID(mock.Anything, tenantInfo, moveID).
		Return(&shipment.ShipmentMove{
			ID:     moveID,
			Status: shipment.MoveStatusInTransit,
		}, nil).
		Once()

	svc := &service{
		l:            zap.NewNop(),
		db:           testDBConnection{},
		repo:         repo,
		shipmentRepo: mocks.NewMockShipmentRepository(t),
		holdRepo:     mocks.NewMockShipmentHoldRepository(t),
		commercial: shipmentcommercial.New(shipmentcommercial.Params{
			Formula:         mocks.NewMockFormulaCalculator(t),
			AccessorialRepo: mocks.NewMockAccessorialChargeRepository(t),
		}),
		shipmentValidator: shipmentservice.NewTestValidator(t),
		coordinator:       shipmentstate.NewCoordinator(),
	}

	err := svc.Unassign(t.Context(), &repositories.UnassignShipmentMoveRequest{
		TenantInfo:     tenantInfo,
		ShipmentMoveID: moveID,
	})

	require.Error(t, err)
	var businessErr *errortypes.BusinessError
	require.ErrorAs(t, err, &businessErr)
}

type testDBConnection struct{}

func (testDBConnection) DB() *bun.DB                          { return nil }
func (testDBConnection) DBForContext(context.Context) bun.IDB { return nil }
func (testDBConnection) HealthCheck(context.Context) error    { return nil }
func (testDBConnection) IsHealthy(context.Context) bool       { return true }
func (testDBConnection) Close() error                         { return nil }
func (testDBConnection) WithTx(
	ctx context.Context,
	_ ports.TxOptions,
	fn func(context.Context, bun.Tx) error,
) error {
	return fn(ctx, bun.Tx{})
}

func validShipment(
	shipmentID, moveID pulid.ID,
	tenantInfo pagination.TenantInfo,
) *shipment.Shipment {
	return &shipment.Shipment{
		ID:                shipmentID,
		OrganizationID:    tenantInfo.OrgID,
		BusinessUnitID:    tenantInfo.BuID,
		ServiceTypeID:     pulid.MustNew("svc_"),
		CustomerID:        pulid.MustNew("cust_"),
		BOL:               "BOL-1",
		FormulaTemplateID: pulid.MustNew("ft_"),
		Status:            shipment.StatusNew,
		Moves: []*shipment.ShipmentMove{
			{
				ID:             moveID,
				ShipmentID:     shipmentID,
				OrganizationID: tenantInfo.OrgID,
				BusinessUnitID: tenantInfo.BuID,
				Status:         shipment.MoveStatusNew,
				Loaded:         true,
				Stops: []*shipment.Stop{
					{
						ID:                   pulid.MustNew("stp_"),
						ShipmentMoveID:       moveID,
						OrganizationID:       tenantInfo.OrgID,
						BusinessUnitID:       tenantInfo.BuID,
						Sequence:             0,
						Status:               shipment.StopStatusNew,
						Type:                 shipment.StopTypePickup,
						LocationID:           pulid.MustNew("loc_"),
						ScheduleType:         shipment.StopScheduleTypeOpen,
						ScheduledWindowStart: 1,
						ScheduledWindowEnd:   ptrInt64(2),
					},
					{
						ID:                   pulid.MustNew("stp_"),
						ShipmentMoveID:       moveID,
						OrganizationID:       tenantInfo.OrgID,
						BusinessUnitID:       tenantInfo.BuID,
						Sequence:             1,
						Status:               shipment.StopStatusNew,
						Type:                 shipment.StopTypeDelivery,
						LocationID:           pulid.MustNew("loc_"),
						ScheduleType:         shipment.StopScheduleTypeOpen,
						ScheduledWindowStart: 3,
						ScheduledWindowEnd:   ptrInt64(4),
					},
				},
			},
		},
	}
}

//go:fix inline
func ptrInt16(v int16) *int16 {
	return &v
}

func ptrInt64(v int64) *int64 {
	return &v
}

func TestResolveDelayThresholdMinutes_DisablesAutomaticDelayWhenToggleOff(t *testing.T) {
	t.Parallel()

	assert.Equal(t, shipmentstate.DisabledDelayThresholdMinutes, resolveDelayThresholdMinutes(nil))
	assert.Equal(t, shipmentstate.DisabledDelayThresholdMinutes, resolveDelayThresholdMinutes(&tenant.ShipmentControl{
		AutoDelayShipments:          false,
		AutoDelayShipmentsThreshold: ptrInt16(30),
	}))
	assert.Equal(t, int16(30), resolveDelayThresholdMinutes(&tenant.ShipmentControl{
		AutoDelayShipments:          true,
		AutoDelayShipmentsThreshold: ptrInt16(30),
	}))
}
