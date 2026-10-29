package shipmentflows

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentcontinuity"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmentcontinuityports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/shipmentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentservicetest"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/assignmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	internaltestutil "github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks/accessorialchargeportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/assignmentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/equipmentcontinuityportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/realtimeportsmock"
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

func TestStateCoordinator_PrepareForCreate_DerivesStatusesAndTimestamps(t *testing.T) {
	t.Parallel()

	actualPickupArrival := int64(110)
	actualPickupDeparture := int64(120)
	actualDeliveryArrival := int64(190)
	actualDeliveryDeparture := int64(200)

	entity := shipmentservicetest.ValidShipment()
	entity.Moves[0].Stops[0].ActualArrival = &actualPickupArrival
	entity.Moves[0].Stops[0].ActualDeparture = &actualPickupDeparture
	entity.Moves[0].Stops[1].ActualArrival = &actualDeliveryArrival
	entity.Moves[0].Stops[1].ActualDeparture = &actualDeliveryDeparture

	coordinator := shipmentstate.NewCoordinatorWithClock(func() int64 { return 10 })

	multiErr := coordinator.PrepareForCreateWithDelayThreshold(entity, 30)

	require.Nil(t, multiErr)
	assert.Equal(t, shipmentvocab.StopStatusCompleted, entity.Moves[0].Stops[0].Status)
	assert.Equal(t, shipmentvocab.StopStatusCompleted, entity.Moves[0].Stops[1].Status)
	assert.Equal(t, shipmentvocab.MoveStatusCompleted, entity.Moves[0].Status)
	assert.Equal(t, shipmentvocab.StatusCompleted, entity.Status)
	require.NotNil(t, entity.ActualShipDate)
	require.NotNil(t, entity.ActualDeliveryDate)
	assert.Equal(t, actualPickupDeparture, *entity.ActualShipDate)
	assert.Equal(t, actualDeliveryArrival, *entity.ActualDeliveryDate)
}

func TestStateCoordinator_PrepareForCreate_PreservesAssignedStatusesWithoutOperationalSignal(
	t *testing.T,
) {
	t.Parallel()

	entity := shipmentservicetest.ValidShipment()
	entity.Moves[0].Assignment = &shipment.Assignment{ID: pulid.MustNew("asn_")}

	coordinator := shipmentstate.NewCoordinatorWithClock(func() int64 { return 10 })

	multiErr := coordinator.PrepareForCreateWithDelayThreshold(entity, 30)

	require.Nil(t, multiErr)
	assert.Equal(t, shipmentvocab.MoveStatusAssigned, entity.Moves[0].Status)
	assert.Equal(t, shipmentvocab.StatusAssigned, entity.Status)
}

func TestStateCoordinator_PrepareForCreate_DerivesPartiallyAssignedShipment(t *testing.T) {
	t.Parallel()

	entity := shipmentservicetest.ValidShipment()
	entity.Moves = []*shipment.ShipmentMove{
		shipmentservicetest.ValidMove(),
		shipmentservicetest.ValidMove(),
	}
	entity.Moves[0].Assignment = &shipment.Assignment{ID: pulid.MustNew("asn_")}

	coordinator := shipmentstate.NewCoordinatorWithClock(func() int64 { return 10 })

	multiErr := coordinator.PrepareForCreateWithDelayThreshold(entity, 30)

	require.Nil(t, multiErr)
	assert.Equal(t, shipmentvocab.MoveStatusAssigned, entity.Moves[0].Status)
	assert.Equal(t, shipmentvocab.MoveStatusNew, entity.Moves[1].Status)
	assert.Equal(t, shipmentvocab.StatusPartiallyAssigned, entity.Status)
}

func TestStateCoordinator_PrepareForCreate_DerivesDelayedShipment(t *testing.T) {
	t.Parallel()

	entity := shipmentservicetest.ValidShipment()
	entity.Moves[0].Stops[0].ActualArrival = new(int64(150))
	entity.Moves[0].Stops[0].ScheduledWindowEnd = new(int64(100))

	coordinator := shipmentstate.NewCoordinatorWithClock(func() int64 { return 200 })

	multiErr := coordinator.PrepareForCreateWithDelayThreshold(entity, 1)

	require.Nil(t, multiErr)
	assert.Equal(t, shipmentvocab.StopStatusInTransit, entity.Moves[0].Stops[0].Status)
	assert.Equal(t, shipmentvocab.MoveStatusInTransit, entity.Moves[0].Status)
	assert.Equal(t, shipmentvocab.StatusDelayed, entity.Status)
}

func TestStateCoordinator_PrepareForUpdate_AllowsCompletedStopTimestampCorrection(t *testing.T) {
	t.Parallel()

	original := completedShipmentForCoordinator()
	entity := shipmentservicetest.CloneShipment(original)
	correctedArrival := int64(130)
	correctedDeparture := int64(140)
	entity.Moves[0].Stops[0].ActualArrival = &correctedArrival
	entity.Moves[0].Stops[0].ActualDeparture = &correctedDeparture

	coordinator := shipmentstate.NewCoordinatorWithClock(func() int64 { return 1000 })

	multiErr := coordinator.PrepareForUpdateWithDelayThreshold(original, entity, 30)

	require.Nil(t, multiErr)
	assert.Equal(t, shipmentvocab.StopStatusCompleted, entity.Moves[0].Stops[0].Status)
	assert.Equal(t, shipmentvocab.MoveStatusCompleted, entity.Moves[0].Status)
	assert.Equal(t, shipmentvocab.StatusCompleted, entity.Status)
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
			entity := shipmentservicetest.CloneShipment(original)
			tt.mutate(entity.Moves[0].Stops[0])

			coordinator := shipmentstate.NewCoordinatorWithClock(func() int64 { return 1000 })

			multiErr := coordinator.PrepareForUpdateWithDelayThreshold(original, entity, 30)

			require.NotNil(t, multiErr)
			shipmentservicetest.AssertErrorField(t, multiErr, tt.expectedField)
			assert.Equal(t, shipmentvocab.StopStatusCompleted, entity.Moves[0].Stops[0].Status)
			assert.Equal(t, shipmentvocab.MoveStatusCompleted, entity.Moves[0].Status)
		})
	}
}

func TestStateCoordinator_PrepareForUpdate_RejectsCompletedMoveReopenByOmittedStops(t *testing.T) {
	t.Parallel()

	original := completedShipmentForCoordinator()
	entity := shipmentservicetest.CloneShipment(original)
	entity.Moves[0].Stops = []*shipment.Stop{}

	coordinator := shipmentstate.NewCoordinatorWithClock(func() int64 { return 1000 })

	multiErr := coordinator.PrepareForUpdateWithDelayThreshold(original, entity, 30)

	require.NotNil(t, multiErr)
	shipmentservicetest.AssertErrorField(t, multiErr, "moves[0].status")
	assert.Equal(t, shipmentvocab.MoveStatusCompleted, entity.Moves[0].Status)
	assert.Equal(t, shipmentvocab.StatusCompleted, entity.Status)
}

func TestStateCoordinator_RefreshShipmentState_PreservesBillingAndTerminalStatuses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		status         shipmentvocab.Status
		expectedStatus shipmentvocab.Status
	}{
		{
			name:           "ready to invoice",
			status:         shipmentvocab.StatusReadyToInvoice,
			expectedStatus: shipmentvocab.StatusReadyToInvoice,
		},
		{
			name:           "invoiced",
			status:         shipmentvocab.StatusInvoiced,
			expectedStatus: shipmentvocab.StatusInvoiced,
		},
		{
			name:           "canceled",
			status:         shipmentvocab.StatusCanceled,
			expectedStatus: shipmentvocab.StatusCanceled,
		},
		{
			name:           "completed",
			status:         shipmentvocab.StatusCompleted,
			expectedStatus: shipmentvocab.StatusCompleted,
		},
		{
			name:           "active in transit",
			status:         shipmentvocab.StatusInTransit,
			expectedStatus: shipmentvocab.StatusDelayed,
		},
		{
			name:           "partially completed active",
			status:         shipmentvocab.StatusPartiallyCompleted,
			expectedStatus: shipmentvocab.StatusDelayed,
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
	entity := shipmentservicetest.ValidShipment()
	entity.Moves = []*shipment.ShipmentMove{
		{
			Status:   shipmentvocab.MoveStatusCompleted,
			Sequence: 10,
			Stops: []*shipment.Stop{
				completedStopForCoordinator(shipmentvocab.StopTypePickup, 0, 300, 310),
				completedStopForCoordinator(
					shipmentvocab.StopTypeDelivery,
					1,
					actualDeliveryDate,
					410,
				),
			},
		},
		{
			Status:   shipmentvocab.MoveStatusCompleted,
			Sequence: 0,
			Stops: []*shipment.Stop{
				completedStopForCoordinator(shipmentvocab.StopTypeDelivery, 1, 200, 210),
				completedStopForCoordinator(shipmentvocab.StopTypePickup, 0, 100, actualShipDate),
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

	original := shipmentservicetest.ValidShipment()
	original.ID = pulid.MustNew("shp_")
	original.Version = 1

	entity := shipmentservicetest.ValidShipment()
	entity.ID = original.ID
	entity.Version = original.Version
	entity.Status = shipmentvocab.StatusReadyToInvoice

	repo := shipmentportsmock.NewMockShipmentRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*shipmentports.GetShipmentByIDRequest")).
		RunAndReturn(func(
			_ context.Context,
			req *shipmentports.GetShipmentByIDRequest,
		) (*shipment.Shipment, error) {
			assert.True(t, req.ExpandShipmentDetails)
			return shipmentservicetest.CloneShipment(original), nil
		}).
		Once()
	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, tenantports.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		}).
		Return(&shipmentcontrol.ShipmentControl{}, nil).
		Once()
	userID := pulid.MustNew("usr_")
	svc := assembled(&rig{
		l:            zap.NewNop(),
		repo:         repo,
		controlRepo:  controlRepo,
		validator:    newServiceValidator(t),
		auditService: auditportsmock.NewMockAuditService(t),
		eventService: noopShipmentEventService{},
		coordinator:  shipmentstate.NewCoordinatorWithClock(func() int64 { return 10 }),
	})

	updated, err := svc.writer.Update(
		t.Context(),
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.Nil(t, updated)
	require.Error(t, err)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	shipmentservicetest.AssertErrorField(t, multiErr, "status")
}

func TestServiceUpdate_DerivesAuthoritativeStatusesBeforePersist(t *testing.T) {
	t.Parallel()

	actualPickupArrival := int64(110)
	actualPickupDeparture := int64(120)
	actualDeliveryArrival := int64(190)
	actualDeliveryDeparture := int64(200)

	original := shipmentservicetest.ValidShipment()
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

	entity := shipmentservicetest.CloneShipment(original)
	entity.Status = shipmentvocab.StatusNew
	entity.Moves[0].Status = shipmentvocab.MoveStatusNew
	entity.Moves[0].Stops[0].Status = shipmentvocab.StopStatusNew
	entity.Moves[0].Stops[1].Status = shipmentvocab.StopStatusNew
	entity.Moves[0].Stops[0].ActualArrival = &actualPickupArrival
	entity.Moves[0].Stops[0].ActualDeparture = &actualPickupDeparture
	entity.Moves[0].Stops[1].ActualArrival = &actualDeliveryArrival
	entity.Moves[0].Stops[1].ActualDeparture = &actualDeliveryDeparture

	repo := shipmentportsmock.NewMockShipmentRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*shipmentports.GetShipmentByIDRequest")).
		RunAndReturn(func(
			_ context.Context,
			_ *shipmentports.GetShipmentByIDRequest,
		) (*shipment.Shipment, error) {
			return shipmentservicetest.CloneShipment(original), nil
		}).
		Once()
	repo.EXPECT().
		Update(mock.Anything, mock.AnythingOfType("*shipment.Shipment")).
		RunAndReturn(func(
			_ context.Context,
			entity *shipment.Shipment,
		) (*shipment.Shipment, error) {
			assert.Equal(t, shipmentvocab.StopStatusCompleted, entity.Moves[0].Stops[0].Status)
			assert.Equal(t, shipmentvocab.StopStatusCompleted, entity.Moves[0].Stops[1].Status)
			assert.Equal(t, shipmentvocab.MoveStatusCompleted, entity.Moves[0].Status)
			assert.Equal(t, shipmentvocab.StatusCompleted, entity.Status)
			require.NotNil(t, entity.ActualShipDate)
			require.NotNil(t, entity.ActualDeliveryDate)
			assert.Equal(t, actualPickupDeparture, *entity.ActualShipDate)
			assert.Equal(t, actualDeliveryArrival, *entity.ActualDeliveryDate)
			return entity, nil
		}).
		Once()
	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, tenantports.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		}).
		Return(&shipmentcontrol.ShipmentControl{
			CheckForDuplicateBOLs:       true,
			AutoDelayShipmentsThreshold: new(int16(30)),
		}, nil).
		Once()
	repo.EXPECT().
		CheckForDuplicateBOLs(mock.Anything, mock.AnythingOfType("*shipmentports.DuplicateBOLCheckRequest")).
		Return([]*shipmentports.DuplicateBOLResult{}, nil).
		Once()
	freight := decimal.Zero
	auditService := auditportsmock.NewMockAuditService(t)
	auditService.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil).Once()
	realtime := realtimeportsmock.NewMockRealtimeService(t)
	realtime.EXPECT().PublishResourceInvalidation(mock.Anything, mock.Anything).Return(nil).Once()

	svc := assembled(&rig{
		l:            zap.NewNop(),
		repo:         repo,
		controlRepo:  controlRepo,
		validator:    newServiceValidator(t),
		auditService: auditService,
		commercial: newTestCommercialCalculator(
			t,
			freight,
			accessorialchargeportsmock.NewMockAccessorialChargeRepository(t),
		),
		invalidator:  newTestInvalidator(realtime),
		eventService: noopShipmentEventService{},
		coordinator:  shipmentstate.NewCoordinatorWithClock(func() int64 { return 10 }),
	})

	userID := pulid.MustNew("usr_")
	updated, err := svc.writer.Update(
		t.Context(),
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, shipmentvocab.StatusCompleted, updated.Status)
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

	original := shipmentservicetest.ValidShipment()
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
		Status:          shipmentvocab.AssignmentStatusNew,
	}
	original.Moves[0].Status = shipmentvocab.MoveStatusAssigned

	entity := shipmentservicetest.CloneShipment(original)
	entity.Moves[0].Assignment = nil
	entity.Moves[0].Stops[0].ActualArrival = &actualPickupArrival
	entity.Moves[0].Stops[0].ActualDeparture = &actualPickupDeparture
	entity.Moves[0].Stops[1].ActualArrival = &actualDeliveryArrival
	entity.Moves[0].Stops[1].ActualDeparture = &actualDeliveryDeparture

	repo := shipmentportsmock.NewMockShipmentRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*shipmentports.GetShipmentByIDRequest")).
		Return(shipmentservicetest.CloneShipment(original), nil).
		Once()
	repo.EXPECT().
		Update(mock.Anything, mock.AnythingOfType("*shipment.Shipment")).
		RunAndReturn(func(_ context.Context, entity *shipment.Shipment) (*shipment.Shipment, error) {
			require.Equal(t, shipmentvocab.MoveStatusCompleted, entity.Moves[0].Status)
			return entity, nil
		}).
		Once()
	repo.EXPECT().
		CheckForDuplicateBOLs(mock.Anything, mock.AnythingOfType("*shipmentports.DuplicateBOLCheckRequest")).
		Return([]*shipmentports.DuplicateBOLResult{}, nil).
		Once()

	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, tenantports.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		}).
		Return(&shipmentcontrol.ShipmentControl{
			CheckForDuplicateBOLs:       true,
			AutoDelayShipmentsThreshold: new(int16(30)),
		}, nil).
		Once()

	continuityRepo := equipmentcontinuityportsmock.NewMockEquipmentContinuityRepository(t)
	continuityRepo.EXPECT().
		Advance(mock.Anything, mock.MatchedBy(func(req equipmentcontinuityports.CreateEquipmentContinuityRequest) bool {
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
		Advance(mock.Anything, mock.MatchedBy(func(req equipmentcontinuityports.CreateEquipmentContinuityRequest) bool {
			return req.EquipmentType == equipmentcontinuity.EquipmentTypeTrailer &&
				req.EquipmentID == trailerID &&
				req.CurrentLocationID == entity.Moves[0].Stops[1].LocationID &&
				req.SourceShipmentID == entity.ID &&
				req.SourceShipmentMoveID == entity.Moves[0].ID &&
				req.SourceAssignmentID == assignmentID
		})).
		Return(&equipmentcontinuity.EquipmentContinuity{ID: pulid.MustNew("eqc_")}, nil).
		Once()

	freight := decimal.Zero

	auditService := auditportsmock.NewMockAuditService(t)
	auditService.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil).Once()
	realtime := realtimeportsmock.NewMockRealtimeService(t)
	realtime.EXPECT().PublishResourceInvalidation(mock.Anything, mock.Anything).Return(nil).Once()

	svc := assembled(&rig{
		l:              zap.NewNop(),
		repo:           repo,
		controlRepo:    controlRepo,
		continuityRepo: continuityRepo,
		validator:      newServiceValidator(t),
		auditService:   auditService,
		commercial: newTestCommercialCalculator(
			t,
			freight,
			accessorialchargeportsmock.NewMockAccessorialChargeRepository(t),
		),
		invalidator:  newTestInvalidator(realtime),
		eventService: noopShipmentEventService{},
		coordinator:  shipmentstate.NewCoordinatorWithClock(func() int64 { return 10 }),
	})

	userID := pulid.MustNew("usr_")
	updated, err := svc.writer.Update(
		t.Context(),
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, shipmentvocab.StatusCompleted, updated.Status)
}

func TestServiceUpdate_RejectsMoveTransitionToInTransitWhenEquipmentActiveElsewhere(t *testing.T) {
	t.Parallel()

	actualPickupArrival := int64(110)
	actualPickupDeparture := int64(120)
	trailerID := pulid.MustNew("tr_")
	tractorID := pulid.MustNew("trc_")

	original := shipmentservicetest.ValidShipment()
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
		Status:          shipmentvocab.AssignmentStatusNew,
	}
	original.Moves[0].Status = shipmentvocab.MoveStatusAssigned

	entity := shipmentservicetest.CloneShipment(original)
	entity.Moves[0].Stops[0].ActualArrival = &actualPickupArrival
	entity.Moves[0].Stops[0].ActualDeparture = &actualPickupDeparture

	repo := shipmentportsmock.NewMockShipmentRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*shipmentports.GetShipmentByIDRequest")).
		Return(shipmentservicetest.CloneShipment(original), nil).
		Once()

	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, tenantports.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		}).
		Return(&shipmentcontrol.ShipmentControl{
			CheckForDuplicateBOLs:       true,
			AutoDelayShipmentsThreshold: new(int16(30)),
		}, nil).
		Once()

	freight := decimal.Zero

	assignmentRepo := assignmentportsmock.NewMockAssignmentRepository(t)
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

	svc := assembled(&rig{
		l:              zap.NewNop(),
		repo:           repo,
		assignmentRepo: assignmentRepo,
		controlRepo:    controlRepo,
		validator:      newServiceValidator(t),
		auditService:   auditportsmock.NewMockAuditService(t),
		commercial: newTestCommercialCalculator(
			t,
			freight,
			accessorialchargeportsmock.NewMockAccessorialChargeRepository(t),
		),
		invalidator:  newTestInvalidator(realtimeportsmock.NewMockRealtimeService(t)),
		eventService: noopShipmentEventService{},
		coordinator:  shipmentstate.NewCoordinatorWithClock(func() int64 { return 10 }),
	})

	userID := pulid.MustNew("usr_")
	updated, err := svc.writer.Update(
		t.Context(),
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.Nil(t, updated)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Equal(t, "Tractor is currently in progress on another move", err.Error())
}

func TestServiceUpdate_RejectsActualArrivalWhenTractorAndWorkerOverlapPersistedWindow(
	t *testing.T,
) {
	t.Parallel()

	actualPickupArrival := int64(1773837060)
	trailerID := pulid.MustNew("tr_")
	tractorID := pulid.MustNew("trc_")
	workerID := pulid.MustNew("wrk_")

	original := shipmentservicetest.ValidShipment()
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
		Status:          shipmentvocab.AssignmentStatusNew,
	}
	original.Moves[0].Status = shipmentvocab.MoveStatusAssigned

	entity := shipmentservicetest.CloneShipment(original)
	entity.Moves[0].Assignment = nil
	entity.Moves[0].Stops[0].ActualArrival = &actualPickupArrival

	repo := shipmentportsmock.NewMockShipmentRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*shipmentports.GetShipmentByIDRequest")).
		Return(shipmentservicetest.CloneShipment(original), nil).
		Once()

	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, tenantports.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		}).
		Return(&shipmentcontrol.ShipmentControl{
			CheckForDuplicateBOLs:       true,
			AutoDelayShipmentsThreshold: new(int16(30)),
		}, nil).
		Once()

	freight := decimal.Zero

	assignmentRepo := assignmentportsmock.NewMockAssignmentRepository(t)
	assignmentRepo.EXPECT().
		FindNearestActualEventByTractorID(
			mock.Anything,
			mock.AnythingOfType("assignmentports.FindNearestActualTimelineEventRequest"),
			tractorID,
		).
		Return(nil, nil).
		Twice()
	assignmentRepo.EXPECT().
		FindNearestActualEventByPrimaryWorkerID(
			mock.Anything,
			mock.AnythingOfType("assignmentports.FindNearestActualTimelineEventRequest"),
			workerID,
		).
		Return(nil, nil).
		Twice()
	assignmentRepo.EXPECT().
		FindOverlappingActualWindowByTractorID(
			mock.Anything,
			mock.AnythingOfType("assignmentports.FindOverlappingActualTimelineWindowRequest"),
			tractorID,
		).
		Return(&assignmentports.ActualTimelineWindow{
			StartTimestamp: 1773835200,
			EndTimestamp:   1773837120,
			ShipmentID:     pulid.MustNew("shp_"),
			ShipmentMoveID: pulid.MustNew("sm_"),
		}, nil).
		Once()
	assignmentRepo.EXPECT().
		FindOverlappingActualWindowByPrimaryWorkerID(
			mock.Anything,
			mock.AnythingOfType("assignmentports.FindOverlappingActualTimelineWindowRequest"),
			workerID,
		).
		Return(&assignmentports.ActualTimelineWindow{
			StartTimestamp: 1773835200,
			EndTimestamp:   1773837120,
			ShipmentID:     pulid.MustNew("shp_"),
			ShipmentMoveID: pulid.MustNew("sm_"),
		}, nil).
		Once()

	svc := assembled(&rig{
		l:            zap.NewNop(),
		repo:         repo,
		controlRepo:  controlRepo,
		validator:    newServiceValidatorWithAssignmentRepo(t, assignmentRepo),
		auditService: auditportsmock.NewMockAuditService(t),
		commercial: newTestCommercialCalculator(
			t,
			freight,
			accessorialchargeportsmock.NewMockAccessorialChargeRepository(t),
		),
		invalidator:  newTestInvalidator(realtimeportsmock.NewMockRealtimeService(t)),
		eventService: noopShipmentEventService{},
		coordinator:  shipmentstate.NewCoordinatorWithClock(func() int64 { return 10 }),
	})

	userID := pulid.MustNew("usr_")
	updated, err := svc.writer.Update(
		t.Context(),
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.Nil(t, updated)
	require.Error(t, err)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	shipmentservicetest.AssertErrorField(t, multiErr, "moves[0].stops[0].actualArrival")
	require.Len(t, multiErr.Errors, 1)
	assert.Contains(t, multiErr.Errors[0].Error(), "tractor and primary worker")
}

func TestServiceUpdate_RejectsTwoMovesGoingInTransitWithSameTrailerInPayload(t *testing.T) {
	t.Parallel()

	actualPickupArrival := int64(110)
	actualPickupDeparture := int64(120)
	trailerID := pulid.MustNew("tr_")

	original := shipmentservicetest.ValidShipment()
	original.ID = pulid.MustNew("shp_")
	original.Version = 2
	original.Moves = []*shipment.ShipmentMove{
		shipmentservicetest.ValidMove(),
		shipmentservicetest.ValidMove(),
	}
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
			Status:          shipmentvocab.AssignmentStatusNew,
		}
		move.Status = shipmentvocab.MoveStatusAssigned
	}

	entity := shipmentservicetest.CloneShipment(original)
	for _, move := range entity.Moves {
		move.Assignment = nil
		move.Stops[0].ActualArrival = &actualPickupArrival
		move.Stops[0].ActualDeparture = &actualPickupDeparture
	}

	repo := shipmentportsmock.NewMockShipmentRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*shipmentports.GetShipmentByIDRequest")).
		Return(shipmentservicetest.CloneShipment(original), nil).
		Once()

	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, tenantports.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		}).
		Return(&shipmentcontrol.ShipmentControl{
			CheckForDuplicateBOLs:       true,
			AutoDelayShipmentsThreshold: new(int16(30)),
		}, nil).
		Once()

	freight := decimal.Zero

	assignmentRepo := assignmentportsmock.NewMockAssignmentRepository(t)
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

	svc := assembled(&rig{
		l:              zap.NewNop(),
		repo:           repo,
		assignmentRepo: assignmentRepo,
		controlRepo:    controlRepo,
		validator:      newServiceValidator(t),
		auditService:   auditportsmock.NewMockAuditService(t),
		commercial: newTestCommercialCalculator(
			t,
			freight,
			accessorialchargeportsmock.NewMockAccessorialChargeRepository(t),
		),
		invalidator:  newTestInvalidator(realtimeportsmock.NewMockRealtimeService(t)),
		eventService: noopShipmentEventService{},
		coordinator:  shipmentstate.NewCoordinatorWithClock(func() int64 { return 10 }),
	})

	userID := pulid.MustNew("usr_")
	updated, err := svc.writer.Update(
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

	original := shipmentservicetest.ValidShipment()
	original.ID = pulid.MustNew("shp_")
	original.Version = 1
	original.Moves[0].ID = pulid.MustNew("sm_")
	original.Moves[0].Version = 1
	original.Moves[0].Stops[0].ID = pulid.MustNew("stp_")
	original.Moves[0].Stops[0].ShipmentMoveID = original.Moves[0].ID
	original.Moves[0].Stops[1].ID = pulid.MustNew("stp_")
	original.Moves[0].Stops[1].ShipmentMoveID = original.Moves[0].ID
	original.Moves[0].Assignment = &shipment.Assignment{ID: pulid.MustNew("asn_")}
	original.Moves[0].Status = shipmentvocab.MoveStatusAssigned
	original.Status = shipmentvocab.StatusAssigned

	entity := shipmentservicetest.CloneShipment(original)
	entity.Status = shipmentvocab.StatusNew
	entity.Moves[0].Status = shipmentvocab.MoveStatusNew
	entity.Moves[0].Assignment = nil
	entity.BOL = "BOL-UPDATED"

	repo := shipmentportsmock.NewMockShipmentRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*shipmentports.GetShipmentByIDRequest")).
		RunAndReturn(func(
			_ context.Context,
			_ *shipmentports.GetShipmentByIDRequest,
		) (*shipment.Shipment, error) {
			return shipmentservicetest.CloneShipment(original), nil
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
			assert.Equal(t, shipmentvocab.MoveStatusAssigned, entity.Moves[0].Status)
			assert.Equal(t, shipmentvocab.StatusAssigned, entity.Status)
			return entity, nil
		}).
		Once()
	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, tenantports.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		}).
		Return(&shipmentcontrol.ShipmentControl{
			CheckForDuplicateBOLs:       true,
			AutoDelayShipmentsThreshold: new(int16(30)),
		}, nil).
		Once()
	repo.EXPECT().
		CheckForDuplicateBOLs(mock.Anything, mock.AnythingOfType("*shipmentports.DuplicateBOLCheckRequest")).
		Return([]*shipmentports.DuplicateBOLResult{}, nil).
		Once()
	freight := decimal.Zero
	auditService := auditportsmock.NewMockAuditService(t)
	auditService.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil).Once()
	realtime := realtimeportsmock.NewMockRealtimeService(t)
	realtime.EXPECT().PublishResourceInvalidation(mock.Anything, mock.Anything).Return(nil).Once()

	svc := assembled(&rig{
		l:            zap.NewNop(),
		repo:         repo,
		controlRepo:  controlRepo,
		validator:    newServiceValidator(t),
		auditService: auditService,
		commercial: newTestCommercialCalculator(
			t,
			freight,
			accessorialchargeportsmock.NewMockAccessorialChargeRepository(t),
		),
		invalidator:  newTestInvalidator(realtime),
		eventService: noopShipmentEventService{},
		coordinator:  shipmentstate.NewCoordinatorWithClock(func() int64 { return 10 }),
	})

	userID := pulid.MustNew("usr_")
	updated, err := svc.writer.Update(
		t.Context(),
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, shipmentvocab.StatusAssigned, updated.Status)
}

func TestServiceUpdate_RejectsDirectInvoiceFromCompleted(t *testing.T) {
	t.Parallel()

	original := shipmentservicetest.ValidShipment()
	original.ID = pulid.MustNew("shp_")
	original.Version = 1
	original.Status = shipmentvocab.StatusCompleted
	original.Moves[0].Status = shipmentvocab.MoveStatusCompleted
	original.Moves[0].Stops[0].Status = shipmentvocab.StopStatusCompleted
	original.Moves[0].Stops[1].Status = shipmentvocab.StopStatusCompleted

	entity := shipmentservicetest.CloneShipment(original)
	entity.Status = shipmentvocab.StatusInvoiced

	repo := shipmentportsmock.NewMockShipmentRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*shipmentports.GetShipmentByIDRequest")).
		RunAndReturn(func(
			_ context.Context,
			_ *shipmentports.GetShipmentByIDRequest,
		) (*shipment.Shipment, error) {
			return shipmentservicetest.CloneShipment(original), nil
		}).
		Once()
	controlRepo := tenantportsmock.NewMockShipmentControlRepository(t)
	controlRepo.EXPECT().
		Get(mock.Anything, tenantports.GetShipmentControlRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		}).
		Return(&shipmentcontrol.ShipmentControl{}, nil).
		Once()
	svc := assembled(&rig{
		l:            zap.NewNop(),
		repo:         repo,
		controlRepo:  controlRepo,
		validator:    newServiceValidator(t),
		auditService: auditportsmock.NewMockAuditService(t),
		eventService: noopShipmentEventService{},
		coordinator:  newStateCoordinator(),
	})
	userID := pulid.MustNew("usr_")
	updated, err := svc.writer.Update(
		t.Context(),
		entity,
		internaltestutil.NewSessionActor(userID, entity.OrganizationID, entity.BusinessUnitID),
	)

	require.Nil(t, updated)
	require.Error(t, err)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	shipmentservicetest.AssertErrorField(t, multiErr, "status")
}

func completedShipmentForCoordinator() *shipment.Shipment {
	entity := shipmentservicetest.ValidShipment()
	entity.ID = pulid.MustNew("shp_")
	entity.Version = 1
	entity.Status = shipmentvocab.StatusCompleted

	move := entity.Moves[0]
	move.ID = pulid.MustNew("sm_")
	move.ShipmentID = entity.ID
	move.Status = shipmentvocab.MoveStatusCompleted
	move.Version = 1

	for stopIndex, stop := range move.Stops {
		stop.ID = pulid.MustNew("stp_")
		stop.ShipmentMoveID = move.ID
		stop.Status = shipmentvocab.StopStatusCompleted
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

func shipmentWithOverdueStopForRefresh(status shipmentvocab.Status) *shipment.Shipment {
	entity := shipmentservicetest.ValidShipment()
	entity.Status = status
	entity.Moves[0].Status = shipmentvocab.MoveStatusInTransit
	entity.Moves[0].Stops[0].Status = shipmentvocab.StopStatusInTransit
	entity.Moves[0].Stops[0].ScheduledWindowEnd = new(int64(100))

	if status != shipmentvocab.StatusPartiallyCompleted {
		return entity
	}

	completedMove := shipmentservicetest.ValidMove()
	completedMove.Status = shipmentvocab.MoveStatusCompleted
	completedMove.Sequence = 0
	for _, stop := range completedMove.Stops {
		stop.Status = shipmentvocab.StopStatusCompleted
		stop.ActualArrival = new(int64(10))
		stop.ActualDeparture = new(int64(20))
	}

	activeMove := entity.Moves[0]
	activeMove.Sequence = 1
	entity.Moves = []*shipment.ShipmentMove{completedMove, activeMove}
	return entity
}

func completedStopForCoordinator(
	stopType shipmentvocab.StopType,
	sequence int64,
	actualArrival int64,
	actualDeparture int64,
) *shipment.Stop {
	stop := shipmentservicetest.MakeStop(stopType, sequence, 1, 2)
	stop.Status = shipmentvocab.StopStatusCompleted
	stop.ActualArrival = &actualArrival
	stop.ActualDeparture = &actualDeparture
	return stop
}

//go:fix inline
//go:fix inline
func ptrInt64(value int64) *int64 {
	return new(value)
}
