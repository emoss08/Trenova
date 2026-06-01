package servicefailureservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
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

func TestShouldEvaluateStopHonorsServiceIncidentPolicy(t *testing.T) {
	override := true
	shipperStop := &shipment.Stop{ID: pulid.MustNew("stp_"), Type: shipment.StopTypePickup}
	otherPickup := &shipment.Stop{ID: pulid.MustNew("stp_"), Type: shipment.StopTypePickup}
	delivery := &shipment.Stop{ID: pulid.MustNew("stp_"), Type: shipment.StopTypeDelivery}
	splitDelivery := &shipment.Stop{ID: pulid.MustNew("stp_"), Type: shipment.StopTypeSplitDelivery}
	overrideDelivery := &shipment.Stop{
		ID:                pulid.MustNew("stp_"),
		Type:              shipment.StopTypeDelivery,
		CountLateOverride: &override,
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

func TestCreateManualCreatesFailureWithScopedStopContext(t *testing.T) {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	shipmentID := pulid.MustNew("sp_")
	moveID := pulid.MustNew("sm_")
	stopID := pulid.MustNew("stp_")
	reasonID := pulid.MustNew("sfrc_")
	userID := pulid.MustNew("usr_")
	actualArrival := int64(1_301)
	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID}
	source := serviceFailureShipmentFixture(orgID, buID, shipmentID, moveID, stopID, shipment.StopTypeDelivery, actualArrival)

	shipmentRepo := mocks.NewMockShipmentRepository(t)
	reasonRepo := mocks.NewMockServiceFailureReasonCodeRepository(t)
	dispatchRepo := mocks.NewMockDispatchControlRepository(t)
	repo := mocks.NewMockServiceFailureRepository(t)
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
	reasonRepo.EXPECT().
		GetByID(mock.Anything, repositories.GetServiceFailureReasonCodeByIDRequest{
			ID:         reasonID,
			TenantInfo: tenantInfo,
		}).
		Return(&servicefailure.ReasonCode{
			ID:             reasonID,
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Code:           "LATE",
			Label:          "Late delivery",
			Active:         true,
			AppliesTo:      servicefailure.ReasonCodeAppliesToDelivery,
		}, nil).
		Once()
	dispatchRepo.EXPECT().
		GetOrCreate(mock.Anything, orgID, buID).
		Return(&dispatchcontrol.DispatchControl{}, nil).
		Once()
	repo.EXPECT().
		FindUnresolvedByStop(mock.Anything, mock.AnythingOfType("*repositories.ServiceFailureActiveStopRequest")).
		Return(nil, errortypes.NewNotFoundError("not found")).
		Once()
	repo.EXPECT().
		Create(mock.Anything, mock.AnythingOfType("*servicefailure.ServiceFailure")).
		RunAndReturn(func(_ context.Context, entity *servicefailure.ServiceFailure) (*servicefailure.ServiceFailure, error) {
			require.Equal(t, shipmentID, entity.ShipmentID)
			require.Equal(t, moveID, entity.ShipmentMoveID)
			require.Equal(t, stopID, entity.StopID)
			require.Equal(t, servicefailure.TypeLateDelivery, entity.Type)
			require.Equal(t, servicefailure.SourceManual, entity.Source)
			require.Equal(t, servicefailure.StatusOpen, entity.Status)
			require.NotNil(t, entity.CreatedByID)
			require.Equal(t, userID, *entity.CreatedByID)
			require.Equal(t, int64(1), entity.LateMinutes)
			return entity, nil
		}).
		Once()
	audit.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil).Once()
	realtime.EXPECT().PublishResourceInvalidation(mock.Anything, mock.Anything).Return(nil).Once()

	created, err := svc.CreateManual(t.Context(), &serviceports.CreateManualServiceFailureRequest{
		TenantInfo:     tenantInfo,
		ShipmentID:     shipmentID,
		ShipmentMoveID: moveID,
		StopID:         stopID,
		ReasonCodeID:   reasonID,
		Type:           servicefailure.TypeLateDelivery,
	}, &serviceports.RequestActor{
		PrincipalType:  serviceports.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	})

	require.NoError(t, err)
	require.NotNil(t, created)
}

func TestCreateManualRejectsStopTypeMismatch(t *testing.T) {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	shipmentID := pulid.MustNew("sp_")
	moveID := pulid.MustNew("sm_")
	stopID := pulid.MustNew("stp_")
	reasonID := pulid.MustNew("sfrc_")
	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID}
	source := serviceFailureShipmentFixture(orgID, buID, shipmentID, moveID, stopID, shipment.StopTypePickup, 1_301)
	shipmentRepo := mocks.NewMockShipmentRepository(t)
	svc := &service{
		l:            zap.NewNop(),
		shipmentRepo: shipmentRepo,
	}

	shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*repositories.GetShipmentByIDRequest")).
		Return(source, nil).
		Once()

	_, err := svc.CreateManual(t.Context(), &serviceports.CreateManualServiceFailureRequest{
		TenantInfo:     tenantInfo,
		ShipmentID:     shipmentID,
		ShipmentMoveID: moveID,
		StopID:         stopID,
		ReasonCodeID:   reasonID,
		Type:           servicefailure.TypeLateDelivery,
	}, nil)

	require.Error(t, err)
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

func TestLifecycleReviewRequiresUserAndRecordsAudit(t *testing.T) {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	entity := &servicefailure.ServiceFailure{
		ID:                 pulid.MustNew("sf_"),
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

type fakeDelayedShipmentMarker struct {
	called  bool
	params  delayedShipmentMarkParams
	updated *shipment.Shipment
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
