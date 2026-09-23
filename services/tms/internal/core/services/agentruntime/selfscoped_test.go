package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/emoss08/trenova/internal/core/services/agenttoolcatalog"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// selfScopedAction is a write on the person's own records, like the home page.
type selfScopedAction struct {
	*agentruntimetest.StubActionTool
}

func (selfScopedAction) SelfScoped() bool { return true }

func selfScopedRuntime(
	completion *scriptedCompletion,
) (*Service, *agentruntimetest.StubPermissions) {
	permissions := &agentruntimetest.StubPermissions{Denied: map[string]bool{
		string(permission.ResourceHomeLayoutPreset) + ":" + string(permission.OpUpdate): true,
	}}
	action := &stubActionRegistry{Tools: []serviceports.AgentTool{
		selfScopedAction{actionTool("add_home_widget", agent.TierPropose, nil)},
	}}
	action.Tools[0].(selfScopedAction).Resource = permission.ResourceHomeLayoutPreset

	return newRuntime(completion, &stubQueryRegistry{}, action, permissions), permissions
}

/*
A person may arrange their own home page without a role grant for it, as the
home page's own editor lets them; the runtime does not ask the permission
engine about a tool that only touches the person's own records. What the model
sent as the owner is replaced by who is actually in the conversation.
*/
func TestRun_OffersASelfScopedToolToThePersonWithoutAGrant(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("add_home_widget", map[string]any{
			"key":                            "kpi",
			serviceports.SelfScopeOwnerParam: pulid.MustNew("usr_").String(),
		}),
		textTurn("Proposed."),
	}}
	service, permissions := selfScopedRuntime(completion)
	actor := testActor()

	result, err := service.Run(t.Context(), &serviceports.RunRequest{
		Definition: testDefinition("add_home_widget"),
		Actor:      actor,
		Input:      "put on-time on my dashboard",
	})
	require.NoError(t, err)

	assert.Contains(t, requestTools(completion.Requests[0]), "add_home_widget")
	require.Len(t, result.Actions, 1)
	assert.Equal(t, actor.UserID.String(),
		result.Actions[0].Arguments[serviceports.SelfScopeOwnerParam],
		"the owner is the person in the conversation, whatever the model wrote")
	for _, request := range permissions.Requests {
		assert.NotEqual(t, string(permission.ResourceHomeLayoutPreset), request.Resource)
	}
}

// Nobody is in a background run, so there are no own records to touch.
func TestRun_WithholdsASelfScopedToolFromARunNobodyIsWatching(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("add_home_widget", map[string]any{"key": "kpi"}),
		textTurn("done"),
	}}
	service, _ := selfScopedRuntime(completion)

	result, err := service.Run(t.Context(), &serviceports.RunRequest{
		Definition: testDefinition("add_home_widget"),
		Actor:      testActor(),
		Input:      "tidy the home page",
		Unattended: true,
	})
	require.NoError(t, err)

	assert.NotContains(t, requestTools(completion.Requests[0]), "add_home_widget")
	assert.Empty(t, result.Actions)
	assert.True(t, result.Messages[2].ToolFailed)
}

func TestRun_WithholdsASelfScopedToolFromAnAgentPrincipal(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		textTurn("done"),
	}}
	service, _ := selfScopedRuntime(completion)
	actor := testActor()
	actor.PrincipalType = serviceports.PrincipalTypeAgent

	_, err := service.Run(t.Context(), &serviceports.RunRequest{
		Definition: testDefinition("add_home_widget"),
		Actor:      actor,
		Input:      "tidy the home page",
	})
	require.NoError(t, err)

	assert.NotContains(t, requestTools(completion.Requests[0]), "add_home_widget")
}

// searchableQuery is a read with the words people use for it.
type searchableQuery struct {
	*agentruntimetest.StubQueryTool
	terms []string
}

func (s searchableQuery) SearchTerms() []string { return s.terms }

/*
The Homepage Widget Builder transcript, replayed: "tell me what I have on my
dashboard" reaches the home page, not only the report dashboards.
*/
func TestNewToolSet_MyDashboardReachesTheHomePage(t *testing.T) {
	t.Parallel()

	service, names := wideRuntime(t)
	query := service.queryTools.(*stubQueryRegistry)
	query.Tools = append(query.Tools,
		searchableQuery{
			StubQueryTool: describedTool("get_my_home_layout",
				"Read what is on the person's own home page: each widget and its settings."),
			terms: []string{"my dashboard", "home page", "widgets"},
		},
		describedTool("list_dashboards", "List the report dashboards under Reports, with their tiles."),
	)
	service.catalog = agenttoolcatalog.NewFromRegistries(agenttoolcatalog.Params{
		QueryTools:  query,
		ActionTools: service.actionTools,
	})

	set := service.newToolSet(t.Context(), toolSetRequest{
		definition: testDefinition(append(names, "get_my_home_layout", "list_dashboards")...),
		actor:      testActor(),
		input:      "Tell me what I have on my dashboard currently",
	})

	require.True(t, set.disclosed)
	sent := specNames(set.specs)
	assert.Contains(t, sent, "get_my_home_layout")
	assert.Contains(t, sent, "list_dashboards")
}
