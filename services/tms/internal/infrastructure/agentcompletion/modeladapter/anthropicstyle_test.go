package modeladapter

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// styledThinking is the thinking and output_config a request carried for a
// provider whose operator chose how its model takes thinking.
func styledThinking(
	t *testing.T,
	model string,
	style aiprovider.ThinkingStyle,
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
	call.Provider.ThinkingStyle = style
	_, _ = streamWith(t, NewAnthropicAdapter(), call)

	thinking, _ := (*captured)["thinking"].(map[string]any)
	output, _ := (*captured)["output_config"].(map[string]any)

	return thinking, output
}

/*
A gateway names a model whatever it likes, and an id the adapter cannot read
keeps the token budget, which every current Claude model refuses with a 400.
An operator who knows the model behind the alias thinks by effort says so, and
the levels are asked for by effort however the id reads.
*/
func TestAnthropicAdapter_ThinksByEffortWhenTheOperatorSaysSo(t *testing.T) {
	t.Parallel()

	for level, effort := range map[aiprovider.ReasoningEffort]string{
		aiprovider.ReasoningMinimal: "low",
		aiprovider.ReasoningLow:     "low",
		aiprovider.ReasoningMedium:  "medium",
		aiprovider.ReasoningHigh:    "high",
	} {
		thinking, output := styledThinking(t, "acme-reasoner", aiprovider.ThinkingStyleEffort, level)
		require.NotNil(t, thinking, level)
		assert.Equal(t, "adaptive", thinking["type"], level)
		assert.Equal(t, "summarized", thinking["display"], level)
		assert.NotContains(t, thinking, "budget_tokens", level)
		require.NotNil(t, output, level)
		assert.Equal(t, effort, output["effort"], level)
	}
}

/*
Behind an alias the adapter cannot tell a model that may stop thinking from
one that cannot, so None asks for the least thinking every effort model
accepts: adaptive at low effort. Sending nothing would leave a model that
always thinks at its own default.
*/
func TestAnthropicAdapter_NoneOnADeclaredEffortModelAsksForTheLeast(t *testing.T) {
	t.Parallel()

	thinking, output := styledThinking(t, "acme-reasoner", aiprovider.ThinkingStyleEffort, aiprovider.ReasoningNone)
	require.NotNil(t, thinking)
	assert.Equal(t, "adaptive", thinking["type"])
	require.NotNil(t, output)
	assert.Equal(t, "low", output["effort"])

	thinking, output = styledThinking(t, "acme-reasoner", aiprovider.ThinkingStyleEffort, aiprovider.ReasoningOff)
	assert.Nil(t, thinking)
	assert.Nil(t, output)
}

// An operator who knows the model behind an id takes a budget gets one, even
// when the id reads like a model that thinks by effort.
func TestAnthropicAdapter_KeepsTheBudgetWhenTheOperatorSaysSo(t *testing.T) {
	t.Parallel()

	thinking, output := styledThinking(t, "claude-opus-5-5", aiprovider.ThinkingStyleBudget, aiprovider.ReasoningHigh)
	require.NotNil(t, thinking)
	assert.Equal(t, "enabled", thinking["type"])
	assert.InDelta(t, aiprovider.ReasoningHigh.ThinkingBudget(), thinking["budget_tokens"], 0)
	assert.NotContains(t, thinking, "block_binding")
	if output != nil {
		assert.NotContains(t, output, "effort")
	}
}

// Auto, and a provider saved before the setting existed, read the model id.
func TestAnthropicAdapter_AutoReadsTheModelID(t *testing.T) {
	t.Parallel()

	for _, style := range []aiprovider.ThinkingStyle{aiprovider.ThinkingStyleAuto, ""} {
		thinking, _ := styledThinking(t, "claude-opus-5-5", style, aiprovider.ReasoningHigh)
		require.NotNil(t, thinking, style)
		assert.Equal(t, "adaptive", thinking["type"], style)

		thinking, _ = styledThinking(t, "claude-haiku-4-5", style, aiprovider.ReasoningHigh)
		require.NotNil(t, thinking, style)
		assert.Equal(t, "enabled", thinking["type"], style)
	}
}
