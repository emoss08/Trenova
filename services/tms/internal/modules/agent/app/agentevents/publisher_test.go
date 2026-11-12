package agentevents_test

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentevents"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentwaitports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeDefinitions struct {
	agentdefinitionports.AgentDefinitionRepository
	items   []*agentdefinition.Definition
	lastReq agentdefinitionports.ListAgentDefinitionsByTriggerRequest
}

func (f *fakeDefinitions) ListEnabledByTrigger(
	_ context.Context,
	req agentdefinitionports.ListAgentDefinitionsByTriggerRequest,
) ([]*agentdefinition.Definition, error) {
	f.lastReq = req
	return f.items, nil
}

type fakeEventRuns struct {
	queued []*agenteventports.EventRunRequest
	err    error
}

func (f *fakeEventRuns) Enqueue(_ context.Context, req *agenteventports.EventRunRequest) error {
	f.queued = append(f.queued, req)

	return f.err
}

// fakeRoutine takes one event kind for every agent.
type fakeRoutine struct {
	kind  agenttrigger.EventKind
	taken []*agenteventports.AgentEvent
}

func (r *fakeRoutine) Handles(_ *agentdefinition.Definition, kind agenttrigger.EventKind) bool {
	return kind == r.kind
}

func (r *fakeRoutine) Take(
	_ context.Context,
	_ *agentdefinition.Definition,
	event *agenteventports.AgentEvent,
) error {
	r.taken = append(r.taken, event)

	return nil
}

type quietWaits struct {
	agentwaitports.AgentWaitNotifier
}

func (quietWaits) NotifyEvent(context.Context, *agenteventports.AgentEvent) {}

var tenantInfo = pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

type fixture struct {
	definitions *fakeDefinitions
	eventRuns   *fakeEventRuns
	routine     *fakeRoutine
	publisher   agenteventports.AgentEventPublisher
}

func newFixture(defs ...*agentdefinition.Definition) *fixture {
	f := &fixture{
		definitions: &fakeDefinitions{items: defs},
		eventRuns:   &fakeEventRuns{},
		routine:     &fakeRoutine{kind: agenttrigger.EventShipmentPODMissing},
	}
	f.publisher = agentevents.New(agentevents.Params{
		Logger:      zap.NewNop(),
		Waits:       quietWaits{},
		Definitions: f.definitions,
		EventRuns:   f.eventRuns,
		Routines:    []agenteventports.EventRoutine{f.routine},
	})
	return f
}

func TestPublish_StartsEachSubscribedDefinition(t *testing.T) {
	t.Parallel()

	first := &agentdefinition.Definition{ID: pulid.MustNew("agd_")}
	second := &agentdefinition.Definition{ID: pulid.MustNew("agd_")}
	f := newFixture(first, second)
	subject := pulid.MustNew("bqi_")

	f.publisher.Publish(t.Context(), &agenteventports.AgentEvent{
		Kind:       agenttrigger.EventBillingQueueItemException,
		SubjectID:  subject,
		TenantInfo: tenantInfo,
	})

	require.Equal(t, agentdefinition.TriggerEvent, f.definitions.lastReq.Mode)
	require.Equal(t, agenttrigger.EventBillingQueueItemException, f.definitions.lastReq.EventKind)
	require.Equal(t, tenantInfo, f.definitions.lastReq.TenantInfo)
	require.Len(t, f.eventRuns.queued, 2, "each agent's run is queued behind its cap")
	for i, req := range f.eventRuns.queued {
		require.Equal(t, []pulid.ID{first.ID, second.ID}[i], req.DefinitionID)
		require.Equal(t, agenttrigger.EventBillingQueueItemException, req.Kind)
		require.Equal(t, subject, req.SubjectID)
		require.Equal(t, tenantInfo, req.TenantInfo)
	}
}

// A routine that does an event's part of the job takes the event instead of
// the agent's model run.
func TestPublish_HandsAnEventARoutineTakesToTheRoutine(t *testing.T) {
	t.Parallel()

	f := newFixture(&agentdefinition.Definition{ID: pulid.MustNew("agd_")})

	f.publisher.Publish(t.Context(), &agenteventports.AgentEvent{
		Kind:       agenttrigger.EventShipmentPODMissing,
		SubjectID:  pulid.MustNew("shp_"),
		TenantInfo: tenantInfo,
	})

	require.Len(t, f.routine.taken, 1)
	require.Empty(t, f.eventRuns.queued, "the routine's event starts no model run")
}

func TestPublish_IgnoresUnknownKind(t *testing.T) {
	t.Parallel()

	f := newFixture(&agentdefinition.Definition{ID: pulid.MustNew("agd_")})

	f.publisher.Publish(t.Context(), &agenteventports.AgentEvent{
		Kind:       agenttrigger.EventKind("not.a.kind"),
		SubjectID:  pulid.MustNew("shp_"),
		TenantInfo: tenantInfo,
	})

	require.Empty(t, f.eventRuns.queued)
}

func TestPublish_IgnoresMissingSubjectOrTenant(t *testing.T) {
	t.Parallel()

	f := newFixture(&agentdefinition.Definition{ID: pulid.MustNew("agd_")})

	f.publisher.Publish(t.Context(), &agenteventports.AgentEvent{
		Kind:       agenttrigger.EventShipmentCreated,
		TenantInfo: tenantInfo,
	})
	f.publisher.Publish(t.Context(), &agenteventports.AgentEvent{
		Kind:      agenttrigger.EventShipmentCreated,
		SubjectID: pulid.MustNew("shp_"),
	})

	require.Empty(t, f.eventRuns.queued)
}

func TestPublish_QueueFailureDoesNotPanicOrStopOthers(t *testing.T) {
	t.Parallel()

	f := newFixture(
		&agentdefinition.Definition{ID: pulid.MustNew("agd_")},
		&agentdefinition.Definition{ID: pulid.MustNew("agd_")},
	)
	f.eventRuns.err = errors.New("temporal down")

	f.publisher.Publish(t.Context(), &agenteventports.AgentEvent{
		Kind:       agenttrigger.EventDocumentExtracted,
		SubjectID:  pulid.MustNew("doc_"),
		TenantInfo: tenantInfo,
	})

	require.Len(t, f.eventRuns.queued, 2)
}
