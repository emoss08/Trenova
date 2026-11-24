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

// An entity's reader is internal to records_query and record_get, so a direct
// call is refused with the way to read those records, even when the registry
// still resolves it.
func TestDispatch_RefusesADirectCallToAnEntityReader(t *testing.T) {
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

	assert.True(t, outcome.Failed)
	assert.Zero(t, tool.Calls)
	assert.Contains(t, outcome.Content, `"invoice_list" is not a tool`)
	assert.Contains(t, outcome.Content, "records_query or record_get")
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
