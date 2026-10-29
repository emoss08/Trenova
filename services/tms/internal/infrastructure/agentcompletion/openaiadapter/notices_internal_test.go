package openaiadapter

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladaptertest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIResponses_SendsANoticeAsADeveloperMessage(t *testing.T) {
	t.Parallel()

	items := encodedItems(t, toResponsesInput("", modeladaptertest.NoticeConversation(), nil))

	require.Len(t, items, 6)
	assert.Equal(t, "user", items[3]["role"])
	assert.Equal(t, "developer", items[4]["role"])
	assert.Equal(t, "developer", items[5]["role"])
}

func TestOpenAIChat_SendsANoticeAsText(t *testing.T) {
	t.Parallel()

	out := toChatMessages("", modeladaptertest.NoticeConversation(), pulid.Nil, "gpt-5.2")

	require.Len(t, out, 6)
	assert.Equal(t, "user", out[4].Role,
		"an OpenAI-compatible host may refuse a system message mid-conversation")
	assert.Equal(t, modeladaptertest.SteerNotice, out[4].Content)
}
