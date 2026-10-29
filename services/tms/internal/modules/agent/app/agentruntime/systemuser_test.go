package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimecontext"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
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

			completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
				callsTurn(llmports.ToolCall{
					ID:        "home",
					Name:      "home_add_widget",
					Arguments: map[string]any{"key": "kpi"},
				}),
				runtimetest.TextTurn("done"),
			}}
			service, permissions := selfScopedRuntime(completion)
			actor := runtimetest.SystemUserActor()
			req := &agentruntimeports.RunRequest{
				Definition: runtimetest.Definition("home_add_widget"),
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
				assert.Equal(t, permissionports.PrincipalTypeAgent, request.PrincipalType,
					"every check is made as the agent, never as the system user")
			}
		})
	}
}

func TestRun_AnUnattendedRunIsCheckedAsTheAgentNotItsSystemUser(t *testing.T) {
	t.Parallel()

	tool := actionTool("update_worker", agent.TierAutoExecute, nil)
	tool.Resource = permission.ResourceWorker
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		callsTurn(llmports.ToolCall{
			ID:        "write",
			Name:      "update_worker",
			Arguments: map[string]any{"workerId": "wrk_1"},
		}),
		runtimetest.TextTurn("done"),
	}}
	permissions := &agenttest.StubPermissions{}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agenttoolports.AgentTool{tool}}, permissions)
	actor := runtimetest.SystemUserActor()

	_, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.AutoDefinition("update_worker"),
		Actor:      actor,
		Input:      "A worker changed.",
		Unattended: true,
	})
	require.NoError(t, err)

	require.NotEmpty(t, permissions.Requests)
	for _, request := range permissions.Requests {
		assert.Equal(t, permissionports.PrincipalTypeAgent, request.PrincipalType)
		assert.Equal(t, permissionports.AgentPrincipalID, request.PrincipalID)
	}
	assert.Equal(t, actor.UserID, tool.LastParams.Actor.UserID,
		"the write is attributed to the system user")
	assert.Equal(t, permissionports.PrincipalTypeAgent, tool.LastParams.Actor.PrincipalType)
}

func TestContextBuilder_NeverDescribesTheSystemUserAsThePerson(t *testing.T) {
	t.Parallel()

	builder := runtimecontext.NewBuilder(runtimecontext.BuilderParams{
		Logger: zap.NewNop(),
		Organizations: &runtimetest.StubOrganizations{
			Org: &tenant.Organization{Timezone: "America/Denver"},
		},
		Users: &runtimetest.StubUsers{User: &tenant.User{Name: "System Account"}},
		Runtime: newRuntime(
			&scriptedCompletion{},
			&stubQueryRegistry{},
			&stubActionRegistry{},
			nil,
		),
	})
	definition := runtimetest.Definition()
	definition.ContextProviders = []agentdefinition.ContextProvider{agentdefinition.ContextUser}

	unattended, err := builder.Build(t.Context(), &agentruntimeports.RuntimeContextRequest{
		Definition: definition,
		Actor:      runtimetest.SystemUserActor(),
	})
	require.NoError(t, err)
	assert.Nil(t, unattended.User, "the system user is not someone the agent works for")

	person, err := builder.Build(t.Context(), &agentruntimeports.RuntimeContextRequest{
		Definition: definition,
		Actor:      runtimetest.Actor(),
	})
	require.NoError(t, err)
	require.NotNil(t, person.User)
}
