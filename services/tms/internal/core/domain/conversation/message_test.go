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

func TestStampUnstamped_KeepsEachMessagesOwnTime(t *testing.T) {
	t.Parallel()

	messages := []Message{
		{Role: RoleUser},
		{Role: RoleAssistant, CreatedAt: 110},
		{Role: RoleTool, CreatedAt: 140},
		{Role: RoleAssistant, CreatedAt: 260},
		{Role: RoleAssistant},
	}

	StampUnstamped(messages, 300)

	assert.Equal(t, []int64{110, 110, 140, 260, 300}, stamps(messages),
		"an unstamped question takes the first step's time and an unstamped note the save's")
}

func TestStampUnstamped_NeverStampsAheadOfWhatFollowsOrOfTheSave(t *testing.T) {
	t.Parallel()

	messages := []Message{
		{Role: RoleUser, CreatedAt: 100},
		{Role: RoleAssistant, CreatedAt: 500},
		{Role: RoleTool, CreatedAt: 200},
		{Role: RoleAssistant, CreatedAt: 900},
	}

	StampUnstamped(messages, 400)

	assert.Equal(t, []int64{100, 200, 200, 400}, stamps(messages))
}

func stamps(messages []Message) []int64 {
	out := make([]int64, 0, len(messages))
	for idx := range messages {
		out = append(out, messages[idx].CreatedAt)
	}

	return out
}
