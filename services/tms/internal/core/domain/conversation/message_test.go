package conversation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReasoningTrace_ReplayableOnlyByItsOwnProtocol(t *testing.T) {
	t.Parallel()

	var missing *ReasoningTrace
	assert.False(t, missing.ReplayableBy("AnthropicMessages"))
	assert.True(
		t,
		(&ReasoningTrace{Signature: "s"}).ReplayableBy("AnthropicMessages"),
		"an untagged trace replays as before",
	)
	assert.True(
		t,
		(&ReasoningTrace{ProviderKind: "OpenAIResponses"}).ReplayableBy("OpenAIResponses"),
	)
	assert.False(
		t,
		(&ReasoningTrace{ProviderKind: "OpenAIResponses"}).ReplayableBy("AnthropicMessages"),
	)
}
