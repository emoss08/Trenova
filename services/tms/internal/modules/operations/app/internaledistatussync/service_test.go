package internaledistatussync

import (
	"context"
	"maps"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edipayload"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edisync"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/app/internaledilifecycle"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentinvalidation"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopactual"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentmoveports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/orderportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestObserverAppliesTargetStatusToSourceShipment(t *testing.T) {
	t.Parallel()

	fixture := newObserverFixture(t)
	fixture.source.Status = shipmentvocab.StatusNew
	fixture.target.Status = shipmentvocab.StatusInTransit

	err := fixture.observer.OnShipmentEvent(
		t.Context(),
		targetStatusEvent(fixture, shipmentvocab.StatusInTransit),
	)

	require.NoError(t, err)
	require.Len(t, fixture.transferRepo.created, 1)
	require.Equal(t, edisync.TransferChangeStatusApplied, fixture.transferRepo.created[0].Status)
	require.Equal(
		t,
		edisync.TransferChangeDirectionTargetToSource,
		fixture.transferRepo.created[0].Direction,
	)
	require.Equal(t, shipmentvocab.StatusInTransit, fixture.source.Status)
	require.Len(t, fixture.eventRepo.inserted, 1)
	require.Equal(
		t,
		fixture.event.ID.String(),
		fixture.eventRepo.inserted[0].Metadata[edisync.InternalEDIMirroredFromEventIDKey],
	)
	require.Len(t, fixture.realtime.published, 3)
}

func TestObserverDuplicateExecutionIsIdempotent(t *testing.T) {
	t.Parallel()

	fixture := newObserverFixture(t)
	event := targetStatusEvent(fixture, shipmentvocab.StatusInTransit)
	key := fixture.idempotencyKey(
		event,
		edisync.TransferChangeDirectionTargetToSource,
		edisync.TransferChangeTypeShipmentStatus214,
	)
	fixture.transferRepo.existing[key] = &edisync.TransferChange{
		ID:             pulid.MustNew("editc_"),
		BusinessUnitID: fixture.link.BusinessUnitID,
		ShipmentLinkID: fixture.link.ID,
		Direction:      edisync.TransferChangeDirectionTargetToSource,
		ChangeType:     edisync.TransferChangeTypeShipmentStatus214,
		IdempotencyKey: key,
		Status:         edisync.TransferChangeStatusApplied,
	}

	err := fixture.observer.OnShipmentEvent(t.Context(), event)

	require.NoError(t, err)
	require.Empty(t, fixture.shipmentRepo.updates)
	require.Empty(t, fixture.eventRepo.inserted)
	require.Empty(t, fixture.transferRepo.created)
}

func TestObserverAutoAppliesLifecycleActualsThroughStopActuals(t *testing.T) {
	t.Parallel()

	fixture := newObserverFixture(t)
	fixture.installLifecycleStops()
	arrival := int64(1_000)
	departure := int64(1_100)
	fixture.target.Moves[0].Stops[0].ActualArrival = &arrival
	fixture.target.Moves[0].Stops[0].ActualDeparture = &departure

	err := fixture.observer.OnShipmentEvent(
		t.Context(),
		targetStatusEvent(fixture, shipmentvocab.StatusInTransit),
	)

	require.NoError(t, err)
	require.Empty(t, fixture.shipmentRepo.updates)
	require.Len(t, fixture.stopActuals.calls, 1)
	call := fixture.stopActuals.calls[0]
	require.Equal(t, fixture.source.Moves[0].ID, call.req.MoveID)
	require.Equal(t, fixture.link.SourceOrganizationID, call.req.TenantInfo.OrgID)
	require.Equal(t, []stopactual.Change{{
		StopID:    fixture.source.Moves[0].Stops[0].ID,
		Arrival:   &arrival,
		Departure: &departure,
	}}, call.applied)
	synced := fixture.source.Moves[0].Stops[0]
	require.Equal(t, shipmentvocab.StopStatusCompleted, synced.Status)
	require.Equal(t, arrival, *synced.ActualArrival)
	require.Equal(t, departure, *synced.ActualDeparture)
	require.Equal(t, shipmentvocab.StopStatusNew, fixture.source.Moves[0].Stops[1].Status)
	require.Len(t, fixture.transferRepo.created, 1)
	change := fixture.transferRepo.created[0]
	require.Equal(t, edisync.TransferChangeTypeShipmentLifecycle214, change.ChangeType)
	require.Equal(t, edisync.TransferChangeStatusApplied, change.Status)
	require.NotEmpty(t, change.Payload["matchedStopActualDiffs"])
	require.Len(t, fixture.eventRepo.inserted, 1)
	require.Equal(
		t,
		"Lifecycle synced from internal EDI 214",
		fixture.eventRepo.inserted[0].Summary,
	)
}

func TestObserverManualReviewLifecycleDoesNotMutateShipment(t *testing.T) {
	t.Parallel()

	fixture := newObserverFixture(t)
	fixture.link.SyncPolicy = edisync.ShipmentSyncPolicyManualReview
	fixture.installLifecycleStops()
	arrival := int64(1_000)
	fixture.target.Moves[0].Stops[0].ActualArrival = &arrival

	err := fixture.observer.OnShipmentEvent(
		t.Context(),
		targetStatusEvent(fixture, shipmentvocab.StatusInTransit),
	)

	require.NoError(t, err)
	require.Empty(t, fixture.stopActuals.calls)
	require.Empty(t, fixture.eventRepo.inserted)
	require.Len(t, fixture.transferRepo.created, 1)
	require.Equal(
		t,
		edisync.TransferChangeStatusPendingReview,
		fixture.transferRepo.created[0].Status,
	)
	require.Equal(
		t,
		edisync.TransferChangeTypeShipmentLifecycle214,
		fixture.transferRepo.created[0].ChangeType,
	)
}

func TestObserverLifecycleMappingConflictCreatesPendingReview(t *testing.T) {
	t.Parallel()

	fixture := newObserverFixture(t)
	fixture.installLifecycleStops()
	fixture.source.Moves[0].Stops = fixture.source.Moves[0].Stops[:1]
	arrival := int64(1_000)
	fixture.target.Moves[0].Stops[1].ActualArrival = &arrival

	err := fixture.observer.OnShipmentEvent(
		t.Context(),
		targetStatusEvent(fixture, shipmentvocab.StatusInTransit),
	)

	require.NoError(t, err)
	require.Empty(t, fixture.stopActuals.calls)
	require.Empty(t, fixture.eventRepo.inserted)
	require.Len(t, fixture.transferRepo.created, 1)
	change := fixture.transferRepo.created[0]
	require.Equal(t, edisync.TransferChangeStatusPendingReview, change.Status)
	require.Equal(t, edisync.TransferChangeConflictConflict, change.ConflictStatus)
	require.NotEmpty(t, change.ConflictReason)
	require.NotEmpty(t, change.Payload["conflicts"])
}

func TestObserverIgnoresMirroredEvents(t *testing.T) {
	t.Parallel()

	fixture := newObserverFixture(t)
	event := targetStatusEvent(fixture, shipmentvocab.StatusInTransit)
	event.Metadata[edisync.InternalEDIMirroredFromEventIDKey] = pulid.MustNew("se_").String()

	err := fixture.observer.OnShipmentEvent(t.Context(), event)

	require.NoError(t, err)
	require.Zero(t, fixture.linkRepo.calls)
	require.Empty(t, fixture.transferRepo.created)
}

func TestObserverDoesNotApplyInactiveOrReadOnlyLinks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status edisync.ShipmentLinkStatus
		policy edisync.ShipmentSyncPolicy
	}{
		{
			name:   "suspended",
			status: edisync.ShipmentLinkStatusSuspended,
			policy: edisync.ShipmentSyncPolicyAutoOperational,
		},
		{
			name:   "closed",
			status: edisync.ShipmentLinkStatusClosed,
			policy: edisync.ShipmentSyncPolicyAutoOperational,
		},
		{
			name:   "read only",
			status: edisync.ShipmentLinkStatusActive,
			policy: edisync.ShipmentSyncPolicyReadOnly,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := newObserverFixture(t)
			fixture.link.Status = tt.status
			fixture.link.SyncPolicy = tt.policy

			err := fixture.observer.OnShipmentEvent(
				t.Context(),
				targetStatusEvent(fixture, shipmentvocab.StatusInTransit),
			)

			require.NoError(t, err)
			require.Empty(t, fixture.shipmentRepo.updates)
			require.Empty(t, fixture.eventRepo.inserted)
			require.Len(t, fixture.transferRepo.created, 1)
			require.Equal(
				t,
				edisync.TransferChangeStatusIgnored,
				fixture.transferRepo.created[0].Status,
			)
		})
	}
}

func TestObserverInvalidTransitionCreatesPendingConflict(t *testing.T) {
	t.Parallel()

	fixture := newObserverFixture(t)
	fixture.source.Status = shipmentvocab.StatusInvoiced
	fixture.target.Status = shipmentvocab.StatusCompleted

	err := fixture.observer.OnShipmentEvent(
		t.Context(),
		targetStatusEvent(fixture, shipmentvocab.StatusCompleted),
	)

	require.NoError(t, err)
	require.Empty(t, fixture.shipmentRepo.updates)
	require.Empty(t, fixture.eventRepo.inserted)
	require.Len(t, fixture.transferRepo.created, 1)
	change := fixture.transferRepo.created[0]
	require.Equal(t, edisync.TransferChangeStatusPendingReview, change.Status)
	require.Equal(t, edisync.TransferChangeConflictConflict, change.ConflictStatus)
	require.NotEmpty(t, change.ConflictReason)
}

func TestObserverAutoAppliesSourceCancellationToTargetShipment(t *testing.T) {
	t.Parallel()

	fixture := newObserverFixture(t)
	fixture.source.Status = shipmentvocab.StatusCanceled
	fixture.target.Status = shipmentvocab.StatusInTransit

	err := fixture.observer.OnShipmentEvent(
		t.Context(),
		sourceCancelEvent(fixture, "Customer canceled"),
	)

	require.NoError(t, err)
	require.Empty(t, fixture.shipmentRepo.updates)
	require.Len(t, fixture.shipmentRepo.cancels, 1)
	require.Equal(t, fixture.link.TargetShipmentID, fixture.shipmentRepo.cancels[0].ShipmentID)
	require.Equal(t, "Customer canceled", fixture.shipmentRepo.cancels[0].CancelReason)
	require.Equal(t, shipmentvocab.StatusCanceled, fixture.target.Status)
	require.Len(t, fixture.transferRepo.created, 1)
	change := fixture.transferRepo.created[0]
	require.Equal(t, edisync.TransferChangeTypeShipmentCancel214, change.ChangeType)
	require.Equal(t, edisync.TransferChangeStatusApplied, change.Status)
	require.Equal(
		t,
		"A7",
		change.Payload["shipmentStatus"].(edipayload.ShipmentStatusPayload).StatusCode,
	)
	require.Len(t, fixture.eventRepo.inserted, 1)
	require.Equal(t, shipmentevent.TypeShipmentCanceled, fixture.eventRepo.inserted[0].Type)
	require.Equal(
		t,
		fixture.event.ID.String(),
		fixture.eventRepo.inserted[0].Metadata[edisync.InternalEDIMirroredFromEventIDKey],
	)
	require.Len(t, fixture.realtime.published, 3)
}

func TestObserverAutoAppliesTargetCancellationToSourceShipment(t *testing.T) {
	t.Parallel()

	fixture := newObserverFixture(t)
	fixture.source.Status = shipmentvocab.StatusAssigned
	fixture.target.Status = shipmentvocab.StatusCanceled

	err := fixture.observer.OnShipmentEvent(
		t.Context(),
		targetCancelEvent(fixture, "Carrier canceled"),
	)

	require.NoError(t, err)
	require.Empty(t, fixture.shipmentRepo.updates)
	require.Len(t, fixture.shipmentRepo.cancels, 1)
	require.Equal(t, fixture.link.SourceShipmentID, fixture.shipmentRepo.cancels[0].ShipmentID)
	require.Equal(t, shipmentvocab.StatusCanceled, fixture.source.Status)
	require.Len(t, fixture.transferRepo.created, 1)
	require.Equal(
		t,
		edisync.TransferChangeDirectionTargetToSource,
		fixture.transferRepo.created[0].Direction,
	)
}

func TestObserverManualReviewCancellationDoesNotMutateShipment(t *testing.T) {
	t.Parallel()

	fixture := newObserverFixture(t)
	fixture.link.SyncPolicy = edisync.ShipmentSyncPolicyManualReview
	fixture.source.Status = shipmentvocab.StatusCanceled
	fixture.target.Status = shipmentvocab.StatusInTransit

	err := fixture.observer.OnShipmentEvent(
		t.Context(),
		sourceCancelEvent(fixture, "Manual review"),
	)

	require.NoError(t, err)
	require.Empty(t, fixture.shipmentRepo.updates)
	require.Empty(t, fixture.shipmentRepo.cancels)
	require.Empty(t, fixture.eventRepo.inserted)
	require.Len(t, fixture.transferRepo.created, 1)
	require.Equal(
		t,
		edisync.TransferChangeStatusPendingReview,
		fixture.transferRepo.created[0].Status,
	)
}

func TestObserverIgnoresAlreadyCanceledOppositeShipment(t *testing.T) {
	t.Parallel()

	fixture := newObserverFixture(t)
	fixture.source.Status = shipmentvocab.StatusCanceled
	fixture.target.Status = shipmentvocab.StatusCanceled

	err := fixture.observer.OnShipmentEvent(
		t.Context(),
		sourceCancelEvent(fixture, "Already canceled"),
	)

	require.NoError(t, err)
	require.Empty(t, fixture.shipmentRepo.cancels)
	require.Empty(t, fixture.eventRepo.inserted)
	require.Len(t, fixture.transferRepo.created, 1)
	require.Equal(t, edisync.TransferChangeStatusIgnored, fixture.transferRepo.created[0].Status)
}

func TestObserverInvalidCancellationTransitionCreatesPendingConflict(t *testing.T) {
	t.Parallel()

	fixture := newObserverFixture(t)
	fixture.source.Status = shipmentvocab.StatusCanceled
	fixture.target.Status = shipmentvocab.StatusCompleted

	err := fixture.observer.OnShipmentEvent(t.Context(), sourceCancelEvent(fixture, "Invalid"))

	require.NoError(t, err)
	require.Empty(t, fixture.shipmentRepo.cancels)
	require.Empty(t, fixture.eventRepo.inserted)
	require.Len(t, fixture.transferRepo.created, 1)
	change := fixture.transferRepo.created[0]
	require.Equal(t, edisync.TransferChangeStatusPendingReview, change.Status)
	require.Equal(t, edisync.TransferChangeConflictConflict, change.ConflictStatus)
	require.NotEmpty(t, change.ConflictReason)
}

func TestObserverDuplicateCancellationExecutionIsIdempotent(t *testing.T) {
	t.Parallel()

	fixture := newObserverFixture(t)
	event := sourceCancelEvent(fixture, "Duplicate")
	key := fixture.idempotencyKey(
		event,
		edisync.TransferChangeDirectionSourceToTarget,
		edisync.TransferChangeTypeShipmentCancel214,
	)
	fixture.transferRepo.existing[key] = &edisync.TransferChange{
		ID:             pulid.MustNew("editc_"),
		BusinessUnitID: fixture.link.BusinessUnitID,
		ShipmentLinkID: fixture.link.ID,
		Direction:      edisync.TransferChangeDirectionSourceToTarget,
		ChangeType:     edisync.TransferChangeTypeShipmentCancel214,
		IdempotencyKey: key,
		Status:         edisync.TransferChangeStatusApplied,
	}

	err := fixture.observer.OnShipmentEvent(t.Context(), event)

	require.NoError(t, err)
	require.Empty(t, fixture.shipmentRepo.cancels)
	require.Empty(t, fixture.eventRepo.inserted)
	require.Empty(t, fixture.transferRepo.created)
}

type observerFixture struct {
	observer     *Observer
	link         *edisync.ShipmentLink
	source       *shipment.Shipment
	target       *shipment.Shipment
	event        *shipmentevent.Event
	shipmentRepo *fakeShipmentRepository
	linkRepo     *fakeShipmentLinkRepository
	transferRepo *fakeTransferChangeRepository
	eventRepo    *fakeShipmentEventRepository
	realtime     *fakeRealtimeService
	stopActuals  *fakeStopActuals
}

func newObserverFixture(t *testing.T) *observerFixture {
	t.Helper()

	sourceOrgID := pulid.MustNew("org_")
	targetOrgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	sourceID := pulid.MustNew("shp_")
	targetID := pulid.MustNew("shp_")
	link := &edisync.ShipmentLink{
		ID:                   pulid.MustNew("edislink_"),
		BusinessUnitID:       buID,
		SourceOrganizationID: sourceOrgID,
		TargetOrganizationID: targetOrgID,
		SourceShipmentID:     sourceID,
		TargetShipmentID:     targetID,
		SyncPolicy:           edisync.ShipmentSyncPolicyAutoOperational,
		Status:               edisync.ShipmentLinkStatusActive,
	}
	source := &shipment.Shipment{
		ID:             sourceID,
		OrganizationID: sourceOrgID,
		BusinessUnitID: buID,
		ProNumber:      "SOURCE-1",
		Status:         shipmentvocab.StatusNew,
		Version:        2,
	}
	target := &shipment.Shipment{
		ID:             targetID,
		OrganizationID: targetOrgID,
		BusinessUnitID: buID,
		ProNumber:      "TARGET-1",
		Status:         shipmentvocab.StatusInTransit,
		Version:        4,
	}

	shipmentRepo := &fakeShipmentRepository{shipments: map[pulid.ID]*shipment.Shipment{
		sourceID: source,
		targetID: target,
	}}
	linkRepo := &fakeShipmentLinkRepository{links: []*edisync.ShipmentLink{link}}
	transferRepo := &fakeTransferChangeRepository{existing: map[string]*edisync.TransferChange{}}
	eventRepo := &fakeShipmentEventRepository{}
	realtime := &fakeRealtimeService{}
	stopActuals := &fakeStopActuals{shipments: shipmentRepo.shipments}

	derivation := orderportsmock.NewMockOrderDerivationService(t)
	derivation.EXPECT().
		RecomputeForShipment(mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Maybe()

	fixture := &observerFixture{
		link:         link,
		source:       source,
		target:       target,
		shipmentRepo: shipmentRepo,
		linkRepo:     linkRepo,
		transferRepo: transferRepo,
		eventRepo:    eventRepo,
		realtime:     realtime,
		stopActuals:  stopActuals,
	}
	coordinator := shipmentstate.NewCoordinatorWithClock(func() int64 { return 2_000 })
	fixture.observer = &Observer{
		l:                  zap.NewNop(),
		db:                 rollbackDB{transfers: transferRepo},
		shipmentRepo:       shipmentRepo,
		shipmentEventRepo:  eventRepo,
		shipmentLinkRepo:   linkRepo,
		transferChangeRepo: transferRepo,
		realtime:           realtime,
		invalidator: shipmentinvalidation.New(shipmentinvalidation.Params{
			Realtime: realtime,
			Epochs:   &mocks.NoopShipmentBoardEpochBumper{},
			Logger:   zap.NewNop(),
		}),
		orderDerivation: derivation,
		coordinator:     coordinator,
	}
	fixture.observer.SetStopActuals(stopActuals)
	return fixture
}

func targetStatusEvent(fixture *observerFixture, status shipmentvocab.Status) *shipmentevent.Event {
	fixture.event = &shipmentevent.Event{
		ID:             pulid.MustNew("se_"),
		OrganizationID: fixture.link.TargetOrganizationID,
		BusinessUnitID: fixture.link.BusinessUnitID,
		ShipmentID:     fixture.link.TargetShipmentID,
		Type:           shipmentevent.TypeStatusChanged,
		Summary:        "Status updated",
		Metadata: map[string]any{
			"previousStatus": string(shipmentvocab.StatusAssigned),
			"newStatus":      string(status),
		},
		OccurredAt: 1780418506,
	}
	return fixture.event
}

func sourceCancelEvent(fixture *observerFixture, reason string) *shipmentevent.Event {
	fixture.event = &shipmentevent.Event{
		ID:             pulid.MustNew("se_"),
		OrganizationID: fixture.link.SourceOrganizationID,
		BusinessUnitID: fixture.link.BusinessUnitID,
		ShipmentID:     fixture.link.SourceShipmentID,
		Type:           shipmentevent.TypeShipmentCanceled,
		Summary:        "Shipment canceled",
		ActorType:      shipmentevent.ActorUser,
		ActorID:        pulid.MustNew("usr_"),
		Metadata: map[string]any{
			"reason": reason,
		},
		OccurredAt: 1780418506,
	}
	return fixture.event
}

func targetCancelEvent(fixture *observerFixture, reason string) *shipmentevent.Event {
	fixture.event = &shipmentevent.Event{
		ID:             pulid.MustNew("se_"),
		OrganizationID: fixture.link.TargetOrganizationID,
		BusinessUnitID: fixture.link.BusinessUnitID,
		ShipmentID:     fixture.link.TargetShipmentID,
		Type:           shipmentevent.TypeShipmentCanceled,
		Summary:        "Shipment canceled",
		Metadata: map[string]any{
			"reason": reason,
		},
		OccurredAt: 1780418506,
	}
	return fixture.event
}

func (f *observerFixture) installLifecycleStops() {
	f.source.Moves = []*shipment.ShipmentMove{
		lifecycleMove(f.source, shipmentvocab.MoveStatusNew),
	}
	f.target.Moves = []*shipment.ShipmentMove{
		lifecycleMove(f.target, shipmentvocab.MoveStatusNew),
	}
}

func lifecycleMove(
	entity *shipment.Shipment,
	status shipmentvocab.MoveStatus,
) *shipment.ShipmentMove {
	moveID := pulid.MustNew("sm_")
	return &shipment.ShipmentMove{
		ID:             moveID,
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
		ShipmentID:     entity.ID,
		Status:         status,
		Sequence:       0,
		Stops: []*shipment.Stop{
			lifecycleStop(entity, moveID, shipmentvocab.StopTypePickup, 0),
			lifecycleStop(entity, moveID, shipmentvocab.StopTypeDelivery, 1),
		},
	}
}

func lifecycleStop(
	entity *shipment.Shipment,
	moveID pulid.ID,
	stopType shipmentvocab.StopType,
	sequence int64,
) *shipment.Stop {
	return &shipment.Stop{
		ID:                   pulid.MustNew("stp_"),
		OrganizationID:       entity.OrganizationID,
		BusinessUnitID:       entity.BusinessUnitID,
		ShipmentMoveID:       moveID,
		LocationID:           pulid.MustNew("loc_"),
		Status:               shipmentvocab.StopStatusNew,
		Type:                 stopType,
		Sequence:             sequence,
		ScheduledWindowStart: 500 + sequence,
	}
}

func (f *observerFixture) idempotencyKey(
	event *shipmentevent.Event,
	direction edisync.TransferChangeDirection,
	changeType string,
) string {
	return f.link.ID.String() + ":" + event.ID.String() + ":" + string(direction) + ":" + changeType
}

type fakeShipmentRepository struct {
	shipments map[pulid.ID]*shipment.Shipment
	updates   []*shipmentports.UpdateShipmentStatusRequest
	cancels   []*shipmentports.CancelShipmentRequest
}

func (r *fakeShipmentRepository) GetByID(
	_ context.Context,
	req *shipmentports.GetShipmentByIDRequest,
) (*shipment.Shipment, error) {
	return r.shipments[req.ID], nil
}

func (r *fakeShipmentRepository) UpdateStatus(
	_ context.Context,
	req *shipmentports.UpdateShipmentStatusRequest,
) (*shipment.Shipment, error) {
	r.updates = append(r.updates, req)
	entity := r.shipments[req.ShipmentID]
	entity.Status = req.Status
	entity.Version++
	return entity, nil
}

func (r *fakeShipmentRepository) Cancel(
	_ context.Context,
	req *shipmentports.CancelShipmentRequest,
) (*shipment.Shipment, error) {
	r.cancels = append(r.cancels, req)
	entity := r.shipments[req.ShipmentID]
	entity.Status = shipmentvocab.StatusCanceled
	entity.CancelReason = req.CancelReason
	entity.CanceledAt = &req.CanceledAt
	entity.CanceledByID = req.CanceledByID
	entity.Version++
	return entity, nil
}

type fakeShipmentLinkRepository struct {
	links []*edisync.ShipmentLink
	calls int
}

func (r *fakeShipmentLinkRepository) GetShipmentLinksByShipmentID(
	context.Context,
	editenderports.GetEDIShipmentLinksByShipmentIDRequest,
) ([]*edisync.ShipmentLink, error) {
	r.calls++
	return r.links, nil
}

type fakeTransferChangeRepository struct {
	existing map[string]*edisync.TransferChange
	created  []*edisync.TransferChange
}

func (r *fakeTransferChangeRepository) CreateTransferChangeIdempotent(
	_ context.Context,
	entity *edisync.TransferChange,
) (*editenderports.CreateEDITransferChangeIdempotentResult, error) {
	if existing := r.existing[entity.IdempotencyKey]; existing != nil {
		return &editenderports.CreateEDITransferChangeIdempotentResult{
			TransferChange: existing,
			Created:        false,
		}, nil
	}
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("editc_")
	}
	r.existing[entity.IdempotencyKey] = entity
	r.created = append(r.created, entity)
	return &editenderports.CreateEDITransferChangeIdempotentResult{
		TransferChange: entity,
		Created:        true,
	}, nil
}

type fakeShipmentEventRepository struct {
	inserted []*shipmentevent.Event
}

func (r *fakeShipmentEventRepository) Insert(
	_ context.Context,
	entity *shipmentevent.Event,
) error {
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("se_")
	}
	r.inserted = append(r.inserted, entity)
	return nil
}

type fakeRealtimeService struct {
	published []*realtimeports.PublishResourceInvalidationRequest
}

func (s *fakeRealtimeService) PublishResourceInvalidation(
	_ context.Context,
	req *realtimeports.PublishResourceInvalidationRequest,
) error {
	s.published = append(s.published, req)
	return nil
}

func TestStopActualEventsSyncTheLinkedShipmentsLifecycle(t *testing.T) {
	t.Parallel()

	for _, eventType := range []shipmentevent.Type{
		shipmentevent.TypeStopArrived,
		shipmentevent.TypeStopDeparted,
		shipmentevent.TypeStopArrivalCorrected,
		shipmentevent.TypeStopDepartureCorrected,
	} {
		event := &shipmentevent.Event{Type: eventType}
		require.True(t, isLifecycleCandidate(event), eventType)
		require.True(t, eventCanSync(event), eventType)
	}
}

func TestObserverSyncedArrivalReconcilesTheLinkedStopOnce(t *testing.T) {
	t.Parallel()

	fixture := newObserverFixture(t)
	fixture.installLifecycleStops()
	arrival := int64(1_000)
	fixture.source.Moves[0].Stops[0].ActualArrival = &arrival
	event := stopEvent(fixture, fixture.source, shipmentevent.TypeStopArrived)

	require.NoError(t, fixture.observer.OnShipmentEvent(t.Context(), event))

	require.Len(t, fixture.stopActuals.calls, 1)
	call := fixture.stopActuals.calls[0]
	linked := fixture.target.Moves[0].Stops[0]
	require.Equal(t, fixture.target.Moves[0].ID, call.req.MoveID)
	require.Equal(t, fixture.link.TargetOrganizationID, call.req.TenantInfo.OrgID)
	require.Equal(t, fixture.link.BusinessUnitID, call.req.TenantInfo.BuID)
	require.Nil(t, call.before[linked.ID].arrival)
	require.Equal(t, []stopactual.Change{{StopID: linked.ID, Arrival: &arrival}}, call.applied)
	require.Equal(t, arrival, *linked.ActualArrival)
	require.Equal(t, shipmentvocab.StopStatusInTransit, linked.Status)
	require.NotNil(t, call.req.Origin)
	require.Equal(t, internaledilifecycle.ActorLabel, call.req.Origin.ActorLabel)
	require.Equal(t, event.CorrelationID, call.req.Origin.CorrelationID)
	require.Equal(
		t,
		event.ID.String(),
		call.req.Origin.Metadata[edisync.InternalEDIMirroredFromEventIDKey],
	)
	require.Equal(
		t,
		fixture.link.ID.String(),
		call.req.Origin.Metadata[edisync.InternalEDIShipmentLinkIDKey],
	)
	require.Len(t, fixture.transferRepo.created, 1)
	change := fixture.transferRepo.created[0]
	require.Equal(t, edisync.TransferChangeTypeShipmentLifecycle214, change.ChangeType)
	require.Equal(t, edisync.TransferChangeStatusApplied, change.Status)
	require.Len(t, fixture.eventRepo.inserted, 1)

	require.NoError(t, fixture.observer.OnShipmentEvent(t.Context(), event))
	require.NoError(t, fixture.observer.OnShipmentEvent(
		t.Context(),
		stopEvent(fixture, fixture.source, shipmentevent.TypeStopArrived),
	))

	require.Len(t, fixture.stopActuals.calls, 1, "a repeated sync reconciles nothing")
	require.Len(t, fixture.transferRepo.created, 1, "and records no second change")
	require.Len(t, fixture.eventRepo.inserted, 1)
}

func TestObserverSyncedCorrectionReconcilesTheCorrectedTime(t *testing.T) {
	t.Parallel()

	fixture := newObserverFixture(t)
	fixture.installLifecycleStops()
	first := int64(1_000)
	fixture.source.Moves[0].Stops[0].ActualArrival = &first
	require.NoError(t, fixture.observer.OnShipmentEvent(
		t.Context(),
		stopEvent(fixture, fixture.source, shipmentevent.TypeStopArrived),
	))

	corrected := int64(1_200)
	fixture.source.Moves[0].Stops[0].ActualArrival = &corrected
	require.NoError(t, fixture.observer.OnShipmentEvent(
		t.Context(),
		stopEvent(fixture, fixture.source, shipmentevent.TypeStopArrivalCorrected),
	))

	require.Len(t, fixture.stopActuals.calls, 2)
	call := fixture.stopActuals.calls[1]
	linked := fixture.target.Moves[0].Stops[0]
	require.Equal(t, first, *call.before[linked.ID].arrival)
	require.Equal(t, []stopactual.Change{{StopID: linked.ID, Arrival: &corrected}}, call.applied)
	require.Equal(t, corrected, *linked.ActualArrival)
	require.Len(t, fixture.transferRepo.created, 2)
}

func TestObserverDoesNotEchoSyncedStopEventsBack(t *testing.T) {
	t.Parallel()

	fixture := newObserverFixture(t)
	fixture.installLifecycleStops()
	arrival := int64(1_000)
	fixture.source.Moves[0].Stops[0].ActualArrival = &arrival
	require.NoError(t, fixture.observer.OnShipmentEvent(
		t.Context(),
		stopEvent(fixture, fixture.source, shipmentevent.TypeStopArrived),
	))
	require.Len(t, fixture.stopActuals.calls, 1)
	origin := fixture.stopActuals.calls[0].req.Origin
	linkCalls := fixture.linkRepo.calls

	echo := stopEvent(fixture, fixture.target, shipmentevent.TypeStopArrived)
	echo.ActorLabel = origin.ActorLabel
	maps.Copy(echo.Metadata, origin.Metadata)
	require.NoError(t, fixture.observer.OnShipmentEvent(t.Context(), echo))

	require.Equal(t, linkCalls, fixture.linkRepo.calls, "a synced stop event is not synced back")
	require.Len(t, fixture.stopActuals.calls, 1)
	require.Len(t, fixture.transferRepo.created, 1)

	require.NoError(t, fixture.observer.OnShipmentEvent(
		t.Context(),
		stopEvent(fixture, fixture.target, shipmentevent.TypeStopArrived),
	))

	require.Len(t, fixture.stopActuals.calls, 1, "an untagged stop event finds nothing to sync")
	require.Len(t, fixture.transferRepo.created, 1)
	require.Equal(t, arrival, *fixture.source.Moves[0].Stops[0].ActualArrival)
}

func TestObserverDoesNotClearTimesTheLinkedShipmentAlreadyHas(t *testing.T) {
	t.Parallel()

	fixture := newObserverFixture(t)
	fixture.installLifecycleStops()
	arrival := int64(1_000)
	departure := int64(1_100)
	fixture.source.Moves[0].Stops[0].ActualArrival = &arrival
	linked := fixture.target.Moves[0].Stops[0]
	linkedArrival := arrival
	linked.ActualArrival = &linkedArrival
	linked.ActualDeparture = &departure

	require.NoError(t, fixture.observer.OnShipmentEvent(
		t.Context(),
		stopEvent(fixture, fixture.source, shipmentevent.TypeStopArrived),
	))

	require.Empty(t, fixture.stopActuals.calls)
	require.Empty(t, fixture.transferRepo.created)
	require.Equal(t, departure, *linked.ActualDeparture)
}

func TestObserverRefusedStopActualsBecomeAPendingConflict(t *testing.T) {
	t.Parallel()

	fixture := newObserverFixture(t)
	fixture.installLifecycleStops()
	arrival := int64(1_000)
	fixture.source.Moves[0].Stops[0].ActualArrival = &arrival
	fixture.stopActuals.err = errortypes.NewBusinessError("This load has been canceled")

	require.NoError(t, fixture.observer.OnShipmentEvent(
		t.Context(),
		stopEvent(fixture, fixture.source, shipmentevent.TypeStopArrived),
	))

	require.Empty(t, fixture.eventRepo.inserted)
	require.Nil(t, fixture.target.Moves[0].Stops[0].ActualArrival)
	require.Len(t, fixture.transferRepo.created, 1)
	change := fixture.transferRepo.created[0]
	require.Equal(t, edisync.TransferChangeTypeShipmentLifecycle214, change.ChangeType)
	require.Equal(t, edisync.TransferChangeStatusPendingReview, change.Status)
	require.Equal(t, edisync.TransferChangeConflictConflict, change.ConflictStatus)
	require.Contains(t, change.ConflictReason, "This load has been canceled")
	require.Nil(t, change.AppliedAt)
}

func stopEvent(
	fixture *observerFixture,
	entity *shipment.Shipment,
	eventType shipmentevent.Type,
) *shipmentevent.Event {
	move := entity.Moves[0]
	event := &shipmentevent.Event{
		ID:             pulid.MustNew("se_"),
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
		ShipmentID:     entity.ID,
		MoveID:         move.ID,
		StopID:         move.Stops[0].ID,
		Type:           eventType,
		Summary:        "Arrived at stop",
		Metadata:       map[string]any{},
		OccurredAt:     1780418506,
		CorrelationID:  pulid.MustNew("corr_").String(),
	}
	fixture.event = event
	return event
}

type rollbackDB struct {
	transfers *fakeTransferChangeRepository
}

func (d rollbackDB) Transact(
	ctx context.Context,
	_ kernelports.TxOptions,
	fn func(context.Context) error,
) error {
	existing := maps.Clone(d.transfers.existing)
	created := slices.Clone(d.transfers.created)
	if err := fn(ctx); err != nil {
		d.transfers.existing = existing
		d.transfers.created = created
		return err
	}
	return nil
}

type stopTimes struct {
	arrival   *int64
	departure *int64
}

type stopActualsCall struct {
	req     *shipmentmoveports.ReconcileStopActualsRequest
	before  map[pulid.ID]stopTimes
	applied []stopactual.Change
}

type fakeStopActuals struct {
	shipments map[pulid.ID]*shipment.Shipment
	calls     []stopActualsCall
	err       error
}

func (f *fakeStopActuals) ReconcileStopActuals(
	ctx context.Context,
	req *shipmentmoveports.ReconcileStopActualsRequest,
) (*shipmentmoveports.ReconcileStopActualsResult, error) {
	if f.err != nil {
		return nil, f.err
	}
	move := f.move(req.MoveID)
	if move == nil {
		return nil, errortypes.NewNotFoundError("Move not found")
	}

	call := stopActualsCall{req: req, before: make(map[pulid.ID]stopTimes, len(move.Stops))}
	for _, stop := range move.Stops {
		call.before[stop.ID] = stopTimes{
			arrival:   stop.ActualArrival,
			departure: stop.ActualDeparture,
		}
	}
	for range len(move.Stops) + 1 {
		changes, err := req.Plan(ctx, move)
		if err != nil {
			return nil, err
		}
		if len(changes) == 0 {
			break
		}
		if _, err = stopactual.Apply(move, changes); err != nil {
			return nil, err
		}
		call.applied = append(call.applied, changes...)
	}
	f.calls = append(f.calls, call)

	return &shipmentmoveports.ReconcileStopActualsResult{Move: move, Changes: call.applied}, nil
}

func (f *fakeStopActuals) move(id pulid.ID) *shipment.ShipmentMove {
	for _, entity := range f.shipments {
		for _, move := range entity.Moves {
			if move.ID == id {
				return move
			}
		}
	}
	return nil
}
