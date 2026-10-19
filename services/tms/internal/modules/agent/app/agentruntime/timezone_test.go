package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
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
	completion := &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
		toolTurn("report_update", map[string]any{"definitionId": "rd_1"}),
		textTurn("Done."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []agentports.AgentTool{tool}}, nil)

	_, err := rt.Run(t.Context(), &agentports.RunRequest{
		Definition: autoDefinition("report_update"),
		Actor:      testActor(),
		Input:      "update",
		Context:    agentdefinition.RuntimeContext{Timezone: "America/Chicago"},
	})
	require.NoError(t, err)

	assert.Equal(t, "America/Chicago", tool.lastSeen.Timezone, "the check reads times in it")
	assert.Equal(t, "America/Chicago", tool.LastParams.Timezone, "the write reads times in it")
}

func TestRun_TheFilingBaselineReadsTheRunsTimezone(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	tool := &targetedStubTool{actionTool("shipment_place_hold", agent.TierPropose, nil)}
	completion := &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
		toolTurn("shipment_place_hold", map[string]any{"shipmentId": shipmentID.String()}),
		textTurn("I have proposed a hold."),
	}}
	previews := &recordingPreviews{target: &agentports.ProposalTarget{
		Resource: permission.ResourceShipment,
		ID:       shipmentID,
		Version:  7,
	}}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{
		Tools: []agentports.AgentTool{tool},
	}, nil)
	rt.versions = stubVersions{version: 7}
	rt.previews = previews

	_, err := rt.Run(t.Context(), &agentports.RunRequest{
		Definition: testDefinition("shipment_place_hold"),
		Actor:      testActor(),
		Input:      "hold it",
		Context:    agentdefinition.RuntimeContext{Timezone: "America/Denver"},
	})
	require.NoError(t, err)

	require.Len(t, previews.asked, 1)
	assert.Equal(t, "America/Denver", previews.asked[0].Params.Timezone)
}
