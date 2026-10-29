package assistantcompaction

import (
	"errors"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationchapter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTranscriptLines_OpensEachChapterWithItsTitle(t *testing.T) {
	t.Parallel()

	older := []conversation.Message{
		assistanttest.EarlierSummary(40, 1),
		{Sequence: 2, Role: conversation.RoleUser, Content: "Which loads are late?"},
		{Sequence: 3, Role: conversation.RoleAssistant, Content: "Three are late."},
		{Sequence: 4, Role: conversation.RoleUser, Content: "Open the Acme dispute"},
		{Sequence: 5, Role: conversation.RoleAssistant, Content: "It is open."},
	}
	titles := map[int]string{40: "Earlier", 2: "Late loads", 4: "Acme invoice dispute"}

	lines := transcriptLines(older, compactionToolChars, titles)

	assert.Equal(t, []string{
		"Summary of the conversation before this:\n- Looked at the late loads.",
		"Chapter: Late loads",
		"Person: Which loads are late?",
		"Agent: Three are late.",
		"Chapter: Acme invoice dispute",
		"Person: Open the Acme dispute",
		"Agent: It is open.",
	}, lines, "a summary is never headed as a chapter")
}

func TestTranscriptLines_WithoutChaptersHasNoChapterLines(t *testing.T) {
	t.Parallel()

	lines := transcriptLines(assistanttest.SpokenStretch(0, 1), compactionToolChars, nil)

	assert.Equal(t, []string{"Person: m0", "Agent: m1"}, lines)
}

func TestCompactionTranscript_CarriesTheChapterTitles(t *testing.T) {
	t.Parallel()

	transcript := compactionTranscript(
		assistanttest.SpokenStretch(0, 3),
		map[int]string{0: "Late loads", 2: "Acme"},
	)

	assert.Equal(
		t,
		"Chapter: Late loads\n\nPerson: m0\n\nAgent: m1\n\nChapter: Acme\n\nPerson: m2\n\nAgent: m3",
		transcript,
	)
}

func TestChaptered_OnlyWhenAChapterStartsAtASpokenMessage(t *testing.T) {
	t.Parallel()

	older := append(
		[]conversation.Message{assistanttest.EarlierSummary(30, 9)},
		assistanttest.SpokenStretch(10, 12)...)

	assert.True(t, chaptered(older, map[int]string{11: "Acme"}))
	assert.False(t, chaptered(older, map[int]string{30: "Earlier"}))
	assert.False(t, chaptered(older, map[int]string{2: "Before"}))
	assert.False(t, chaptered(older, nil))
}

func TestCompactionPrompt_AsksToKeepChaptersTogetherOnlyWhenThereAreAny(t *testing.T) {
	t.Parallel()

	with := compactionPrompt("Dispatch desk", true)
	without := compactionPrompt("Dispatch desk", false)

	instruction := "The conversation is divided into chapters, each opened by a line " +
		"\"Chapter: <title>\". Keep each chapter's points together, in order, and " +
		"start the first bullet of each chapter with its title in square brackets, " +
		"so the agent still knows what each chapter was about.\n\n"
	assert.Contains(t, with, instruction)
	assert.NotContains(t, without, "Chapter: <title>")
	assert.Equal(t, without, strings.Replace(with, instruction, "", 1),
		"the chapter instruction is the only difference")
	assert.Contains(t, with, "Dispatch desk")
}

func TestPrepareCompaction_SummarizesUpToTheLatestChapterStart(t *testing.T) {
	t.Parallel()

	svc, conversations := newCompactionService(assistanttest.Definition())
	conversations.Messages = assistanttest.LongConversation(12)
	conversations.Thread.ContextUsage = &conversation.ContextUsage{
		Instructions: 6200,
		Model:        "claude-sonnet-4-5",
	}
	svc.chapters = &assistanttest.StubChapters{Chapters: []*conversationchapter.Chapter{
		{ID: "achp_1", Title: "Customers 0 to 5", StartSequence: assistanttest.StartedAt(0)},
		{ID: "achp_2", Title: "Customers 6 to 9", StartSequence: assistanttest.StartedAt(24)},
		{ID: "achp_3", Title: "Being answered"},
	}}
	actor := assistanttest.Actor()

	plan, err := svc.PrepareCompaction(t.Context(), &CompactRequest{
		ThreadID:   conversations.Thread.ID,
		TenantInfo: actor.TenantInfo(),
	}, actor)
	require.NoError(t, err)

	assert.Equal(t, 23, plan.Through, "the summary ends before the chapter that starts at 24")
	assert.Equal(t, 24, plan.Summarized)
	assert.Contains(t, plan.Request.System, "divided into chapters")
	transcript := plan.Request.Messages[0].Content
	assert.True(
		t,
		strings.HasPrefix(
			transcript,
			"Chapter: Customers 0 to 5\n\nPerson: Which loads for customer 0?",
		),
	)
	assert.NotContains(t, transcript, "Customers 6 to 9")
	assert.NotContains(t, transcript, "customer 6?")
}

func TestPrepareCompaction_ReadsOnWhenTheChaptersCannotBeRead(t *testing.T) {
	t.Parallel()

	svc, conversations := newCompactionService(assistanttest.Definition())
	conversations.Messages = assistanttest.LongConversation(12)
	conversations.Thread.ContextUsage = &conversation.ContextUsage{
		Instructions: 6200,
		Model:        "claude-sonnet-4-5",
	}
	svc.chapters = &assistanttest.StubChapters{Err: errors.New("database is down")}
	actor := assistanttest.Actor()

	plan, err := svc.PrepareCompaction(t.Context(), &CompactRequest{
		ThreadID:   conversations.Thread.ID,
		TenantInfo: actor.TenantInfo(),
	}, actor)
	require.NoError(t, err)

	assert.Equal(t, 10*4-1, plan.Through)
	assert.NotContains(t, plan.Request.System, "divided into chapters")
	assert.NotContains(t, plan.Request.Messages[0].Content, "Chapter: ")
}
