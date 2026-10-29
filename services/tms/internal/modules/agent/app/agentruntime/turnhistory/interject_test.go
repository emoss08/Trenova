package turnhistory

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/worldwatch"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplayHistory_FramesASteerAndKeepsItInsideItsTurn(t *testing.T) {
	t.Parallel()

	history := []conversation.Message{
		{Role: conversation.RoleUser, Content: "Find a carrier for S-1001."},
		{Role: conversation.RoleAssistant, Content: "Looking."},
		{Role: conversation.RoleUser, Kind: conversation.MessageKindSteer, Content: "Only Werner.",
			Mentions: []agentpagecontext.EntityRef{
				{Type: "carrier", ID: "car_01J9WERNERWERNERWERNER000", Label: "Werner"},
			}},
		{Role: conversation.RoleUser, Kind: conversation.MessageKindWorldChange, Content: "notice"},
		{Role: conversation.RoleAssistant, Content: "Tendered to Werner."},
	}

	messages, _ := replayHistory(history, nil)
	require.Len(t, messages, 6)
	assert.Equal(
		t,
		worldwatch.SteerPrompt("Only Werner.", history[2].Mentions),
		messages[2].Content,
	)
	assert.Equal(t, llmports.RoleUser, messages[2].Role, "the person's words stay theirs")
	assert.Equal(t, llmports.NoticeMessage(worldwatch.SteerNotice), messages[3],
		"what to make of them is the system's")
	assert.Equal(t, llmports.NoticeMessage("notice"), messages[4])
	assert.Equal(t, 0, recentTurnStart(history, 1),
		"a steer and a notice arrive inside a turn and open none of their own")
}
