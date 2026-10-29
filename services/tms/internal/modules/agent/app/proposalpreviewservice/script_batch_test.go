package proposalpreviewservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scriptBatch is a script's batch of two intents, each on its own shipment
// through its own tool, as the recorder files it.
type scriptBatch struct {
	f     *fixture
	other *previewingTool
	plan  *agentplan.AgentPlan
}

func newScriptBatch() *scriptBatch {
	other := &previewingTool{
		baseTool: baseTool{name: "hold_shipment", resource: permission.ResourceShipment},
		state: &shipmentState{
			ID:         pulid.MustNew("shp_"),
			ProNumber:  "PRO-200",
			Status:     "New",
			RateAmount: decimal.RequireFromString("100"),
			Version:    5,
		},
	}
	f := newFixture(other)
	f.versions.versions[other.state.ID] = 5
	plan := &agentplan.AgentPlan{
		ID:             pulid.MustNew("apl_"),
		OrganizationID: f.org,
		BusinessUnitID: f.bu,
		ScriptStepKey:  "script-step",
	}

	first := f.proposal(f.cancelParams())
	first.PlanID, first.PlanStep, first.ScriptStepKey = &plan.ID, 1, plan.ScriptStepKey
	second := f.proposal(map[string]any{"shipmentId": other.state.ID.String(), "status": "Hold"})
	second.ToolName, second.TargetID, second.TargetVersion = other.name, other.state.ID, 5
	second.PlanID, second.PlanStep, second.ScriptStepKey = &plan.ID, 2, plan.ScriptStepKey
	f.steps.steps = []*agent.AgentProposal{first, second}

	return &scriptBatch{f: f, other: other, plan: plan}
}

func (b *scriptBatch) preview(t *testing.T) *agentpreview.PlanPreview {
	t.Helper()

	preview, err := b.f.svc.ForPlan(t.Context(), &agentruntimeports.PlanPreviewRequest{
		Plan:   b.plan,
		Viewer: b.f.viewer(),
	})
	require.NoError(t, err)
	require.Len(t, preview.Steps, 2)

	return preview
}

func TestForPlan_AnUnchangedScriptBatchPreviewsTheSameEachTime(t *testing.T) {
	t.Parallel()

	batch := newScriptBatch()
	filed := batch.preview(t)
	decided := batch.preview(t)

	assert.False(t, decided.Stale)
	assert.Equal(t, filed.Digest, decided.Digest, "an approval of what was shown matches")
	assert.Equal(t, filed.StepDigests(), decided.StepDigests())
	want, err := agentpreview.PlanPreviewDigest(batch.plan.ID, decided.StepDigests())
	require.NoError(t, err)
	assert.Equal(t, want, decided.Digest, "the batch's digest covers every step's")
	for _, step := range decided.Steps {
		assert.Zero(
			t,
			changeOf(t, step.Preview).DependsOnStep,
			"each intent stands on its own record",
		)
	}
}

func TestForPlan_AScriptBatchIsStaleWhenOneStepsRecordMoved(t *testing.T) {
	t.Parallel()

	batch := newScriptBatch()
	shown := batch.preview(t)
	batch.f.versions.versions[batch.other.state.ID] = 6

	decided := batch.preview(t)

	assert.True(t, decided.Stale, "one moved step makes the whole batch stale")
	assert.False(t, decided.Steps[0].Preview.IsStale())
	assert.True(t, decided.Steps[1].Preview.IsStale())
	assert.True(t, decided.Steps[1].Preview.HasWarning(agentpreview.PreviewWarningTargetChanged))
	assert.NotEqual(t, shown.Digest, decided.Digest)
}

func TestForPlan_AScriptBatchIsStaleWhenOneStepsRecordWasDeleted(t *testing.T) {
	t.Parallel()

	batch := newScriptBatch()
	batch.f.versions.missing[batch.other.state.ID] = true

	decided := batch.preview(t)

	assert.True(t, decided.Stale)
	assert.True(t, decided.Steps[1].Preview.Staleness.Missing)
	assert.True(t, decided.Steps[1].Preview.HasWarning(agentpreview.PreviewWarningRecordMissing))
}

func TestForPlan_AScriptBatchWhosePreviewChangedHasANewDigest(t *testing.T) {
	t.Parallel()

	batch := newScriptBatch()
	shown := batch.preview(t)
	batch.other.state.Status = "Delayed"

	decided := batch.preview(t)

	assert.False(t, decided.Stale, "the version did not move")
	assert.Equal(t, shown.Steps[0].Preview.Digest, decided.Steps[0].Preview.Digest)
	assert.NotEqual(t, shown.Steps[1].Preview.Digest, decided.Steps[1].Preview.Digest)
	assert.NotEqual(t, shown.Digest, decided.Digest, "an approval of the old preview is a conflict")
}
