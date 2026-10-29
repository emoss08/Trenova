package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The model sent an owner for a write on the person's own records. The key
// is not a parameter the tool declares, so a closed schema would refuse it;
// it is the runtime's, set aside before the check and stamped from the
// person in the conversation after.
func TestRun_StripsTheOwnerTheModelSentAndStampsThePerson(t *testing.T) {
	t.Parallel()

	stub := actionTool("home_add_widget", agent.TierPropose, nil)
	stub.Resource = permission.ResourceHomeLayoutPreset
	stub.Schema = runtimetest.ClosedSchema("key")
	tool := selfScopedAction{stub}
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn("home_add_widget", map[string]any{
			"key":                              "kpi",
			agenttoolports.SelfScopeOwnerParam: pulid.MustNew("usr_").String(),
		}),
		runtimetest.TextTurn("Proposed."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agenttoolports.AgentTool{tool}}, nil)
	actor := runtimetest.Actor()

	result, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition("home_add_widget"),
		Actor:      actor,
		Input:      "put on-time on my dashboard",
	})
	require.NoError(t, err)

	require.Len(t, result.Actions, 1, "the call fit the schema once the owner was set aside")
	assert.Equal(t, map[string]any{
		"key":                              "kpi",
		agenttoolports.SelfScopeOwnerParam: actor.UserID.String(),
	}, result.Actions[0].Arguments)
}

// A read is held to its schema as a write is. A list tool given a filter it
// does not declare used to list everything, and the model read the answer
// as if the filter had applied.
func TestRun_ValidatesAQueryToolsArguments(t *testing.T) {
	t.Parallel()

	tool := &agenttest.StubQueryTool{
		ToolName: "shipment_list",
		Result:   map[string]any{"items": []any{}},
		Schema:   runtimetest.ClosedSchema("status"),
	}
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn("shipment_list", map[string]any{"status": "New", "customer": "Acme"}),
		runtimetest.TextTurn("I could not list them."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{tool}},
		&stubActionRegistry{}, nil)

	result, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition("shipment_list"),
		Actor:      runtimetest.Actor(),
		Input:      "Acme's new loads",
	})
	require.NoError(t, err)

	assert.Zero(t, tool.Calls, "the read never ran")
	assert.Equal(t, 1, result.ToolCallsUsed)
	refusals := toolMessages(result)
	require.Len(t, refusals, 1)
	assert.True(t, refusals[0].ToolFailed)
	requireRefusalNames(t, refusals[0].Content, []string{"customer"})
}
