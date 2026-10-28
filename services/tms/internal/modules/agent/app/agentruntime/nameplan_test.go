package agentruntime

import (
	"slices"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNamePlan_NamesThePlanTheTurnFiles(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		toolTurn("shipment_create", map[string]any{"customerId": "cus_1"}),
		toolTurn("shipment_create", map[string]any{"customerId": "cus_2"}),
		toolTurn(namePlanName, map[string]any{
			"title": "  Enter the Acme\nand Globex loads  ",
		}),
		textTurn("I proposed both loads."),
	}}

	result, err := proposingRuntime(completion).Run(t.Context(), proposingRequest())
	require.NoError(t, err)

	assert.Equal(t, "Enter the Acme and Globex loads", result.PlanTitle)
	content, failed := toolResult(result, namePlanName)
	assert.False(t, failed)
	assert.Equal(t, `The plan this turn files will be named "Enter the Acme and Globex loads". `+
		"Nothing has changed.", content)
	assert.Contains(t, offered(completion.Requests[0]), namePlanName,
		"declared before the first proposal, so the tool list never changes")
	assertSameToolsEveryCall(t, completion.Requests)
}

func TestNamePlan_SaysTheNameIsUnusedUntilTwoProposalsAreFiled(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		toolTurn("shipment_create", map[string]any{"customerId": "cus_1"}),
		toolTurn(namePlanName, map[string]any{"title": "Enter the Acme load"}),
		textTurn("Proposed."),
	}}

	result, err := proposingRuntime(completion).Run(t.Context(), proposingRequest())
	require.NoError(t, err)

	content, failed := toolResult(result, namePlanName)
	assert.False(t, failed)
	assert.Contains(t, content, "It is used only if this turn files two or more proposals.")
	assert.Equal(t, "Enter the Acme load", result.PlanTitle)
}

func TestNamePlan_TheLastNameGivenIsKept(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		toolTurn(namePlanName, map[string]any{"title": "First name"}),
		toolTurn(namePlanName, map[string]any{"title": "Second name"}),
		textTurn("Done."),
	}}

	result, err := proposingRuntime(completion).Run(t.Context(), proposingRequest())
	require.NoError(t, err)
	assert.Equal(t, "Second name", result.PlanTitle)
}

func TestNamePlan_RefusesANameThatIsEmptyOrTooLong(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		title any
		want  string
	}{
		"empty":     {title: "   ", want: "title must name the plan"},
		"missing":   {title: nil, want: "title must name the plan"},
		"too long":  {title: strings.Repeat("a", 81), want: "title is at most 80 characters"},
		"not words": {title: 42, want: "title must name the plan"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			arguments := map[string]any{}
			if tc.title != nil {
				arguments["title"] = tc.title
			}
			completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
				toolTurn(namePlanName, arguments),
				textTurn("Done."),
			}}

			result, err := proposingRuntime(completion).Run(t.Context(), proposingRequest())
			require.NoError(t, err)

			content, failed := toolResult(result, namePlanName)
			assert.True(t, failed)
			assert.Equal(t, `Tool "name_plan" was not run: `+tc.want, content)
			assert.Empty(t, result.PlanTitle)
		})
	}
}

func TestNamePlan_AcceptsANameOfExactlyTheLimit(t *testing.T) {
	t.Parallel()

	title := strings.Repeat("é", 80)
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		toolTurn(namePlanName, map[string]any{"title": title}),
		textTurn("Done."),
	}}

	result, err := proposingRuntime(completion).Run(t.Context(), proposingRequest())
	require.NoError(t, err)
	assert.Equal(t, title, result.PlanTitle, "the limit counts characters, not bytes")
}

func TestNamePlan_IsNeverOfferedToADelegate(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		toolTurn(namePlanName, map[string]any{"title": "Mine"}),
		textTurn("Proposed."),
	}}
	req := proposingRequest()
	req.Delegation = &agentruntimeports.Delegation{CallID: "call_parent", StepScope: "d1"}

	result, err := proposingRuntime(completion).Run(t.Context(), req)
	require.NoError(t, err)

	for _, request := range completion.Requests {
		assert.False(t, slices.Contains(offered(request), namePlanName))
	}
	content, failed := toolResult(result, namePlanName)
	assert.True(t, failed)
	assert.Equal(t, delegatedNaming, content)
	assert.Empty(t, result.PlanTitle)
}

func TestNamePlan_IsNotDeclaredWhereNothingCanBeProposed(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		toolTurn(namePlanName, map[string]any{"title": "Nothing"}),
		textTurn("Nothing to do."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{
		Tools: []agenttoolports.AgentQueryTool{queryTool("shipment_get", map[string]any{}, nil)},
	}, &stubActionRegistry{}, nil)

	result, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: testDefinition("shipment_get"),
		Actor:      testActor(),
		Input:      "where is S-1",
	})
	require.NoError(t, err)
	assert.NotContains(t, offered(completion.Requests[0]), namePlanName)
	content, failed := toolResult(result, namePlanName)
	assert.True(t, failed)
	assert.Equal(t, unofferedNaming, content)
}
