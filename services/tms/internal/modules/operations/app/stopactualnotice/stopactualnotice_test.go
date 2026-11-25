package stopactualnotice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fixture struct {
	tenantInfo pagination.TenantInfo
	actor      kernelports.AuditActor
	move       *shipment.ShipmentMove
	pickup     *shipment.Stop
	drop       *shipment.Stop
	final      *shipment.Stop
}

func newFixture() *fixture {
	arrival, departure := int64(1_000), int64(2_000)
	moveID := pulid.MustNew("sm_")
	pickup := &shipment.Stop{
		ID:              pulid.MustNew("stp_"),
		ShipmentMoveID:  moveID,
		Type:            shipmentvocab.StopTypePickup,
		Status:          shipmentvocab.StopStatusCompleted,
		Sequence:        0,
		ActualArrival:   &arrival,
		ActualDeparture: &departure,
		Location:        &location.Location{Name: "Cargill Protein Plant"},
	}
	canceled := &shipment.Stop{
		ID:             pulid.MustNew("stp_"),
		ShipmentMoveID: moveID,
		Type:           shipmentvocab.StopTypeDelivery,
		Status:         shipmentvocab.StopStatusCanceled,
		Sequence:       1,
	}
	drop := &shipment.Stop{
		ID:             pulid.MustNew("stp_"),
		ShipmentMoveID: moveID,
		Type:           shipmentvocab.StopTypeSplitDelivery,
		Status:         shipmentvocab.StopStatusNew,
		Sequence:       2,
		AddressLine:    "400 Commerce St",
	}
	final := &shipment.Stop{
		ID:             pulid.MustNew("stp_"),
		ShipmentMoveID: moveID,
		Type:           shipmentvocab.StopTypeDelivery,
		Status:         shipmentvocab.StopStatusNew,
		Sequence:       3,
		Location:       &location.Location{Name: "Dallas Terminal"},
	}
	userID := pulid.MustNew("usr_")

	return &fixture{
		tenantInfo: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: userID,
		},
		actor: kernelports.AuditActor{
			PrincipalType: permissionports.PrincipalTypeUser,
			PrincipalID:   userID,
			UserID:        userID,
		},
		move: &shipment.ShipmentMove{
			ID:         moveID,
			ShipmentID: pulid.MustNew("shp_"),
			Status:     shipmentvocab.MoveStatusInTransit,
			Stops:      []*shipment.Stop{final, drop, canceled, pickup},
		},
		pickup: pickup,
		drop:   drop,
		final:  final,
	}
}

func TestEvents_ArrivalAtAMiddleStopNamesTheStop(t *testing.T) {
	t.Parallel()

	f := newFixture()
	before := Take(f.move)
	arrived := int64(3_000)
	f.drop.ActualArrival = &arrived

	events := Events(f.tenantInfo, f.actor, f.move, before)

	require.Len(t, events, 1)
	event := events[0]
	assert.Equal(t, shipmentevent.TypeStopArrived, event.Type)
	assert.Equal(t, f.move.ShipmentID, event.ShipmentID)
	assert.Equal(t, f.move.ID, event.MoveID)
	assert.Equal(t, f.drop.ID, event.StopID)
	assert.Equal(t, arrived, event.OccurredAt)
	assert.Equal(t, 2, event.Metadata[shipmentevent.MetadataStopNumber],
		"the canceled stop is not counted")
	assert.Equal(t, int64(2), event.Metadata[shipmentevent.MetadataStopSequence])
	assert.Equal(t, "SplitDelivery", event.Metadata[shipmentevent.MetadataStopType])
	assert.Equal(t, "400 Commerce St", event.Metadata[shipmentevent.MetadataLocationName])
	assert.Equal(t, arrived, event.Metadata[shipmentevent.MetadataActualAt])
	assert.NotContains(t, event.Metadata, shipmentevent.MetadataPreviousActualAt)
	assert.Equal(t, f.tenantInfo.UserID, event.Actor.UserID)
}

func TestEvents_ArrivalAndDepartureInOneRecordAreTwoEventsInOrder(t *testing.T) {
	t.Parallel()

	f := newFixture()
	before := Take(f.move)
	arrived, departed := int64(3_000), int64(4_000)
	f.final.ActualArrival = &arrived
	f.final.ActualDeparture = &departed

	events := Events(f.tenantInfo, f.actor, f.move, before)

	require.Len(t, events, 2)
	assert.Equal(t, shipmentevent.TypeStopArrived, events[0].Type)
	assert.Equal(t, arrived, events[0].OccurredAt)
	assert.Equal(t, shipmentevent.TypeStopDeparted, events[1].Type)
	assert.Equal(t, departed, events[1].OccurredAt)
	assert.Equal(t, "Dallas Terminal", events[1].Metadata[shipmentevent.MetadataLocationName])
	assert.Equal(t, 3, events[1].Metadata[shipmentevent.MetadataStopNumber])
}

func TestEvents_TheSameActualAgainIsNoEvent(t *testing.T) {
	t.Parallel()

	f := newFixture()
	before := Take(f.move)
	same := *f.pickup.ActualArrival
	f.pickup.ActualArrival = &same

	assert.Empty(t, Events(f.tenantInfo, f.actor, f.move, before))
}

func TestEvents_ACorrectedActualIsACorrection(t *testing.T) {
	t.Parallel()

	f := newFixture()
	before := Take(f.move)
	arrival, departure := int64(1_200), int64(2_400)
	f.pickup.ActualArrival = &arrival
	f.pickup.ActualDeparture = &departure

	events := Events(f.tenantInfo, f.actor, f.move, before)

	require.Len(t, events, 2)
	assert.Equal(t, shipmentevent.TypeStopArrivalCorrected, events[0].Type)
	assert.Equal(t, int64(1_200), events[0].Metadata[shipmentevent.MetadataActualAt])
	assert.Equal(t, int64(1_000), events[0].Metadata[shipmentevent.MetadataPreviousActualAt])
	assert.Zero(t, events[0].OccurredAt, "a correction happens when it is recorded")
	assert.Equal(t, shipmentevent.TypeStopDepartureCorrected, events[1].Type)
	assert.Equal(t, int64(2_400), events[1].Metadata[shipmentevent.MetadataActualAt])
	assert.Equal(t, int64(2_000), events[1].Metadata[shipmentevent.MetadataPreviousActualAt])
}

func TestEvents_AMoveNobodySawBeforeRecordsEveryActual(t *testing.T) {
	t.Parallel()

	f := newFixture()

	events := Events(f.tenantInfo, f.actor, f.move, TakeAll(nil)[f.move.ID])

	require.Len(t, events, 2)
	assert.Equal(t, shipmentevent.TypeStopArrived, events[0].Type)
	assert.Equal(t, shipmentevent.TypeStopDeparted, events[1].Type)
}

func TestVisits_EveryStopTimeEnteredAtOnceIsEveryVisitInStopOrder(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.pickup.ActualArrival, f.pickup.ActualDeparture = nil, nil
	before := Take(f.move)
	times := []int64{1_000, 2_000, 3_000, 4_000, 5_000, 6_000}
	f.pickup.ActualArrival, f.pickup.ActualDeparture = &times[0], &times[1]
	f.drop.ActualArrival, f.drop.ActualDeparture = &times[2], &times[3]
	f.final.ActualArrival, f.final.ActualDeparture = &times[4], &times[5]

	visits := Visits(f.move, before)

	assert.Equal(t, []Visit{
		{MoveID: f.move.ID, StopID: f.pickup.ID, At: 1_000},
		{MoveID: f.move.ID, StopID: f.pickup.ID, Departed: true, At: 2_000},
		{MoveID: f.move.ID, StopID: f.drop.ID, At: 3_000},
		{MoveID: f.move.ID, StopID: f.drop.ID, Departed: true, At: 4_000},
		{MoveID: f.move.ID, StopID: f.final.ID, At: 5_000},
		{MoveID: f.move.ID, StopID: f.final.ID, Departed: true, At: 6_000},
	}, visits)
}

func TestVisits_ACorrectedTimeIsNoNewVisit(t *testing.T) {
	t.Parallel()

	f := newFixture()
	before := Take(f.move)
	corrected := int64(1_500)
	f.pickup.ActualArrival = &corrected

	assert.Empty(t, Visits(f.move, before))
}

type recordingPublisher struct {
	events []*agenteventports.AgentEvent
}

func (r *recordingPublisher) Publish(_ context.Context, event *agenteventports.AgentEvent) {
	r.events = append(r.events, event)
}

func TestPublish_NamesTheMoveAndTheStop(t *testing.T) {
	t.Parallel()

	f := newFixture()
	publisher := &recordingPublisher{}

	Publish(t.Context(), publisher, f.tenantInfo, Visit{MoveID: f.move.ID, StopID: f.drop.ID, At: 0})
	Publish(t.Context(), publisher, f.tenantInfo,
		Visit{MoveID: f.move.ID, StopID: f.drop.ID, Departed: true, At: 60})

	require.Len(t, publisher.events, 2)
	assert.Equal(t, agenttrigger.EventShipmentMoveArrived, publisher.events[0].Kind)
	assert.Equal(t, f.move.ID, publisher.events[0].SubjectID)
	assert.Equal(t, []pulid.ID{f.drop.ID}, publisher.events[0].Related)
	assert.Equal(t, f.tenantInfo, publisher.events[0].TenantInfo)
	assert.Equal(t, agenttrigger.EventShipmentMoveDeparted, publisher.events[1].Kind)
}
