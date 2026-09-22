package development

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func artifactRefs(t *testing.T) *artifactSeedRefs {
	t.Helper()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	shipments := make([]*shipment.Shipment, 0, 3)
	for i := range 3 {
		shipments = append(shipments, &shipment.Shipment{
			ID:        pulid.MustNew("shp_"),
			ProNumber: "S0000" + string(rune('1'+i)),
			Status:    shipment.StatusInTransit,
		})
	}

	return &artifactSeedRefs{
		org: orgStub(orgID, buID),
		threads: []*conversation.Thread{
			{ID: pulid.MustNew("thr_")},
			{ID: pulid.MustNew("thr_")},
		},
		shipments: shipments,
		plan: &agent.AgentPlan{
			ID:        pulid.MustNew("apln_"),
			Title:     "Ready S00002 for invoicing",
			Summary:   "Three things have to happen first.",
			StepCount: 3,
		},
		steps: []*agent.AgentProposal{
			{ID: pulid.MustNew("aprp_"), PlanStep: 1, ToolName: "correct_charge_code"},
			{ID: pulid.MustNew("aprp_"), PlanStep: 2, ToolName: "email_customer"},
		},
		draft: &agent.AgentProposal{
			ID:    pulid.MustNew("aprp_"),
			RunID: pulid.MustNew("arun_"),
		},
		now: 1_800_000_000,
	}
}

func TestAssistantArtifactSeed_EveryArtifactIsValid(t *testing.T) {
	seed := NewAssistantArtifactSeed()

	for _, artifact := range seed.artifacts(artifactRefs(t)) {
		multiErr := errortypes.NewMultiError()
		artifact.Validate(multiErr)
		assert.Falsef(t, multiErr.HasErrors(),
			"artifact %q is not valid: %v", artifact.Title, multiErr)
	}
}

/*
The pane reads each payload through a reader that refuses to guess: a value
that is not what the kind promises reads as absent, and the card renders
empty. An empty card is worse than no card — it teaches a developer that the
kind looks like nothing. So the keys each reader needs are asserted here,
where a rename on either side shows up as a failure rather than as a blank
pane somebody has to notice.
*/
func TestAssistantArtifactSeed_PayloadsCarryWhatTheirRendererReads(t *testing.T) {
	required := map[assistantartifact.Kind][]string{
		assistantartifact.KindReportPreview:   {"name", "columns", "rows"},
		assistantartifact.KindTableView:       {"tool", "entity", "columns", "rows"},
		assistantartifact.KindEmailDraft:      {"to", "subject", "body"},
		assistantartifact.KindPlan:            {"title", "stepCount", "steps"},
		assistantartifact.KindEntityCard:      {"entity", "record"},
		assistantartifact.KindRateExplanation: {"winner", "components", "totals"},
	}

	seeded := map[assistantartifact.Kind]bool{}
	for _, artifact := range NewAssistantArtifactSeed().artifacts(artifactRefs(t)) {
		seeded[artifact.Kind] = true
		keys, known := required[artifact.Kind]
		require.Truef(t, known, "artifact kind %q has no expected payload here", artifact.Kind)

		for _, key := range keys {
			value, ok := artifact.Payload[key]
			assert.Truef(t, ok, "%s payload has no %q", artifact.Kind, key)
			assert.NotNilf(t, value, "%s payload has a nil %q", artifact.Kind, key)
		}
	}

	for kind := range required {
		assert.Truef(t, seeded[kind], "no artifact was seeded for kind %q", kind)
	}
}

// A plan artifact is a view over a plan, not a second copy of it: its steps
// have to be the plan's own proposals, or approving from the pane and
// approving from the queue are two different decisions.
func TestAssistantArtifactSeed_PlanArtifactCitesTheRealSteps(t *testing.T) {
	refs := artifactRefs(t)

	var payload map[string]any
	for _, artifact := range NewAssistantArtifactSeed().artifacts(refs) {
		if artifact.Kind == assistantartifact.KindPlan {
			payload = artifact.Payload
		}
	}
	require.NotNil(t, payload)

	steps, ok := payload["steps"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, steps, len(refs.steps))

	for i, step := range steps {
		assert.Equal(t, refs.steps[i].ID.String(), step["proposalId"])
		assert.Equal(t, refs.steps[i].ToolName, step["toolName"])
	}
}
