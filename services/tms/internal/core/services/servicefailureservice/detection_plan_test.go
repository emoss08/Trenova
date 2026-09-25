package servicefailureservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type detectionFixture struct {
	tenant  pagination.TenantInfo
	source  *shipment.Shipment
	lateID  pulid.ID
	onTime  pulid.ID
	grace   int
	svc     *service
	repo    *mocks.MockServiceFailureRepository
	marker  *fakeDelayedShipmentMarker
	request *serviceports.EvaluateShipmentServiceFailuresRequest
}

func newDetectionFixture(t *testing.T) *detectionFixture {
	t.Helper()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	shipmentID := pulid.MustNew("sp_")
	moveID := pulid.MustNew("sm_")
	late := serviceFailureStopFixture(
		orgID,
		buID,
		moveID,
		pulid.MustNew("stp_"),
		shipment.StopTypeDelivery,
		1_360,
	)
	onTime := serviceFailureStopFixture(
		orgID,
		buID,
		moveID,
		pulid.MustNew("stp_"),
		shipment.StopTypePickup,
		1_000,
	)
	source := serviceFailureShipmentWithStops(orgID, buID, shipmentID, moveID, onTime)
	source.Status = shipment.StatusInTransit
	source.Moves[0].Stops = append(source.Moves[0].Stops, late)

	fixture := &detectionFixture{
		tenant: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		source: source,
		lateID: late.ID,
		onTime: onTime.ID,
		grace:  5,
		repo:   mocks.NewMockServiceFailureRepository(t),
		marker: &fakeDelayedShipmentMarker{},
	}
	shipmentRepo := mocks.NewMockShipmentRepository(t)
	shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*repositories.GetShipmentByIDRequest")).
		Return(source, nil).
		Once()
	dispatchRepo := mocks.NewMockDispatchControlRepository(t)
	dispatchRepo.EXPECT().
		GetOrCreate(mock.Anything, orgID, buID).
		Return(&dispatchcontrol.DispatchControl{
			RecordServiceFailures:     dispatchcontrol.ServiceIncidentTypePickupDelivery,
			ServiceFailureGracePeriod: &fixture.grace,
		}, nil).
		Once()
	reasonRepo := mocks.NewMockServiceFailureReasonCodeRepository(t)
	reasonRepo.EXPECT().
		FindDefault(mock.Anything, fixture.tenant, servicefailure.ReasonCodeAppliesToDelivery).
		Return(nil, errortypes.NewNotFoundError("not found")).
		Once()

	fixture.svc = &service{
		l:              zap.NewNop(),
		repo:           fixture.repo,
		reasonCodeRepo: reasonRepo,
		shipmentRepo:   shipmentRepo,
		dispatchRepo:   dispatchRepo,
		delayedMarker:  fixture.marker,
	}
	fixture.request = &serviceports.EvaluateShipmentServiceFailuresRequest{
		TenantInfo: fixture.tenant,
		ShipmentID: shipmentID,
	}

	return fixture
}

func TestPreviewEvaluateShipment_PlansTheFailureWithoutSavingIt(t *testing.T) {
	t.Parallel()

	fixture := newDetectionFixture(t)
	fixture.repo.EXPECT().
		FindUnresolvedByStop(mock.Anything, mock.AnythingOfType("*repositories.ServiceFailureActiveStopRequest")).
		Return(nil, errortypes.NewNotFoundError("not found")).
		Once()

	plan, err := fixture.svc.PreviewEvaluateShipment(t.Context(), fixture.request)
	require.NoError(t, err)

	require.Len(t, plan.Detected, 1)
	detected := plan.Detected[0]
	assert.Nil(t, detected.Existing)
	assert.Equal(t, fixture.lateID, detected.Failure.StopID)
	assert.Equal(t, servicefailure.TypeLateDelivery, detected.Failure.Type)
	assert.Equal(t, int64(1), detected.Failure.LateMinutes)
	assert.NotEmpty(t, detected.Failure.Notes)
	require.Len(t, plan.SkippedStops, 1)
	assert.Equal(t, fixture.onTime, plan.SkippedStops[0].StopID)
	assert.Equal(t, "not late after grace", plan.SkippedStops[0].Reason)
	assert.True(t, plan.MarksDelayed)
	assert.False(t, fixture.marker.called)
	assert.Equal(t, shipment.StatusInTransit, fixture.source.Status)
}

func TestPreviewEvaluateShipment_RefreshesAnOpenFailureInsteadOfOpeningAnother(t *testing.T) {
	t.Parallel()

	fixture := newDetectionFixture(t)
	existing := &servicefailure.ServiceFailure{
		ID:              pulid.MustNew("sf_"),
		StopID:          fixture.lateID,
		Status:          servicefailure.StatusOpen,
		ScheduledCutoff: 900,
		ActualArrival:   1_250,
		LateMinutes:     1,
	}
	fixture.repo.EXPECT().
		FindUnresolvedByStop(mock.Anything, mock.AnythingOfType("*repositories.ServiceFailureActiveStopRequest")).
		Return(existing, nil).
		Once()

	plan, err := fixture.svc.PreviewEvaluateShipment(t.Context(), fixture.request)
	require.NoError(t, err)

	require.Len(t, plan.Detected, 1)
	detected := plan.Detected[0]
	assert.Same(t, existing, detected.Existing)
	assert.Equal(t, existing.ID, detected.Failure.ID)
	assert.Equal(t, int64(1_000), detected.Failure.ScheduledCutoff)
	assert.Equal(t, int64(1_360), detected.Failure.ActualArrival)
	assert.Equal(t, int64(900), existing.ScheduledCutoff)
	assert.False(t, plan.MarksDelayed)
}
