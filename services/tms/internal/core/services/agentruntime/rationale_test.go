package agentruntime

import (
	"strings"
	"testing"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithRationale_OffersItOnEveryToolWithoutChangingTheToolset(t *testing.T) {
	t.Parallel()

	specs := []serviceports.ToolSpec{{
		Name: "list_shipments",
		Parameters: map[string]any{
			"type":       "object",
			"properties": map[string]any{"status": map[string]any{"type": "string"}},
		},
	}, {Name: "get_time"}}
	offered := withRationale(specs)

	for _, spec := range offered {
		properties := spec.Parameters["properties"].(map[string]any)
		assert.Contains(t, properties, StepRationaleParam, spec.Name)
	}
	assert.NotContains(t, specs[0].Parameters["properties"], StepRationaleParam,
		"the toolset's own spec is left as the tool declared it")
}

func TestLiftRationales_TakesTheReasonOffTheArguments(t *testing.T) {
	t.Parallel()

	original := map[string]any{
		"status": "Completed",
		StepRationaleParam: map[string]any{
			"saw":       "  8 items   have no biller ",
			"because":   "billing can't post without one",
			"insteadOf": strings.Repeat("x", 300),
		},
	}
	calls := []serviceports.ToolCall{
		{Name: "list_billing_queue_items", Arguments: original},
		{Name: "get_time", Arguments: map[string]any{}},
	}
	liftRationales(calls)

	assert.Equal(t, map[string]any{"status": "Completed"}, calls[0].Arguments)
	assert.Contains(t, original, StepRationaleParam, "the model's own map is not edited")
	require.NotNil(t, calls[0].Why)
	assert.Equal(t, "8 items have no biller", calls[0].Why.Saw)
	assert.Equal(t, "billing can't post without one", calls[0].Why.Because)
	assert.Len(t, []rune(calls[0].Why.InsteadOf), maxRationaleRunes)
	assert.Nil(t, calls[1].Why)
}

func TestLiftRationales_DropsAnEmptyOrMalformedReason(t *testing.T) {
	t.Parallel()

	calls := []serviceports.ToolCall{
		{Name: "a", Arguments: map[string]any{StepRationaleParam: "not an object"}},
		{Name: "b", Arguments: map[string]any{StepRationaleParam: map[string]any{"saw": " "}}},
	}
	liftRationales(calls)

	for _, call := range calls {
		assert.Nil(t, call.Why, call.Name)
		assert.NotContains(t, call.Arguments, StepRationaleParam, call.Name)
	}
}
