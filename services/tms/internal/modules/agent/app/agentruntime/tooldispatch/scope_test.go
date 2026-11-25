package tooldispatch

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/stretchr/testify/assert"
)

// An entity's reader is not offered on every turn, but a model that names it
// still reads through it: the agent's scope, plan and permission decide, as
// they do for records_query.
func TestDispatch_RunsADirectCallToAnEntityReaderInScope(t *testing.T) {
	t.Parallel()

	tool := &agenttest.StubQueryTool{ToolName: "invoice_list"}
	d := testDispatcher(DispatcherParams{
		QueryTools: &agenttest.StubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{tool}},
	})

	outcome := d.dispatch(t.Context(), &DispatchParams{
		req: &agentruntimeports.RunRequest{
			Definition: runtimetest.Definition(),
			Actor:      runtimetest.Actor(),
		},
		call:  llmports.ToolCall{ID: "call-1", Name: tool.ToolName},
		taint: &agent.RunTaint{},
	})

	assert.False(t, outcome.Failed)
	assert.Equal(t, 1, tool.Calls)
}

func TestDispatch_RefusesAnEntityReaderOutsideTheAgentsScope(t *testing.T) {
	t.Parallel()

	tool := &agenttest.StubQueryTool{ToolName: "invoice_list"}
	d := testDispatcher(DispatcherParams{
		QueryTools: &agenttest.StubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{tool}},
	})

	outcome := d.dispatch(t.Context(), &DispatchParams{
		req: &agentruntimeports.RunRequest{
			Definition: runtimetest.Definition("report_list"),
			Actor:      runtimetest.Actor(),
		},
		call:  llmports.ToolCall{ID: "call-1", Name: tool.ToolName},
		taint: &agent.RunTaint{},
	})

	assert.True(t, outcome.Failed)
	assert.Zero(t, tool.Calls)
	assert.Contains(t, outcome.Content, "outside this agent's scope")
}

func TestDispatch_RefusesAToolOutsideTheAgentsScope(t *testing.T) {
	t.Parallel()

	tool := &agenttest.StubQueryTool{ToolName: "shipment_compare"}
	d := testDispatcher(DispatcherParams{
		QueryTools: &agenttest.StubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{tool}},
	})

	outcome := d.dispatch(t.Context(), &DispatchParams{
		req: &agentruntimeports.RunRequest{
			Definition: runtimetest.Definition("report_list"),
			Actor:      runtimetest.Actor(),
		},
		call:  llmports.ToolCall{ID: "call-1", Name: tool.ToolName},
		taint: &agent.RunTaint{},
	})

	assert.True(t, outcome.Failed)
	assert.Zero(t, tool.Calls)
	assert.Contains(t, outcome.Content, "outside this agent's scope")
}
