package agentevents_test

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentevents"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeDefinitions struct {
	repositories.AgentDefinitionRepository
	items   []*agentdefinition.Definition
	lastReq repositories.ListAgentDefinitionsByTriggerRequest
}

func (f *fakeDefinitions) ListEnabledByTrigger(
	_ context.Context,
	req repositories.ListAgentDefinitionsByTriggerRequest,
) ([]*agentdefinition.Definition, error) {
	f.lastReq = req
	return f.items, nil
}

type fakeRuns struct {
	repositories.AgentRunRepository
	open map[pulid.ID]int
}

func (f *fakeRuns) CountOpen(
	_ context.Context,
	req repositories.CountOpenAgentRunsRequest,
) (int, error) {
	return f.open[req.DefinitionID], nil
}

type fakeRunService struct {
	serviceports.AgentRunService
	started []*serviceports.StartAgentRunForDefinitionRequest
	err     error
}

func (f *fakeRunService) StartForDefinition(
	_ context.Context,
	req *serviceports.StartAgentRunForDefinitionRequest,
	_ *serviceports.RequestActor,
) (*agent.AgentRun, error) {
	f.started = append(f.started, req)
	if f.err != nil {
		return nil, f.err
	}
	return &agent.AgentRun{ID: pulid.MustNew("ar_")}, nil
}

var tenantInfo = pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

type fixture struct {
	definitions *fakeDefinitions
	runs        *fakeRuns
	runService  *fakeRunService
	publisher   serviceports.AgentEventPublisher
}

func newFixture(defs ...*agentdefinition.Definition) *fixture {
	f := &fixture{
		definitions: &fakeDefinitions{items: defs},
		runs:        &fakeRuns{open: map[pulid.ID]int{}},
		runService:  &fakeRunService{},
	}
	f.publisher = agentevents.New(agentevents.Params{
		Logger:      zap.NewNop(),
		Definitions: f.definitions,
		Runs:        f.runs,
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

	f.publisher.Publish(t.Context(), serviceports.AgentEvent{
		Kind:       agent.EventBillingQueueItemException,
		SubjectID:  subject,
		TenantInfo: tenantInfo,
	})

	require.Equal(t, agentdefinition.TriggerEvent, f.definitions.lastReq.Mode)
	require.Equal(t, agent.EventBillingQueueItemException, f.definitions.lastReq.EventKind)
	require.Equal(t, tenantInfo, f.definitions.lastReq.TenantInfo)
	require.Len(t, f.runService.started, 2)
	for _, req := range f.runService.started {
		require.Equal(t, agent.RunTriggerEvent, req.Trigger)
		require.Equal(t, agent.EventBillingQueueItemException, req.EventKind)
		require.Equal(t, agent.SubjectBillingQueueItem, req.SubjectType)
		require.Equal(t, subject, req.SubjectID)
		require.Equal(t, tenantInfo, req.TenantInfo)
	}
}

func TestPublish_SkipsDefinitionWithOpenRunForSubject(t *testing.T) {
	t.Parallel()

	busy := &agentdefinition.Definition{ID: pulid.MustNew("agd_")}
	idle := &agentdefinition.Definition{ID: pulid.MustNew("agd_")}
	f := newFixture(busy, idle)
	f.runs.open[busy.ID] = 1

	f.publisher.Publish(t.Context(), serviceports.AgentEvent{
		Kind:       agent.EventShipmentMoveUnassigned,
		SubjectID:  pulid.MustNew("smv_"),
		TenantInfo: tenantInfo,
	})

	require.Len(t, f.runService.started, 1)
	require.Equal(t, idle.ID, f.runService.started[0].DefinitionID)
}

func TestPublish_IgnoresUnknownKind(t *testing.T) {
	t.Parallel()

	f := newFixture(&agentdefinition.Definition{ID: pulid.MustNew("agd_")})

	f.publisher.Publish(t.Context(), serviceports.AgentEvent{
		Kind:       agent.EventKind("not.a.kind"),
		SubjectID:  pulid.MustNew("shp_"),
		TenantInfo: tenantInfo,
	})

	require.Empty(t, f.runService.started)
}

func TestPublish_IgnoresMissingSubjectOrTenant(t *testing.T) {
	t.Parallel()

	f := newFixture(&agentdefinition.Definition{ID: pulid.MustNew("agd_")})

	f.publisher.Publish(t.Context(), serviceports.AgentEvent{
		Kind:       agent.EventShipmentCreated,
		TenantInfo: tenantInfo,
	})
	f.publisher.Publish(t.Context(), serviceports.AgentEvent{
		Kind:      agent.EventShipmentCreated,
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

	f.publisher.Publish(t.Context(), serviceports.AgentEvent{
		Kind:       agent.EventDocumentExtracted,
		SubjectID:  pulid.MustNew("doc_"),
		TenantInfo: tenantInfo,
	})

	require.Len(t, f.runService.started, 2)
}
