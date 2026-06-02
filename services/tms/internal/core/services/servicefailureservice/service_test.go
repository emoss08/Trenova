package servicefailureservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/edix12"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestLateMinutesAfterGraceUsesCeilingSemantics(t *testing.T) {
	tests := []struct {
		name   string
		actual int64
		cutoff int64
		grace  int
		want   int64
	}{
		{name: "not late at cutoff plus grace", actual: 1_300, cutoff: 1_000, grace: 5, want: 0},
		{name: "one second late rounds to one minute", actual: 1_301, cutoff: 1_000, grace: 5, want: 1},
		{name: "exact minute late stays one minute", actual: 1_360, cutoff: 1_000, grace: 5, want: 1},
		{name: "partial second minute rounds up", actual: 1_361, cutoff: 1_000, grace: 5, want: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := lateMinutesAfterGrace(tt.actual, tt.cutoff, tt.grace); got != tt.want {
				t.Fatalf("lateMinutesAfterGrace() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestNormalizedGracePeriodFallsBackToDispatchDefault(t *testing.T) {
	positive := 45
	zero := 0

	tests := []struct {
		name    string
		control *dispatchcontrol.DispatchControl
		want    int
	}{
		{name: "nil control", want: dispatchcontrol.DefaultServiceFailureGracePeriod},
		{name: "nil configured grace", control: &dispatchcontrol.DispatchControl{}, want: dispatchcontrol.DefaultServiceFailureGracePeriod},
		{name: "zero configured grace", control: &dispatchcontrol.DispatchControl{ServiceFailureGracePeriod: &zero}, want: dispatchcontrol.DefaultServiceFailureGracePeriod},
		{name: "positive configured grace", control: &dispatchcontrol.DispatchControl{ServiceFailureGracePeriod: &positive}, want: positive},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizedGracePeriod(tt.control); got != tt.want {
				t.Fatalf("normalizedGracePeriod() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestNewServiceFailureEvaluationResultUsesEmptySlices(t *testing.T) {
	result := newServiceFailureEvaluationResult()

	require.NotNil(t, result.CreatedIDs)
	require.NotNil(t, result.UpdatedIDs)
	require.NotNil(t, result.CreatedStops)
	require.NotNil(t, result.UpdatedStops)
	require.NotNil(t, result.SkippedStops)
	require.Empty(t, result.CreatedIDs)
	require.Empty(t, result.UpdatedIDs)
	require.Empty(t, result.CreatedStops)
	require.Empty(t, result.UpdatedStops)
	require.Empty(t, result.SkippedStops)
}

func TestShouldEvaluateStopHonorsServiceIncidentPolicy(t *testing.T) {
	override := true
	disabledOverride := false
	shipperStop := &shipment.Stop{ID: pulid.MustNew("stp_"), Type: shipment.StopTypePickup}
	otherPickup := &shipment.Stop{ID: pulid.MustNew("stp_"), Type: shipment.StopTypePickup}
	delivery := &shipment.Stop{ID: pulid.MustNew("stp_"), Type: shipment.StopTypeDelivery}
	splitDelivery := &shipment.Stop{ID: pulid.MustNew("stp_"), Type: shipment.StopTypeSplitDelivery}
	overrideDelivery := &shipment.Stop{
		ID:                pulid.MustNew("stp_"),
		Type:              shipment.StopTypeDelivery,
		CountLateOverride: &override,
	}
	disabledPickup := &shipment.Stop{
		ID:                pulid.MustNew("stp_"),
		Type:              shipment.StopTypePickup,
		CountLateOverride: &disabledOverride,
	}

	tests := []struct {
		name   string
		stop   *shipment.Stop
		policy dispatchcontrol.ServiceIncidentType
		want   bool
	}{
		{name: "pickup policy accepts origin stop", stop: otherPickup, policy: dispatchcontrol.ServiceIncidentTypePickup, want: true},
		{name: "pickup policy rejects delivery stop", stop: delivery, policy: dispatchcontrol.ServiceIncidentTypePickup},
		{name: "delivery policy accepts destination stop", stop: delivery, policy: dispatchcontrol.ServiceIncidentTypeDelivery, want: true},
		{name: "delivery policy accepts split delivery stop", stop: splitDelivery, policy: dispatchcontrol.ServiceIncidentTypeDelivery, want: true},
		{name: "pickup delivery policy accepts pickup", stop: otherPickup, policy: dispatchcontrol.ServiceIncidentTypePickupDelivery, want: true},
		{name: "pickup delivery policy accepts delivery", stop: delivery, policy: dispatchcontrol.ServiceIncidentTypePickupDelivery, want: true},
		{name: "all except shipper excludes shipper stop", stop: shipperStop, policy: dispatchcontrol.ServiceIncidentTypeAllExceptShipper},
		{name: "all except shipper accepts other pickup", stop: otherPickup, policy: dispatchcontrol.ServiceIncidentTypeAllExceptShipper, want: true},
		{name: "count late override bypasses stop type policy", stop: overrideDelivery, policy: dispatchcontrol.ServiceIncidentTypePickup, want: true},
		{name: "count late override false skips otherwise matching stop", stop: disabledPickup, policy: dispatchcontrol.ServiceIncidentTypePickup},
		{name: "never policy skips stop", stop: delivery, policy: dispatchcontrol.ServiceIncidentTypeNever},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldEvaluateStop(shouldEvaluateStopParams{
				stop:        tt.stop,
				shipperStop: shipperStop,
				policy:      tt.policy,
			})
			if got != tt.want {
				t.Fatalf("shouldEvaluateStop() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateManualRejectsWithoutTouchingRepositories(t *testing.T) {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID}
	svc := &service{l: zap.NewNop()}

	_, err := svc.CreateManual(t.Context(), &serviceports.CreateManualServiceFailureRequest{
		TenantInfo:     tenantInfo,
		ShipmentID:     pulid.MustNew("sp_"),
		ShipmentMoveID: pulid.MustNew("sm_"),
		StopID:         pulid.MustNew("stp_"),
		ReasonCodeID:   pulid.MustNew("sfrc_"),
		Type:           servicefailure.TypeLateDelivery,
	}, nil)

	require.Error(t, err)
	require.ErrorContains(t, err, "Manual service failure creation is disabled")
}

func TestCreateOrUpdateDetectedUpdatesExistingSnapshotIdempotently(t *testing.T) {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	reasonID := pulid.MustNew("sfrc_")
	entity := &servicefailure.ServiceFailure{
		ShipmentID:         pulid.MustNew("sp_"),
		ShipmentMoveID:     pulid.MustNew("sm_"),
		StopID:             pulid.MustNew("stp_"),
		OrganizationID:     orgID,
		BusinessUnitID:     buID,
		Type:               servicefailure.TypeLateDelivery,
		Source:             servicefailure.SourceDetected,
		Status:             servicefailure.StatusOpen,
		StopType:           shipment.StopTypeDelivery,
		ScheduledCutoff:    1_000,
		ActualArrival:      1_301,
		GracePeriodMinutes: 5,
		LateMinutes:        1,
	}
	existing := *entity
	existing.ID = pulid.MustNew("sf_")
	existing.ReasonCodeID = &reasonID
	existing.Notes = "existing reason"

	repo := mocks.NewMockServiceFailureRepository(t)
	reasonRepo := mocks.NewMockServiceFailureReasonCodeRepository(t)
	svc := &service{
		l:              zap.NewNop(),
		repo:           repo,
		reasonCodeRepo: reasonRepo,
	}

	reasonRepo.EXPECT().
		FindDefault(mock.Anything, pagination.TenantInfo{OrgID: orgID, BuID: buID}, servicefailure.ReasonCodeAppliesToDelivery).
		Return(nil, errortypes.NewNotFoundError("not found")).
		Once()
	repo.EXPECT().
		FindUnresolvedByStop(mock.Anything, mock.AnythingOfType("*repositories.ServiceFailureActiveStopRequest")).
		Return(&existing, nil).
		Once()
	repo.EXPECT().
		UpdateDetectionSnapshot(mock.Anything, mock.AnythingOfType("*servicefailure.ServiceFailure")).
		RunAndReturn(func(_ context.Context, updated *servicefailure.ServiceFailure) (*servicefailure.ServiceFailure, error) {
			require.Equal(t, existing.ID, updated.ID)
			require.Equal(t, reasonID, *updated.ReasonCodeID)
			require.Equal(t, "existing reason", updated.Notes)
			require.Equal(t, int64(1_301), updated.ActualArrival)
			require.Equal(t, int64(1), updated.LateMinutes)
			return updated, nil
		}).
		Once()

	updated, err := svc.createOrUpdateDetected(t.Context(), &detectedAction{entity: entity}, nil)

	require.NoError(t, err)
	require.Equal(t, existing.ID, updated.ID)
}

func TestCreateOrUpdateDetectedCreatesWithGeneratedNumber(t *testing.T) {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	entity := &servicefailure.ServiceFailure{
		ShipmentID:         pulid.MustNew("sp_"),
		ShipmentMoveID:     pulid.MustNew("sm_"),
		StopID:             pulid.MustNew("stp_"),
		OrganizationID:     orgID,
		BusinessUnitID:     buID,
		Type:               servicefailure.TypeLateDelivery,
		Source:             servicefailure.SourceDetected,
		Status:             servicefailure.StatusOpen,
		StopType:           shipment.StopTypeDelivery,
		ScheduledCutoff:    1_000,
		ActualArrival:      1_301,
		GracePeriodMinutes: 5,
		LateMinutes:        1,
	}
	repo := mocks.NewMockServiceFailureRepository(t)
	reasonRepo := mocks.NewMockServiceFailureReasonCodeRepository(t)
	audit := mocks.NewMockAuditService(t)
	realtime := mocks.NewMockRealtimeService(t)
	svc := &service{
		l:              zap.NewNop(),
		repo:           repo,
		reasonCodeRepo: reasonRepo,
		auditService:   audit,
		realtime:       realtime,
	}

	reasonRepo.EXPECT().
		FindDefault(mock.Anything, pagination.TenantInfo{OrgID: orgID, BuID: buID}, servicefailure.ReasonCodeAppliesToDelivery).
		Return(nil, errortypes.NewNotFoundError("not found")).
		Once()
	repo.EXPECT().
		FindUnresolvedByStop(mock.Anything, mock.AnythingOfType("*repositories.ServiceFailureActiveStopRequest")).
		Return(nil, errortypes.NewNotFoundError("not found")).
		Once()
	repo.EXPECT().
		Create(mock.Anything, mock.AnythingOfType("*servicefailure.ServiceFailure")).
		RunAndReturn(func(_ context.Context, created *servicefailure.ServiceFailure) (*servicefailure.ServiceFailure, error) {
			require.False(t, created.ID.IsNil())
			require.NotEmpty(t, created.Number)
			require.Contains(t, created.Number, "SF-")
			return created, nil
		}).
		Once()
	audit.EXPECT().LogAction(mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	realtime.EXPECT().PublishResourceInvalidation(mock.Anything, mock.Anything).Return(nil).Once()

	created, err := svc.createOrUpdateDetected(t.Context(), &detectedAction{entity: entity}, nil)

	require.NoError(t, err)
	require.NotEmpty(t, created.Number)
}

func TestEvaluateShipmentIncludesCreatedStopSummaries(t *testing.T) {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	shipmentID := pulid.MustNew("sp_")
	moveID := pulid.MustNew("sm_")
	stopID := pulid.MustNew("stp_")
	userID := pulid.MustNew("usr_")
	gracePeriod := 5
	actualArrival := int64(1_360)
	stop := serviceFailureStopFixture(orgID, buID, moveID, stopID, shipment.StopTypeDelivery, actualArrival)
	source := serviceFailureShipmentWithStops(orgID, buID, shipmentID, moveID, stop)

	repo := mocks.NewMockServiceFailureRepository(t)
	reasonRepo := mocks.NewMockServiceFailureReasonCodeRepository(t)
	shipmentRepo := mocks.NewMockShipmentRepository(t)
	dispatchRepo := mocks.NewMockDispatchControlRepository(t)
	audit := mocks.NewMockAuditService(t)
	realtime := mocks.NewMockRealtimeService(t)
	svc := &service{
		l:              zap.NewNop(),
		repo:           repo,
		reasonCodeRepo: reasonRepo,
		shipmentRepo:   shipmentRepo,
		dispatchRepo:   dispatchRepo,
		auditService:   audit,
		realtime:       realtime,
	}

	shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*repositories.GetShipmentByIDRequest")).
		Return(source, nil).
		Once()
	dispatchRepo.EXPECT().
		GetOrCreate(mock.Anything, orgID, buID).
		Return(&dispatchcontrol.DispatchControl{
			RecordServiceFailures:     dispatchcontrol.ServiceIncidentTypePickupDelivery,
			ServiceFailureGracePeriod: &gracePeriod,
		}, nil).
		Once()
	reasonRepo.EXPECT().
		FindDefault(mock.Anything, pagination.TenantInfo{OrgID: orgID, BuID: buID}, servicefailure.ReasonCodeAppliesToDelivery).
		Return(nil, errortypes.NewNotFoundError("not found")).
		Once()
	repo.EXPECT().
		FindUnresolvedByStop(mock.Anything, mock.AnythingOfType("*repositories.ServiceFailureActiveStopRequest")).
		Return(nil, errortypes.NewNotFoundError("not found")).
		Once()
	repo.EXPECT().
		Create(mock.Anything, mock.AnythingOfType("*servicefailure.ServiceFailure")).
		RunAndReturn(func(_ context.Context, created *servicefailure.ServiceFailure) (*servicefailure.ServiceFailure, error) {
			created.Stop = stop
			return created, nil
		}).
		Once()
	audit.EXPECT().LogAction(mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	realtime.EXPECT().PublishResourceInvalidation(mock.Anything, mock.Anything).Return(nil).Once()

	result, err := svc.EvaluateShipment(t.Context(), &serviceports.EvaluateShipmentServiceFailuresRequest{
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		ShipmentID: shipmentID,
	}, &serviceports.RequestActor{
		PrincipalType:  serviceports.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	})

	require.NoError(t, err)
	require.Len(t, result.CreatedIDs, 1)
	require.Len(t, result.CreatedStops, 1)
	summary := result.CreatedStops[0]
	require.Equal(t, result.CreatedIDs[0], summary.ServiceFailureID)
	require.Equal(t, shipmentID, summary.ShipmentID)
	require.Equal(t, moveID, summary.ShipmentMoveID)
	require.Equal(t, stopID, summary.StopID)
	require.Equal(t, int64(2), summary.StopSequence)
	require.Equal(t, shipment.StopTypeDelivery, summary.StopType)
	require.Equal(t, "Warehouse 12", summary.LocationName)
	require.Equal(t, "WH12", summary.LocationCode)
	require.Equal(t, "Austin", summary.City)
	require.Equal(t, "TX", summary.StateCode)
	require.Equal(t, int64(1_000), summary.ScheduledCutoff)
	require.Equal(t, actualArrival, summary.ActualArrival)
	require.Equal(t, gracePeriod, summary.GracePeriodMinutes)
	require.Equal(t, int64(1), summary.LateMinutes)
}

func TestEvaluateShipmentIncludesUpdatedStopSummaries(t *testing.T) {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	shipmentID := pulid.MustNew("sp_")
	moveID := pulid.MustNew("sm_")
	stopID := pulid.MustNew("stp_")
	gracePeriod := 5
	actualArrival := int64(1_360)
	stop := serviceFailureStopFixture(orgID, buID, moveID, stopID, shipment.StopTypeDelivery, actualArrival)
	source := serviceFailureShipmentWithStops(orgID, buID, shipmentID, moveID, stop)
	existing := &servicefailure.ServiceFailure{
		ID:                 pulid.MustNew("sf_"),
		ShipmentID:         shipmentID,
		ShipmentMoveID:     moveID,
		StopID:             stopID,
		OrganizationID:     orgID,
		BusinessUnitID:     buID,
		Type:               servicefailure.TypeLateDelivery,
		Source:             servicefailure.SourceDetected,
		Status:             servicefailure.StatusOpen,
		StopType:           shipment.StopTypeDelivery,
		ScheduledCutoff:    900,
		ActualArrival:      1_250,
		GracePeriodMinutes: gracePeriod,
		LateMinutes:        1,
	}

	repo := mocks.NewMockServiceFailureRepository(t)
	reasonRepo := mocks.NewMockServiceFailureReasonCodeRepository(t)
	shipmentRepo := mocks.NewMockShipmentRepository(t)
	dispatchRepo := mocks.NewMockDispatchControlRepository(t)
	svc := &service{
		l:              zap.NewNop(),
		repo:           repo,
		reasonCodeRepo: reasonRepo,
		shipmentRepo:   shipmentRepo,
		dispatchRepo:   dispatchRepo,
	}

	shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*repositories.GetShipmentByIDRequest")).
		Return(source, nil).
		Once()
	dispatchRepo.EXPECT().
		GetOrCreate(mock.Anything, orgID, buID).
		Return(&dispatchcontrol.DispatchControl{
			RecordServiceFailures:     dispatchcontrol.ServiceIncidentTypePickupDelivery,
			ServiceFailureGracePeriod: &gracePeriod,
		}, nil).
		Once()
	reasonRepo.EXPECT().
		FindDefault(mock.Anything, pagination.TenantInfo{OrgID: orgID, BuID: buID}, servicefailure.ReasonCodeAppliesToDelivery).
		Return(nil, errortypes.NewNotFoundError("not found")).
		Once()
	repo.EXPECT().
		FindUnresolvedByStop(mock.Anything, mock.AnythingOfType("*repositories.ServiceFailureActiveStopRequest")).
		Return(existing, nil).
		Once()
	repo.EXPECT().
		UpdateDetectionSnapshot(mock.Anything, mock.AnythingOfType("*servicefailure.ServiceFailure")).
		RunAndReturn(func(_ context.Context, updated *servicefailure.ServiceFailure) (*servicefailure.ServiceFailure, error) {
			updated.Stop = stop
			return updated, nil
		}).
		Once()

	result, err := svc.EvaluateShipment(t.Context(), &serviceports.EvaluateShipmentServiceFailuresRequest{
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		ShipmentID: shipmentID,
	}, nil)

	require.NoError(t, err)
	require.Len(t, result.UpdatedIDs, 1)
	require.Len(t, result.UpdatedStops, 1)
	summary := result.UpdatedStops[0]
	require.Equal(t, existing.ID, summary.ServiceFailureID)
	require.Equal(t, shipmentID, summary.ShipmentID)
	require.Equal(t, moveID, summary.ShipmentMoveID)
	require.Equal(t, stopID, summary.StopID)
	require.Equal(t, "Warehouse 12", summary.LocationName)
	require.Equal(t, "TX", summary.StateCode)
	require.Equal(t, int64(1_000), summary.ScheduledCutoff)
	require.Equal(t, actualArrival, summary.ActualArrival)
	require.Equal(t, int64(1), summary.LateMinutes)
}

func TestEvaluateShipmentSkippedStopsIncludeStopContext(t *testing.T) {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	shipmentID := pulid.MustNew("sp_")
	moveID := pulid.MustNew("sm_")
	stopID := pulid.MustNew("stp_")
	gracePeriod := 5
	stop := serviceFailureStopFixture(orgID, buID, moveID, stopID, shipment.StopTypeDelivery, 0)
	stop.ActualArrival = nil
	source := serviceFailureShipmentWithStops(orgID, buID, shipmentID, moveID, stop)

	shipmentRepo := mocks.NewMockShipmentRepository(t)
	dispatchRepo := mocks.NewMockDispatchControlRepository(t)
	svc := &service{
		l:            zap.NewNop(),
		shipmentRepo: shipmentRepo,
		dispatchRepo: dispatchRepo,
	}

	shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*repositories.GetShipmentByIDRequest")).
		Return(source, nil).
		Once()
	dispatchRepo.EXPECT().
		GetOrCreate(mock.Anything, orgID, buID).
		Return(&dispatchcontrol.DispatchControl{
			RecordServiceFailures:     dispatchcontrol.ServiceIncidentTypePickupDelivery,
			ServiceFailureGracePeriod: &gracePeriod,
		}, nil).
		Once()

	result, err := svc.EvaluateShipment(t.Context(), &serviceports.EvaluateShipmentServiceFailuresRequest{
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		ShipmentID: shipmentID,
	}, nil)

	require.NoError(t, err)
	require.Equal(t, 1, result.Skipped)
	require.Len(t, result.SkippedStops, 1)
	summary := result.SkippedStops[0]
	require.Equal(t, "missing actual arrival", summary.Reason)
	require.Equal(t, shipmentID, summary.ShipmentID)
	require.Equal(t, moveID, summary.ShipmentMoveID)
	require.Equal(t, stopID, summary.StopID)
	require.Equal(t, int64(2), summary.StopSequence)
	require.Equal(t, shipment.StopTypeDelivery, summary.StopType)
	require.Equal(t, "Warehouse 12", summary.LocationName)
	require.Equal(t, "WH12", summary.LocationCode)
	require.Equal(t, "Austin", summary.City)
	require.Equal(t, "TX", summary.StateCode)
	require.Equal(t, int64(1_000), summary.ScheduledCutoff)
	require.Equal(t, int64(0), summary.ActualArrival)
	require.Equal(t, gracePeriod, summary.GracePeriodMinutes)
	require.Equal(t, int64(0), summary.LateMinutes)
}

func TestLifecycleReviewRequiresUserAndRecordsAudit(t *testing.T) {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	reasonID := pulid.MustNew("sfrc_")
	entity := &servicefailure.ServiceFailure{
		ID:                 pulid.MustNew("sf_"),
		ShipmentID:         pulid.MustNew("sp_"),
		ShipmentMoveID:     pulid.MustNew("sm_"),
		StopID:             pulid.MustNew("stp_"),
		ReasonCodeID:       &reasonID,
		OrganizationID:     orgID,
		BusinessUnitID:     buID,
		Type:               servicefailure.TypeLateDelivery,
		Source:             servicefailure.SourceDetected,
		Status:             servicefailure.StatusOpen,
		StopType:           shipment.StopTypeDelivery,
		ScheduledCutoff:    1_000,
		ActualArrival:      1_301,
		GracePeriodMinutes: 5,
		LateMinutes:        1,
	}
	repo := mocks.NewMockServiceFailureRepository(t)
	audit := mocks.NewMockAuditService(t)
	realtime := mocks.NewMockRealtimeService(t)
	svc := &service{
		l:            zap.NewNop(),
		repo:         repo,
		auditService: audit,
		realtime:     realtime,
	}
	req := &serviceports.ServiceFailureLifecycleRequest{
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		ID:         entity.ID,
		ShipmentID: entity.ShipmentID,
		Version:    2,
		Notes:      "reviewed",
	}

	repo.EXPECT().
		GetByShipment(mock.Anything, mock.AnythingOfType("*repositories.GetServiceFailureByShipmentRequest")).
		Return(entity, nil).
		Once()
	repo.EXPECT().
		Update(mock.Anything, mock.AnythingOfType("*servicefailure.ServiceFailure")).
		RunAndReturn(func(_ context.Context, updated *servicefailure.ServiceFailure) (*servicefailure.ServiceFailure, error) {
			require.Equal(t, servicefailure.StatusReviewed, updated.Status)
			require.Equal(t, int64(2), updated.Version)
			require.NotNil(t, updated.ReviewedByID)
			require.Equal(t, userID, *updated.ReviewedByID)
			require.Equal(t, "reviewed", updated.InternalNotes)
			return updated, nil
		}).
		Once()
	audit.EXPECT().LogAction(mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	realtime.EXPECT().PublishResourceInvalidation(mock.Anything, mock.Anything).Return(nil).Once()

	updated, err := svc.Review(t.Context(), req, &serviceports.RequestActor{
		PrincipalType:  serviceports.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	})

	require.NoError(t, err)
	require.Equal(t, servicefailure.StatusReviewed, updated.Status)
}

func TestTransitionShipmentToDelayedUsesMarker(t *testing.T) {
	source := &shipment.Shipment{
		ID:             pulid.MustNew("sp_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Status:         shipment.StatusInTransit,
	}
	updated := *source
	updated.Status = shipment.StatusDelayed
	marker := &fakeDelayedShipmentMarker{updated: &updated}
	svc := &service{l: zap.NewNop(), delayedMarker: marker}
	failure := &servicefailure.ServiceFailure{ID: pulid.MustNew("sf_")}

	svc.transitionShipmentToDelayed(t.Context(), source, nil, failure)

	require.True(t, marker.called)
	require.Equal(t, source.ID, marker.params.shipment.ID)
	require.Equal(t, failure.ID, marker.params.failure.ID)
	require.Equal(t, shipment.StatusDelayed, source.Status)
}

func TestPreflightServiceFailure214BlocksMandatoryDiagnostics(t *testing.T) {
	t.Parallel()

	failure := serviceFailureLifecycleFixture(servicefailure.StatusReviewed)
	ediSvc := mocks.NewMockEDIService(t)
	svc := &service{l: zap.NewNop(), ediService: ediSvc}

	ediSvc.EXPECT().
		PreviewServiceFailure214ForLifecycle(mock.Anything, mock.AnythingOfType("*services.ServiceFailure214LifecycleRequest")).
		Return(&serviceports.ServiceFailure214LifecycleResult{
			Trigger:   serviceports.ServiceFailureEDITriggerReviewed,
			Action:    serviceports.ServiceFailureEDIActionBlocked,
			Mandatory: true,
			Diagnostics: []edix12.Diagnostic{{
				Severity: edi.ValidationSeverityError,
				Path:     "shipmentStatus.statusReasonCode",
				Message:  "status reason required",
			}},
		}, nil).
		Once()

	err := svc.preflightServiceFailure214(t.Context(), serviceFailure214Params{current: failure})

	require.Error(t, err)
	require.Contains(t, err.Error(), "status reason required")
}

func TestPreflightServiceFailure214AllowsNonMandatoryBlockedAndSkipped(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		result *serviceports.ServiceFailure214LifecycleResult
	}{
		{
			name: "non mandatory blocked",
			result: &serviceports.ServiceFailure214LifecycleResult{
				Trigger: serviceports.ServiceFailureEDITriggerReviewed,
				Action:  serviceports.ServiceFailureEDIActionBlocked,
			},
		},
		{
			name: "skipped disabled",
			result: &serviceports.ServiceFailure214LifecycleResult{
				Trigger:       serviceports.ServiceFailureEDITriggerReviewed,
				Action:        serviceports.ServiceFailureEDIActionSkipped,
				SkippedReason: "service failure 214 trigger disabled",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failure := serviceFailureLifecycleFixture(servicefailure.StatusReviewed)
			ediSvc := mocks.NewMockEDIService(t)
			svc := &service{l: zap.NewNop(), ediService: ediSvc}
			ediSvc.EXPECT().
				PreviewServiceFailure214ForLifecycle(mock.Anything, mock.Anything).
				Return(tt.result, nil).
				Once()

			err := svc.preflightServiceFailure214(t.Context(), serviceFailure214Params{current: failure})

			require.NoError(t, err)
		})
	}
}

func TestGenerateServiceFailure214RecordsGeneratedAndDuplicateOnly(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		action      serviceports.ServiceFailureEDIAction
		wantComment bool
	}{
		{name: "generated", action: serviceports.ServiceFailureEDIActionGenerated, wantComment: true},
		{name: "duplicate", action: serviceports.ServiceFailureEDIActionDuplicate, wantComment: true},
		{name: "skipped", action: serviceports.ServiceFailureEDIActionSkipped},
		{name: "blocked", action: serviceports.ServiceFailureEDIActionBlocked},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failure := serviceFailureLifecycleFixture(servicefailure.StatusReviewed)
			ediSvc := mocks.NewMockEDIService(t)
			audit := mocks.NewMockAuditService(t)
			comments := mocks.NewMockShipmentCommentService(t)
			svc := &service{
				l:              zap.NewNop(),
				ediService:     ediSvc,
				auditService:   audit,
				commentService: comments,
			}
			ediSvc.EXPECT().
				GenerateServiceFailure214ForLifecycle(mock.Anything, mock.Anything).
				Return(&serviceports.ServiceFailure214LifecycleResult{
					Trigger:                  serviceports.ServiceFailureEDITriggerReviewed,
					Action:                   tt.action,
					MessageID:                pulid.MustNew("edimsg_"),
					EDIPartnerID:             pulid.MustNew("edip_"),
					PartnerDocumentProfileID: pulid.MustNew("edidp_"),
				}, nil).
				Once()
			if tt.wantComment {
				audit.EXPECT().LogAction(mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
				comments.EXPECT().
					CreateSystem(mock.Anything, mock.AnythingOfType("*services.CreateSystemShipmentCommentRequest")).
					Return(&shipment.ShipmentComment{}, nil).
					Once()
			}

			svc.generateServiceFailure214(t.Context(), serviceFailure214Params{current: failure})
		})
	}
}

type fakeDelayedShipmentMarker struct {
	called  bool
	params  delayedShipmentMarkParams
	updated *shipment.Shipment
}

func serviceFailureLifecycleFixture(status servicefailure.Status) *servicefailure.ServiceFailure {
	reasonID := pulid.MustNew("sfrc_")
	return &servicefailure.ServiceFailure{
		ID:             pulid.MustNew("sf_"),
		ShipmentID:     pulid.MustNew("sp_"),
		ShipmentMoveID: pulid.MustNew("sm_"),
		StopID:         pulid.MustNew("stp_"),
		ReasonCodeID:   &reasonID,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Number:         "SF-1006",
		Type:           servicefailure.TypeLateDelivery,
		Source:         servicefailure.SourceDetected,
		Status:         status,
	}
}

func (m *fakeDelayedShipmentMarker) MarkDelayedForServiceFailure(
	_ context.Context,
	params delayedShipmentMarkParams,
) (*shipment.Shipment, error) {
	m.called = true
	m.params = params
	return m.updated, nil
}

func serviceFailureShipmentFixture(
	orgID pulid.ID,
	buID pulid.ID,
	shipmentID pulid.ID,
	moveID pulid.ID,
	stopID pulid.ID,
	stopType shipment.StopType,
	actualArrival int64,
) *shipment.Shipment {
	return &shipment.Shipment{
		ID:             shipmentID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Status:         shipment.StatusNew,
		Moves: []*shipment.ShipmentMove{
			{
				ID:     moveID,
				Status: shipment.MoveStatusNew,
				Stops: []*shipment.Stop{
					{
						ID:                   stopID,
						Type:                 stopType,
						Status:               shipment.StopStatusCompleted,
						ScheduledWindowStart: 1_000,
						ActualArrival:        &actualArrival,
					},
				},
			},
		},
	}
}

func serviceFailureStopFixture(
	orgID pulid.ID,
	buID pulid.ID,
	moveID pulid.ID,
	stopID pulid.ID,
	stopType shipment.StopType,
	actualArrival int64,
) *shipment.Stop {
	locationID := pulid.MustNew("loc_")
	return &shipment.Stop{
		ID:                   stopID,
		OrganizationID:       orgID,
		BusinessUnitID:       buID,
		ShipmentMoveID:       moveID,
		LocationID:           locationID,
		Type:                 stopType,
		Status:               shipment.StopStatusCompleted,
		Sequence:             2,
		ScheduledWindowStart: 1_000,
		ActualArrival:        &actualArrival,
		Location: &location.Location{
			ID:    locationID,
			Code:  "WH12",
			Name:  "Warehouse 12",
			City:  "Austin",
			State: &usstate.UsState{Abbreviation: "TX"},
		},
	}
}

func serviceFailureShipmentWithStops(
	orgID pulid.ID,
	buID pulid.ID,
	shipmentID pulid.ID,
	moveID pulid.ID,
	stop *shipment.Stop,
) *shipment.Shipment {
	return &shipment.Shipment{
		ID:             shipmentID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Status:         shipment.StatusNew,
		Moves: []*shipment.ShipmentMove{
			{
				ID:     moveID,
				Status: shipment.MoveStatusNew,
				Stops:  []*shipment.Stop{stop},
			},
		},
	}
}
