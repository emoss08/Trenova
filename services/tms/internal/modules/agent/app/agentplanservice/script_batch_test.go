package agentplanservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const scriptStep = "script-step"

// newScriptHarness is a batch a script prepared: internal writes that the
// same agent could have run on its own, every step an intent of one script.
func newScriptHarness(t *testing.T, steps int) *harness {
	t.Helper()

	h := newHarness(t, steps, false)
	h.plans.plan.ScriptStepKey = scriptStep
	for _, step := range h.steps.steps {
		step.ScriptStepKey = scriptStep
	}

	return h
}

func TestDecide_AScriptsBatchApprovedUnseenIsRefusedWhateverItsClass(t *testing.T) {
	t.Parallel()

	h := newScriptHarness(t, 3)
	h.svc.previews = &fakePreviews{}

	_, err := h.svc.Decide(t.Context(), h.req, h.actor)

	require.Error(t, err)
	assert.True(t, agent.IsReviewRequired(err))
	assert.Empty(t, h.decider.decided, "no step is decided")
	assert.Empty(t, h.plans.statuses, "the batch is not claimed")
}

func TestDecide_AnUnchangedScriptBatchCommitsEveryStepOnItsDigest(t *testing.T) {
	t.Parallel()

	h := newScriptHarness(t, 3)
	h.svc.previews = &fakePreviews{}
	h.req.PreviewDigest = "plan-digest"

	plan, err := h.svc.Decide(t.Context(), h.req, h.actor)
	require.NoError(t, err)
	assert.Equal(t, agentplan.PlanStatusCompleted, plan.Status)

	require.Len(t, h.decider.decided, 3)
	for idx, decided := range h.decider.decided {
		assert.Equal(
			t,
			h.steps.steps[idx].ID,
			decided.ProposalID,
			"steps run in the order prepared",
		)
		require.NotNil(t, decided.StepPreview)
		assert.Equal(t, "step-"+h.steps.steps[idx].ID.String(), decided.StepPreview.Digest)
		assert.True(t, decided.StepPreviewReviewed)
	}
}

func TestDecide_AScriptBatchWithAStepThatMovedIsRefusedWhole(t *testing.T) {
	t.Parallel()

	h := newScriptHarness(t, 3)
	h.svc.previews = &fakePreviews{stale: true, staleStep: 2}
	followUps := &recordedFollowUps{}
	h.svc.followUps = followUps
	h.req.PreviewDigest = "plan-digest"

	plan, err := h.svc.Decide(t.Context(), h.req, h.actor)
	require.NoError(t, err)

	assert.Equal(t, agentplan.PlanStatusFailed, plan.Status)
	require.Len(t, h.decider.decided, 1, "no step commits on stale authority")
	assert.Equal(t, h.steps.steps[1].ID, h.decider.decided[0].ProposalID,
		"only the moved step is recorded, as refused")
	assert.Equal(t, 1, h.steps.skipped, "the batch's other steps are skipped")
	require.Len(t, followUps.requests, 1)
}

func TestDecide_AScriptBatchApprovedOnAnOlderPreviewIsAConflict(t *testing.T) {
	t.Parallel()

	h := newScriptHarness(t, 3)
	h.svc.previews = &fakePreviews{}
	h.req.PreviewDigest = "the-digest-before-a-step-changed"

	_, err := h.svc.Decide(t.Context(), h.req, h.actor)

	var conflict *errortypes.ConflictError
	require.True(t, errors.As(err, &conflict), "%v", err)
	assert.Empty(t, h.decider.decided)
	assert.Empty(t, h.plans.statuses)
}

func TestDecide_AScriptBatchDecidedTwiceCommitsOnce(t *testing.T) {
	t.Parallel()

	h := newScriptHarness(t, 3)
	h.svc.previews = &fakePreviews{}
	h.req.PreviewDigest = "plan-digest"

	_, err := h.svc.Decide(t.Context(), h.req, h.actor)
	require.NoError(t, err)
	for _, step := range h.steps.steps {
		step.Status = agent.ProposalStatusAccepted
	}

	_, err = h.svc.Decide(t.Context(), h.req, h.actor)
	require.Error(t, err, "a batch already decided is not decided again")
	assert.Len(t, h.decider.decided, 3, "every step ran once")
}

func TestDecide_AScriptBatchCanAlwaysBeRejectedUnseen(t *testing.T) {
	t.Parallel()

	h := newScriptHarness(t, 2)
	h.svc.previews = &fakePreviews{stale: true}
	h.req.Decision = agentdecision.DecisionRejected

	plan, err := h.svc.Decide(t.Context(), h.req, h.actor)
	require.NoError(t, err)
	assert.Equal(t, agentplan.PlanStatusRejected, plan.Status)
}

type recordedAudit struct {
	entries []*auditports.LogActionParams
}

func (a *recordedAudit) LogAction(
	params *auditports.LogActionParams,
	_ ...auditports.LogOption,
) error {
	a.entries = append(a.entries, params)

	return nil
}

func TestDecide_AScriptBatchsDecisionIsAuditedAsTheScriptsBatch(t *testing.T) {
	t.Parallel()

	h := newScriptHarness(t, 2)
	h.svc.previews = &fakePreviews{}
	audit := &recordedAudit{}
	h.svc.audit = audit
	h.req.PreviewDigest = "plan-digest"

	_, err := h.svc.Decide(t.Context(), h.req, h.actor)
	require.NoError(t, err)

	require.Len(t, audit.entries, 1)
	entry := audit.entries[0]
	assert.True(t, entry.Critical)
	assert.Equal(t, permission.ResourceAgentProposal, entry.Resource)
	assert.Equal(t, h.plans.plan.ID.String(), entry.ResourceID)
	assert.Equal(t, h.actor.UserID, entry.UserID)
	assert.Equal(t, scriptStep, entry.CurrentState["scriptStepKey"], "the entry names the script")
}

func TestDecide_AScriptBatchThatReadOutsideContentNamesItInTheAudit(t *testing.T) {
	t.Parallel()

	h := newScriptHarness(t, 2)
	h.svc.previews = &fakePreviews{}
	audit := &recordedAudit{}
	h.svc.audit = audit
	h.req.PreviewDigest = "plan-digest"
	taint := &agent.RunTaint{}
	taint.Add(agent.TaintMark{
		Source:   agent.TaintSourceInboundMessage,
		ToolName: "inbox_get_inbound_message",
		CallID:   "script-step:3",
		Ref:      &agentrecord.RecordRef{EntityType: "inbound_message", ID: "imsg_1"},
	})
	for _, step := range h.steps.steps {
		step.Tainted, step.Taint = true, taint
	}

	_, err := h.svc.Decide(t.Context(), h.req, h.actor)
	require.NoError(t, err)

	require.Len(t, audit.entries, 1)
	recorded, ok := audit.entries[0].CurrentState["taint"].(map[string]any)
	require.True(t, ok, "the entry records what outside content the batch read")
	assert.NotEmpty(t, recorded)
	assert.Equal(t, true, audit.entries[0].CurrentState["tainted"])
}

func TestDecide_AnUntaintedBatchRecordsNoTaint(t *testing.T) {
	t.Parallel()

	h := newScriptHarness(t, 2)
	h.svc.previews = &fakePreviews{}
	audit := &recordedAudit{}
	h.svc.audit = audit
	h.req.PreviewDigest = "plan-digest"

	_, err := h.svc.Decide(t.Context(), h.req, h.actor)
	require.NoError(t, err)

	require.Len(t, audit.entries, 1)
	assert.NotContains(t, audit.entries[0].CurrentState, "taint")
	assert.NotContains(t, audit.entries[0].CurrentState, "tainted")
}
