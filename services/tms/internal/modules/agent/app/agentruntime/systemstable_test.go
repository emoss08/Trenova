package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The shared part of the system prompt has to reach the provider on every call
// of a turn, including a turn workflow code rebuilt from its state, or the
// cache mark lands on the whole prompt and misses on every new page or memory.
func TestTurn_CarriesTheSharedPartOfItsPromptThroughItsState(t *testing.T) {
	t.Parallel()

	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	req := &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition(),
		Actor:      runtimetest.Actor(),
		Input:      "Where is S1?",
		Context: agentturn.RuntimeContext{
			Page: &agentturn.PageContext{
				Path:       "/shipments/shp_1",
				EntityType: "shipment",
				EntityID:   "shp_1",
			},
		},
	}

	turn := rt.OpenTurn(t.Context(), req)
	state := turn.State()

	require.Positive(t, state.SystemStable)
	require.Less(t, state.SystemStable, len(state.System))
	assert.Equal(t, state.SystemStable, turn.completionRequest().SystemStable)

	restored := rt.RestoreTurn(req, &state)
	assert.Equal(t, state.SystemStable, restored.completionRequest().SystemStable)
}
