package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_HandsAToolTheRunsTimezoneWhereverItIsAsked(t *testing.T) {
	t.Parallel()

	tool := &validatingActionTool{
		stubActionToolAlias: actionTool("report_update", agent.TierAutoExecute, nil),
	}
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn("report_update", map[string]any{"definitionId": "rd_1"}),
		runtimetest.TextTurn("Done."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agenttoolports.AgentTool{tool}}, nil)

	_, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.AutoDefinition("report_update"),
		Actor:      runtimetest.Actor(),
		Input:      "update",
		Context:    agentturn.RuntimeContext{Timezone: "America/Chicago"},
	})
	require.NoError(t, err)

	assert.Equal(t, "America/Chicago", tool.lastSeen.Timezone, "the check reads times in it")
	assert.Equal(t, "America/Chicago", tool.LastParams.Timezone, "the write reads times in it")
}

func TestRun_TheFilingBaselineReadsTheRunsTimezone(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	tool := &targetedStubTool{actionTool("shipment_place_hold", agent.TierPropose, nil)}
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		runtimetest.ToolTurn(
			"shipment_place_hold",
			map[string]any{"shipmentId": shipmentID.String()},
		),
		runtimetest.TextTurn("I have proposed a hold."),
	}}
	previews := &recordingPreviews{target: &agentruntimeports.ProposalTarget{
		Resource: permission.ResourceShipment,
		ID:       shipmentID,
		Version:  7,
	}}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{
		Tools: []agenttoolports.AgentTool{tool},
	}, nil)
	rewire(rt, func(p *Params) { p.Versions = stubVersions{version: 7} })
	rewire(rt, func(p *Params) { p.Previews = previews })

	_, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition("shipment_place_hold"),
		Actor:      runtimetest.Actor(),
		Input:      "hold it",
		Context:    agentturn.RuntimeContext{Timezone: "America/Denver"},
	})
	require.NoError(t, err)

	require.Len(t, previews.asked, 1)
	assert.Equal(t, "America/Denver", previews.asked[0].Params.Timezone)
}
