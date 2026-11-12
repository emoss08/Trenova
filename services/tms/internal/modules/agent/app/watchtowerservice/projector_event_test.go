package watchtowerservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type heardEvents struct {
	agenteventports.AgentEventPublisher

	events []*agenteventports.AgentEvent
}

func (h *heardEvents) Publish(_ context.Context, event *agenteventports.AgentEvent) {
	h.events = append(h.events, event)
}

/*
A sweep that finds a condition has no moment of its own to wake an agent
from, so the projector does it: the item's event goes to the agents that
subscribe when the item is first raised, once, and not on every later sweep
that finds the same condition still there.
*/
func TestUpsert_RaisesAnItemsEventOnceWhenASweepFirstFindsIt(t *testing.T) {
	t.Parallel()

	repo := newStubRepo()
	projector, _ := newRoutingProjector(
		repo,
		&agenttest.StubPermissions{},
		&dispatchcontrol.DispatchControl{},
	)
	heard := &heardEvents{}
	projector.events = heard
	tenant := testActor().TenantInfo()
	shipment := pulid.MustNew("shp_")

	in := input(tenant, watchtower.SourcePodMissing, shipment.String(), watchtower.SeverityWarning, 100)
	in.SubjectType = agent.SubjectShipment
	in.SubjectID = shipment
	in.EventKind = agenttrigger.EventShipmentPODMissing
	in.RaisesEvent = true

	require.NoError(t, projector.upsert(t.Context(), in))
	require.NoError(t, projector.upsert(t.Context(), in))

	require.Len(t, heard.events, 1)
	assert.Equal(t, agenttrigger.EventShipmentPODMissing, heard.events[0].Kind)
	assert.Equal(t, shipment, heard.events[0].SubjectID)
}

// A source that publishes its own event when the thing happens labels its
// item with the kind for hand-off, and the projector stays quiet.
func TestUpsert_LeavesTheEventToASourceThatPublishesItsOwn(t *testing.T) {
	t.Parallel()

	repo := newStubRepo()
	projector, _ := newRoutingProjector(
		repo,
		&agenttest.StubPermissions{},
		&dispatchcontrol.DispatchControl{},
	)
	heard := &heardEvents{}
	projector.events = heard
	worker := pulid.MustNew("wrk_")

	in := input(
		testActor().TenantInfo(),
		watchtower.SourceWorkerCredential,
		worker.String(),
		watchtower.SeverityWarning,
		100,
	)
	in.SubjectType = agent.SubjectWorker
	in.SubjectID = worker
	in.EventKind = agenttrigger.EventWorkerCredentialExpiring

	require.NoError(t, projector.upsert(t.Context(), in))

	assert.Empty(t, heard.events)
}
