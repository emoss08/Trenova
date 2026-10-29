package conversationreplay

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/stretchr/testify/assert"
)

func sequenced(role conversation.Role, sequence int, content string) conversation.Message {
	return conversation.Message{
		Role:     role,
		Sequence: sequence,
		Content:  content,
		Kind:     conversation.MessageKindMessage,
	}
}

func summaryOf(sequence, through int, content string) conversation.Message {
	return conversation.Message{
		Role:       conversation.RoleUser,
		Kind:       conversation.MessageKindCompaction,
		Sequence:   sequence,
		Content:    content,
		Compaction: &conversation.Compaction{Through: through},
	}
}

func contents(messages []conversation.Message) []string {
	out := make([]string, 0, len(messages))
	for _, message := range messages {
		out = append(out, message.Content)
	}

	return out
}

func TestReplayOrder_LeavesAnUncompactedConversationAsItIs(t *testing.T) {
	t.Parallel()

	history := []conversation.Message{
		sequenced(conversation.RoleUser, 0, "q1"),
		sequenced(conversation.RoleAssistant, 1, "a1"),
	}

	assert.Equal(t, history, ReplayOrder(history))
}

/*
The summary is written after the turns it keeps whole, so in sequence order it
follows them. The model has to read it first: it is the account of everything
before those turns, and read after them the conversation would run backwards.
*/
func TestReplayOrder_ReadsTheSummaryBeforeTheTurnsItKeptWhole(t *testing.T) {
	t.Parallel()

	history := []conversation.Message{
		sequenced(conversation.RoleUser, 4, "q3"),
		sequenced(conversation.RoleAssistant, 5, "a3"),
		sequenced(conversation.RoleUser, 6, "q4"),
		sequenced(conversation.RoleAssistant, 7, "a4"),
		summaryOf(8, 3, "summary of q1-a2"),
		sequenced(conversation.RoleUser, 9, "q5"),
	}

	assert.Equal(t,
		[]string{"summary of q1-a2", "q3", "a3", "q4", "a4", "q5"},
		contents(ReplayOrder(history)),
	)
}

/*
A second compaction summarizes the first summary along with what followed it.
The first is folded into the second, so it is never read again, even though it
is numbered after the stretch the second one replaces.
*/
func TestReplayOrder_FoldsAnEarlierSummaryIntoTheLatest(t *testing.T) {
	t.Parallel()

	history := []conversation.Message{
		sequenced(conversation.RoleUser, 6, "q4"),
		sequenced(conversation.RoleAssistant, 7, "a4"),
		summaryOf(8, 3, "first summary"),
		sequenced(conversation.RoleUser, 9, "q5"),
		sequenced(conversation.RoleAssistant, 10, "a5"),
		summaryOf(11, 5, "second summary"),
	}

	assert.Equal(t,
		[]string{"second summary", "q4", "a4", "q5", "a5"},
		contents(ReplayOrder(history)),
	)
}
