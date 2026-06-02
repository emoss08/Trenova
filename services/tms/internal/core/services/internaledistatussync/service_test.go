package internaledistatussync

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	coreports "github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/internaledilifecycle"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func TestObserverAppliesTargetStatusToSourceShipment(t *testing.T) {
	t.Parallel()

	fixture := newObserverFixture(t)
	fixture.source.Status = shipment.StatusNew
	fixture.target.Status = shipment.StatusInTransit

	err := fixture.observer.OnShipmentEvent(t.Context(), targetStatusEvent(fixture, shipment.StatusInTransit))

	require.NoError(t, err)
	require.Len(t, fixture.transferRepo.created, 1)
	require.Equal(t, edi.TransferChangeStatusApplied, fixture.transferRepo.created[0].Status)
	require.Equal(t, edi.TransferChangeDirectionTargetToSource, fixture.transferRepo.created[0].Direction)
	require.Equal(t, shipment.StatusInTransit, fixture.source.Status)
	require.Len(t, fixture.eventRepo.inserted, 1)
	require.Equal(
		t,
		fixture.event.ID.String(),
		fixture.eventRepo.inserted[0].Metadata[edi.InternalEDIMirroredFromEventIDKey],
	)
	require.Len(t, fixture.realtime.published, 3)
}

func TestObserverDuplicateExecutionIsIdempotent(t *testing.T) {
	t.Parallel()

	fixture := newObserverFixture(t)
	event := targetStatusEvent(fixture, shipment.StatusInTransit)
	key := fixture.idempotencyKey(
		event,
		edi.TransferChangeDirectionTargetToSource,
		edi.TransferChangeTypeShipmentStatus214,
	)
	fixture.transferRepo.existing[key] = &edi.TransferChange{
		ID:             pulid.MustNew("editc_"),
		BusinessUnitID: fixture.link.BusinessUnitID,
		ShipmentLinkID: fixture.link.ID,
		Direction:      edi.TransferChangeDirectionTargetToSource,
		ChangeType:     edi.TransferChangeTypeShipmentStatus214,
		IdempotencyKey: key,
		Status:         edi.TransferChangeStatusApplied,
	}

	err := fixture.observer.OnShipmentEvent(t.Context(), event)

	require.NoError(t, err)
	require.Empty(t, fixture.shipmentRepo.updates)
	require.Empty(t, fixture.eventRepo.inserted)
	require.Empty(t, fixture.transferRepo.created)
}

func TestObserverAutoAppliesLifecycleActualsThroughCoordinator(t *testing.T) {
	t.Parallel()

	fixture := newObserverFixture(t)
	fixture.installLifecycleStops()
	arrival := int64(1_000)
	departure := int64(1_100)
	fixture.target.Moves[0].Stops[0].ActualArrival = &arrival
	fixture.target.Moves[0].Stops[0].ActualDeparture = &departure

	err := fixture.observer.OnShipmentEvent(t.Context(), targetStatusEvent(fixture, shipment.StatusInTransit))

	require.NoError(t, err)
	require.Empty(t, fixture.shipmentRepo.updates)
	require.Len(t, fixture.shipmentRepo.lifecycleUpdates, 1)
	updated := fixture.shipmentRepo.lifecycleUpdates[0]
	require.Equal(t, shipment.StatusInTransit, updated.Status)
	require.Equal(t, shipment.MoveStatusInTransit, updated.Moves[0].Status)
	require.Equal(t, shipment.StopStatusCompleted, updated.Moves[0].Stops[0].Status)
	require.Equal(t, shipment.StopStatusNew, updated.Moves[0].Stops[1].Status)
	require.Equal(t, arrival, *updated.Moves[0].Stops[0].ActualArrival)
	require.Equal(t, departure, *updated.Moves[0].Stops[0].ActualDeparture)
	require.Equal(t, departure, *updated.ActualShipDate)
	require.Nil(t, updated.ActualDeliveryDate)
	require.Len(t, fixture.transferRepo.created, 1)
	change := fixture.transferRepo.created[0]
	require.Equal(t, edi.TransferChangeTypeShipmentLifecycle214, change.ChangeType)
	require.Equal(t, edi.TransferChangeStatusApplied, change.Status)
	require.NotEmpty(t, change.Payload["matchedStopActualDiffs"])
	require.Len(t, fixture.eventRepo.inserted, 1)
	require.Equal(t, "Lifecycle synced from internal EDI 214", fixture.eventRepo.inserted[0].Summary)
}

func TestObserverManualReviewLifecycleDoesNotMutateShipment(t *testing.T) {
	t.Parallel()

	fixture := newObserverFixture(t)
	fixture.link.SyncPolicy = edi.ShipmentSyncPolicyManualReview
	fixture.installLifecycleStops()
	arrival := int64(1_000)
	fixture.target.Moves[0].Stops[0].ActualArrival = &arrival

	err := fixture.observer.OnShipmentEvent(t.Context(), targetStatusEvent(fixture, shipment.StatusInTransit))

	require.NoError(t, err)
	require.Empty(t, fixture.shipmentRepo.lifecycleUpdates)
	require.Empty(t, fixture.eventRepo.inserted)
	require.Len(t, fixture.transferRepo.created, 1)
	require.Equal(t, edi.TransferChangeStatusPendingReview, fixture.transferRepo.created[0].Status)
	require.Equal(t, edi.TransferChangeTypeShipmentLifecycle214, fixture.transferRepo.created[0].ChangeType)
}

func TestObserverLifecycleMappingConflictCreatesPendingReview(t *testing.T) {
	t.Parallel()

	fixture := newObserverFixture(t)
	fixture.installLifecycleStops()
	fixture.source.Moves[0].Stops = fixture.source.Moves[0].Stops[:1]
	arrival := int64(1_000)
	fixture.target.Moves[0].Stops[1].ActualArrival = &arrival

	err := fixture.observer.OnShipmentEvent(t.Context(), targetStatusEvent(fixture, shipment.StatusInTransit))

	require.NoError(t, err)
	require.Empty(t, fixture.shipmentRepo.lifecycleUpdates)
	require.Empty(t, fixture.eventRepo.inserted)
	require.Len(t, fixture.transferRepo.created, 1)
	change := fixture.transferRepo.created[0]
	require.Equal(t, edi.TransferChangeStatusPendingReview, change.Status)
	require.Equal(t, edi.TransferChangeConflictConflict, change.ConflictStatus)
	require.NotEmpty(t, change.ConflictReason)
	require.NotEmpty(t, change.Payload["conflicts"])
}

func TestObserverIgnoresMirroredEvents(t *testing.T) {
	t.Parallel()

	fixture := newObserverFixture(t)
	event := targetStatusEvent(fixture, shipment.StatusInTransit)
	event.Metadata[edi.InternalEDIMirroredFromEventIDKey] = pulid.MustNew("se_").String()

	err := fixture.observer.OnShipmentEvent(t.Context(), event)

	require.NoError(t, err)
	require.Zero(t, fixture.linkRepo.calls)
	require.Empty(t, fixture.transferRepo.created)
}

func TestObserverDoesNotApplyInactiveOrReadOnlyLinks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status edi.ShipmentLinkStatus
		policy edi.ShipmentSyncPolicy
	}{
		{
			name:   "suspended",
			status: edi.ShipmentLinkStatusSuspended,
			policy: edi.ShipmentSyncPolicyAutoOperational,
		},
		{
			name:   "closed",
			status: edi.ShipmentLinkStatusClosed,
			policy: edi.ShipmentSyncPolicyAutoOperational,
		},
		{
			name:   "read only",
			status: edi.ShipmentLinkStatusActive,
			policy: edi.ShipmentSyncPolicyReadOnly,
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
				targetStatusEvent(fixture, shipment.StatusInTransit),
			)

			require.NoError(t, err)
			require.Empty(t, fixture.shipmentRepo.updates)
			require.Empty(t, fixture.eventRepo.inserted)
			require.Len(t, fixture.transferRepo.created, 1)
			require.Equal(t, edi.TransferChangeStatusIgnored, fixture.transferRepo.created[0].Status)
		})
	}
}

func TestObserverInvalidTransitionCreatesPendingConflict(t *testing.T) {
	t.Parallel()

	fixture := newObserverFixture(t)
	fixture.source.Status = shipment.StatusInvoiced
	fixture.target.Status = shipment.StatusCompleted

	err := fixture.observer.OnShipmentEvent(t.Context(), targetStatusEvent(fixture, shipment.StatusCompleted))

	require.NoError(t, err)
	require.Empty(t, fixture.shipmentRepo.updates)
	require.Empty(t, fixture.eventRepo.inserted)
	require.Len(t, fixture.transferRepo.created, 1)
	change := fixture.transferRepo.created[0]
	require.Equal(t, edi.TransferChangeStatusPendingReview, change.Status)
	require.Equal(t, edi.TransferChangeConflictConflict, change.ConflictStatus)
	require.NotEmpty(t, change.ConflictReason)
}

func TestObserverAutoAppliesSourceCancellationToTargetShipment(t *testing.T) {
	t.Parallel()

	fixture := newObserverFixture(t)
	fixture.source.Status = shipment.StatusCanceled
	fixture.target.Status = shipment.StatusInTransit

	err := fixture.observer.OnShipmentEvent(t.Context(), sourceCancelEvent(fixture, "Customer canceled"))

	require.NoError(t, err)
	require.Empty(t, fixture.shipmentRepo.updates)
	require.Len(t, fixture.shipmentRepo.cancels, 1)
	require.Equal(t, fixture.link.TargetShipmentID, fixture.shipmentRepo.cancels[0].ShipmentID)
	require.Equal(t, "Customer canceled", fixture.shipmentRepo.cancels[0].CancelReason)
	require.Equal(t, shipment.StatusCanceled, fixture.target.Status)
	require.Len(t, fixture.transferRepo.created, 1)
	change := fixture.transferRepo.created[0]
	require.Equal(t, edi.TransferChangeTypeShipmentCancel214, change.ChangeType)
	require.Equal(t, edi.TransferChangeStatusApplied, change.Status)
	require.Equal(t, "A7", change.Payload["shipmentStatus"].(edi.ShipmentStatusPayload).StatusCode)
	require.Len(t, fixture.eventRepo.inserted, 1)
	require.Equal(t, shipmentevent.TypeShipmentCanceled, fixture.eventRepo.inserted[0].Type)
	require.Equal(
		t,
		fixture.event.ID.String(),
		fixture.eventRepo.inserted[0].Metadata[edi.InternalEDIMirroredFromEventIDKey],
	)
	require.Len(t, fixture.realtime.published, 3)
}

func TestObserverAutoAppliesTargetCancellationToSourceShipment(t *testing.T) {
	t.Parallel()

	fixture := newObserverFixture(t)
	fixture.source.Status = shipment.StatusAssigned
	fixture.target.Status = shipment.StatusCanceled

	err := fixture.observer.OnShipmentEvent(t.Context(), targetCancelEvent(fixture, "Carrier canceled"))

	require.NoError(t, err)
	require.Empty(t, fixture.shipmentRepo.updates)
	require.Len(t, fixture.shipmentRepo.cancels, 1)
	require.Equal(t, fixture.link.SourceShipmentID, fixture.shipmentRepo.cancels[0].ShipmentID)
	require.Equal(t, shipment.StatusCanceled, fixture.source.Status)
	require.Len(t, fixture.transferRepo.created, 1)
	require.Equal(t, edi.TransferChangeDirectionTargetToSource, fixture.transferRepo.created[0].Direction)
}

func TestObserverManualReviewCancellationDoesNotMutateShipment(t *testing.T) {
	t.Parallel()

	fixture := newObserverFixture(t)
	fixture.link.SyncPolicy = edi.ShipmentSyncPolicyManualReview
	fixture.source.Status = shipment.StatusCanceled
	fixture.target.Status = shipment.StatusInTransit

	err := fixture.observer.OnShipmentEvent(t.Context(), sourceCancelEvent(fixture, "Manual review"))

	require.NoError(t, err)
	require.Empty(t, fixture.shipmentRepo.updates)
	require.Empty(t, fixture.shipmentRepo.cancels)
	require.Empty(t, fixture.eventRepo.inserted)
	require.Len(t, fixture.transferRepo.created, 1)
	require.Equal(t, edi.TransferChangeStatusPendingReview, fixture.transferRepo.created[0].Status)
}

func TestObserverIgnoresAlreadyCanceledOppositeShipment(t *testing.T) {
	t.Parallel()

	fixture := newObserverFixture(t)
	fixture.source.Status = shipment.StatusCanceled
	fixture.target.Status = shipment.StatusCanceled

	err := fixture.observer.OnShipmentEvent(t.Context(), sourceCancelEvent(fixture, "Already canceled"))

	require.NoError(t, err)
	require.Empty(t, fixture.shipmentRepo.cancels)
	require.Empty(t, fixture.eventRepo.inserted)
	require.Len(t, fixture.transferRepo.created, 1)
	require.Equal(t, edi.TransferChangeStatusIgnored, fixture.transferRepo.created[0].Status)
}

func TestObserverInvalidCancellationTransitionCreatesPendingConflict(t *testing.T) {
	t.Parallel()

	fixture := newObserverFixture(t)
	fixture.source.Status = shipment.StatusCanceled
	fixture.target.Status = shipment.StatusCompleted

	err := fixture.observer.OnShipmentEvent(t.Context(), sourceCancelEvent(fixture, "Invalid"))

	require.NoError(t, err)
	require.Empty(t, fixture.shipmentRepo.cancels)
	require.Empty(t, fixture.eventRepo.inserted)
	require.Len(t, fixture.transferRepo.created, 1)
	change := fixture.transferRepo.created[0]
	require.Equal(t, edi.TransferChangeStatusPendingReview, change.Status)
	require.Equal(t, edi.TransferChangeConflictConflict, change.ConflictStatus)
	require.NotEmpty(t, change.ConflictReason)
}

func TestObserverDuplicateCancellationExecutionIsIdempotent(t *testing.T) {
	t.Parallel()

	fixture := newObserverFixture(t)
	event := sourceCancelEvent(fixture, "Duplicate")
	key := fixture.idempotencyKey(event, edi.TransferChangeDirectionSourceToTarget, edi.TransferChangeTypeShipmentCancel214)
	fixture.transferRepo.existing[key] = &edi.TransferChange{
		ID:             pulid.MustNew("editc_"),
		BusinessUnitID: fixture.link.BusinessUnitID,
		ShipmentLinkID: fixture.link.ID,
		Direction:      edi.TransferChangeDirectionSourceToTarget,
		ChangeType:     edi.TransferChangeTypeShipmentCancel214,
		IdempotencyKey: key,
		Status:         edi.TransferChangeStatusApplied,
	}

	err := fixture.observer.OnShipmentEvent(t.Context(), event)

	require.NoError(t, err)
	require.Empty(t, fixture.shipmentRepo.cancels)
	require.Empty(t, fixture.eventRepo.inserted)
	require.Empty(t, fixture.transferRepo.created)
}

type observerFixture struct {
	observer     *Observer
	link         *edi.ShipmentLink
	source       *shipment.Shipment
	target       *shipment.Shipment
	event        *shipmentevent.Event
	shipmentRepo *fakeShipmentRepository
	linkRepo     *fakeShipmentLinkRepository
	transferRepo *fakeTransferChangeRepository
	eventRepo    *fakeShipmentEventRepository
	realtime     *fakeRealtimeService
}

func newObserverFixture(t *testing.T) *observerFixture {
	t.Helper()

	sourceOrgID := pulid.MustNew("org_")
	targetOrgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	sourceID := pulid.MustNew("shp_")
	targetID := pulid.MustNew("shp_")
	link := &edi.ShipmentLink{
		ID:                   pulid.MustNew("edislink_"),
		BusinessUnitID:       buID,
		SourceOrganizationID: sourceOrgID,
		TargetOrganizationID: targetOrgID,
		SourceShipmentID:     sourceID,
		TargetShipmentID:     targetID,
		SyncPolicy:           edi.ShipmentSyncPolicyAutoOperational,
		Status:               edi.ShipmentLinkStatusActive,
	}
	source := &shipment.Shipment{
		ID:             sourceID,
		OrganizationID: sourceOrgID,
		BusinessUnitID: buID,
		ProNumber:      "SOURCE-1",
		Status:         shipment.StatusNew,
		Version:        2,
	}
	target := &shipment.Shipment{
		ID:             targetID,
		OrganizationID: targetOrgID,
		BusinessUnitID: buID,
		ProNumber:      "TARGET-1",
		Status:         shipment.StatusInTransit,
		Version:        4,
	}

	shipmentRepo := &fakeShipmentRepository{shipments: map[pulid.ID]*shipment.Shipment{
		sourceID: source,
		targetID: target,
	}}
	linkRepo := &fakeShipmentLinkRepository{links: []*edi.ShipmentLink{link}}
	transferRepo := &fakeTransferChangeRepository{existing: map[string]*edi.TransferChange{}}
	eventRepo := &fakeShipmentEventRepository{}
	realtime := &fakeRealtimeService{}

	fixture := &observerFixture{
		link:         link,
		source:       source,
		target:       target,
		shipmentRepo: shipmentRepo,
		linkRepo:     linkRepo,
		transferRepo: transferRepo,
		eventRepo:    eventRepo,
		realtime:     realtime,
	}
	fixture.observer = &Observer{
		l:                  zap.NewNop(),
		db:                 fakeDBConnection{},
		shipmentRepo:       shipmentRepo,
		shipmentEventRepo:  eventRepo,
		shipmentLinkRepo:   linkRepo,
		transferChangeRepo: transferRepo,
		realtime:           realtime,
		lifecycleApplier: internaledilifecycle.New(internaledilifecycle.Params{
			ShipmentRepo: shipmentRepo,
			Coordinator:  shipmentstate.NewCoordinatorWithClock(func() int64 { return 2_000 }),
		}),
	}
	return fixture
}

func targetStatusEvent(fixture *observerFixture, status shipment.Status) *shipmentevent.Event {
	fixture.event = &shipmentevent.Event{
		ID:             pulid.MustNew("se_"),
		OrganizationID: fixture.link.TargetOrganizationID,
		BusinessUnitID: fixture.link.BusinessUnitID,
		ShipmentID:     fixture.link.TargetShipmentID,
		Type:           shipmentevent.TypeStatusChanged,
		Summary:        "Status updated",
		Metadata: map[string]any{
			"previousStatus": string(shipment.StatusAssigned),
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
		lifecycleMove(f.source, shipment.MoveStatusNew),
	}
	f.target.Moves = []*shipment.ShipmentMove{
		lifecycleMove(f.target, shipment.MoveStatusNew),
	}
}

func lifecycleMove(entity *shipment.Shipment, status shipment.MoveStatus) *shipment.ShipmentMove {
	moveID := pulid.MustNew("sm_")
	return &shipment.ShipmentMove{
		ID:             moveID,
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
		ShipmentID:     entity.ID,
		Status:         status,
		Sequence:       0,
		Stops: []*shipment.Stop{
			lifecycleStop(entity, moveID, shipment.StopTypePickup, 0),
			lifecycleStop(entity, moveID, shipment.StopTypeDelivery, 1),
		},
	}
}

func lifecycleStop(
	entity *shipment.Shipment,
	moveID pulid.ID,
	stopType shipment.StopType,
	sequence int64,
) *shipment.Stop {
	return &shipment.Stop{
		ID:                   pulid.MustNew("stp_"),
		OrganizationID:       entity.OrganizationID,
		BusinessUnitID:       entity.BusinessUnitID,
		ShipmentMoveID:       moveID,
		LocationID:           pulid.MustNew("loc_"),
		Status:               shipment.StopStatusNew,
		Type:                 stopType,
		Sequence:             sequence,
		ScheduledWindowStart: 500 + sequence,
	}
}

func (f *observerFixture) idempotencyKey(
	event *shipmentevent.Event,
	direction edi.TransferChangeDirection,
	changeType string,
) string {
	return f.link.ID.String() + ":" + event.ID.String() + ":" + string(direction) + ":" + changeType
}

type fakeDBConnection struct{}

func (fakeDBConnection) DB() *bun.DB                          { return nil }
func (fakeDBConnection) DBForContext(context.Context) bun.IDB { return nil }
func (fakeDBConnection) HealthCheck(context.Context) error    { return nil }
func (fakeDBConnection) IsHealthy(context.Context) bool       { return true }
func (fakeDBConnection) Close() error                         { return nil }
func (fakeDBConnection) WithTx(
	ctx context.Context,
	_ coreports.TxOptions,
	fn func(context.Context, bun.Tx) error,
) error {
	return fn(ctx, bun.Tx{})
}

type fakeShipmentRepository struct {
	shipments        map[pulid.ID]*shipment.Shipment
	updates          []*repositories.UpdateShipmentStatusRequest
	lifecycleUpdates []*shipment.Shipment
	cancels          []*repositories.CancelShipmentRequest
}

func (r *fakeShipmentRepository) GetByID(
	_ context.Context,
	req *repositories.GetShipmentByIDRequest,
) (*shipment.Shipment, error) {
	return r.shipments[req.ID], nil
}

func (r *fakeShipmentRepository) UpdateStatus(
	_ context.Context,
	req *repositories.UpdateShipmentStatusRequest,
) (*shipment.Shipment, error) {
	r.updates = append(r.updates, req)
	entity := r.shipments[req.ShipmentID]
	entity.Status = req.Status
	entity.Version++
	return entity, nil
}

func (r *fakeShipmentRepository) UpdateOperationalLifecycle(
	_ context.Context,
	entity *shipment.Shipment,
) (*shipment.Shipment, error) {
	r.lifecycleUpdates = append(r.lifecycleUpdates, entity)
	r.shipments[entity.ID] = entity
	entity.Version++
	return entity, nil
}

func (r *fakeShipmentRepository) Cancel(
	_ context.Context,
	req *repositories.CancelShipmentRequest,
) (*shipment.Shipment, error) {
	r.cancels = append(r.cancels, req)
	entity := r.shipments[req.ShipmentID]
	entity.Status = shipment.StatusCanceled
	entity.CancelReason = req.CancelReason
	entity.CanceledAt = &req.CanceledAt
	entity.CanceledByID = req.CanceledByID
	entity.Version++
	return entity, nil
}

type fakeShipmentLinkRepository struct {
	links []*edi.ShipmentLink
	calls int
}

func (r *fakeShipmentLinkRepository) GetShipmentLinksByShipmentID(
	context.Context,
	repositories.GetEDIShipmentLinksByShipmentIDRequest,
) ([]*edi.ShipmentLink, error) {
	r.calls++
	return r.links, nil
}

type fakeTransferChangeRepository struct {
	existing map[string]*edi.TransferChange
	created  []*edi.TransferChange
}

func (r *fakeTransferChangeRepository) CreateTransferChangeIdempotent(
	_ context.Context,
	entity *edi.TransferChange,
) (*repositories.CreateEDITransferChangeIdempotentResult, error) {
	if existing := r.existing[entity.IdempotencyKey]; existing != nil {
		return &repositories.CreateEDITransferChangeIdempotentResult{
			TransferChange: existing,
			Created:        false,
		}, nil
	}
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("editc_")
	}
	r.existing[entity.IdempotencyKey] = entity
	r.created = append(r.created, entity)
	return &repositories.CreateEDITransferChangeIdempotentResult{
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
	published []*services.PublishResourceInvalidationRequest
}

func (fakeRealtimeService) CreateTokenRequest(
	*services.CreateRealtimeTokenRequest,
) (*services.RealtimeTokenRequest, error) {
	return nil, nil
}

func (s *fakeRealtimeService) PublishResourceInvalidation(
	_ context.Context,
	req *services.PublishResourceInvalidationRequest,
) error {
	s.published = append(s.published, req)
	return nil
}
