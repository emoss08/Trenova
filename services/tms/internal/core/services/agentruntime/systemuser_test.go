package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRun_AnUnattendedSystemUserCannotReachASelfScopedTool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		unattended bool
	}{
		{name: "an unattended run", unattended: true},
		{name: "an agent principal outside an unattended run", unattended: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
				callsTurn(serviceports.ToolCall{
					ID:        "home",
					Name:      "add_home_widget",
					Arguments: map[string]any{"key": "kpi"},
				}),
				textTurn("done"),
			}}
			service, permissions := selfScopedRuntime(completion)
			actor := systemUserActor()
			req := &serviceports.RunRequest{
				Definition: testDefinition("add_home_widget"),
				Actor:      actor,
				Input:      "tidy the home page",
				Unattended: tt.unattended,
			}
			events := recordEvents(req)

			result, err := service.Run(t.Context(), req)
			require.NoError(t, err)

			assert.Empty(t, result.Actions, "a self-scoped write is never made for the system user")
			require.Len(t, toolMessages(result), 1)
			assert.True(t, toolMessages(result)[0].ToolFailed)
			assert.Contains(t, toolMessages(result)[0].Content, "nobody is in this one")
			assert.Equal(t, aitrace.OutcomeDenied, finishedVerdicts(events)["home"])
			for _, request := range permissions.Requests {
				assert.Equal(t, serviceports.PrincipalTypeAgent, request.PrincipalType,
					"every check is made as the agent, never as the system user")
			}
		})
	}
}

func TestRun_AnUnattendedRunIsCheckedAsTheAgentNotItsSystemUser(t *testing.T) {
	t.Parallel()

	tool := actionTool("update_worker", agent.TierAutoExecute, nil)
	tool.Resource = permission.ResourceWorker
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		callsTurn(serviceports.ToolCall{
			ID:        "write",
			Name:      "update_worker",
			Arguments: map[string]any{"workerId": "wrk_1"},
		}),
		textTurn("done"),
	}}
	permissions := &agentruntimetest.StubPermissions{}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []serviceports.AgentTool{tool}}, permissions)
	actor := systemUserActor()

	_, err := rt.Run(t.Context(), &serviceports.RunRequest{
		Definition: autoDefinition("update_worker"),
		Actor:      actor,
		Input:      "A worker changed.",
		Unattended: true,
	})
	require.NoError(t, err)

	require.NotEmpty(t, permissions.Requests)
	for _, request := range permissions.Requests {
		assert.Equal(t, serviceports.PrincipalTypeAgent, request.PrincipalType)
		assert.Equal(t, serviceports.AgentPrincipalID, request.PrincipalID)
	}
	assert.Equal(t, actor.UserID, tool.LastParams.Actor.UserID,
		"the write is attributed to the system user")
	assert.Equal(t, serviceports.PrincipalTypeAgent, tool.LastParams.Actor.PrincipalType)
}

func TestContextBuilder_NeverDescribesTheSystemUserAsThePerson(t *testing.T) {
	t.Parallel()

	builder := &ContextBuilder{
		logger:        zap.NewNop(),
		organizations: &stubOrganizations{org: &tenant.Organization{Timezone: "America/Denver"}},
		users:         &stubUsers{user: &tenant.User{Name: "System Account"}},
		runtime: newRuntime(
			&scriptedCompletion{},
			&stubQueryRegistry{},
			&stubActionRegistry{},
			nil,
		),
	}
	definition := testDefinition()
	definition.ContextProviders = []agentdefinition.ContextProvider{agentdefinition.ContextUser}

	unattended, err := builder.Build(t.Context(), &serviceports.RuntimeContextRequest{
		Definition: definition,
		Actor:      systemUserActor(),
	})
	require.NoError(t, err)
	assert.Nil(t, unattended.User, "the system user is not someone the agent works for")

	person, err := builder.Build(t.Context(), &serviceports.RuntimeContextRequest{
		Definition: definition,
		Actor:      testActor(),
	})
	require.NoError(t, err)
	require.NotNil(t, person.User)
}

func TestTurnAttribution_UsageIsNotChargedToTheSystemUser(t *testing.T) {
	t.Parallel()

	unattended := turnAttribution(&serviceports.RunRequest{
		Definition: testDefinition(),
		Actor:      systemUserActor(),
	})
	assert.Equal(t, pulid.Nil, unattended.UserID)

	actor := testActor()
	person := turnAttribution(&serviceports.RunRequest{Definition: testDefinition(), Actor: actor})
	assert.Equal(t, actor.UserID, person.UserID)
}
