package assistantservice

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubChapters struct {
	assistantports.ChapterRepository

	chapters []*conversation.Chapter
	err      error
}

func (s *stubChapters) List(
	context.Context,
	assistantports.ListChaptersRequest,
) ([]*conversation.Chapter, error) {
	return s.chapters, s.err
}

func spokenStretch(from, to int) []conversation.Message {
	messages := make([]conversation.Message, 0, to-from+1)
	for sequence := from; sequence <= to; sequence++ {
		role := conversation.RoleUser
		if sequence%2 == 1 {
			role = conversation.RoleAssistant
		}
		messages = append(messages, conversation.Message{
			Sequence: sequence,
			Role:     role,
			Kind:     conversation.MessageKindMessage,
			Content:  fmt.Sprintf("m%d", sequence),
		})
	}

	return messages
}

func earlierSummary(sequence, through int) conversation.Message {
	return conversation.Message{
		Sequence:   sequence,
		Role:       conversation.RoleUser,
		Kind:       conversation.MessageKindCompaction,
		Content:    "- Looked at the late loads.",
		Compaction: &conversation.Compaction{Through: through},
	}
}

func sequencesOf(messages []conversation.Message) []int {
	out := make([]int, 0, len(messages))
	for idx := range messages {
		out = append(out, messages[idx].Sequence)
	}

	return out
}

func startedAt(sequence int64) *int64 { return &sequence }

func TestSnapToChapter_EndsTheSummaryWhereTheLatestChapterInsideItStarts(t *testing.T) {
	t.Parallel()

	older := spokenStretch(0, 9)

	snapped, through := snapToChapter(
		older,
		9,
		map[int]string{0: "Late loads", 6: "Acme invoice dispute"},
	)

	assert.Equal(t, 5, through)
	assert.Equal(t, []int{0, 1, 2, 3, 4, 5}, sequencesOf(snapped),
		"the chapter that starts inside the stretch is kept whole for the agent")
}

func TestSnapToChapter_TakesTheLatestOfSeveralStarts(t *testing.T) {
	t.Parallel()

	older := spokenStretch(0, 9)

	snapped, through := snapToChapter(older, 9, map[int]string{4: "B", 7: "C"})

	assert.Equal(t, 6, through)
	assert.Equal(t, []int{0, 1, 2, 3, 4, 5, 6}, sequencesOf(snapped))
}

func TestSnapToChapter_KeepsTheCutWhenTheStartWouldSummarizeTooLittle(t *testing.T) {
	t.Parallel()

	older := spokenStretch(0, 9)

	snapped, through := snapToChapter(older, 9, map[int]string{0: "Late loads", 3: "Acme"})

	assert.Equal(t, 9, through, "three of ten messages is less than half the stretch")
	assert.Equal(t, older, snapped)
}

func TestSnapToChapter_SnapsAtExactlyHalf(t *testing.T) {
	t.Parallel()

	older := spokenStretch(0, 9)

	snapped, through := snapToChapter(older, 9, map[int]string{5: "Acme"})

	assert.Equal(t, 4, through)
	assert.Len(t, snapped, 5)
}

func TestSnapToChapter_LeavesTheCutWithoutAChapterStartInside(t *testing.T) {
	t.Parallel()

	older := spokenStretch(0, 9)
	cases := map[string]map[int]string{
		"no chapters":                        nil,
		"empty titles":                       {},
		"a chapter at the stretch's start":   {0: "Late loads"},
		"a chapter after the stretch":        {12: "Acme"},
		"chapters at the start and after it": {0: "Late loads", 10: "Acme"},
	}
	for name, titles := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			snapped, through := snapToChapter(older, 9, titles)

			assert.Equal(t, 9, through)
			assert.Equal(t, older, snapped)
		})
	}
}

func TestSnapToChapter_LeavesAStretchOfOnlyAnEarlierSummary(t *testing.T) {
	t.Parallel()

	older := []conversation.Message{earlierSummary(20, 9)}

	snapped, through := snapToChapter(older, 9, map[int]string{20: "Late loads", 5: "Acme"})

	assert.Equal(t, 9, through)
	assert.Equal(t, older, snapped)
}

func TestSnapToChapter_KeepsAnEarlierSummaryAndCountsOnlyWhatWasSaid(t *testing.T) {
	t.Parallel()

	older := append([]conversation.Message{earlierSummary(30, 9)}, spokenStretch(10, 19)...)

	snapped, through := snapToChapter(older, 19, map[int]string{30: "Old", 16: "Acme"})

	assert.Equal(t, 15, through)
	assert.Equal(t, []int{30, 10, 11, 12, 13, 14, 15}, sequencesOf(snapped),
		"the earlier summary stays first and the six messages before the start are summarized")
}

func TestSnapToChapter_LeavesAnEmptyStretch(t *testing.T) {
	t.Parallel()

	snapped, through := snapToChapter(nil, 4, map[int]string{2: "Acme"})

	assert.Empty(t, snapped)
	assert.Equal(t, 4, through)
}

func TestTranscriptLines_OpensEachChapterWithItsTitle(t *testing.T) {
	t.Parallel()

	older := []conversation.Message{
		earlierSummary(40, 1),
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

	lines := transcriptLines(spokenStretch(0, 1), compactionToolChars, nil)

	assert.Equal(t, []string{"Person: m0", "Agent: m1"}, lines)
}

func TestCompactionTranscript_CarriesTheChapterTitles(t *testing.T) {
	t.Parallel()

	transcript := compactionTranscript(
		spokenStretch(0, 3),
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

	older := append([]conversation.Message{earlierSummary(30, 9)}, spokenStretch(10, 12)...)

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

	svc, conversations := newConversationService(&scriptedCompletion{}, testDefinition())
	conversations.messages = longConversation(12)
	conversations.thread.ContextUsage = &conversation.ContextUsage{
		Instructions: 6200,
		Model:        "claude-sonnet-4-5",
	}
	svc.chapters = &stubChapters{chapters: []*conversation.Chapter{
		{ID: "achp_1", Title: "Customers 0 to 5", StartSequence: startedAt(0)},
		{ID: "achp_2", Title: "Customers 6 to 9", StartSequence: startedAt(24)},
		{ID: "achp_3", Title: "Being answered"},
	}}
	actor := testActor()

	plan, err := svc.PrepareCompaction(t.Context(), &CompactRequest{
		ThreadID:   conversations.thread.ID,
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

	svc, conversations := newConversationService(&scriptedCompletion{}, testDefinition())
	conversations.messages = longConversation(12)
	conversations.thread.ContextUsage = &conversation.ContextUsage{
		Instructions: 6200,
		Model:        "claude-sonnet-4-5",
	}
	svc.chapters = &stubChapters{err: errors.New("database is down")}
	actor := testActor()

	plan, err := svc.PrepareCompaction(t.Context(), &CompactRequest{
		ThreadID:   conversations.thread.ID,
		TenantInfo: actor.TenantInfo(),
	}, actor)
	require.NoError(t, err)

	assert.Equal(t, 10*4-1, plan.Through)
	assert.NotContains(t, plan.Request.System, "divided into chapters")
	assert.NotContains(t, plan.Request.Messages[0].Content, "Chapter: ")
}

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
		&stubConversations{thread: input.Thread, messages: input.Messages},
		&stubProposalRepo{},
	)
	svc.chapters = &stubChapters{chapters: []*conversation.Chapter{
		{ID: "achp_1", Title: "Late loads", StartSequence: startedAt(1)},
		{ID: "achp_2", Title: "Acme invoice dispute", StartSequence: startedAt(3)},
		{ID: "achp_3", Title: "Being answered"},
	}}

	transcript, err := svc.Transcript(
		t.Context(),
		assistantports.GetThreadRequest{ID: input.Thread.ID},
	)
	require.NoError(t, err)

	assert.Contains(t, transcript.Body, "- **Chapters:** 2\n",
		"a chapter with nothing saved yet is not counted")
	assert.Contains(t, transcript.Body, "# Chapter 1 · Late loads\n")
	assert.Contains(t, transcript.Body, "# Chapter 2 · Acme invoice dispute\n")
}
