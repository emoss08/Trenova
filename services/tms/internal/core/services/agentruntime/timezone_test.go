package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_HandsAToolTheRunsTimezoneWhereverItIsAsked(t *testing.T) {
	t.Parallel()

	tool := &validatingActionTool{
		stubActionToolAlias: actionTool("update_report", agent.TierAutoExecute, nil),
	}
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("update_report", map[string]any{"definitionId": "rd_1"}),
		textTurn("Done."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{},
		&stubActionRegistry{Tools: []serviceports.AgentTool{tool}}, nil)

	_, err := rt.Run(t.Context(), &serviceports.RunRequest{
		Definition: autoDefinition("update_report"),
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
	tool := &targetedStubTool{actionTool("place_shipment_hold", agent.TierPropose, nil)}
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("place_shipment_hold", map[string]any{"shipmentId": shipmentID.String()}),
		textTurn("I have proposed a hold."),
	}}
	previews := &recordingPreviews{target: &serviceports.ProposalTarget{
		Resource: permission.ResourceShipment,
		ID:       shipmentID,
		Version:  7,
	}}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{
		Tools: []serviceports.AgentTool{tool},
	}, nil)
	rt.versions = stubVersions{version: 7}
	rt.previews = previews

	_, err := rt.Run(t.Context(), &serviceports.RunRequest{
		Definition: testDefinition("place_shipment_hold"),
		Actor:      testActor(),
		Input:      "hold it",
		Context:    agentdefinition.RuntimeContext{Timezone: "America/Denver"},
	})
	require.NoError(t, err)

	require.Len(t, previews.asked, 1)
	assert.Equal(t, "America/Denver", previews.asked[0].Params.Timezone)
}
