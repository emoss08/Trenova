package agentguard_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentguard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	trailerGoal = "List the trailers whose inspection is due this month."
	trailerNeed = "find trailers by inspection due date; read a trailer's inspection history"
)

func TestEvaluate_AnAllowedRequestCarriesItsGoalAndNeed(t *testing.T) {
	t.Parallel()

	stub := &stubCompletion{
		category: string(agentguard.CategoryTransportationOperations),
		goal:     trailerGoal,
		need:     trailerNeed,
	}
	decision := newGuard(t, stub).Evaluate(t.Context(), agentguard.EvaluateRequest{
		TenantInfo: tenant(),
		Input:      "which ones are due?",
	})

	require.True(t, decision.Allowed)
	assert.Equal(t, trailerGoal, decision.Goal)
	assert.Equal(t, trailerNeed, decision.Need)

	schema := stub.lastReq.OutputSchema
	assert.ElementsMatch(t, []string{"category", "reasoning", "goal", "need"},
		schema["required"], "a strict schema names every field it returns")
	assert.Contains(t, stub.lastReq.System, "never a tool or a field name",
		"the need is written in the work's words, not the catalog's")
}

func TestEvaluate_ARefusedRequestCarriesNoGoal(t *testing.T) {
	t.Parallel()

	stub := &stubCompletion{
		category: string(agentguard.CategoryGeneralKnowledge),
		goal:     "Tell a joke.",
		need:     "jokes",
	}
	decision := newGuard(t, stub).Evaluate(t.Context(), agentguard.EvaluateRequest{
		TenantInfo: tenant(),
		Input:      "Tell me a joke",
	})

	require.False(t, decision.Allowed)
	assert.Empty(t, decision.Goal)
	assert.Empty(t, decision.Need)
}

func TestEvaluate_TheSharedStoreKeepsNoParaphraseOfTheRequest(t *testing.T) {
	t.Parallel()

	shared := newFakeShared()
	tenantInfo := tenant()
	first := &stubCompletion{
		category: string(agentguard.CategoryTransportationOperations),
		goal:     trailerGoal,
		need:     trailerNeed,
	}
	local := guardSharing(t, first, shared).Evaluate(t.Context(), agentguard.EvaluateRequest{
		TenantInfo: tenantInfo,
		Input:      hazmat,
	})
	require.Equal(t, trailerNeed, local.Need)

	other := guardSharing(t, &stubCompletion{}, shared).Evaluate(t.Context(),
		agentguard.EvaluateRequest{TenantInfo: tenantInfo, Input: hazmat})

	require.True(t, other.Allowed, "the shared verdict still admits the request")
	assert.Empty(t, other.Goal,
		"the store every replica reads holds no text derived from what the person asked")
	assert.Empty(t, other.Need)
}
