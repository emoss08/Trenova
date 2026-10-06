package conversation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func sequenced(role Role, sequence int, content string) Message {
	return Message{Role: role, Sequence: sequence, Content: content, Kind: MessageKindMessage}
}

func summaryOf(sequence, through int, content string) Message {
	return Message{
		Role:       RoleUser,
		Kind:       MessageKindCompaction,
		Sequence:   sequence,
		Content:    content,
		Compaction: &Compaction{Through: through},
	}
}

func contents(messages []Message) []string {
	out := make([]string, 0, len(messages))
	for _, message := range messages {
		out = append(out, message.Content)
	}

	return out
}

func TestReplayOrder_LeavesAnUncompactedConversationAsItIs(t *testing.T) {
	t.Parallel()

	history := []Message{
		sequenced(RoleUser, 0, "q1"),
		sequenced(RoleAssistant, 1, "a1"),
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

	history := []Message{
		sequenced(RoleUser, 4, "q3"),
		sequenced(RoleAssistant, 5, "a3"),
		sequenced(RoleUser, 6, "q4"),
		sequenced(RoleAssistant, 7, "a4"),
		summaryOf(8, 3, "summary of q1-a2"),
		sequenced(RoleUser, 9, "q5"),
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

	history := []Message{
		sequenced(RoleUser, 6, "q4"),
		sequenced(RoleAssistant, 7, "a4"),
		summaryOf(8, 3, "first summary"),
		sequenced(RoleUser, 9, "q5"),
		sequenced(RoleAssistant, 10, "a5"),
		summaryOf(11, 5, "second summary"),
	}

	assert.Equal(t,
		[]string{"second summary", "q4", "a4", "q5", "a5"},
		contents(ReplayOrder(history)),
	)
}

func TestContextUsage_CompactsItselfOnlyPastTheShareAndWithEnoughToFree(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		usage *ContextUsage
		want  bool
	}{
		{"nothing measured", nil, false},
		{
			"below the share",
			&ContextUsage{Instructions: 6000, Messages: 100_000, Compactable: 90_000, Window: 200_000},
			false,
		},
		{
			"past the share with plenty behind the latest turns",
			&ContextUsage{
				Instructions: 6000, Messages: 40_000, ToolResults: 130_000,
				Compactable: 150_000, Window: 200_000,
			},
			true,
		},
		{
			"past the share but all of it in the latest turns",
			&ContextUsage{Instructions: 6000, ToolResults: 170_000, Compactable: 3000, Window: 200_000},
			false,
		},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, tc.usage.NeedsCompaction(), tc.name)
	}
}

func TestContextUsage_FreesTheStretchLessItsSummary(t *testing.T) {
	t.Parallel()

	usage := &ContextUsage{Compactable: 100_000}
	assert.Equal(t, 100_000-SummaryTokenBudget, usage.Frees())

	small := &ContextUsage{Compactable: 5000}
	assert.Equal(t, 4000, small.Frees(), "a short stretch is expected to summarize to a fifth")
	assert.False(t, small.WorthCompacting(), "freeing no more than the minimum is not worth a model call")
}
