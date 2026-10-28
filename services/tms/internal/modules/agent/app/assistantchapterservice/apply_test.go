package assistantchapterservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func answer(text string) *agentports.StructuredCompletionResult {
	return &agentports.StructuredCompletionResult{Text: text, ModelIdentifier: "test-model"}
}

func (f *fixture) openingPlan(t *testing.T, question string) *agentports.ChapterPlan {
	t.Helper()

	turn := f.turn(conversation.AssistantTurnOriginPerson, conversation.AssistantTurnStatusRunning, question)
	plan, err := f.svc.Plan(t.Context(), f.request(turn.ID, agentports.ChapterPhaseOpened))
	require.NoError(t, err)
	require.True(t, plan.Opens)

	return plan
}

func (f *fixture) decidingPlan(t *testing.T, question string) *agentports.ChapterPlan {
	t.Helper()

	earlier := f.turn(conversation.AssistantTurnOriginPerson, conversation.AssistantTurnStatusCompleted, "Late loads")
	f.chapter(earlier.ID, "Late loads on the Dallas lane", conversation.ChapterTitleModel, sequence(0))
	f.conversations.recent = []conversation.Message{
		{Role: conversation.RoleUser, Content: "Which loads are late on the Dallas lane?"},
		{Role: conversation.RoleAssistant, Content: "Three loads are late."},
	}
	turn := f.turn(conversation.AssistantTurnOriginPerson, conversation.AssistantTurnStatusRunning, question)
	plan, err := f.svc.Plan(t.Context(), f.request(turn.ID, agentports.ChapterPhaseOpened))
	require.NoError(t, err)
	require.True(t, plan.Decides)

	return plan
}

func (f *fixture) assertAnnounced(t *testing.T, times int) {
	t.Helper()

	require.Len(t, f.realtime.published, times)
	for _, published := range f.realtime.published {
		assert.Equal(t, agentports.ChaptersResource, published.Resource)
		assert.Equal(t, "updated", published.Action)
		assert.Equal(t, f.thread.ID, published.RecordID)
		assert.Equal(t, f.tenant.OrgID, published.OrganizationID)
		assert.Equal(t, f.tenant.BuID, published.BusinessUnitID)
		assert.Equal(t, f.tenant.UserID, published.AudienceUserID)
		assert.Equal(t, map[string]string{"threadId": f.thread.ID.String()}, published.Entity)
	}
}

func TestApply_ANewSubjectOpensAChapterWithTheModelsCleanTitle(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	plan := f.decidingPlan(t, "Now the Acme invoice dispute")

	outcome, err := f.svc.Apply(t.Context(), plan,
		answer(`{"newChapter":true,"title":"acme invoice dispute."}`))
	require.NoError(t, err)

	require.Len(t, f.chapters.created, 1)
	created := f.chapters.created[0]
	assert.Equal(t, "Acme invoice dispute", created.Title)
	assert.Equal(t, conversation.ChapterTitleModel, created.TitleSource)
	assert.Equal(t, plan.Turn.TurnID, created.TurnID)
	assert.Equal(t, f.thread.ID, created.ThreadID)
	assert.Equal(t, f.tenant.OrgID, created.OrganizationID)
	assert.Equal(t, f.tenant.BuID, created.BusinessUnitID)
	assert.Equal(t, &agentports.ChapterOutcome{ChapterID: created.ID, Opened: true}, outcome)
	f.assertAnnounced(t, 1)
}

func TestApply_AFollowUpStaysInItsChapter(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	plan := f.decidingPlan(t, "and the second one?")

	outcome, err := f.svc.Apply(t.Context(), plan, answer(`{"newChapter":false,"title":""}`))
	require.NoError(t, err)

	assert.Empty(t, f.chapters.created)
	assert.Equal(t, &agentports.ChapterOutcome{}, outcome)
	f.assertAnnounced(t, 0)
}

func TestApply_ANewChapterWithoutATitleIsNotOpened(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	plan := f.decidingPlan(t, "Now the Acme invoice dispute")

	_, err := f.svc.Apply(t.Context(), plan, answer(`{"newChapter":true,"title":"  **  "}`))
	require.NoError(t, err)

	assert.Empty(t, f.chapters.created)
}

func TestApply_TheFirstQuestionFallsBackToItsOwnWordsWithoutAnAnswer(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	plan := f.openingPlan(t, "why is load 1200 late?")

	outcome, err := f.svc.Apply(t.Context(), plan, nil)
	require.NoError(t, err)

	require.Len(t, f.chapters.created, 1)
	created := f.chapters.created[0]
	assert.Equal(t, "Why is load 1200 late?", created.Title)
	assert.Equal(t, conversation.ChapterTitleQuestion, created.TitleSource)
	assert.Equal(t,
		&agentports.ChapterOutcome{ChapterID: created.ID, Opened: true, Fallback: true}, outcome)
	f.assertAnnounced(t, 1)
}

func TestApply_TheFirstQuestionTakesTheModelsTitle(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	plan := f.openingPlan(t, "why is load 1200 late?")

	outcome, err := f.svc.Apply(t.Context(), plan, answer(`{"title":"Late load 1200"}`))
	require.NoError(t, err)

	require.Len(t, f.chapters.created, 1)
	assert.Equal(t, "Late load 1200", f.chapters.created[0].Title)
	assert.Equal(t, conversation.ChapterTitleModel, f.chapters.created[0].TitleSource)
	assert.False(t, outcome.Fallback)
	assert.True(t, outcome.Opened)
}

/*
A title is a claim about the conversation. One that names a load the person
never mentioned is a wrong label on the table of contents, so it is dropped
and the chapter keeps the person's own words, or is not opened at all.
*/
func TestApply_ATitleNamingANumberTheConversationLacksIsRefused(t *testing.T) {
	t.Parallel()

	t.Run("opening", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)
		plan := f.openingPlan(t, "why is load 1200 late")

		outcome, err := f.svc.Apply(t.Context(), plan, answer(`{"title":"Load 4471 delay"}`))
		require.NoError(t, err)

		require.Len(t, f.chapters.created, 1)
		assert.Equal(t, "Why is load 1200 late", f.chapters.created[0].Title)
		assert.Equal(t, conversation.ChapterTitleQuestion, f.chapters.created[0].TitleSource)
		assert.True(t, outcome.Fallback)
	})

	t.Run("deciding", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)
		plan := f.decidingPlan(t, "why is load 1200 late")

		outcome, err := f.svc.Apply(t.Context(), plan,
			answer(`{"newChapter":true,"title":"Load 4471 delay"}`))
		require.NoError(t, err)

		assert.Empty(t, f.chapters.created)
		assert.False(t, outcome.Opened)
		f.assertAnnounced(t, 0)
	})
}

func TestApply_AnAnswerThatCannotBeReadCountsAsNone(t *testing.T) {
	t.Parallel()

	t.Run("opening", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)
		plan := f.openingPlan(t, "Detention policy setup")

		outcome, err := f.svc.Apply(t.Context(), plan, answer("Sure! The title is Detention."))
		require.NoError(t, err)

		require.Len(t, f.chapters.created, 1)
		assert.Equal(t, "Detention policy setup", f.chapters.created[0].Title)
		assert.True(t, outcome.Fallback)
	})

	t.Run("deciding", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)
		plan := f.decidingPlan(t, "Now the Acme invoice dispute")

		outcome, err := f.svc.Apply(t.Context(), plan, answer(`{"newChapter":tru`))
		require.NoError(t, err)

		assert.Empty(t, f.chapters.created)
		assert.Equal(t, &agentports.ChapterOutcome{}, outcome)
	})
}

func TestApply_ASkippedPlanChangesNothing(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	plan := &agentports.ChapterPlan{Turn: *f.request("atrn_a", agentports.ChapterPhaseOpened), Skip: skipNoTurn}

	outcome, err := f.svc.Apply(t.Context(), plan, answer(`{"title":"Late loads"}`))
	require.NoError(t, err)

	assert.Equal(t, &agentports.ChapterOutcome{}, outcome)
	assert.Empty(t, f.chapters.created)
	f.assertAnnounced(t, 0)
}

func TestApply_RemovesAnEmptyChapterButNeverOneThePersonNamed(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	turn := f.turn(conversation.AssistantTurnOriginPerson, conversation.AssistantTurnStatusStopped, "Late loads")
	chapter := f.chapter(turn.ID, "Late loads", conversation.ChapterTitleQuestion, nil)
	plan, err := f.svc.Plan(t.Context(), f.request(turn.ID, agentports.ChapterPhaseAnswered))
	require.NoError(t, err)

	outcome, err := f.svc.Apply(t.Context(), plan, nil)
	require.NoError(t, err)

	assert.Equal(t, &agentports.ChapterOutcome{ChapterID: chapter.ID, Removed: true}, outcome)
	require.Len(t, f.chapters.deleted, 1)
	assert.Equal(t, chapter.ID, f.chapters.deleted[0].ID)
	assert.Equal(t, f.thread.ID, f.chapters.deleted[0].ThreadID)
	assert.True(t, f.chapters.deleted[0].KeepPerson)
	assert.Empty(t, f.chapters.chapters)
	f.assertAnnounced(t, 1)
}

func TestApply_ARemovalThatFindsNothingIsNotAnnounced(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	plan := &agentports.ChapterPlan{
		Turn:      *f.request("atrn_a", agentports.ChapterPhaseAnswered),
		ChapterID: "achp_gone",
		Remove:    true,
	}

	outcome, err := f.svc.Apply(t.Context(), plan, nil)
	require.NoError(t, err)

	assert.False(t, outcome.Removed)
	f.assertAnnounced(t, 0)
}

func TestApply_RenamesAnAnsweredChapterWithTheModelsTitle(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	question := "which loads are late on the Dallas lane?"
	turn := f.turn(conversation.AssistantTurnOriginPerson, conversation.AssistantTurnStatusCompleted, question)
	chapter := f.chapter(turn.ID, "Which loads are late on the Dallas lane?", conversation.ChapterTitleQuestion, sequence(0))
	f.conversations.byTurn[turn.ID] = []conversation.Message{
		{Role: conversation.RoleUser, Content: question},
		{Role: conversation.RoleAssistant, Content: "Three loads are late."},
	}
	plan, err := f.svc.Plan(t.Context(), f.request(turn.ID, agentports.ChapterPhaseAnswered))
	require.NoError(t, err)

	outcome, err := f.svc.Apply(t.Context(), plan, answer(`{"title":"late loads on the Dallas lane"}`))
	require.NoError(t, err)

	assert.Equal(t, &agentports.ChapterOutcome{ChapterID: chapter.ID, Retitled: true}, outcome)
	require.Len(t, f.chapters.retitled, 1)
	retitled := f.chapters.retitled[0]
	assert.Equal(t, chapter.ID, retitled.ID)
	assert.Equal(t, f.thread.ID, retitled.ThreadID)
	assert.Equal(t, "Late loads on the Dallas lane", retitled.Title)
	assert.Equal(t, conversation.ChapterTitleModel, retitled.TitleSource)
	assert.Nil(t, retitled.Version, "a model's rename never carries a person's version")
	f.assertAnnounced(t, 1)
}

func TestApply_ARenameWithoutAnAnswerKeepsTheTitle(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	turn := f.turn(conversation.AssistantTurnOriginPerson, conversation.AssistantTurnStatusCompleted, "late loads")
	chapter := f.chapter(turn.ID, "Late loads", conversation.ChapterTitleQuestion, sequence(0))
	f.conversations.byTurn[turn.ID] = []conversation.Message{
		{Role: conversation.RoleUser, Content: "late loads"},
	}
	plan, err := f.svc.Plan(t.Context(), f.request(turn.ID, agentports.ChapterPhaseAnswered))
	require.NoError(t, err)

	outcome, err := f.svc.Apply(t.Context(), plan, nil)
	require.NoError(t, err)

	assert.Equal(t, &agentports.ChapterOutcome{ChapterID: chapter.ID}, outcome)
	assert.Empty(t, f.chapters.retitled)
	f.assertAnnounced(t, 0)
}

func TestApply_ARenameOfAChapterSinceRemovedIsQuiet(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	plan := &agentports.ChapterPlan{
		Turn:        *f.request("atrn_a", agentports.ChapterPhaseAnswered),
		ChapterID:   "achp_gone",
		ModelSource: conversation.ChapterTitleModel,
		Evidence:    []string{"late loads"},
	}

	outcome, err := f.svc.Apply(t.Context(), plan, answer(`{"title":"Late loads"}`))
	require.NoError(t, err)

	assert.False(t, outcome.Retitled)
	f.assertAnnounced(t, 0)
}
