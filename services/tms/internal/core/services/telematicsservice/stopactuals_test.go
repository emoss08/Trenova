package telematicsservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type outcomeRecorder struct {
	repositories.TelematicsRepository
	recorded []*telematics.TelematicsEvent
}

func (r *outcomeRecorder) RecordStopOutcome(
	_ context.Context,
	event *telematics.TelematicsEvent,
) error {
	copied := *event
	r.recorded = append(r.recorded, &copied)
	return nil
}

type towerRecorder struct {
	upserts  []services.WatchtowerItemInput
	resolved []string
}

func (w *towerRecorder) Upsert(_ context.Context, input services.WatchtowerItemInput) {
	w.upserts = append(w.upserts, input)
}

func (w *towerRecorder) Resolve(
	_ context.Context,
	_ pagination.TenantInfo,
	kind watchtower.SourceKind,
	sourceID string,
) {
	if kind == watchtower.SourceTelematicsStopVisit {
		w.resolved = append(w.resolved, sourceID)
	}
}

type stopHarness struct {
	service *Service
	repo    *outcomeRecorder
	tower   *towerRecorder
	moves   *mocks.MockShipmentMoveService
	move    *shipment.ShipmentMove
	tenant  pagination.TenantInfo
	tractor pulid.ID
}

func newStopHarness(t *testing.T, stops ...*shipment.Stop) *stopHarness {
	t.Helper()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	tractor := pulid.MustNew("tr_")
	move := &shipment.ShipmentMove{
		ID:         pulid.MustNew("sm_"),
		ShipmentID: pulid.MustNew("shp_"),
		Status:     shipment.MoveStatusInTransit,
		Stops:      stops,
	}

	dispatch := mocks.NewMockDispatchControlRepository(t)
	dispatch.EXPECT().GetOrCreate(mock.Anything, tenant.OrgID, tenant.BuID).
		Return(&dispatchcontrol.DispatchControl{EnableAutoStopActuals: true}, nil).Maybe()

	assignments := mocks.NewMockAssignmentRepository(t)
	assignments.EXPECT().FindActiveByTractorID(mock.Anything, tenant, tractor).
		Return(&shipment.Assignment{ShipmentMoveID: move.ID}, nil).Maybe()

	moveRepo := mocks.NewMockShipmentMoveRepository(t)
	moveRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(move, nil).Maybe()

	moves := mocks.NewMockShipmentMoveService(t)
	repo := &outcomeRecorder{}
	tower := &towerRecorder{}

	return &stopHarness{
		service: &Service{
			repo:                repo,
			assignmentRepo:      assignments,
			shipmentMoveRepo:    moveRepo,
			shipmentMoveService: moves,
			dispatchControlRepo: dispatch,
			watchtower:          tower,
			l:                   zap.NewNop(),
		},
		repo:    repo,
		tower:   tower,
		moves:   moves,
		move:    move,
		tenant:  tenant,
		tractor: tractor,
	}
}

func (h *stopHarness) event(
	locationID pulid.ID,
	visit shipment.VisitKind,
	providerStop bool,
) *stopEvent {
	return &stopEvent{
		tenantInfo: h.tenant,
		record: &telematics.TelematicsEvent{
			ID:             telematics.NewEventID(),
			OrganizationID: h.tenant.OrgID,
			BusinessUnitID: h.tenant.BuID,
			OccurredAt:     1_700_000_000,
			TractorID:      h.tractor,
			LocationID:     locationID,
			AddressName:    "Acme DC",
		},
		visit:        visit,
		providerStop: providerStop,
	}
}

func openStop(sequence int64, locationID pulid.ID) *shipment.Stop {
	return &shipment.Stop{
		ID:         pulid.MustNew("stp_"),
		Sequence:   sequence,
		LocationID: locationID,
		Status:     shipment.StopStatusNew,
	}
}

func TestApplyStopEvent_RecordsTheMatchedStopAtTheEventTime(t *testing.T) {
	t.Parallel()

	location := pulid.MustNew("loc_")
	first := openStop(1, pulid.MustNew("loc_"))
	second := openStop(2, location)
	h := newStopHarness(t, first, second)
	event := h.event(location, shipment.VisitArrival, false)

	h.moves.EXPECT().RecordStopActual(mock.Anything, mock.MatchedBy(
		func(req *repositories.RecordStopActualRequest) bool {
			return req.StopID == second.ID &&
				req.Action == repositories.StopActualActionArrive &&
				req.OccurredAt != nil && *req.OccurredAt == 1_700_000_000
		},
	)).Return(h.move, nil).Once()

	h.service.applyStopEvent(t.Context(), event)

	require.Len(t, h.repo.recorded, 1)
	assert.Equal(t, telematics.StopOutcomeRecorded, h.repo.recorded[0].StopOutcome)
	assert.Equal(t, second.ID, h.repo.recorded[0].StopID)
	assert.Empty(t, h.tower.upserts)
	assert.Equal(t, []string{h.move.ID.String() + ":" + second.ID.String() + ":Arrival"}, h.tower.resolved)
}

func TestApplyStopEvent_NeverFallsBackToTheNextOpenStop(t *testing.T) {
	t.Parallel()

	t.Run("a geofence at a place not on the load is ignored", func(t *testing.T) {
		t.Parallel()
		h := newStopHarness(t, openStop(1, pulid.MustNew("loc_")))

		h.service.applyStopEvent(
			t.Context(),
			h.event(pulid.MustNew("loc_"), shipment.VisitArrival, false),
		)
		h.service.applyStopEvent(t.Context(), h.event(pulid.Nil, shipment.VisitDeparture, false))

		assert.Empty(t, h.repo.recorded)
		assert.Empty(t, h.tower.upserts)
	})

	t.Run("a provider stop event that matches no stop is left for review", func(t *testing.T) {
		t.Parallel()
		h := newStopHarness(t, openStop(1, pulid.MustNew("loc_")))

		h.service.applyStopEvent(t.Context(), h.event(pulid.Nil, shipment.VisitArrival, true))

		require.Len(t, h.repo.recorded, 1)
		recorded := h.repo.recorded[0]
		assert.Equal(t, telematics.StopOutcomeUnmatched, recorded.StopOutcome)
		assert.Equal(t, h.move.ID, recorded.ShipmentMoveID)
		assert.True(t, recorded.StopID.IsNil())
		assert.Contains(t, recorded.StopOutcomeReason, "not linked to a Trenova location")

		require.Len(t, h.tower.upserts, 1)
		item := h.tower.upserts[0]
		assert.Equal(t, watchtower.SourceTelematicsStopVisit, item.SourceKind)
		assert.Equal(t, "Arrival at Acme DC not recorded", item.Title)
		assert.Equal(t, h.move.ID, item.SubjectID)
		assert.Equal(t, int64(1_700_000_000), item.OccurredAt)
		assert.Contains(t, item.Path, h.move.ShipmentID.String())
	})
}

func TestApplyStopEvent_RefusedStopIsLeftForReview(t *testing.T) {
	t.Parallel()

	location := pulid.MustNew("loc_")
	stop := openStop(2, location)
	h := newStopHarness(t, openStop(1, pulid.MustNew("loc_")), stop)

	h.moves.EXPECT().RecordStopActual(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewBusinessError("Complete the earlier stops on this load first")).
		Once()

	h.service.applyStopEvent(t.Context(), h.event(location, shipment.VisitArrival, false))

	require.Len(t, h.repo.recorded, 1)
	recorded := h.repo.recorded[0]
	assert.Equal(t, telematics.StopOutcomeRefused, recorded.StopOutcome)
	assert.Equal(t, stop.ID, recorded.StopID)
	assert.Contains(t, recorded.StopOutcomeReason, "Complete the earlier stops on this load first")
	assert.Contains(t, recorded.StopOutcomeReason, "stop 2")

	require.Len(t, h.tower.upserts, 1)
	assert.Equal(
		t,
		h.move.ID.String()+":"+stop.ID.String()+":Arrival",
		h.tower.upserts[0].SourceID,
	)
}

func TestApplyStopEvent_RepeatSignalsAreRecordedAsDuplicates(t *testing.T) {
	t.Parallel()

	location := pulid.MustNew("loc_")
	at := int64(10)
	onSite := openStop(1, location)
	onSite.ActualArrival = &at
	h := newStopHarness(t, onSite)

	h.service.applyStopEvent(t.Context(), h.event(location, shipment.VisitArrival, true))

	require.Len(t, h.repo.recorded, 1)
	assert.Equal(t, telematics.StopOutcomeDuplicate, h.repo.recorded[0].StopOutcome)
	assert.Empty(t, h.tower.upserts)
	assert.Empty(t, h.tower.resolved)
}

func TestApplyStopEvent_DoesNothingWhenAutoStopActualsAreOff(t *testing.T) {
	t.Parallel()

	h := newStopHarness(t, openStop(1, pulid.MustNew("loc_")))
	dispatch := mocks.NewMockDispatchControlRepository(t)
	dispatch.EXPECT().GetOrCreate(mock.Anything, h.tenant.OrgID, h.tenant.BuID).
		Return(&dispatchcontrol.DispatchControl{}, nil).Once()
	h.service.dispatchControlRepo = dispatch

	h.service.applyStopEvent(t.Context(), h.event(pulid.Nil, shipment.VisitArrival, true))

	assert.Empty(t, h.repo.recorded)
	assert.Empty(t, h.tower.upserts)
}

type recordedWaitNotices struct {
	services.AgentWaitNotifier
	events []*services.AgentEvent
}

func (r *recordedWaitNotices) NotifyEvent(_ context.Context, event *services.AgentEvent) {
	r.events = append(r.events, event)
}

func TestApplyStopEvent_ReachesTheWaitsWhenAutoStopActualsAreOff(t *testing.T) {
	t.Parallel()

	location := pulid.MustNew("loc_")
	stop := openStop(1, location)
	h := newStopHarness(t, stop)
	dispatch := mocks.NewMockDispatchControlRepository(t)
	dispatch.EXPECT().GetOrCreate(mock.Anything, h.tenant.OrgID, h.tenant.BuID).
		Return(&dispatchcontrol.DispatchControl{}, nil).Once()
	h.service.dispatchControlRepo = dispatch
	waits := &recordedWaitNotices{}
	h.service.waits = waits

	h.service.applyStopEvent(t.Context(), h.event(location, shipment.VisitArrival, false))

	require.Len(t, waits.events, 1)
	assert.Equal(t, h.move.ID, waits.events[0].SubjectID)
	assert.Equal(t, []pulid.ID{stop.ID}, waits.events[0].Related)
	assert.Contains(t, waits.events[0].Detail, "entered the geofence")
	assert.Empty(t, h.repo.recorded, "no stop actual is recorded when the setting is off")
}

func TestApplyStopEvent_AVisitAtNoStopOfTheMoveReachesNoWait(t *testing.T) {
	t.Parallel()

	h := newStopHarness(t, openStop(1, pulid.MustNew("loc_")))
	dispatch := mocks.NewMockDispatchControlRepository(t)
	dispatch.EXPECT().GetOrCreate(mock.Anything, h.tenant.OrgID, h.tenant.BuID).
		Return(&dispatchcontrol.DispatchControl{}, nil).Once()
	h.service.dispatchControlRepo = dispatch
	waits := &recordedWaitNotices{}
	h.service.waits = waits

	h.service.applyStopEvent(t.Context(), h.event(pulid.MustNew("loc_"), shipment.VisitArrival, false))

	assert.Empty(t, waits.events)
}
