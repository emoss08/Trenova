package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The shared part of the system prompt has to reach the provider on every call
// of a turn, including a turn workflow code rebuilt from its state, or the
// cache mark lands on the whole prompt and misses on every new page or memory.
func TestTurn_CarriesTheSharedPartOfItsPromptThroughItsState(t *testing.T) {
	t.Parallel()

	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	req := &serviceports.RunRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Input:      "Where is S1?",
		Context: agentdefinition.RuntimeContext{
			Page: &agentdefinition.PageContext{Path: "/shipments/shp_1", EntityType: "shipment", EntityID: "shp_1"},
		},
	}

	turn := rt.OpenTurn(t.Context(), req)
	state := turn.State()

	require.Positive(t, state.SystemStable)
	require.Less(t, state.SystemStable, len(state.System))
	assert.Equal(t, state.SystemStable, turn.completionRequest().SystemStable)

	restored := rt.RestoreTurn(req, state)
	assert.Equal(t, state.SystemStable, restored.completionRequest().SystemStable)
}
