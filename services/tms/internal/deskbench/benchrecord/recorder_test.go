package benchrecord

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aireasoning"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecorderPin_HoldsARunToItsCell(t *testing.T) {
	t.Parallel()

	recorder := NewRecorder(nil)
	definitionID, providerID := pulid.MustNew("agd_"), pulid.MustNew("aiprv_")
	release := recorder.Pin(definitionID, ProviderPin{
		ProviderID: providerID,
		Effort:     aireasoning.ReasoningLow,
	})

	req := &llmports.ChatCompletionRequest{
		Attribution: aiusageports.AIUsageAttribution{AgentDefinitionID: definitionID},
	}
	recorder.pin(req)
	assert.Equal(t, providerID, req.PreferredProviderID)
	assert.True(t, req.PinPreferred)
	assert.Equal(t, aireasoning.ReasoningLow, req.Reasoning)
	assert.Nil(t, req.Advisor)

	release()
	other := &llmports.ChatCompletionRequest{
		Attribution: aiusageports.AIUsageAttribution{AgentDefinitionID: definitionID},
	}
	recorder.pin(other)
	assert.False(t, other.PinPreferred)
}

func TestRecorderPin_HandsTheCellsAdvisorToEveryCall(t *testing.T) {
	t.Parallel()

	recorder := NewRecorder(nil)
	threadID := pulid.MustNew("athr_")
	release := recorder.Pin(threadID, ProviderPin{
		Effort:  aireasoning.ReasoningLow,
		Advisor: llmports.DefaultAdvisor(""),
	})
	defer release()

	first := &llmports.ChatCompletionRequest{
		Attribution: aiusageports.AIUsageAttribution{ThreadID: threadID},
	}
	recorder.pin(first)
	require.NotNil(t, first.Advisor)
	assert.Equal(t, llmports.AdvisorSpec{Model: "claude-opus-5-5", MaxUses: 2}, *first.Advisor)
	assert.Equal(t, aireasoning.ReasoningLow, first.Reasoning)

	first.Advisor.MaxUses = 9
	second := &llmports.ChatCompletionRequest{
		Attribution: aiusageports.AIUsageAttribution{ThreadID: threadID},
	}
	recorder.pin(second)
	assert.Equal(t, 2, second.Advisor.MaxUses, "each call gets its own copy of the pin's advisor")
}
