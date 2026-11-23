package assistanttranscripts

import (
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOversightTranscriptNamesWhoseConversationItIs(t *testing.T) {
	t.Parallel()

	steer := conversation.Message{
		Sequence:  3,
		Role:      conversation.RoleUser,
		Kind:      conversation.MessageKindSteer,
		Content:   "Only last quarter.",
		CreatedAt: 1_790_000_010,
	}
	body := renderTranscript(&transcriptInput{
		Thread:     transcriptThread(),
		AgentName:  "Billing exceptions",
		PersonName: "Dana Reyes",
		Messages:   append(transcriptMessages(), steer),
		ExportedAt: 1_790_000_100,
	})

	assert.Contains(t, body, "- **Person:** Dana Reyes\n- **Agent:** Billing exceptions\n")
	assert.Contains(t, body, "## Dana Reyes · ")
	assert.Contains(t, body, "## Dana Reyes, while it worked · ")
	assert.NotContains(t, body, "## You")
}

func TestOwnersTranscriptStillSpeaksToThem(t *testing.T) {
	t.Parallel()

	body := renderTranscript(&transcriptInput{
		Thread:     transcriptThread(),
		AgentName:  "Billing exceptions",
		Messages:   transcriptMessages(),
		ExportedAt: 1_790_000_100,
	})

	assert.NotContains(t, body, "**Person:**")
	assert.Contains(t, body, "## You · ")
}

func TestOversightNamesAreSafeFileNamesWithoutIDs(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Rate check for Acme", oversightName("  Rate check / for: Acme?  ", "x"))
	assert.Equal(t, "José Núñez", oversightName("José Núñez", "x"))
	assert.Equal(t, "Conversation", oversightName("../..", "Conversation"))
	assert.Len(t, []rune(oversightName(strings.Repeat("a", 200), "x")), oversightNameRunes)
}

func TestOversightDateIsTheDayInTheOrganizationsZone(t *testing.T) {
	t.Parallel()

	loc, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	assert.Equal(
		t,
		"2026-03-19",
		oversightDate(time.Date(2026, 3, 20, 2, 0, 0, 0, time.UTC).Unix(), loc),
	)
	assert.Equal(t, "undated", oversightDate(0, loc))
}

func TestConversationRecordKeepsToolCallsByNameWithTheirResults(t *testing.T) {
	t.Parallel()

	thread := transcriptThread()
	thread.Origin = conversation.ThreadOriginDesk
	record := conversationRecord(&recordInput{
		Thread:     thread,
		OwnerName:  "Dana Reyes",
		OwnerEmail: "dana@example.com",
		AgentName:  "Billing exceptions",
		Messages: []conversation.Message{
			{Role: conversation.RoleUser, Content: "Run it.", CreatedAt: 1_790_000_000},
			{
				Role:      conversation.RoleAssistant,
				CreatedAt: 1_790_000_001,
				ToolCalls: []conversation.ToolCallRecord{{ID: "call_1", Name: "report_run"}},
			},
			{
				Role:        conversation.RoleTool,
				ToolName:    "report_run",
				ToolCallID:  "call_1",
				ToolSummary: "Ran the revenue report",
				Content:     "12 rows",
				CreatedAt:   1_790_000_002,
			},
		},
	})

	assert.Equal(t, "Dana Reyes", record.Owner.Name)
	assert.Equal(t, "dana@example.com", record.Owner.Email)
	assert.Equal(t, "Desk", record.Origin)
	require.Len(t, record.Messages, 3)
	assert.Equal(t, "Run it.", record.Messages[0].Text)
	require.Len(t, record.Messages[1].ToolCalls, 1)
	assert.Equal(t, "report_run", record.Messages[1].ToolCalls[0].Name)
	require.NotNil(t, record.Messages[2].Tool)
	assert.Equal(t, "report_run", record.Messages[2].Tool.Name)
	assert.Equal(t, "Ran the revenue report", record.Messages[2].Tool.Summary)
	assert.Equal(t, "12 rows", record.Messages[2].Tool.Result)
	assert.Equal(t, "2026-09-21T14:13:20Z", record.StartedAt)
}
