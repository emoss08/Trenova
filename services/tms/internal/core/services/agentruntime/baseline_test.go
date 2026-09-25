package agentruntime

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const bulkySummary = "Would change PRO-100 in ways that take many words to say."

type recordingPreviews struct {
	serviceports.ProposalPreviewService

	mu     sync.Mutex
	asked  []*serviceports.ProposalBaselineRequest
	target *serviceports.ProposalTarget
}

func (p *recordingPreviews) Baseline(
	_ context.Context,
	req *serviceports.ProposalBaselineRequest,
) *serviceports.ProposalBaselineResult {
	p.mu.Lock()
	p.asked = append(p.asked, req)
	p.mu.Unlock()

	fields := make([]agent.PreviewFieldChange, 0, 40)
	for range 40 {
		fields = append(fields, agent.PreviewFieldChange{
			Path:  "notes",
			Label: "Notes",
			After: strings.Repeat("n", 1000),
		})
	}

	return &serviceports.ProposalBaselineResult{
		Target: p.target,
		Preview: &agent.ToolPreview{
			Summary: bulkySummary,
			Changes: []agent.RecordChange{{Operation: agent.PreviewOperationUpdate, Fields: fields}},
		},
	}
}

func heldRun(
	t *testing.T,
	previews serviceports.ProposalPreviewService,
	purpose serviceports.AIUsagePurpose,
	shipmentID pulid.ID,
) *serviceports.RunResult {
	t.Helper()

	tool := &targetedStubTool{actionTool("place_shipment_hold", agent.TierPropose, nil)}
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("place_shipment_hold", map[string]any{"shipmentId": shipmentID.String()}),
		textTurn("I have proposed a hold."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{
		Tools: []serviceports.AgentTool{tool},
	}, nil)
	rt.versions = stubVersions{version: 7}
	rt.previews = previews

	result, err := rt.Run(t.Context(), &serviceports.RunRequest{
		Definition:   testDefinition("place_shipment_hold"),
		Actor:        testActor(),
		Input:        "hold it",
		UsagePurpose: purpose,
	})
	require.NoError(t, err)
	require.Len(t, result.Actions, 1)

	return result
}

// The baseline is kept beside the proposal, keyed by the id minted for it,
// never on the action: the action is an activity result that every later
// tool activity of the turn carries, so what it holds is paid for in the
// workflow's history once per proposal and per later call.
func TestRun_KeepsTheBaselineOffTheAction(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	without := heldRun(t, nil, serviceports.AIUsagePurposeLive, shipmentID)

	previews := &recordingPreviews{target: &serviceports.ProposalTarget{
		Resource: permission.ResourceShipment,
		ID:       shipmentID,
		Version:  7,
	}}
	with := heldRun(t, previews, serviceports.AIUsagePurposeLive, shipmentID)

	require.Len(t, previews.asked, 1)
	asked := previews.asked[0]
	assert.True(t, asked.Persist)
	assert.Equal(t, with.Actions[0].ProposalID, asked.ProposalID,
		"the baseline is keyed by the proposal the action becomes")
	assert.Equal(t, shipmentID.String(), asked.Params.Params["shipmentId"])

	action := with.Actions[0]
	assert.Nil(t, action.Simulation)
	encoded, err := sonic.Marshal(action)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), bulkySummary)

	before, err := sonic.Marshal(without.Actions[0])
	require.NoError(t, err)
	assert.Len(t, encoded, len(before), "the action is exactly as big as it was before baselines")
	assert.Equal(t, without.Actions[0].Target, action.Target)
}

func TestRun_TakesTheTargetFromTheBaselineSnapshot(t *testing.T) {
	t.Parallel()

	previews := &recordingPreviews{target: &serviceports.ProposalTarget{
		Resource: permission.ResourceShipment,
		ID:       pulid.MustNew("shp_"),
		Version:  11,
	}}
	result := heldRun(t, previews, serviceports.AIUsagePurposeLive, pulid.MustNew("shp_"))

	require.NotNil(t, result.Actions[0].Target)
	assert.Equal(t, int64(11), result.Actions[0].Target.Version,
		"the version is the one read with the preview, not a second read")
}

// An automatic write in simulation is previewed from the same snapshot as its
// pin, and the simulation it records is read from that preview.
func TestRun_ASimulatedWriteReadsTheBaselinePreview(t *testing.T) {
	t.Parallel()

	tool := &targetedStubTool{actionTool("place_shipment_hold", agent.TierAutoExecute, nil)}
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("place_shipment_hold", map[string]any{"shipmentId": pulid.MustNew("shp_").String()}),
		textTurn("held"),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{
		Tools: []serviceports.AgentTool{tool},
	}, nil)
	previews := &recordingPreviews{}
	rt.previews = previews
	definition := testDefinition("place_shipment_hold")
	definition.AutonomyCeiling = agent.TierAutoExecute
	definition.SimulationMode = true

	result, err := rt.Run(t.Context(), &serviceports.RunRequest{
		Definition: definition,
		Actor:      testActor(),
		Input:      "hold it",
	})
	require.NoError(t, err)

	require.Len(t, result.Actions, 1)
	action := result.Actions[0]
	require.True(t, action.Simulated)
	require.NotNil(t, action.Simulation)
	assert.Equal(t, bulkySummary, action.Simulation.Summary)
	assert.LessOrEqual(t, agent.EncodedPreviewSize(action.Simulation),
		agent.MaxPreviewSimulationBytes+1024, "the simulation on the action stays bounded")
	assert.Zero(t, tool.Calls, "nothing ran")
	require.Len(t, previews.asked, 1)
	assert.False(t, previews.asked[0].Persist, "a simulated write files no baseline")
}

func TestRun_AnEvaluationKeepsNoBaseline(t *testing.T) {
	t.Parallel()

	previews := &recordingPreviews{}
	heldRun(t, previews, serviceports.AIUsagePurposeEvaluation, pulid.MustNew("shp_"))

	require.Len(t, previews.asked, 1)
	assert.False(t, previews.asked[0].Persist)
}
