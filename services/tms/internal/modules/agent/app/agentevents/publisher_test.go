package agentevents_test

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentevents"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
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

type fakeRunService struct {
	agentruntimeports.AgentRunService
	started []*agentruntimeports.StartAgentRunForDefinitionRequest
	err     error
	// open are the definitions with a run already open, which the run
	// service refuses to start again.
	open map[pulid.ID]bool
}

func (f *fakeRunService) StartForDefinition(
	_ context.Context,
	req *agentruntimeports.StartAgentRunForDefinitionRequest,
	_ *permissionports.RequestActor,
) (*agent.AgentRun, error) {
	f.started = append(f.started, req)
	if f.open[req.DefinitionID] {
		return nil, agentruntimeports.ErrAgentRunAlreadyOpen
	}
	if f.err != nil {
		return nil, f.err
	}
	return &agent.AgentRun{ID: pulid.MustNew("ar_")}, nil
}

var tenantInfo = pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

type fixture struct {
	definitions *fakeDefinitions
	runService  *fakeRunService
	publisher   agenteventports.AgentEventPublisher
}

func newFixture(defs ...*agentdefinition.Definition) *fixture {
	f := &fixture{
		definitions: &fakeDefinitions{items: defs},
		runService:  &fakeRunService{open: map[pulid.ID]bool{}},
	}
	f.publisher = agentevents.New(agentevents.Params{
		Logger:      zap.NewNop(),
		Definitions: f.definitions,
		RunService:  f.runService,
	})
	return f
}

func TestPublish_StartsEachSubscribedDefinition(t *testing.T) {
	t.Parallel()

	first := &agentdefinition.Definition{ID: pulid.MustNew("agd_")}
	second := &agentdefinition.Definition{ID: pulid.MustNew("agd_")}
	f := newFixture(first, second)
	subject := pulid.MustNew("bqi_")

	f.publisher.Publish(t.Context(), agenteventports.AgentEvent{
		Kind:       agenttrigger.EventBillingQueueItemException,
		SubjectID:  subject,
		TenantInfo: tenantInfo,
	})

	require.Equal(t, agentdefinition.TriggerEvent, f.definitions.lastReq.Mode)
	require.Equal(t, agenttrigger.EventBillingQueueItemException, f.definitions.lastReq.EventKind)
	require.Equal(t, tenantInfo, f.definitions.lastReq.TenantInfo)
	require.Len(t, f.runService.started, 2)
	for _, req := range f.runService.started {
		require.Equal(t, agent.RunTriggerEvent, req.Trigger)
		require.Equal(t, agenttrigger.EventBillingQueueItemException, req.EventKind)
		require.Equal(t, agent.SubjectBillingQueueItem, req.SubjectType)
		require.Equal(t, subject, req.SubjectID)
		require.Equal(t, tenantInfo, req.TenantInfo)
	}
}

// A definition with a run already open for the subject is refused where the
// run starts; the refusal is a skip, and the other definitions still run.
func TestPublish_SkipsDefinitionWithOpenRunForSubject(t *testing.T) {
	t.Parallel()

	busy := &agentdefinition.Definition{ID: pulid.MustNew("agd_")}
	idle := &agentdefinition.Definition{ID: pulid.MustNew("agd_")}
	f := newFixture(busy, idle)
	f.runService.open[busy.ID] = true

	f.publisher.Publish(t.Context(), agenteventports.AgentEvent{
		Kind:       agenttrigger.EventShipmentMoveUnassigned,
		SubjectID:  pulid.MustNew("smv_"),
		TenantInfo: tenantInfo,
	})

	require.Len(t, f.runService.started, 2, "both are asked; the open one refuses")
}

func TestPublish_IgnoresUnknownKind(t *testing.T) {
	t.Parallel()

	f := newFixture(&agentdefinition.Definition{ID: pulid.MustNew("agd_")})

	f.publisher.Publish(t.Context(), agenteventports.AgentEvent{
		Kind:       agenttrigger.EventKind("not.a.kind"),
		SubjectID:  pulid.MustNew("shp_"),
		TenantInfo: tenantInfo,
	})

	require.Empty(t, f.runService.started)
}

func TestPublish_IgnoresMissingSubjectOrTenant(t *testing.T) {
	t.Parallel()

	f := newFixture(&agentdefinition.Definition{ID: pulid.MustNew("agd_")})

	f.publisher.Publish(t.Context(), agenteventports.AgentEvent{
		Kind:       agenttrigger.EventShipmentCreated,
		TenantInfo: tenantInfo,
	})
	f.publisher.Publish(t.Context(), agenteventports.AgentEvent{
		Kind:      agenttrigger.EventShipmentCreated,
		SubjectID: pulid.MustNew("shp_"),
	})

	require.Empty(t, f.runService.started)
}

func TestPublish_StartFailureDoesNotPanicOrStopOthers(t *testing.T) {
	t.Parallel()

	f := newFixture(
		&agentdefinition.Definition{ID: pulid.MustNew("agd_")},
		&agentdefinition.Definition{ID: pulid.MustNew("agd_")},
	)
	f.runService.err = errors.New("temporal down")

	f.publisher.Publish(t.Context(), agenteventports.AgentEvent{
		Kind:       agenttrigger.EventDocumentExtracted,
		SubjectID:  pulid.MustNew("doc_"),
		TenantInfo: tenantInfo,
	})

	require.Len(t, f.runService.started, 2)
}
