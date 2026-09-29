package agentruntime

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type resolvingTool struct {
	*agentruntimetest.StubActionTool

	resolved map[string]any
	err      error
	asked    []map[string]any
}

func (r *resolvingTool) ResolveSelection(
	_ context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the interface passes params by value
) (map[string]any, error) {
	r.asked = append(r.asked, params.Params)
	if r.err != nil {
		return nil, r.err
	}

	return r.resolved, nil
}

func runSelection(
	t *testing.T,
	tool *resolvingTool,
	auto bool,
) *serviceports.RunResult {
	t.Helper()

	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("transfer_to_billing", map[string]any{"allTransferable": true}),
		textTurn("Done."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []serviceports.AgentTool{tool}}, nil)
	definition := testDefinition("transfer_to_billing")
	if auto {
		definition = autoDefinition("transfer_to_billing")
	}

	result, err := rt.Run(t.Context(), &serviceports.RunRequest{
		Definition: definition,
		Actor:      testActor(),
		Input:      "Transfer all of them.",
	})
	require.NoError(t, err)

	return result
}

func TestDispatch_ProposesTheRecordsASelectionResolvesTo(t *testing.T) {
	t.Parallel()

	tool := &resolvingTool{
		StubActionTool: &agentruntimetest.StubActionTool{
			ToolName: "transfer_to_billing",
			Tier:     agent.TierPropose,
		},
		resolved: map[string]any{"shipmentIds": []any{"shp_1", "shp_2"}},
	}

	result := runSelection(t, tool, false)

	require.Len(t, tool.asked, 1)
	assert.Equal(t, map[string]any{"allTransferable": true}, tool.asked[0])
	require.Len(t, result.Actions, 1)
	assert.Equal(t,
		map[string]any{"shipmentIds": []any{"shp_1", "shp_2"}},
		result.Actions[0].Arguments,
		"what a person approves is the records, never the criteria")
	assert.Zero(t, tool.Calls)
	var sent map[string]any
	for _, message := range result.Messages {
		if len(message.ToolCalls) > 0 {
			sent = message.ToolCalls[0].Arguments
		}
	}
	assert.Equal(t, map[string]any{"allTransferable": true}, sent,
		"the model's own call is kept as it sent it")
}

func TestDispatch_RunsAnAutomaticWriteOnTheResolvedRecords(t *testing.T) {
	t.Parallel()

	tool := &resolvingTool{
		StubActionTool: &agentruntimetest.StubActionTool{
			ToolName: "transfer_to_billing",
			Tier:     agent.TierAutoExecute,
		},
		resolved: map[string]any{"shipmentIds": []any{"shp_1"}},
	}

	runSelection(t, tool, true)

	require.Equal(t, 1, tool.Calls)
	assert.Equal(t, map[string]any{"shipmentIds": []any{"shp_1"}}, tool.LastParams.Params)
}

func TestDispatch_RefusesASelectionThatCannotBeResolved(t *testing.T) {
	t.Parallel()

	tool := &resolvingTool{
		StubActionTool: &agentruntimetest.StubActionTool{
			ToolName: "transfer_to_billing",
			Tier:     agent.TierPropose,
		},
		err: errors.New("none of the 4 shipments these filters select would transfer"),
	}

	result := runSelection(t, tool, false)

	assert.Empty(t, result.Actions)
	refusals := toolMessages(result)
	require.Len(t, refusals, 1)
	assert.True(t, refusals[0].ToolFailed)
	assert.Contains(t, refusals[0].Content, "none of the 4 shipments")
	assert.Contains(t, refusals[0].Content, "was not proposed")
}
