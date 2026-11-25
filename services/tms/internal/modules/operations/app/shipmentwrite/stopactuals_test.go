package shipmentwrite

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenteffects"
	"github.com/emoss08/trenova/internal/modules/operations/app/stopactualnotice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordedShipmentEvents struct {
	recorded []*shipmenteventports.RecordShipmentEventParams
}

func (r *recordedShipmentEvents) Record(
	_ context.Context,
	params *shipmenteventports.RecordShipmentEventParams,
) error {
	r.recorded = append(r.recorded, params)
	return nil
}

func (r *recordedShipmentEvents) List(
	context.Context,
	*shipmenteventports.ListShipmentEventsRequest,
) ([]*shipmentevent.Event, error) {
	return nil, nil
}

type recordedAgentEvents struct {
	events []*agenteventports.AgentEvent
}

func (r *recordedAgentEvents) Publish(_ context.Context, event *agenteventports.AgentEvent) {
	r.events = append(r.events, event)
}

func stopTimesFixture() (*shipment.Shipment, *shipment.Stop, *shipment.Stop) {
	moveID := pulid.MustNew("sm_")
	pickup := &shipment.Stop{
		ID:             pulid.MustNew("stp_"),
		ShipmentMoveID: moveID,
		Type:           shipmentvocab.StopTypePickup,
		Status:         shipmentvocab.StopStatusNew,
		Sequence:       0,
	}
	delivery := &shipment.Stop{
		ID:             pulid.MustNew("stp_"),
		ShipmentMoveID: moveID,
		Type:           shipmentvocab.StopTypeDelivery,
		Status:         shipmentvocab.StopStatusNew,
		Sequence:       1,
	}
	entity := &shipment.Shipment{
		ID:             pulid.MustNew("shp_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Moves: []*shipment.ShipmentMove{{
			ID:     moveID,
			Status: shipmentvocab.MoveStatusAssigned,
			Stops:  []*shipment.Stop{pickup, delivery},
		}},
	}
	entity.Moves[0].ShipmentID = entity.ID

	return entity, pickup, delivery
}

func TestAnnounceStopActuals_StopTimesTypedIntoTheShipmentWakeWhatWatchesTheMove(t *testing.T) {
	t.Parallel()

	entity, pickup, delivery := stopTimesFixture()
	before := stopactualnotice.TakeAll(entity.Moves)
	times := []int64{1_000, 2_000, 3_000, 4_000}
	pickup.ActualArrival, pickup.ActualDeparture = &times[0], &times[1]
	delivery.ActualArrival, delivery.ActualDeparture = &times[2], &times[3]

	shipmentEvents := &recordedShipmentEvents{}
	agentEvents := &recordedAgentEvents{}
	svc := &Service{
		effects: shipmenteffects.New(shipmenteffects.Params{
			Logger:       zap.NewNop(),
			EventService: shipmentEvents,
		}),
		agentEvents: agentEvents,
	}
	userID := pulid.MustNew("usr_")
	actor := kernelports.AuditActor{
		PrincipalType: permissionports.PrincipalTypeUser,
		PrincipalID:   userID,
		UserID:        userID,
	}

	svc.announceStopActuals(t.Context(), entity, before, actor)

	types := make([]shipmentevent.Type, 0, len(shipmentEvents.recorded))
	for _, params := range shipmentEvents.recorded {
		types = append(types, params.Type)
		assert.Equal(t, userID, params.Actor.UserID)
	}
	assert.Equal(t, []shipmentevent.Type{
		shipmentevent.TypeStopArrived,
		shipmentevent.TypeStopDeparted,
		shipmentevent.TypeStopArrived,
		shipmentevent.TypeStopDeparted,
	}, types)

	require.Len(t, agentEvents.events, 4)
	first := agentEvents.events[0]
	assert.Equal(t, agenttrigger.EventShipmentMoveArrived, first.Kind)
	assert.Equal(t, entity.Moves[0].ID, first.SubjectID)
	assert.Equal(t, []pulid.ID{pickup.ID}, first.Related)
	assert.Equal(t, entity.OrganizationID, first.TenantInfo.OrgID)
	assert.Equal(t, entity.BusinessUnitID, first.TenantInfo.BuID)
	assert.Equal(t, agenttrigger.EventShipmentMoveDeparted, agentEvents.events[3].Kind)
	assert.Equal(t, []pulid.ID{delivery.ID}, agentEvents.events[3].Related)
}

func TestAnnounceStopActuals_ASaveThatLeavesTheStopTimesAloneSaysNothing(t *testing.T) {
	t.Parallel()

	entity, pickup, _ := stopTimesFixture()
	arrived := int64(1_000)
	pickup.ActualArrival = &arrived
	before := stopactualnotice.TakeAll(entity.Moves)

	shipmentEvents := &recordedShipmentEvents{}
	agentEvents := &recordedAgentEvents{}
	svc := &Service{
		effects: shipmenteffects.New(shipmenteffects.Params{
			Logger:       zap.NewNop(),
			EventService: shipmentEvents,
		}),
		agentEvents: agentEvents,
	}

	svc.announceStopActuals(t.Context(), entity, before, kernelports.AuditActor{})

	assert.Empty(t, shipmentEvents.recorded)
	assert.Empty(t, agentEvents.events)
}
