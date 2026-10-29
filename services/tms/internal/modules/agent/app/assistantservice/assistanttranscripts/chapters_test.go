package assistanttranscripts

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationchapter"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func chapterTranscriptInput(chapters map[int]string) *transcriptInput {
	return &transcriptInput{
		Thread: &conversation.Thread{
			ID:            pulid.MustNew("athr_"),
			Title:         "Monday work",
			CreatedAt:     1_790_000_000,
			LastMessageAt: 1_790_000_060,
		},
		AgentName: "Dispatch desk",
		Messages: []conversation.Message{
			{
				Sequence:  1,
				Role:      conversation.RoleUser,
				Content:   "Which loads are late?",
				CreatedAt: 1_790_000_000,
			},
			{
				Sequence:  2,
				Role:      conversation.RoleAssistant,
				Content:   "Three are late.",
				CreatedAt: 1_790_000_010,
			},
			{
				Sequence:  3,
				Role:      conversation.RoleUser,
				Content:   "Open the Acme dispute",
				CreatedAt: 1_790_000_020,
			},
			{
				Sequence:  4,
				Role:      conversation.RoleAssistant,
				Content:   "It is open.",
				CreatedAt: 1_790_000_030,
			},
		},
		Chapters:   chapters,
		ExportedAt: 1_790_000_070,
	}
}

func TestRenderTranscript_NumbersTheChaptersInTheOrderTheyStart(t *testing.T) {
	t.Parallel()

	body := renderTranscript(chapterTranscriptInput(map[int]string{
		3: "Acme invoice dispute",
		1: "Late loads",
	}))

	assert.Contains(t, body, "- **Chapters:** 2\n")
	assert.Contains(t, body, "\n---\n\n# Chapter 1 · Late loads\n")
	assert.Contains(t, body, "\n---\n\n# Chapter 2 · Acme invoice dispute\n")
	first := strings.Index(body, "# Chapter 1 · Late loads")
	second := strings.Index(body, "# Chapter 2 · Acme invoice dispute")
	assert.Less(t, first, strings.Index(body, "Which loads are late?"))
	assert.Less(t, strings.Index(body, "Three are late."), second)
	assert.Less(t, second, strings.Index(body, "Open the Acme dispute"))
	assert.Less(
		t,
		strings.Index(body, "- **Last message:**"),
		strings.Index(body, "- **Chapters:**"),
	)
	assert.Less(t, strings.Index(body, "- **Chapters:**"), strings.Index(body, "- **Messages:**"))
}

func TestRenderTranscript_WithoutChaptersHasNoChapterHeadings(t *testing.T) {
	t.Parallel()

	body := renderTranscript(chapterTranscriptInput(nil))

	assert.NotContains(t, body, "# Chapter")
	assert.NotContains(t, body, "**Chapters:**")
}

func TestTranscript_HeadsEachChapterFromTheConversationsChapters(t *testing.T) {
	t.Parallel()

	input := chapterTranscriptInput(nil)
	svc := newTranscriptService(
		&assistanttest.StubConversations{Thread: input.Thread, Messages: input.Messages},
		&assistanttest.StubProposalRepo{},
	)
	svc.chapters = &assistanttest.StubChapters{Chapters: []*conversationchapter.Chapter{
		{ID: "achp_1", Title: "Late loads", StartSequence: assistanttest.StartedAt(1)},
		{ID: "achp_2", Title: "Acme invoice dispute", StartSequence: assistanttest.StartedAt(3)},
		{ID: "achp_3", Title: "Being answered"},
	}}

	transcript, err := svc.Transcript(
		t.Context(),
		&conversationports.GetThreadRequest{ID: input.Thread.ID},
	)
	require.NoError(t, err)

	assert.Contains(t, transcript.Body, "- **Chapters:** 2\n",
		"a chapter with nothing saved yet is not counted")
	assert.Contains(t, transcript.Body, "# Chapter 1 · Late loads\n")
	assert.Contains(t, transcript.Body, "# Chapter 2 · Acme invoice dispute\n")
}
