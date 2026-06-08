package shipmentservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/equipmentcontinuity"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	internaltestutil "github.com/emoss08/trenova/internal/testutil"
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

func TestStateCoordinator_PrepareForCreate_DerivesStatusesAndTimestamps(t *testing.T) {
	t.Parallel()

	actualPickupArrival := int64(110)
	actualPickupDeparture := int64(120)
	actualDeliveryArrival := int64(190)
	actualDeliveryDeparture := int64(200)

	entity := validShipmentForValidation()
	entity.Moves[0].Stops[0].ActualArrival = &actualPickupArrival
	entity.Moves[0].Stops[0].ActualDeparture = &actualPickupDeparture
	entity.Moves[0].Stops[1].ActualArrival = &actualDeliveryArrival
	entity.Moves[0].Stops[1].ActualDeparture = &actualDeliveryDeparture

	coordinator := shipmentstate.NewCoordinatorWithClock(func() int64 { return 10 })

	multiErr := coordinator.PrepareForCreateWithDelayThreshold(entity, 30)

	require.Nil(t, multiErr)
	assert.Equal(t, shipment.StopStatusCompleted, entity.Moves[0].Stops[0].Status)
	assert.Equal(t, shipment.StopStatusCompleted, entity.Moves[0].Stops[1].Status)
	assert.Equal(t, shipment.MoveStatusCompleted, entity.Moves[0].Status)
	assert.Equal(t, shipment.StatusCompleted, entity.Status)
	require.NotNil(t, entity.ActualShipDate)
	require.NotNil(t, entity.ActualDeliveryDate)
	assert.Equal(t, actualPickupDeparture, *entity.ActualShipDate)
	assert.Equal(t, actualDeliveryArrival, *entity.ActualDeliveryDate)
}

func TestStateCoordinator_PrepareForCreate_PreservesAssignedStatusesWithoutOperationalSignal(
	t *testing.T,
) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.Moves[0].Assignment = &shipment.Assignment{ID: pulid.MustNew("asn_")}

	coordinator := shipmentstate.NewCoordinatorWithClock(func() int64 { return 10 })

	multiErr := coordinator.PrepareForCreateWithDelayThreshold(entity, 30)

	require.Nil(t, multiErr)
	assert.Equal(t, shipment.MoveStatusAssigned, entity.Moves[0].Status)
	assert.Equal(t, shipment.StatusAssigned, entity.Status)
}

func TestStateCoordinator_PrepareForCreate_DerivesPartiallyAssignedShipment(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.Moves = []*shipment.ShipmentMove{
		validMove(),
		validMove(),
	}
	entity.Moves[0].Assignment = &shipment.Assignment{ID: pulid.MustNew("asn_")}

	coordinator := shipmentstate.NewCoordinatorWithClock(func() int64 { return 10 })

	multiErr := coordinator.PrepareForCreateWithDelayThreshold(entity, 30)

	require.Nil(t, multiErr)
	assert.Equal(t, shipment.MoveStatusAssigned, entity.Moves[0].Status)
	assert.Equal(t, shipment.MoveStatusNew, entity.Moves[1].Status)
	assert.Equal(t, shipment.StatusPartiallyAssigned, entity.Status)
}

func TestStateCoordinator_PrepareForCreate_DerivesDelayedShipment(t *testing.T) {
	t.Parallel()

	entity := validShipmentForValidation()
	entity.Moves[0].Stops[0].ActualArrival = ptrInt64(150)
	entity.Moves[0].Stops[0].ScheduledWindowEnd = ptrInt64(100)

	coordinator := shipmentstate.NewCoordinatorWithClock(func() int64 { return 200 })

	multiErr := coordinator.PrepareForCreateWithDelayThreshold(entity, 1)

	require.Nil(t, multiErr)
	assert.Equal(t, shipment.StopStatusInTransit, entity.Moves[0].Stops[0].Status)
	assert.Equal(t, shipment.MoveStatusInTransit, entity.Moves[0].Status)
	assert.Equal(t, shipment.StatusDelayed, entity.Status)
}

func TestStateCoordinator_PrepareForUpdate_AllowsCompletedStopTimestampCorrection(t *testing.T) {
	t.Parallel()

	original := completedShipmentForCoordinator()
	entity := cloneShipment(original)
	correctedArrival := int64(130)
	correctedDeparture := int64(140)
	entity.Moves[0].Stops[0].ActualArrival = &correctedArrival
	entity.Moves[0].Stops[0].ActualDeparture = &correctedDeparture

	coordinator := shipmentstate.NewCoordinatorWithClock(func() int64 { return 1000 })

	multiErr := coordinator.PrepareForUpdateWithDelayThreshold(original, entity, 30)

	require.Nil(t, multiErr)
	assert.Equal(t, shipment.StopStatusCompleted, entity.Moves[0].Stops[0].Status)
	assert.Equal(t, shipment.MoveStatusCompleted, entity.Moves[0].Status)
	assert.Equal(t, shipment.StatusCompleted, entity.Status)
	assert.Equal(t, correctedDeparture, *entity.ActualShipDate)
}

func TestStateCoordinator_PrepareForUpdate_RejectsCompletedStopTimestampRemoval(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		mutate        func(*shipment.Stop)
		expectedField string
	}{
		{
			name: "clears actual arrival",
			mutate: func(stop *shipment.Stop) {
				stop.ActualArrival = nil
			},
			expectedField: "moves[0].stops[0].actualArrival",
		},
		{
			name: "clears actual departure",
			mutate: func(stop *shipment.Stop) {
				stop.ActualDeparture = nil
			},
			expectedField: "moves[0].stops[0].actualDeparture",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			original := completedShipmentForCoordinator()
			entity := cloneShipment(original)
			tt.mutate(entity.Moves[0].Stops[0])

			coordinator := shipmentstate.NewCoordinatorWithClock(func() int64 { return 1000 })

			multiErr := coordinator.PrepareForUpdateWithDelayThreshold(original, entity, 30)

			require.NotNil(t, multiErr)
			assertErrorField(t, multiErr, tt.expectedField)
			assert.Equal(t, shipment.StopStatusCompleted, entity.Moves[0].Stops[0].Status)
			assert.Equal(t, shipment.MoveStatusCompleted, entity.Moves[0].Status)
		})
	}
}

func TestStateCoordinator_PrepareForUpdate_RejectsCompletedMoveReopenByOmittedStops(t *testing.T) {
	t.Parallel()

	original := completedShipmentForCoordinator()
	entity := cloneShipment(original)
	entity.Moves[0].Stops = []*shipment.Stop{}

	coordinator := shipmentstate.NewCoordinatorWithClock(func() int64 { return 1000 })

	multiErr := coordinator.PrepareForUpdateWithDelayThreshold(original, entity, 30)

	require.NotNil(t, multiErr)
	assertErrorField(t, multiErr, "moves[0].status")
	assert.Equal(t, shipment.MoveStatusCompleted, entity.Moves[0].Status)
	assert.Equal(t, shipment.StatusCompleted, entity.Status)
}

func TestStateCoordinator_RefreshShipmentState_PreservesBillingAndTerminalStatuses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		status         shipment.Status
		expectedStatus shipment.Status
	}{
		{
			name:           "ready to invoice",
			status:         shipment.StatusReadyToInvoice,
			expectedStatus: shipment.StatusReadyToInvoice,
		},
		{
			name:           "invoiced",
			status:         shipment.StatusInvoiced,
			expectedStatus: shipment.StatusInvoiced,
		},
		{
			name:           "canceled",
			status:         shipment.StatusCanceled,
			expectedStatus: shipment.StatusCanceled,
		},
		{
			name:           "completed",
			status:         shipment.StatusCompleted,
			expectedStatus: shipment.StatusCompleted,
		},
		{
			name:           "active in transit",
			status:         shipment.StatusInTransit,
			expectedStatus: shipment.StatusDelayed,
		},
		{
			name:           "partially completed active",
			status:         shipment.StatusPartiallyCompleted,
			expectedStatus: shipment.StatusDelayed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entity := shipmentWithOverdueStopForRefresh(tt.status)
			coordinator := shipmentstate.NewCoordinatorWithClock(func() int64 { return 200 })

			coordinator.RefreshShipmentStateWithDelayThreshold(entity, 1)

			assert.Equal(t, tt.expectedStatus, entity.Status)
		})
	}
}

func TestStateCoordinator_RefreshShipmentState_UsesSequenceOrderForTimestamps(t *testing.T) {
	t.Parallel()

	actualShipDate := int64(120)
	actualDeliveryDate := int64(400)
	entity := validShipmentForValidation()
	entity.Moves = []*shipment.ShipmentMove{
		{
			Status:   shipment.MoveStatusCompleted,
			Sequence: 10,
			Stops: []*shipment.Stop{
				completedStopForCoordinator(shipment.StopTypePickup, 0, 300, 310),
				completedStopForCoordinator(
					shipment.StopTypeDelivery,
					1,
					actualDeliveryDate,
					410,
				),
			},
		},
		{
			Status:   shipment.MoveStatusCompleted,
			Sequence: 0,
			Stops: []*shipment.Stop{
				completedStopForCoordinator(shipment.StopTypeDelivery, 1, 200, 210),
				completedStopForCoordinator(shipment.StopTypePickup, 0, 100, actualShipDate),
			},
		},
	}

	coordinator := shipmentstate.NewCoordinatorWithClock(func() int64 { return 1000 })

	coordinator.RefreshShipmentStateWithDelayThreshold(entity, 30)

	require.NotNil(t, entity.ActualShipDate)
	require.NotNil(t, entity.ActualDeliveryDate)
	assert.Equal(t, actualShipDate, *entity.ActualShipDate)
	assert.Equal(t, actualDeliveryDate, *entity.ActualDeliveryDate)
}

func TestServiceUpdate_RejectsReadyToInvoiceBeforeCompletion(t *testing.T) {
	t.Parallel()

	original := validShipmentForValidation()
	original.ID = pulid.MustNew("shp_")
	original.Version = 1

	entity := validShipmentForValidation()
	entity.ID = original.ID
	entity.Version = original.Version
	entity.Status = shipment.StatusReadyToInvoice

	repo := mocks.NewMockShipmentRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*repositories.GetShipmentByIDRequest")).
		RunAndReturn(func(
			_ context.Context,
			req *repositories.GetShipmentByIDRequest,
		) (*shipment.Shipment, error) {
			assert.True(t, req.ExpandShipmentDetails)
			return cloneShipment(original), nil
		}).
		Once()
	controlRepo := mocks.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, repositories.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		}).
		Return(&tenant.ShipmentControl{}, nil).
		Once()
	userID := pulid.MustNew("usr_")
	svc := &service{
		l:            zap.NewNop(),
		repo:         repo,
		controlRepo:  controlRepo,
		validator:    NewTestValidator(t),
		auditService: mocks.NewMockAuditService(t),
		eventService: noopShipmentEventService{},
		coordinator:  shipmentstate.NewCoordinatorWithClock(func() int64 { return 10 }),
	}

	updated, err := svc.Update(
		t.Context(),
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.Nil(t, updated)
	require.Error(t, err)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assertErrorField(t, multiErr, "status")
}

func TestServiceUpdate_DerivesAuthoritativeStatusesBeforePersist(t *testing.T) {
	t.Parallel()

	actualPickupArrival := int64(110)
	actualPickupDeparture := int64(120)
	actualDeliveryArrival := int64(190)
	actualDeliveryDeparture := int64(200)

	original := validShipmentForValidation()
	original.ID = pulid.MustNew("shp_")
	original.Version = 2
	original.Moves[0].ID = pulid.MustNew("sm_")
	original.Moves[0].ShipmentID = original.ID
	original.Moves[0].Version = 1
	original.Moves[0].Stops[0].ID = pulid.MustNew("stp_")
	original.Moves[0].Stops[0].ShipmentMoveID = original.Moves[0].ID
	original.Moves[0].Stops[0].Version = 1
	original.Moves[0].Stops[1].ID = pulid.MustNew("stp_")
	original.Moves[0].Stops[1].ShipmentMoveID = original.Moves[0].ID
	original.Moves[0].Stops[1].Version = 1

	entity := cloneShipment(original)
	entity.Status = shipment.StatusNew
	entity.Moves[0].Status = shipment.MoveStatusNew
	entity.Moves[0].Stops[0].Status = shipment.StopStatusNew
	entity.Moves[0].Stops[1].Status = shipment.StopStatusNew
	entity.Moves[0].Stops[0].ActualArrival = &actualPickupArrival
	entity.Moves[0].Stops[0].ActualDeparture = &actualPickupDeparture
	entity.Moves[0].Stops[1].ActualArrival = &actualDeliveryArrival
	entity.Moves[0].Stops[1].ActualDeparture = &actualDeliveryDeparture

	repo := mocks.NewMockShipmentRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*repositories.GetShipmentByIDRequest")).
		RunAndReturn(func(
			_ context.Context,
			_ *repositories.GetShipmentByIDRequest,
		) (*shipment.Shipment, error) {
			return cloneShipment(original), nil
		}).
		Once()
	repo.EXPECT().
		Update(mock.Anything, mock.AnythingOfType("*shipment.Shipment")).
		RunAndReturn(func(
			_ context.Context,
			entity *shipment.Shipment,
		) (*shipment.Shipment, error) {
			assert.Equal(t, shipment.StopStatusCompleted, entity.Moves[0].Stops[0].Status)
			assert.Equal(t, shipment.StopStatusCompleted, entity.Moves[0].Stops[1].Status)
			assert.Equal(t, shipment.MoveStatusCompleted, entity.Moves[0].Status)
			assert.Equal(t, shipment.StatusCompleted, entity.Status)
			require.NotNil(t, entity.ActualShipDate)
			require.NotNil(t, entity.ActualDeliveryDate)
			assert.Equal(t, actualPickupDeparture, *entity.ActualShipDate)
			assert.Equal(t, actualDeliveryArrival, *entity.ActualDeliveryDate)
			return entity, nil
		}).
		Once()
	controlRepo := mocks.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, repositories.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		}).
		Return(&tenant.ShipmentControl{
			CheckForDuplicateBOLs:       true,
			AutoDelayShipmentsThreshold: ptrInt16(30),
		}, nil).
		Once()
	repo.EXPECT().
		CheckForDuplicateBOLs(mock.Anything, mock.AnythingOfType("*repositories.DuplicateBOLCheckRequest")).
		Return([]*repositories.DuplicateBOLResult{}, nil).
		Once()
	formula := mocks.NewMockFormulaCalculator(t)
	formula.EXPECT().
		Calculate(mock.Anything, mock.AnythingOfType("*formulatemplatetypes.CalculateRequest")).
		Return(&formulatemplatetypes.CalculateResponse{Amount: decimal.Zero}, nil).
		Once()
	auditService := mocks.NewMockAuditService(t)
	auditService.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil).Once()
	realtime := mocks.NewMockRealtimeService(t)
	realtime.EXPECT().PublishResourceInvalidation(mock.Anything, mock.Anything).Return(nil).Once()

	svc := &service{
		l:            zap.NewNop(),
		repo:         repo,
		controlRepo:  controlRepo,
		validator:    NewTestValidator(t),
		auditService: auditService,
		commercial:   newTestCommercialCalculator(formula, mocks.NewMockAccessorialChargeRepository(t)),
		realtime:     realtime,
		eventService: noopShipmentEventService{},
		coordinator:  shipmentstate.NewCoordinatorWithClock(func() int64 { return 10 }),
	}

	userID := pulid.MustNew("usr_")
	updated, err := svc.Update(
		t.Context(),
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, shipment.StatusCompleted, updated.Status)
}

func TestServiceUpdate_AdvancesContinuityWhenMoveBecomesCompleted(t *testing.T) {
	t.Parallel()

	actualPickupArrival := int64(110)
	actualPickupDeparture := int64(120)
	actualDeliveryArrival := int64(190)
	actualDeliveryDeparture := int64(200)
	trailerID := pulid.MustNew("tr_")
	tractorID := pulid.MustNew("trc_")
	assignmentID := pulid.MustNew("asn_")

	original := validShipmentForValidation()
	original.ID = pulid.MustNew("shp_")
	original.Version = 2
	original.Moves[0].ID = pulid.MustNew("sm_")
	original.Moves[0].ShipmentID = original.ID
	original.Moves[0].Version = 1
	original.Moves[0].Stops[0].ID = pulid.MustNew("stp_")
	original.Moves[0].Stops[0].ShipmentMoveID = original.Moves[0].ID
	original.Moves[0].Stops[0].Version = 1
	original.Moves[0].Stops[1].ID = pulid.MustNew("stp_")
	original.Moves[0].Stops[1].ShipmentMoveID = original.Moves[0].ID
	original.Moves[0].Stops[1].Version = 1
	original.Moves[0].Assignment = &shipment.Assignment{
		ID:              assignmentID,
		OrganizationID:  original.OrganizationID,
		BusinessUnitID:  original.BusinessUnitID,
		ShipmentMoveID:  original.Moves[0].ID,
		PrimaryWorkerID: pulid.Must("wrk_"),
		TractorID:       &tractorID,
		TrailerID:       &trailerID,
		Status:          shipment.AssignmentStatusNew,
	}
	original.Moves[0].Status = shipment.MoveStatusAssigned

	entity := cloneShipment(original)
	entity.Moves[0].Assignment = nil
	entity.Moves[0].Stops[0].ActualArrival = &actualPickupArrival
	entity.Moves[0].Stops[0].ActualDeparture = &actualPickupDeparture
	entity.Moves[0].Stops[1].ActualArrival = &actualDeliveryArrival
	entity.Moves[0].Stops[1].ActualDeparture = &actualDeliveryDeparture

	repo := mocks.NewMockShipmentRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*repositories.GetShipmentByIDRequest")).
		Return(cloneShipment(original), nil).
		Once()
	repo.EXPECT().
		Update(mock.Anything, mock.AnythingOfType("*shipment.Shipment")).
		RunAndReturn(func(_ context.Context, entity *shipment.Shipment) (*shipment.Shipment, error) {
			require.Equal(t, shipment.MoveStatusCompleted, entity.Moves[0].Status)
			return entity, nil
		}).
		Once()
	repo.EXPECT().
		CheckForDuplicateBOLs(mock.Anything, mock.AnythingOfType("*repositories.DuplicateBOLCheckRequest")).
		Return([]*repositories.DuplicateBOLResult{}, nil).
		Once()

	controlRepo := mocks.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, repositories.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		}).
		Return(&tenant.ShipmentControl{
			CheckForDuplicateBOLs:       true,
			AutoDelayShipmentsThreshold: ptrInt16(30),
		}, nil).
		Once()

	continuityRepo := mocks.NewMockEquipmentContinuityRepository(t)
	continuityRepo.EXPECT().
		Advance(mock.Anything, mock.MatchedBy(func(req repositories.CreateEquipmentContinuityRequest) bool {
			return req.EquipmentType == equipmentcontinuity.EquipmentTypeTractor &&
				req.EquipmentID == tractorID &&
				req.CurrentLocationID == entity.Moves[0].Stops[1].LocationID &&
				req.SourceShipmentID == entity.ID &&
				req.SourceShipmentMoveID == entity.Moves[0].ID &&
				req.SourceAssignmentID == assignmentID
		})).
		Return(&equipmentcontinuity.EquipmentContinuity{ID: pulid.MustNew("eqc_")}, nil).
		Once()
	continuityRepo.EXPECT().
		Advance(mock.Anything, mock.MatchedBy(func(req repositories.CreateEquipmentContinuityRequest) bool {
			return req.EquipmentType == equipmentcontinuity.EquipmentTypeTrailer &&
				req.EquipmentID == trailerID &&
				req.CurrentLocationID == entity.Moves[0].Stops[1].LocationID &&
				req.SourceShipmentID == entity.ID &&
				req.SourceShipmentMoveID == entity.Moves[0].ID &&
				req.SourceAssignmentID == assignmentID
		})).
		Return(&equipmentcontinuity.EquipmentContinuity{ID: pulid.MustNew("eqc_")}, nil).
		Once()

	formula := mocks.NewMockFormulaCalculator(t)
	formula.EXPECT().
		Calculate(mock.Anything, mock.AnythingOfType("*formulatemplatetypes.CalculateRequest")).
		Return(&formulatemplatetypes.CalculateResponse{Amount: decimal.Zero}, nil).
		Once()

	auditService := mocks.NewMockAuditService(t)
	auditService.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil).Once()
	realtime := mocks.NewMockRealtimeService(t)
	realtime.EXPECT().PublishResourceInvalidation(mock.Anything, mock.Anything).Return(nil).Once()

	svc := &service{
		l:              zap.NewNop(),
		repo:           repo,
		controlRepo:    controlRepo,
		continuityRepo: continuityRepo,
		validator:      NewTestValidator(t),
		auditService:   auditService,
		commercial:     newTestCommercialCalculator(formula, mocks.NewMockAccessorialChargeRepository(t)),
		realtime:       realtime,
		eventService:   noopShipmentEventService{},
		coordinator:    shipmentstate.NewCoordinatorWithClock(func() int64 { return 10 }),
	}

	userID := pulid.MustNew("usr_")
	updated, err := svc.Update(
		t.Context(),
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, shipment.StatusCompleted, updated.Status)
}

func TestServiceUpdate_RejectsMoveTransitionToInTransitWhenEquipmentActiveElsewhere(t *testing.T) {
	t.Parallel()

	actualPickupArrival := int64(110)
	actualPickupDeparture := int64(120)
	trailerID := pulid.MustNew("tr_")
	tractorID := pulid.MustNew("trc_")

	original := validShipmentForValidation()
	original.ID = pulid.MustNew("shp_")
	original.Version = 2
	original.Moves[0].ID = pulid.MustNew("sm_")
	original.Moves[0].ShipmentID = original.ID
	original.Moves[0].Version = 1
	original.Moves[0].Stops[0].ID = pulid.MustNew("stp_")
	original.Moves[0].Stops[0].ShipmentMoveID = original.Moves[0].ID
	original.Moves[0].Stops[1].ID = pulid.MustNew("stp_")
	original.Moves[0].Stops[1].ShipmentMoveID = original.Moves[0].ID
	original.Moves[0].Assignment = &shipment.Assignment{
		ID:              pulid.MustNew("asn_"),
		OrganizationID:  original.OrganizationID,
		BusinessUnitID:  original.BusinessUnitID,
		ShipmentMoveID:  original.Moves[0].ID,
		PrimaryWorkerID: pulid.Must("wrk_"),
		TractorID:       &tractorID,
		TrailerID:       &trailerID,
		Status:          shipment.AssignmentStatusNew,
	}
	original.Moves[0].Status = shipment.MoveStatusAssigned

	entity := cloneShipment(original)
	entity.Moves[0].Stops[0].ActualArrival = &actualPickupArrival
	entity.Moves[0].Stops[0].ActualDeparture = &actualPickupDeparture

	repo := mocks.NewMockShipmentRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*repositories.GetShipmentByIDRequest")).
		Return(cloneShipment(original), nil).
		Once()

	controlRepo := mocks.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, repositories.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		}).
		Return(&tenant.ShipmentControl{
			CheckForDuplicateBOLs:       true,
			AutoDelayShipmentsThreshold: ptrInt16(30),
		}, nil).
		Once()

	formula := mocks.NewMockFormulaCalculator(t)
	formula.EXPECT().
		Calculate(mock.Anything, mock.AnythingOfType("*formulatemplatetypes.CalculateRequest")).
		Return(&formulatemplatetypes.CalculateResponse{Amount: decimal.Zero}, nil).
		Once()

	assignmentRepo := mocks.NewMockAssignmentRepository(t)
	assignmentRepo.EXPECT().
		FindInProgressByTractorID(
			mock.Anything,
			pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
			tractorID,
			original.Moves[0].ID,
		).
		Return(&shipment.Assignment{
			ID:             pulid.MustNew("asn_"),
			ShipmentMoveID: pulid.MustNew("sm_"),
			TractorID:      &tractorID,
		}, nil).
		Once()

	svc := &service{
		l:              zap.NewNop(),
		repo:           repo,
		assignmentRepo: assignmentRepo,
		controlRepo:    controlRepo,
		validator:      NewTestValidator(t),
		auditService:   mocks.NewMockAuditService(t),
		commercial:     newTestCommercialCalculator(formula, mocks.NewMockAccessorialChargeRepository(t)),
		realtime:       mocks.NewMockRealtimeService(t),
		eventService:   noopShipmentEventService{},
		coordinator:    shipmentstate.NewCoordinatorWithClock(func() int64 { return 10 }),
	}

	userID := pulid.MustNew("usr_")
	updated, err := svc.Update(
		t.Context(),
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.Nil(t, updated)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Equal(t, "Tractor is currently in progress on another move", err.Error())
}

func TestServiceUpdate_RejectsActualArrivalWhenTractorAndWorkerOverlapPersistedWindow(t *testing.T) {
	t.Parallel()

	actualPickupArrival := int64(1773837060)
	trailerID := pulid.MustNew("tr_")
	tractorID := pulid.MustNew("trc_")
	workerID := pulid.MustNew("wrk_")

	original := validShipmentForValidation()
	original.ID = pulid.MustNew("shp_")
	original.Version = 2
	original.Moves[0].ID = pulid.MustNew("sm_")
	original.Moves[0].ShipmentID = original.ID
	original.Moves[0].Version = 1
	original.Moves[0].Stops[0].ID = pulid.MustNew("stp_")
	original.Moves[0].Stops[0].ShipmentMoveID = original.Moves[0].ID
	original.Moves[0].Stops[1].ID = pulid.MustNew("stp_")
	original.Moves[0].Stops[1].ShipmentMoveID = original.Moves[0].ID
	original.Moves[0].Assignment = &shipment.Assignment{
		ID:              pulid.MustNew("asn_"),
		OrganizationID:  original.OrganizationID,
		BusinessUnitID:  original.BusinessUnitID,
		ShipmentMoveID:  original.Moves[0].ID,
		PrimaryWorkerID: &workerID,
		TractorID:       &tractorID,
		TrailerID:       &trailerID,
		Status:          shipment.AssignmentStatusNew,
	}
	original.Moves[0].Status = shipment.MoveStatusAssigned

	entity := cloneShipment(original)
	entity.Moves[0].Assignment = nil
	entity.Moves[0].Stops[0].ActualArrival = &actualPickupArrival

	repo := mocks.NewMockShipmentRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*repositories.GetShipmentByIDRequest")).
		Return(cloneShipment(original), nil).
		Once()

	controlRepo := mocks.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, repositories.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		}).
		Return(&tenant.ShipmentControl{
			CheckForDuplicateBOLs:       true,
			AutoDelayShipmentsThreshold: ptrInt16(30),
		}, nil).
		Once()

	formula := mocks.NewMockFormulaCalculator(t)
	formula.EXPECT().
		Calculate(mock.Anything, mock.AnythingOfType("*formulatemplatetypes.CalculateRequest")).
		Return(&formulatemplatetypes.CalculateResponse{Amount: decimal.Zero}, nil).
		Once()

	assignmentRepo := mocks.NewMockAssignmentRepository(t)
	assignmentRepo.EXPECT().
		FindNearestActualEventByTractorID(
			mock.Anything,
			mock.AnythingOfType("repositories.FindNearestActualTimelineEventRequest"),
			tractorID,
		).
		Return(nil, nil).
		Twice()
	assignmentRepo.EXPECT().
		FindNearestActualEventByPrimaryWorkerID(
			mock.Anything,
			mock.AnythingOfType("repositories.FindNearestActualTimelineEventRequest"),
			workerID,
		).
		Return(nil, nil).
		Twice()
	assignmentRepo.EXPECT().
		FindOverlappingActualWindowByTractorID(
			mock.Anything,
			mock.AnythingOfType("repositories.FindOverlappingActualTimelineWindowRequest"),
			tractorID,
		).
		Return(&repositories.ActualTimelineWindow{
			StartTimestamp: 1773835200,
			EndTimestamp:   1773837120,
			ShipmentID:     pulid.MustNew("shp_"),
			ShipmentMoveID: pulid.MustNew("sm_"),
		}, nil).
		Once()
	assignmentRepo.EXPECT().
		FindOverlappingActualWindowByPrimaryWorkerID(
			mock.Anything,
			mock.AnythingOfType("repositories.FindOverlappingActualTimelineWindowRequest"),
			workerID,
		).
		Return(&repositories.ActualTimelineWindow{
			StartTimestamp: 1773835200,
			EndTimestamp:   1773837120,
			ShipmentID:     pulid.MustNew("shp_"),
			ShipmentMoveID: pulid.MustNew("sm_"),
		}, nil).
		Once()

	svc := &service{
		l:            zap.NewNop(),
		repo:         repo,
		controlRepo:  controlRepo,
		validator:    NewTestValidatorWithAssignmentRepo(t, assignmentRepo),
		auditService: mocks.NewMockAuditService(t),
		commercial:   newTestCommercialCalculator(formula, mocks.NewMockAccessorialChargeRepository(t)),
		realtime:     mocks.NewMockRealtimeService(t),
		eventService: noopShipmentEventService{},
		coordinator:  shipmentstate.NewCoordinatorWithClock(func() int64 { return 10 }),
	}

	userID := pulid.MustNew("usr_")
	updated, err := svc.Update(
		t.Context(),
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.Nil(t, updated)
	require.Error(t, err)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assertErrorField(t, multiErr, "moves[0].stops[0].actualArrival")
	require.Len(t, multiErr.Errors, 1)
	assert.Contains(t, multiErr.Errors[0].Message, "tractor and primary worker")
}

func TestServiceUpdate_RejectsTwoMovesGoingInTransitWithSameTrailerInPayload(t *testing.T) {
	t.Parallel()

	actualPickupArrival := int64(110)
	actualPickupDeparture := int64(120)
	trailerID := pulid.MustNew("tr_")

	original := validShipmentForValidation()
	original.ID = pulid.MustNew("shp_")
	original.Version = 2
	original.Moves = []*shipment.ShipmentMove{validMove(), validMove()}
	for idx, move := range original.Moves {
		move.ID = pulid.MustNew("sm_")
		move.ShipmentID = original.ID
		move.Sequence = int64(idx)
		move.Version = 1
		move.Stops[0].ID = pulid.MustNew("stp_")
		move.Stops[0].ShipmentMoveID = move.ID
		move.Stops[1].ID = pulid.MustNew("stp_")
		move.Stops[1].ShipmentMoveID = move.ID
		tractorID := pulid.MustNew("trc_")
		move.Assignment = &shipment.Assignment{
			ID:              pulid.MustNew("asn_"),
			OrganizationID:  original.OrganizationID,
			BusinessUnitID:  original.BusinessUnitID,
			ShipmentMoveID:  move.ID,
			PrimaryWorkerID: pulid.Must("wrk_"),
			TractorID:       &tractorID,
			TrailerID:       &trailerID,
			Status:          shipment.AssignmentStatusNew,
		}
		move.Status = shipment.MoveStatusAssigned
	}

	entity := cloneShipment(original)
	for _, move := range entity.Moves {
		move.Assignment = nil
		move.Stops[0].ActualArrival = &actualPickupArrival
		move.Stops[0].ActualDeparture = &actualPickupDeparture
	}

	repo := mocks.NewMockShipmentRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*repositories.GetShipmentByIDRequest")).
		Return(cloneShipment(original), nil).
		Once()

	controlRepo := mocks.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, repositories.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		}).
		Return(&tenant.ShipmentControl{
			CheckForDuplicateBOLs:       true,
			AutoDelayShipmentsThreshold: ptrInt16(30),
		}, nil).
		Once()

	formula := mocks.NewMockFormulaCalculator(t)
	formula.EXPECT().
		Calculate(mock.Anything, mock.AnythingOfType("*formulatemplatetypes.CalculateRequest")).
		Return(&formulatemplatetypes.CalculateResponse{Amount: decimal.Zero}, nil).
		Once()

	assignmentRepo := mocks.NewMockAssignmentRepository(t)
	firstMove := original.Moves[0]
	firstTractorID := *firstMove.Assignment.TractorID
	assignmentRepo.EXPECT().
		FindInProgressByTractorID(
			mock.Anything,
			pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
			firstTractorID,
			firstMove.ID,
		).
		Return(nil, nil).
		Once()
	assignmentRepo.EXPECT().
		FindInProgressByTrailerID(
			mock.Anything,
			pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
			trailerID,
			firstMove.ID,
		).
		Return(nil, nil).
		Once()

	svc := &service{
		l:              zap.NewNop(),
		repo:           repo,
		assignmentRepo: assignmentRepo,
		controlRepo:    controlRepo,
		validator:      NewTestValidator(t),
		auditService:   mocks.NewMockAuditService(t),
		commercial:     newTestCommercialCalculator(formula, mocks.NewMockAccessorialChargeRepository(t)),
		realtime:       mocks.NewMockRealtimeService(t),
		eventService:   noopShipmentEventService{},
		coordinator:    shipmentstate.NewCoordinatorWithClock(func() int64 { return 10 }),
	}

	userID := pulid.MustNew("usr_")
	updated, err := svc.Update(
		t.Context(),
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.Nil(t, updated)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Equal(t, "Trailer is currently in progress on another move", err.Error())
}

func TestServiceUpdate_PreservesAssignedStateWhenPayloadSendsNew(t *testing.T) {
	t.Parallel()

	original := validShipmentForValidation()
	original.ID = pulid.MustNew("shp_")
	original.Version = 1
	original.Moves[0].ID = pulid.MustNew("sm_")
	original.Moves[0].Version = 1
	original.Moves[0].Stops[0].ID = pulid.MustNew("stp_")
	original.Moves[0].Stops[0].ShipmentMoveID = original.Moves[0].ID
	original.Moves[0].Stops[1].ID = pulid.MustNew("stp_")
	original.Moves[0].Stops[1].ShipmentMoveID = original.Moves[0].ID
	original.Moves[0].Assignment = &shipment.Assignment{ID: pulid.MustNew("asn_")}
	original.Moves[0].Status = shipment.MoveStatusAssigned
	original.Status = shipment.StatusAssigned

	entity := cloneShipment(original)
	entity.Status = shipment.StatusNew
	entity.Moves[0].Status = shipment.MoveStatusNew
	entity.Moves[0].Assignment = nil
	entity.BOL = "BOL-UPDATED"

	repo := mocks.NewMockShipmentRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*repositories.GetShipmentByIDRequest")).
		RunAndReturn(func(
			_ context.Context,
			_ *repositories.GetShipmentByIDRequest,
		) (*shipment.Shipment, error) {
			return cloneShipment(original), nil
		}).
		Once()
	repo.EXPECT().
		Update(mock.Anything, mock.AnythingOfType("*shipment.Shipment")).
		RunAndReturn(func(
			_ context.Context,
			entity *shipment.Shipment,
		) (*shipment.Shipment, error) {
			require.NotNil(t, entity.Moves[0].Assignment)
			assert.Equal(t, original.Moves[0].Assignment.ID, entity.Moves[0].Assignment.ID)
			assert.Equal(t, shipment.MoveStatusAssigned, entity.Moves[0].Status)
			assert.Equal(t, shipment.StatusAssigned, entity.Status)
			return entity, nil
		}).
		Once()
	controlRepo := mocks.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, repositories.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		}).
		Return(&tenant.ShipmentControl{
			CheckForDuplicateBOLs:       true,
			AutoDelayShipmentsThreshold: ptrInt16(30),
		}, nil).
		Once()
	repo.EXPECT().
		CheckForDuplicateBOLs(mock.Anything, mock.AnythingOfType("*repositories.DuplicateBOLCheckRequest")).
		Return([]*repositories.DuplicateBOLResult{}, nil).
		Once()
	formula := mocks.NewMockFormulaCalculator(t)
	formula.EXPECT().
		Calculate(mock.Anything, mock.AnythingOfType("*formulatemplatetypes.CalculateRequest")).
		Return(&formulatemplatetypes.CalculateResponse{Amount: decimal.Zero}, nil).
		Once()
	auditService := mocks.NewMockAuditService(t)
	auditService.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil).Once()
	realtime := mocks.NewMockRealtimeService(t)
	realtime.EXPECT().PublishResourceInvalidation(mock.Anything, mock.Anything).Return(nil).Once()

	svc := &service{
		l:            zap.NewNop(),
		repo:         repo,
		controlRepo:  controlRepo,
		validator:    NewTestValidator(t),
		auditService: auditService,
		commercial:   newTestCommercialCalculator(formula, mocks.NewMockAccessorialChargeRepository(t)),
		realtime:     realtime,
		eventService: noopShipmentEventService{},
		coordinator:  shipmentstate.NewCoordinatorWithClock(func() int64 { return 10 }),
	}

	userID := pulid.MustNew("usr_")
	updated, err := svc.Update(
		t.Context(),
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, shipment.StatusAssigned, updated.Status)
}

func TestServiceUpdate_RejectsDirectInvoiceFromCompleted(t *testing.T) {
	t.Parallel()

	original := validShipmentForValidation()
	original.ID = pulid.MustNew("shp_")
	original.Version = 1
	original.Status = shipment.StatusCompleted
	original.Moves[0].Status = shipment.MoveStatusCompleted
	original.Moves[0].Stops[0].Status = shipment.StopStatusCompleted
	original.Moves[0].Stops[1].Status = shipment.StopStatusCompleted

	entity := cloneShipment(original)
	entity.Status = shipment.StatusInvoiced

	repo := mocks.NewMockShipmentRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*repositories.GetShipmentByIDRequest")).
		RunAndReturn(func(
			_ context.Context,
			_ *repositories.GetShipmentByIDRequest,
		) (*shipment.Shipment, error) {
			return cloneShipment(original), nil
		}).
		Once()
	controlRepo := mocks.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, repositories.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		}).
		Return(&tenant.ShipmentControl{}, nil).
		Once()
	svc := &service{
		l:            zap.NewNop(),
		repo:         repo,
		controlRepo:  controlRepo,
		validator:    NewTestValidator(t),
		auditService: mocks.NewMockAuditService(t),
		eventService: noopShipmentEventService{},
		coordinator:  newStateCoordinator(),
	}
	userID := pulid.MustNew("usr_")
	updated, err := svc.Update(
		t.Context(),
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.Nil(t, updated)
	require.Error(t, err)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assertErrorField(t, multiErr, "status")
}

func cloneShipment(source *shipment.Shipment) *shipment.Shipment {
	if source == nil {
		return nil
	}

	clone := *source
	clone.Moves = make([]*shipment.ShipmentMove, 0, len(source.Moves))

	for _, move := range source.Moves {
		if move == nil {
			clone.Moves = append(clone.Moves, nil)
			continue
		}

		moveClone := *move
		moveClone.Stops = make([]*shipment.Stop, 0, len(move.Stops))
		if move.Assignment != nil {
			assignmentClone := *move.Assignment
			moveClone.Assignment = &assignmentClone
		}

		for _, stop := range move.Stops {
			if stop == nil {
				moveClone.Stops = append(moveClone.Stops, nil)
				continue
			}

			stopClone := *stop
			moveClone.Stops = append(moveClone.Stops, &stopClone)
		}

		clone.Moves = append(clone.Moves, &moveClone)
	}

	return &clone
}

func completedShipmentForCoordinator() *shipment.Shipment {
	entity := validShipmentForValidation()
	entity.ID = pulid.MustNew("shp_")
	entity.Version = 1
	entity.Status = shipment.StatusCompleted

	move := entity.Moves[0]
	move.ID = pulid.MustNew("sm_")
	move.ShipmentID = entity.ID
	move.Status = shipment.MoveStatusCompleted
	move.Version = 1

	for stopIndex, stop := range move.Stops {
		stop.ID = pulid.MustNew("stp_")
		stop.ShipmentMoveID = move.ID
		stop.Status = shipment.StopStatusCompleted
		stop.Version = 1
		arrival := int64(100 + stopIndex*100)
		departure := arrival + 10
		stop.ActualArrival = &arrival
		stop.ActualDeparture = &departure
	}

	entity.ActualShipDate = move.Stops[0].ActualDeparture
	entity.ActualDeliveryDate = move.Stops[1].ActualArrival
	return entity
}

func shipmentWithOverdueStopForRefresh(status shipment.Status) *shipment.Shipment {
	entity := validShipmentForValidation()
	entity.Status = status
	entity.Moves[0].Status = shipment.MoveStatusInTransit
	entity.Moves[0].Stops[0].Status = shipment.StopStatusInTransit
	entity.Moves[0].Stops[0].ScheduledWindowEnd = ptrInt64(100)

	if status != shipment.StatusPartiallyCompleted {
		return entity
	}

	completedMove := validMove()
	completedMove.Status = shipment.MoveStatusCompleted
	completedMove.Sequence = 0
	for _, stop := range completedMove.Stops {
		stop.Status = shipment.StopStatusCompleted
		stop.ActualArrival = ptrInt64(10)
		stop.ActualDeparture = ptrInt64(20)
	}

	activeMove := entity.Moves[0]
	activeMove.Sequence = 1
	entity.Moves = []*shipment.ShipmentMove{completedMove, activeMove}
	return entity
}

func completedStopForCoordinator(
	stopType shipment.StopType,
	sequence int64,
	actualArrival int64,
	actualDeparture int64,
) *shipment.Stop {
	stop := makeStopForValidation(stopType, sequence, 1, 2)
	stop.Status = shipment.StopStatusCompleted
	stop.ActualArrival = &actualArrival
	stop.ActualDeparture = &actualDeparture
	return stop
}

//go:fix inline
//go:fix inline
func ptrInt64(value int64) *int64 {
	return new(value)
}
