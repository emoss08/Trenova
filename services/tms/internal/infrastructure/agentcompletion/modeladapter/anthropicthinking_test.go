package modeladapter

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sentThinking is the thinking and output_config an Anthropic request carried.
func sentThinking(
	t *testing.T,
	model string,
	effort aiprovider.ReasoningEffort,
) (map[string]any, map[string]any) {
	t.Helper()

	server, captured := streamServer(t, "text/event-stream", sse(
		[2]string{"message_start", `{"type":"message_start","message":{"model":"m","usage":{"input_tokens":1}}}`},
		[2]string{"message_stop", `{"type":"message_stop"}`},
	))
	call := reasoningCall(aiprovider.KindAnthropicMessages, server.URL, effort,
		&Request{Messages: UserMessage("hi")})
	call.Provider.Model = model
	_, _ = streamWith(t, NewAnthropicAdapter(), call)

	thinking, _ := (*captured)["thinking"].(map[string]any)
	output, _ := (*captured)["output_config"].(map[string]any)

	return thinking, output
}

/*
The current Claude models think by effort and refuse a token budget outright:
Low, Medium and High sent budget_tokens and every one of them was a 400 on
Opus 5.5, Sonnet 5.5 and Fable. They ask for adaptive thinking at that effort,
with the summary shown, which those models leave out unless asked.
*/
func TestAnthropicAdapter_AsksCurrentModelsToThinkByEffort(t *testing.T) {
	t.Parallel()

	levels := map[aiprovider.ReasoningEffort]string{
		aiprovider.ReasoningMinimal: "low",
		aiprovider.ReasoningLow:     "low",
		aiprovider.ReasoningMedium:  "medium",
		aiprovider.ReasoningHigh:    "high",
	}
	for _, model := range []string{"claude-opus-5-5", "claude-sonnet-5-5", "claude-fable-5-1", "claude-opus-4-8", "claude-sonnet-4-6"} {
		for level, effort := range levels {
			thinking, output := sentThinking(t, model, level)
			require.NotNil(t, thinking, "%s %s", model, level)
			assert.Equal(t, "adaptive", thinking["type"], "%s %s", model, level)
			assert.Equal(t, "summarized", thinking["display"], "%s %s", model, level)
			assert.NotContains(t, thinking, "budget_tokens", "%s %s", model, level)
			require.NotNil(t, output, "%s %s", model, level)
			assert.Equal(t, effort, output["effort"], "%s %s", model, level)
			assert.NotContains(t, output, "format")
		}
	}
}

// A model from before adaptive thinking keeps the budget it always took, and
// nothing about effort, which those models reject.
func TestAnthropicAdapter_KeepsTheBudgetForOlderModels(t *testing.T) {
	t.Parallel()

	for _, model := range []string{"claude-haiku-4-5", "claude-sonnet-4-5", "my-gateway-alias"} {
		thinking, output := sentThinking(t, model, aiprovider.ReasoningHigh)
		require.NotNil(t, thinking, model)
		assert.Equal(t, "enabled", thinking["type"], model)
		assert.InDelta(t, 16384, thinking["budget_tokens"], 0, model)
		assert.Nil(t, output, model)

		none, _ := sentThinking(t, model, aiprovider.ReasoningNone)
		assert.Nil(t, none, model)
	}
}

/*
None is the least thinking each model allows. Opus 5.5 and Fable always think
and Sonnet 5.5 refuses "disabled", so they think at low effort; Opus 5 takes
"disabled" at this effort; Opus 4.8 and earlier adaptive models do not think
unless asked, so they are sent nothing.
*/
func TestAnthropicAdapter_SendsTheLeastThinkingEachModelAllowsForNone(t *testing.T) {
	t.Parallel()

	for _, model := range []string{"claude-opus-5-5", "claude-sonnet-5-5", "claude-fable-5-1", "claude-fable-5"} {
		thinking, output := sentThinking(t, model, aiprovider.ReasoningNone)
		require.NotNil(t, thinking, model)
		assert.Equal(t, "adaptive", thinking["type"], model)
		assert.NotContains(t, thinking, "display", model)
		require.NotNil(t, output, model)
		assert.Equal(t, "low", output["effort"], model)
	}

	thinking, output := sentThinking(t, "claude-opus-5", aiprovider.ReasoningNone)
	require.NotNil(t, thinking)
	assert.Equal(t, map[string]any{"type": "disabled"}, thinking)
	assert.Nil(t, output)

	for _, model := range []string{"claude-opus-4-8", "claude-sonnet-4-6"} {
		thinking, output := sentThinking(t, model, aiprovider.ReasoningNone)
		assert.Nil(t, thinking, model)
		assert.Nil(t, output, model)
	}
}

// Off sends no reasoning setting, so the model's own default applies.
func TestAnthropicAdapter_SendsNoThinkingSettingForOff(t *testing.T) {
	t.Parallel()

	for _, model := range []string{"claude-opus-4-8", "claude-haiku-4-5", "claude-sonnet-5"} {
		thinking, output := sentThinking(t, model, aiprovider.ReasoningOff)
		assert.Nil(t, thinking, model)
		assert.Nil(t, output, model)
	}
}
