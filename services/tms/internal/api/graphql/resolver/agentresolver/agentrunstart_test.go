package agentresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type startingRuns struct {
	agentruntimeports.AgentRunService

	requests []*agentruntimeports.StartAgentRunForDefinitionRequest
	actors   []*permissionports.RequestActor
}

func (s *startingRuns) StartForDefinition(
	_ context.Context,
	req *agentruntimeports.StartAgentRunForDefinitionRequest,
	actor *permissionports.RequestActor,
) (*agent.AgentRun, error) {
	s.requests = append(s.requests, req)
	s.actors = append(s.actors, actor)
	return &agent.AgentRun{
		ID:                pulid.MustNew("ar_"),
		AgentDefinitionID: req.DefinitionID,
		Status:            agent.RunStatusPending,
	}, nil
}

func TestStartAgentRun_NeedsAgentRunCreate(t *testing.T) {
	t.Parallel()

	engine := resolvertest.RefusingEngine()
	deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
	ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

	_, err := (&MutationResolver{deps}).StartAgentRun(ctx, gqlmodel.StartAgentRunInput{})

	resolvertest.RequireRefused(t, err, engine, permission.ResourceAgentRun, permission.OpCreate)
}

func TestStartAgentRun_StartsTheAgentManuallyAsTheCaller(t *testing.T) {
	t.Parallel()

	runs := &startingRuns{}
	deps := &Deps{
		Core:            &resolvercore.Core{PermissionEngine: &mocks.AllowAllPermissionEngine{}},
		AgentRunService: runs,
	}
	authCtx := resolvertest.UserAuthContext()
	ctx := resolvertest.AuthedContext(t, authCtx)
	definitionID := pulid.MustNew("agdef_")
	raw := definitionID.String()
	subject := agent.SubjectBillingQueueItem
	subjectID := pulid.MustNew("bqi_")
	subjectRaw := subjectID.String()

	run, err := (&MutationResolver{deps}).StartAgentRun(ctx, gqlmodel.StartAgentRunInput{
		AgentDefinitionID: &raw,
		SubjectType:       &subject,
		SubjectID:         &subjectRaw,
	})
	require.NoError(t, err)

	require.Len(t, runs.requests, 1)
	req := runs.requests[0]
	assert.Equal(t, definitionID, req.DefinitionID)
	assert.Equal(t, agent.SubjectBillingQueueItem, req.SubjectType)
	assert.Equal(t, subjectID, req.SubjectID)
	assert.Empty(t, req.SystemKey)
	assert.Equal(t, agent.RunTriggerManual, req.Trigger)
	assert.Equal(t, authCtx.OrganizationID, req.TenantInfo.OrgID)
	assert.Equal(t, authCtx.UserID, req.TenantInfo.UserID)
	assert.Equal(t, authCtx.UserID, runs.actors[0].UserID)
	assert.Equal(t, definitionID, run.AgentDefinitionID)
}
