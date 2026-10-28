package assistantchapterservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/testutil/validationtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRename_KeepsThePersonsTitleOnOneLine(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	turn := f.turn(conversation.AssistantTurnOriginPerson, conversation.AssistantTurnStatusCompleted, "a")
	chapter := f.chapter(turn.ID, "Late loads", conversation.ChapterTitleModel, sequence(0))
	version := int64(3)

	renamed, err := f.svc.Rename(t.Context(), agentports.RenameChapterRequest{
		ID:         chapter.ID,
		ThreadID:   f.thread.ID,
		UserID:     f.tenant.UserID,
		Title:      "  Acme   invoice\n dispute ",
		Version:    &version,
		TenantInfo: f.tenant,
	})
	require.NoError(t, err)

	assert.Equal(t, "Acme invoice dispute", renamed.Title)
	require.Len(t, f.chapters.retitled, 1)
	assert.Equal(t, agentports.RetitleChapterRequest{
		ID:          chapter.ID,
		ThreadID:    f.thread.ID,
		TenantInfo:  f.tenant,
		Title:       "Acme invoice dispute",
		TitleSource: conversation.ChapterTitlePerson,
		Version:     &version,
	}, f.chapters.retitled[0])
	f.assertAnnounced(t, 1)
}

func TestRename_RefusesATitleOverSixtyCharacters(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	turn := f.turn(conversation.AssistantTurnOriginPerson, conversation.AssistantTurnStatusCompleted, "a")
	chapter := f.chapter(turn.ID, "Late loads", conversation.ChapterTitleModel, sequence(0))

	_, err := f.svc.Rename(t.Context(), agentports.RenameChapterRequest{
		ID:         chapter.ID,
		ThreadID:   f.thread.ID,
		UserID:     f.tenant.UserID,
		Title:      strings.Repeat("é", conversation.ChapterTitleRunes+1),
		TenantInfo: f.tenant,
	})

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Equal(t, []string{string(errortypes.ErrInvalid)}, validationtest.FieldCodes(multiErr, "title"))
	assert.Empty(t, f.chapters.retitled)
	f.assertAnnounced(t, 0)
}

func TestRename_RefusesABlankTitle(t *testing.T) {
	t.Parallel()

	f := newFixture(t)

	_, err := f.svc.Rename(t.Context(), agentports.RenameChapterRequest{
		ID:         "achp_a",
		ThreadID:   f.thread.ID,
		UserID:     f.tenant.UserID,
		Title:      " \n\t ",
		TenantInfo: f.tenant,
	})

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Equal(t, []string{string(errortypes.ErrRequired)}, validationtest.FieldCodes(multiErr, "title"))
}

func TestChapters_SomeoneElsesConversationIsNotFound(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	stranger := pulid.MustNew("usr_")

	_, err := f.svc.List(t.Context(), agentports.ThreadChaptersRequest{
		ThreadID:   f.thread.ID,
		UserID:     stranger,
		TenantInfo: f.tenant,
	})
	assert.True(t, errortypes.IsNotFoundError(err))

	_, err = f.svc.Rename(t.Context(), agentports.RenameChapterRequest{
		ID:         "achp_a",
		ThreadID:   f.thread.ID,
		UserID:     stranger,
		Title:      "Mine now",
		TenantInfo: f.tenant,
	})
	assert.True(t, errortypes.IsNotFoundError(err))

	err = f.svc.Remove(t.Context(), agentports.RemoveChapterRequest{
		ID:         "achp_a",
		ThreadID:   f.thread.ID,
		UserID:     stranger,
		TenantInfo: f.tenant,
	})
	assert.True(t, errortypes.IsNotFoundError(err))

	_, err = f.svc.Open(t.Context(), agentports.OpenChapterRequest{
		ThreadID:   f.thread.ID,
		UserID:     stranger,
		MessageID:  "amsg_a",
		TenantInfo: f.tenant,
	})
	assert.True(t, errortypes.IsNotFoundError(err))

	assert.Empty(t, f.chapters.retitled)
	assert.Empty(t, f.chapters.deleted)
	assert.Empty(t, f.chapters.created)
}

func TestRemove_DeletesAnyChapterThePersonPicks(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	turn := f.turn(conversation.AssistantTurnOriginPerson, conversation.AssistantTurnStatusCompleted, "a")
	chapter := f.chapter(turn.ID, "My loads", conversation.ChapterTitlePerson, sequence(0))

	err := f.svc.Remove(t.Context(), agentports.RemoveChapterRequest{
		ID:         chapter.ID,
		ThreadID:   f.thread.ID,
		UserID:     f.tenant.UserID,
		TenantInfo: f.tenant,
	})
	require.NoError(t, err)

	require.Len(t, f.chapters.deleted, 1)
	assert.False(t, f.chapters.deleted[0].KeepPerson, "a person may remove a chapter they named")
	assert.Empty(t, f.chapters.chapters)
	f.assertAnnounced(t, 1)
}

func TestRemove_AMissingChapterIsNotFound(t *testing.T) {
	t.Parallel()

	f := newFixture(t)

	err := f.svc.Remove(t.Context(), agentports.RemoveChapterRequest{
		ID:         "achp_gone",
		ThreadID:   f.thread.ID,
		UserID:     f.tenant.UserID,
		TenantInfo: f.tenant,
	})

	assert.True(t, errortypes.IsNotFoundError(err))
	f.assertAnnounced(t, 0)
}

func TestOpen_StartsAChapterAtTheQuestionOfTheMessagesTurn(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	turn := f.turn(conversation.AssistantTurnOriginPerson, conversation.AssistantTurnStatusCompleted, "late loads")
	messageID := pulid.MustNew("amsg_")
	f.chapters.turnOf[messageID] = turn.ID
	f.conversations.byTurn[turn.ID] = []conversation.Message{
		{Role: conversation.RoleUser, Content: "which loads are late?"},
		{Role: conversation.RoleAssistant, Content: "Three are late."},
	}

	opened, err := f.svc.Open(t.Context(), agentports.OpenChapterRequest{
		ThreadID:   f.thread.ID,
		UserID:     f.tenant.UserID,
		MessageID:  messageID,
		TenantInfo: f.tenant,
	})
	require.NoError(t, err)

	assert.Equal(t, "Which loads are late?", opened.Title)
	assert.Equal(t, conversation.ChapterTitleQuestion, opened.TitleSource)
	assert.Equal(t, turn.ID, opened.TurnID)
	require.Len(t, f.chapters.created, 1)
	assert.Equal(t, []*agentports.ChapterTurnRequest{{
		TenantInfo: f.tenant,
		ThreadID:   f.thread.ID,
		UserID:     f.tenant.UserID,
		TurnID:     turn.ID,
		Phase:      agentports.ChapterPhaseRequested,
	}}, f.scheduler.cues, "the model is asked to name it afterwards")
}

func TestOpen_NamesTheQuietFirstChapterAlongsideTheNewOne(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	first := f.turn(conversation.AssistantTurnOriginPerson, conversation.AssistantTurnStatusCompleted, "late loads")
	f.chapter(first.ID, "Late loads", conversation.ChapterTitleQuestion, sequence(0))
	named := f.turn(conversation.AssistantTurnOriginPerson, conversation.AssistantTurnStatusCompleted, "acme")
	f.chapter(named.ID, "Acme invoice dispute", conversation.ChapterTitlePerson, sequence(4))
	turn := f.turn(conversation.AssistantTurnOriginPerson, conversation.AssistantTurnStatusCompleted, "driver hours")
	messageID := pulid.MustNew("amsg_")
	f.chapters.turnOf[messageID] = turn.ID
	f.conversations.byTurn[turn.ID] = []conversation.Message{
		{Role: conversation.RoleUser, Content: "driver hours for Tuesday"},
	}

	_, err := f.svc.Open(t.Context(), agentports.OpenChapterRequest{
		ThreadID:   f.thread.ID,
		UserID:     f.tenant.UserID,
		MessageID:  messageID,
		TenantInfo: f.tenant,
	})
	require.NoError(t, err)

	cued := make([]pulid.ID, 0, len(f.scheduler.cues))
	for _, cue := range f.scheduler.cues {
		assert.Equal(t, agentports.ChapterPhaseRequested, cue.Phase)
		cued = append(cued, cue.TurnID)
	}
	assert.Equal(t, []pulid.ID{turn.ID, first.ID}, cued,
		"the new chapter and the quiet first one are named; the person's title is kept")
}

func TestOpen_AMessageOutsideATurnCannotStartAChapter(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	messageID := pulid.MustNew("amsg_")
	f.chapters.turnOf[messageID] = ""

	_, err := f.svc.Open(t.Context(), agentports.OpenChapterRequest{
		ThreadID:   f.thread.ID,
		UserID:     f.tenant.UserID,
		MessageID:  messageID,
		TenantInfo: f.tenant,
	})

	assert.True(t, errortypes.IsBusinessError(err))
	assert.Empty(t, f.chapters.created)
	assert.Empty(t, f.scheduler.cues)
}

func TestOpen_AnUnknownMessageIsNotFound(t *testing.T) {
	t.Parallel()

	f := newFixture(t)

	_, err := f.svc.Open(t.Context(), agentports.OpenChapterRequest{
		ThreadID:   f.thread.ID,
		UserID:     f.tenant.UserID,
		MessageID:  "amsg_gone",
		TenantInfo: f.tenant,
	})

	assert.True(t, errortypes.IsNotFoundError(err))
}

func TestOpen_ATurnThatAlreadyOpensAChapterReturnsIt(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	turn := f.turn(conversation.AssistantTurnOriginPerson, conversation.AssistantTurnStatusCompleted, "late loads")
	existing := f.chapter(turn.ID, "Late loads", conversation.ChapterTitleModel, sequence(0))
	messageID := pulid.MustNew("amsg_")
	f.chapters.turnOf[messageID] = turn.ID

	opened, err := f.svc.Open(t.Context(), agentports.OpenChapterRequest{
		ThreadID:   f.thread.ID,
		UserID:     f.tenant.UserID,
		MessageID:  messageID,
		TenantInfo: f.tenant,
	})
	require.NoError(t, err)

	assert.Same(t, existing, opened)
	assert.Empty(t, f.chapters.created)
	assert.Empty(t, f.scheduler.cues)
}

func TestOpen_RefusesPastTheLimit(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	for idx := range conversation.MaxChaptersPerThread {
		earlier := f.turn(conversation.AssistantTurnOriginPerson, conversation.AssistantTurnStatusCompleted, "a")
		f.chapter(earlier.ID, "Chapter", conversation.ChapterTitleModel, sequence(int64(idx)))
	}
	turn := f.turn(conversation.AssistantTurnOriginPerson, conversation.AssistantTurnStatusCompleted, "late loads")
	messageID := pulid.MustNew("amsg_")
	f.chapters.turnOf[messageID] = turn.ID

	_, err := f.svc.Open(t.Context(), agentports.OpenChapterRequest{
		ThreadID:   f.thread.ID,
		UserID:     f.tenant.UserID,
		MessageID:  messageID,
		TenantInfo: f.tenant,
	})

	assert.True(t, errortypes.IsBusinessError(err))
	assert.Empty(t, f.chapters.created)
}

func spanChapters() []*conversation.Chapter {
	return []*conversation.Chapter{
		{ID: "achp_1", Title: "Late loads", StartSequence: sequence(0)},
		{ID: "achp_2", Title: "Acme invoice dispute", StartSequence: sequence(6)},
		{ID: "achp_3", Title: "Still being answered"},
		{ID: "achp_4", Title: "Detention policy", StartSequence: sequence(14)},
	}
}

func intPtr(value int) *int { return &value }

func TestSpanOf_RunsFromAChaptersStartToTheNextOnesStart(t *testing.T) {
	t.Parallel()

	chapters := spanChapters()
	cases := map[pulid.ID]struct {
		from    int
		through *int
	}{
		"achp_1": {from: 0, through: intPtr(5)},
		"achp_2": {from: 6, through: intPtr(13)},
		"achp_4": {from: 14, through: nil},
	}
	for id, want := range cases {
		span, err := SpanOf(chapters, id)
		require.NoError(t, err, id)

		assert.Equal(t, id, span.Chapter.ID)
		assert.Equal(t, want.from, span.From, id)
		assert.Equal(t, want.through, span.Through, id)
	}
}

func TestSpanOf_AChapterWithNothingSavedHasNoSpan(t *testing.T) {
	t.Parallel()

	_, err := SpanOf(spanChapters(), "achp_3")

	assert.True(t, errortypes.IsBusinessError(err))
}

func TestSpanOf_AnUnknownChapterIsNotFound(t *testing.T) {
	t.Parallel()

	_, err := SpanOf(spanChapters(), "achp_gone")

	assert.True(t, errortypes.IsNotFoundError(err))
}

func TestSpan_ReadsTheConversationsChapters(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	first := f.turn(conversation.AssistantTurnOriginPerson, conversation.AssistantTurnStatusCompleted, "a")
	second := f.turn(conversation.AssistantTurnOriginPerson, conversation.AssistantTurnStatusCompleted, "b")
	opening := f.chapter(first.ID, "Late loads", conversation.ChapterTitleModel, sequence(2))
	f.chapter(second.ID, "Acme invoice dispute", conversation.ChapterTitleModel, sequence(9))

	span, err := f.svc.Span(t.Context(), agentports.ChapterSpanRequest{
		ID:         opening.ID,
		ThreadID:   f.thread.ID,
		UserID:     f.tenant.UserID,
		TenantInfo: f.tenant,
	})
	require.NoError(t, err)

	assert.Equal(t, &agentports.ChapterSpan{Chapter: opening, From: 2, Through: intPtr(8)}, span)
}
