package agentruntime

import (
	"context"
	"errors"
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

type failingPreviews struct {
	serviceports.ProposalPreviewService
}

func (failingPreviews) Baseline(
	context.Context,
	*serviceports.ProposalBaselineRequest,
) *serviceports.ProposalBaselineResult {
	return &serviceports.ProposalBaselineResult{
		PreviewErr: errors.New("cannot execute UPDATE in a read-only transaction"),
	}
}

type countingPreviewTool struct {
	*targetedStubTool
	previews int
}

func (t *countingPreviewTool) Preview(
	context.Context,
	serviceports.ToolExecuteParams,
) (*agent.ToolPreview, error) {
	t.previews++

	return &agent.ToolPreview{Summary: "previewed where its write would land"}, nil
}

// A preview the snapshot refused is reported as failed. Asking the tool again
// outside the snapshot would give a write the snapshot stopped a second
// chance to land.
func TestRun_ASimulatedWriteNeverPreviewsAgainOutsideTheSnapshot(t *testing.T) {
	t.Parallel()

	tool := &countingPreviewTool{
		targetedStubTool: &targetedStubTool{
			actionTool("place_shipment_hold", agent.TierAutoExecute, nil),
		},
	}
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("place_shipment_hold", map[string]any{"shipmentId": pulid.MustNew("shp_").String()}),
		textTurn("held"),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{
		Tools: []serviceports.AgentTool{tool},
	}, nil)
	rt.previews = failingPreviews{}
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
	assert.Zero(t, tool.previews, "the tool was not asked again outside the snapshot")
	assert.Zero(t, tool.Calls, "nothing ran")
	assert.False(t, action.Simulation.Previewed)
	assert.Contains(t, action.Simulation.Summary, "read-only transaction")
}

// refusingPreviews answers every baseline with a preview, and the one for
// refuseTool with a preview that says the write would be refused.
type refusingPreviews struct {
	serviceports.ProposalPreviewService

	mu         sync.Mutex
	refuseTool string
	asked      []*serviceports.ProposalBaselineRequest
}

func (p *refusingPreviews) Baseline(
	_ context.Context,
	req *serviceports.ProposalBaselineRequest,
) *serviceports.ProposalBaselineResult {
	p.mu.Lock()
	p.asked = append(p.asked, req)
	p.mu.Unlock()

	shipmentID, _ := req.Params.Params["shipmentId"].(string)
	result := &serviceports.ProposalBaselineResult{
		Target: &serviceports.ProposalTarget{
			Resource: permission.ResourceShipment,
			ID:       pulid.ID(shipmentID),
			Version:  7,
		},
		Preview: &agent.ToolPreview{Summary: "Would change the shipment."},
	}
	if req.Tool.Name() == p.refuseTool {
		result.Preview.Warnings = []agent.PreviewWarning{{
			Code:    agent.PreviewWarningWouldFail,
			Message: "This would be refused as it stands: validation failed:\n- BOL is already in use",
			Reasons: []agent.PreviewReason{{
				Field:   "bol",
				Label:   "BOL",
				Message: "BOL is already in use by shipment(s) with Pro Number(s): SEED-DET-009",
				Param:   "shipment.bol",
			}, {
				Message: "The customer is on credit hold",
			}},
		}}
	}

	return result
}

// A write whose baseline says it would be refused is not filed: a person
// would only ever be able to reject it. The model is told each reason and
// which parameter it is about, and asked to get what is missing.
func TestRun_DoesNotFileAProposalWhoseBaselineWouldBeRefused(t *testing.T) {
	t.Parallel()

	tool := &targetedStubTool{actionTool("create_shipment", agent.TierActWithApproval, nil)}
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("create_shipment", map[string]any{"shipmentId": pulid.MustNew("shp_").String()}),
		textTurn("Which BOL should I use?"),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{
		Tools: []serviceports.AgentTool{tool},
	}, nil)
	previews := &refusingPreviews{refuseTool: "create_shipment"}
	rt.previews = previews
	definition := testDefinition("create_shipment")
	definition.AutonomyCeiling = agent.TierActWithApproval

	result, err := rt.Run(t.Context(), &serviceports.RunRequest{
		Definition: definition,
		Actor:      testActor(),
		Input:      "copy SEED-DET-009",
	})
	require.NoError(t, err)

	assert.Empty(t, result.Actions, "nothing is filed for a write that would be refused")
	assert.Zero(t, tool.Calls)
	require.Len(t, previews.asked, 1)
	assert.False(t, previews.asked[0].FileRefused, "and no baseline is kept for it")

	told := result.Messages[2]
	assert.True(t, told.ToolFailed)
	assert.Contains(t, told.Content, `"create_shipment" was not proposed`)
	assert.Contains(t, told.Content,
		"BOL: BOL is already in use by shipment(s) with Pro Number(s): SEED-DET-009")
	assert.Contains(t, told.Content, "shipment.bol")
	assert.Contains(t, told.Content, "The customer is on credit hold")
	assert.Contains(t, told.Content, "Ask the person for")
	assert.Contains(t, told.Content, "propose again")
	assert.NotContains(t, told.Content, "This would be refused as it stands",
		"the model gets the reasons, not the sentence the card shows")
}

// A later write on a record an earlier step of the same turn changes may be
// refused only because that step has not run yet, so it is filed as a step
// of the plan and its baseline kept; the plan's preview says it depends on
// the earlier step.
func TestRun_FilesAWriteAnEarlierStepMayMakeValid(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	hold := &targetedStubTool{actionTool("place_shipment_hold", agent.TierPropose, nil)}
	cancel := &targetedStubTool{actionTool("cancel_shipment", agent.TierPropose, nil)}
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("place_shipment_hold", map[string]any{"shipmentId": shipmentID.String()}),
		toolTurn("cancel_shipment", map[string]any{"shipmentId": shipmentID.String()}),
		textTurn("Both are waiting for you."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{
		Tools: []serviceports.AgentTool{hold, cancel},
	}, nil)
	previews := &refusingPreviews{refuseTool: "cancel_shipment"}
	rt.previews = previews

	result, err := rt.Run(t.Context(), &serviceports.RunRequest{
		Definition: testDefinition("place_shipment_hold", "cancel_shipment"),
		Actor:      testActor(),
		Input:      "hold then cancel",
	})
	require.NoError(t, err)

	require.Len(t, result.Actions, 2, "the later step is filed with the earlier one")
	assert.Equal(t, "cancel_shipment", result.Actions[1].ToolName)
	require.Len(t, previews.asked, 2)
	assert.False(t, previews.asked[0].FileRefused)
	assert.True(t, previews.asked[1].FileRefused, "its baseline is kept, as any filed step's is")
	assert.False(t, result.Messages[4].ToolFailed)
	assert.Contains(t, result.Messages[4].Content, "Recorded a proposal")
}
