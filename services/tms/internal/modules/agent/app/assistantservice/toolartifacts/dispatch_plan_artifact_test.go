package toolartifacts

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A dispatch plan is shown as the plan a dispatcher acts on, with the
// assignments ready to make, rather than written out again as prose.
func TestArtifactFromObservation_ADispatchPlanBecomesItsCard(t *testing.T) {
	t.Parallel()

	artifact := ArtifactFromObservation(assistanttest.Observation("dispatch_plan", map[string]any{
		"planningMode": "Immediate",
		"cost":         "$867",
		"lines": []any{map[string]any{
			"shipmentMoveId": "smv_1", "proNumber": "S-1", "workerId": "wrk_1",
			"workerName": "Adam Stephens", "cost": "$412",
		}},
		"uncovered": []any{map[string]any{
			"shipmentMoveId": "smv_2", "proNumber": "S-2", "reason": "No driver could take this move.",
		}},
		"assignments": []any{map[string]any{
			"shipmentMoveId": "smv_1", "primaryWorkerId": "wrk_1", "tractorId": "trc_1",
		}},
		"notPlanned": []any{"Shipment shp_9 has no open move picking up in the next 168 hours"},
	}))

	require.NotNil(t, artifact)
	assert.Equal(t, assistantartifact.KindDispatchPlan, artifact.Kind)
	assert.Equal(t, "Drivers for 2 loads", artifact.Title)
	assert.Equal(t, "dispatch_plan", artifact.Payload["tool"])
	assert.Equal(t, "$867", artifact.Payload["cost"])
	assert.Len(t, artifact.Payload["lines"], 1)
	assert.Len(t, artifact.Payload["uncovered"], 1)
	assert.Len(t, artifact.Payload["assignments"], 1)
	assert.InDelta(t, 1, artifact.Payload["notPlannedCount"], 0)
	assert.NotContains(t, artifact.Payload, "notPlanned", "the model's notes name ids; the card counts them")
}

func TestArtifactFromObservation_AnEmptyDispatchPlanGetsNoCard(t *testing.T) {
	t.Parallel()

	assert.Nil(t, ArtifactFromObservation(assistanttest.Observation("dispatch_plan", map[string]any{
		"lines": []any{}, "uncovered": []any{}, "note": "nothing to plan",
	})))
}
