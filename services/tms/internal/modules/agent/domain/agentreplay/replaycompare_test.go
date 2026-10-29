package agentreplay

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func original(
	tool string,
	params map[string]any,
	status agent.ProposalStatus,
	decision agentdecision.DecisionType,
) OriginalProposal {
	return OriginalProposal{
		ID:       pulid.MustNew("aprop_"),
		ToolName: tool,
		Params:   params,
		Status:   status,
		Decision: decision,
	}
}

// The decisions people made on the original are the yardstick. A change a
// person approved that the replay drops is a regression; one a person
// rejected that the replay drops is what the change was for.
func TestCompareReplay_JudgesByHowAPersonDecided(t *testing.T) {
	t.Parallel()

	originals := []OriginalProposal{
		original(
			"dispatch_assign_move",
			map[string]any{"shipmentMoveId": "smv_1", "primaryWorkerId": "wrk_a"},
			agent.ProposalStatusExecuted,
			agentdecision.DecisionAccepted,
		),
		original(
			"shipment_cancel",
			map[string]any{"shipmentId": "shp_1"},
			agent.ProposalStatusRejected,
			agentdecision.DecisionRejected,
		),
		original(
			"shipment_place_hold",
			map[string]any{"shipmentId": "shp_2"},
			agent.ProposalStatusExecuted,
			agentdecision.DecisionAccepted,
		),
		original(
			"shipment_add_comment",
			map[string]any{"shipmentId": "shp_3"},
			agent.ProposalStatusExpired,
			"",
		),
	}
	replay := []ReplayAction{
		{
			ToolName:  "dispatch_assign_move",
			Arguments: map[string]any{"primaryWorkerId": "wrk_a", "shipmentMoveId": "smv_1"},
		},
		{ToolName: "worker_send_message", Arguments: map[string]any{"workerId": "wrk_a"}},
	}

	comparison := CompareReplay(originals, replay)

	verdicts := make(map[string]ReplayVerdict, len(comparison.Matches))
	for _, match := range comparison.Matches {
		verdicts[match.ToolName] = match.Verdict
	}
	assert.Equal(t, VerdictAgreed, verdicts["dispatch_assign_move"])
	assert.Equal(t, VerdictImproved, verdicts["shipment_cancel"])
	assert.Equal(t, VerdictRegressed, verdicts["shipment_place_hold"])
	assert.Equal(t, VerdictUndecided, verdicts["shipment_add_comment"])
	assert.Equal(t, VerdictAdded, verdicts["worker_send_message"])

	assert.Equal(t, 1, comparison.Agreed)
	assert.Equal(t, 1, comparison.Improved)
	assert.Equal(t, 1, comparison.Regressed)
	assert.Equal(t, 1, comparison.Undecided)
	assert.Equal(t, 1, comparison.Added)
	assert.Equal(t, 3, comparison.Decided)
	require.NotNil(t, comparison.Score)
	assert.InDelta(t, 2.0/3.0, *comparison.Score, 0.001)
}

// The same tool with different parameters is neither agreement nor a drop:
// the person is shown what moved.
func TestCompareReplay_ShowsWhatChangedOnTheSameTool(t *testing.T) {
	t.Parallel()

	originals := []OriginalProposal{
		original(
			"dispatch_assign_move",
			map[string]any{
				"shipmentMoveId":  "smv_1",
				"primaryWorkerId": "wrk_a",
				"tractorId":       "trc_1",
			},
			agent.ProposalStatusExecuted,
			agentdecision.DecisionAccepted,
		),
	}
	replay := []ReplayAction{
		{
			ToolName: "dispatch_assign_move",
			Arguments: map[string]any{
				"shipmentMoveId":  "smv_1",
				"primaryWorkerId": "wrk_b",
				"tractorId":       "trc_1",
			},
		},
	}

	comparison := CompareReplay(originals, replay)

	require.Len(t, comparison.Matches, 1)
	assert.Equal(t, VerdictChanged, comparison.Matches[0].Verdict)
	assert.Equal(
		t,
		[]agent.FieldChange{{Field: "primaryWorkerId", From: "wrk_a", To: "wrk_b"}},
		comparison.Matches[0].Changes,
	)
	assert.Equal(t, 1, comparison.Changed)
	assert.Nil(t, comparison.Score, "a changed proposal is not scored either way")
}

// Parameters stored as JSON come back as floats; a model writes integers.
// The two are the same call.
func TestCompareReplay_ReadsNumbersAndOrderAsTheSame(t *testing.T) {
	t.Parallel()

	originals := []OriginalProposal{
		original(
			"equipment_set_tractor_status",
			map[string]any{"tractorIds": []any{"trc_1"}, "status": "Available", "note": ""},
			agent.ProposalStatusExecuted,
			agentdecision.DecisionAccepted,
		),
		original(
			"shipment_request_documents",
			map[string]any{"count": float64(2)},
			agent.ProposalStatusRejected,
			agentdecision.DecisionRejected,
		),
	}
	replay := []ReplayAction{
		{ToolName: "shipment_request_documents", Arguments: map[string]any{"count": 2}},
		{
			ToolName:  "equipment_set_tractor_status",
			Arguments: map[string]any{"status": "Available", "tractorIds": []string{"trc_1"}},
		},
	}

	comparison := CompareReplay(originals, replay)

	assert.Equal(t, VerdictAgreed, comparison.Matches[0].Verdict)
	assert.Equal(t, VerdictRepeated, comparison.Matches[1].Verdict)
	assert.Equal(t, 1, comparison.Repeated)
}

func TestCompareReplay_WithNothingOnEitherSide(t *testing.T) {
	t.Parallel()

	comparison := CompareReplay(nil, nil)

	assert.Empty(t, comparison.Matches)
	assert.Nil(t, comparison.Score)
}

func modifiedOriginal() OriginalProposal {
	proposal := &agent.AgentProposal{
		ID:         pulid.MustNew("aprop_"),
		ToolName:   "update_rate",
		ToolParams: map[string]any{"shipmentId": "shp_1", "rate": float64(1200)},
		Status:     agent.ProposalStatusExecuted,
	}

	return NewOriginalProposal(proposal, []*agentdecision.AgentDecision{{
		Decision:      agentdecision.DecisionModified,
		Modifications: map[string]any{"rate": float64(1350)},
	}})
}

func TestNewOriginalProposal_MergesTheLatestModification(t *testing.T) {
	t.Parallel()

	original := modifiedOriginal()

	assert.Equal(t, agentdecision.DecisionModified, original.Decision)
	assert.Equal(t, map[string]any{"shipmentId": "shp_1", "rate": float64(1200)}, original.Params)
	assert.Equal(
		t,
		map[string]any{"shipmentId": "shp_1", "rate": float64(1350)},
		original.CorrectedParams,
	)
	assert.Equal(t, original.CorrectedParams, original.Expected())
}

func TestNewOriginalProposal_RejectionCorrectsNothing(t *testing.T) {
	t.Parallel()

	proposal := &agent.AgentProposal{
		ID:         pulid.MustNew("aprop_"),
		ToolName:   "update_rate",
		ToolParams: map[string]any{"rate": float64(1200)},
		Status:     agent.ProposalStatusRejected,
	}

	rejected := NewOriginalProposal(proposal, []*agentdecision.AgentDecision{
		{Decision: agentdecision.DecisionRejected},
		{
			Decision:      agentdecision.DecisionModified,
			Modifications: map[string]any{"rate": float64(1)},
		},
	})
	undecided := NewOriginalProposal(proposal, nil)

	assert.Nil(t, rejected.CorrectedParams)
	assert.Equal(t, proposal.ToolParams, rejected.Expected())
	assert.Nil(t, undecided.CorrectedParams)
	assert.Empty(t, undecided.Decision)
}

func TestNewOriginalProposal_KeepsTheLatestModificationUnderALaterApproval(t *testing.T) {
	t.Parallel()

	proposal := &agent.AgentProposal{
		ID:         pulid.MustNew("aprop_"),
		ToolName:   "update_rate",
		ToolParams: map[string]any{"shipmentId": "shp_1", "rate": float64(1200)},
		Status:     agent.ProposalStatusExecuted,
	}

	original := NewOriginalProposal(proposal, []*agentdecision.AgentDecision{
		{Decision: agentdecision.DecisionAccepted},
		{
			Decision:      agentdecision.DecisionModified,
			Modifications: map[string]any{"rate": float64(1400)},
		},
		{
			Decision:      agentdecision.DecisionModified,
			Modifications: map[string]any{"rate": float64(1300)},
		},
	})

	assert.Equal(t, agentdecision.DecisionAccepted, original.Decision)
	assert.Equal(
		t,
		map[string]any{"shipmentId": "shp_1", "rate": float64(1400)},
		original.CorrectedParams,
		"the newest modification is what the person settled on",
	)
	assert.Equal(t, float64(1200), proposal.ToolParams["rate"], "the proposal is not mutated")
}

func TestDecisionsByProposal_KeepsEachProposalsOrder(t *testing.T) {
	t.Parallel()

	first, second := pulid.MustNew("aprop_"), pulid.MustNew("aprop_")
	newest := &agentdecision.AgentDecision{
		ProposalID: &first,
		Decision:   agentdecision.DecisionAccepted,
	}
	older := &agentdecision.AgentDecision{
		ProposalID: &first,
		Decision:   agentdecision.DecisionModified,
	}
	other := &agentdecision.AgentDecision{
		ProposalID: &second,
		Decision:   agentdecision.DecisionRejected,
	}

	grouped := DecisionsByProposal([]*agentdecision.AgentDecision{newest, other, older, {}, nil})

	assert.Equal(t, []*agentdecision.AgentDecision{newest, older}, grouped[first])
	assert.Equal(t, []*agentdecision.AgentDecision{other}, grouped[second])
	assert.Len(t, grouped, 2)
}

func TestCompareReplay_ProposingTheCorrectedVersionAgrees(t *testing.T) {
	t.Parallel()

	comparison := CompareReplay([]OriginalProposal{modifiedOriginal()}, []ReplayAction{{
		ToolName:  "update_rate",
		Arguments: map[string]any{"shipmentId": "shp_1", "rate": 1350},
	}})

	require.Len(t, comparison.Matches, 1)
	match := comparison.Matches[0]
	assert.Equal(t, VerdictAgreed, match.Verdict)
	assert.Equal(t, float64(1350), match.CorrectedParams["rate"])
	assert.Equal(t, float64(1200), match.OriginalParams["rate"])
	assert.Empty(t, match.Changes)
	assert.Equal(t, 1, comparison.Agreed)
}

func TestCompareReplay_RepeatingTheFixedParametersIsAChange(t *testing.T) {
	t.Parallel()

	comparison := CompareReplay([]OriginalProposal{modifiedOriginal()}, []ReplayAction{{
		ToolName:  "update_rate",
		Arguments: map[string]any{"shipmentId": "shp_1", "rate": 1200},
	}})

	require.Len(t, comparison.Matches, 1)
	match := comparison.Matches[0]
	assert.Equal(t, VerdictChanged, match.Verdict)
	require.Len(t, match.Changes, 1)
	assert.Equal(t, agent.FieldChange{Field: "rate", From: "1350", To: "1200"}, match.Changes[0])
	assert.Zero(t, comparison.Agreed)
}
