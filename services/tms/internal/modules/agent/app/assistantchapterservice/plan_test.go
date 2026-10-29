package assistantchapterservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aireasoning"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sectionTitles(req *llmports.StructuredCompletionRequest) []string {
	titles := make([]string, 0, len(req.Context.Sections))
	for _, section := range req.Context.Sections {
		titles = append(titles, section.Title)
	}

	return titles
}

func TestPlan_TheFirstQuestionOpensAQuietChapterWithoutAskingTheModel(t *testing.T) {
	t.Parallel()

	for name, off := range map[string]bool{"naming on": false, "naming off": true} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			f := newFixture(t)
			f.control.AutoChaptersOff = off
			question := "why is load 1200 late?"
			turn := f.turn(
				conversation.AssistantTurnOriginPerson,
				conversation.AssistantTurnStatusRunning,
				question,
			)

			plan, err := f.svc.Plan(
				t.Context(),
				f.request(turn.ID, assistantports.ChapterPhaseOpened),
			)
			require.NoError(t, err)

			assert.False(t, plan.Skipped())
			assert.True(t, plan.Opens)
			assert.False(t, plan.Decides)
			assert.Equal(t, "Why is load 1200 late?", plan.Fallback)
			assert.Equal(t, conversation.ChapterTitleQuestion, plan.FallbackSource)
			assert.Nil(t, plan.Request)
			assert.False(t, plan.NeedsModel())
			assert.True(t, plan.NameEarlier.IsNil())
		})
	}
}

func TestPlan_AutomaticChaptersOffSkipsALaterQuestion(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.control.AutoChaptersOff = true
	earlier := f.turn(
		conversation.AssistantTurnOriginPerson,
		conversation.AssistantTurnStatusCompleted,
		"Late loads",
	)
	f.chapter(earlier.ID, "Late loads", conversation.ChapterTitleQuestion, sequence(0))
	turn := f.turn(
		conversation.AssistantTurnOriginPerson,
		conversation.AssistantTurnStatusRunning,
		"Where is load 1200?",
	)

	plan, err := f.svc.Plan(t.Context(), f.request(turn.ID, assistantports.ChapterPhaseOpened))
	require.NoError(t, err)

	assert.Equal(t, skipAutoOff, plan.Skip)
	assert.Nil(t, plan.Request)
	assert.False(t, plan.Opens)
}

func TestPlan_ANewChapterNamesTheQuietFirstOneWhenItOpens(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	earlier := f.turn(
		conversation.AssistantTurnOriginPerson,
		conversation.AssistantTurnStatusCompleted,
		"Late loads",
	)
	f.chapter(earlier.ID, "Late loads", conversation.ChapterTitleQuestion, sequence(0))
	turn := f.turn(
		conversation.AssistantTurnOriginPerson,
		conversation.AssistantTurnStatusRunning,
		"Now the Acme invoice",
	)

	plan, err := f.svc.Plan(t.Context(), f.request(turn.ID, assistantports.ChapterPhaseOpened))
	require.NoError(t, err)

	assert.True(t, plan.Decides)
	assert.Equal(t, earlier.ID, plan.NameEarlier)

	named := newFixture(t)
	first := named.turn(
		conversation.AssistantTurnOriginPerson,
		conversation.AssistantTurnStatusCompleted,
		"Late loads",
	)
	named.chapter(
		first.ID,
		"Late loads on the Dallas lane",
		conversation.ChapterTitleModel,
		sequence(0),
	)
	next := named.turn(
		conversation.AssistantTurnOriginPerson,
		conversation.AssistantTurnStatusRunning,
		"Now the Acme invoice",
	)

	again, err := named.svc.Plan(
		t.Context(),
		named.request(next.ID, assistantports.ChapterPhaseOpened),
	)
	require.NoError(t, err)

	assert.True(t, again.NameEarlier.IsNil())
}

func TestPlan_ALaterQuestionAsksWhetherItStartsANewChapter(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	earlier := f.turn(
		conversation.AssistantTurnOriginPerson,
		conversation.AssistantTurnStatusCompleted,
		"Late loads",
	)
	f.chapter(
		earlier.ID,
		"Late loads on the Dallas lane",
		conversation.ChapterTitleModel,
		sequence(0),
	)
	f.conversations.recent = []conversation.Message{
		{Role: conversation.RoleUser, Content: "Which loads are late on the Dallas lane?"},
		{Role: conversation.RoleAssistant, Content: "Three loads are late."},
		{Role: conversation.RoleTool, Content: "{\"rows\":3}"},
	}
	question := "Now open the Acme invoice dispute"
	turn := f.turn(
		conversation.AssistantTurnOriginPerson,
		conversation.AssistantTurnStatusRunning,
		question,
	)

	plan, err := f.svc.Plan(t.Context(), f.request(turn.ID, assistantports.ChapterPhaseOpened))
	require.NoError(t, err)

	assert.True(t, plan.Decides)
	assert.False(t, plan.Opens)
	assert.Equal(t, conversation.ChapterTitleModel, plan.ModelSource)
	assert.Equal(t, []string{
		question,
		"Which loads are late on the Dallas lane?",
		"Three loads are late.",
		"{\"rows\":3}",
	}, plan.Evidence)

	req := plan.Request
	require.NotNil(t, req)
	assert.Equal(t, "chapter_decision", req.SchemaName)
	assert.Equal(t, aitask.TaskChapterTitling, req.Task)
	assert.Equal(t, aireasoning.ReasoningOff, req.Reasoning)
	assert.Equal(t, f.tenant, req.TenantInfo)
	assert.Equal(t, aiusage.FeatureConversationChapters, req.Attribution.Feature)
	assert.Equal(t, agentruntimeports.RunStepOwnerAssistantTurn, req.Attribution.OwnerKind)
	assert.Equal(t, turn.ID, req.Attribution.OwnerID)
	assert.Equal(t, f.thread.ID, req.Attribution.ThreadID)
	assert.Equal(t, f.tenant.UserID, req.Attribution.UserID)
	assert.Equal(t, []string{
		"Current chapter title",
		"The conversation just before",
		"The person's new message",
	}, sectionTitles(req))
	assert.Equal(t, "Late loads on the Dallas lane", req.Context.Sections[0].Content)
	assert.Equal(t,
		"Person: Which loads are late on the Dallas lane?\nAgent: Three loads are late.",
		req.Context.Sections[1].Content)
	assert.Equal(t, question, req.Context.Sections[2].Content)

	require.Len(t, f.conversations.listed, 1)
	listed := f.conversations.listed[0]
	assert.Equal(t, recentMessages, listed.Limit)
	assert.True(t, listed.SinceCompaction)
	assert.Equal(t, conversation.ModelHiddenKinds(), listed.ExcludeKinds)
}

func TestPlan_TheCurrentChapterIsTheLatestOne(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	first := f.turn(
		conversation.AssistantTurnOriginPerson,
		conversation.AssistantTurnStatusCompleted,
		"a",
	)
	second := f.turn(
		conversation.AssistantTurnOriginPerson,
		conversation.AssistantTurnStatusCompleted,
		"b",
	)
	f.chapter(first.ID, "Late loads", conversation.ChapterTitleModel, sequence(0))
	f.chapter(second.ID, "Acme invoice dispute", conversation.ChapterTitleModel, sequence(4))
	turn := f.turn(
		conversation.AssistantTurnOriginPerson,
		conversation.AssistantTurnStatusRunning,
		"And the credit?",
	)

	plan, err := f.svc.Plan(t.Context(), f.request(turn.ID, assistantports.ChapterPhaseOpened))
	require.NoError(t, err)

	require.NotNil(t, plan.Request)
	assert.Equal(t, "Acme invoice dispute", plan.Request.Context.Sections[0].Content)
}

func TestPlan_TurnsThatContinueAChapterAreSkipped(t *testing.T) {
	t.Parallel()

	for _, origin := range []conversation.AssistantTurnOrigin{
		conversation.AssistantTurnOriginDecisionFollowUp,
		conversation.AssistantTurnOriginCompaction,
		conversation.AssistantTurnOriginWaitResolved,
	} {
		t.Run(string(origin), func(t *testing.T) {
			t.Parallel()

			f := newFixture(t)
			turn := f.turn(
				origin,
				conversation.AssistantTurnStatusRunning,
				"Approved the rate change",
			)

			plan, err := f.svc.Plan(
				t.Context(),
				f.request(turn.ID, assistantports.ChapterPhaseOpened),
			)
			require.NoError(t, err)

			assert.Equal(t, skipContinues, plan.Skip)
			assert.Nil(t, plan.Request)
			assert.False(t, plan.Opens)
			assert.False(t, plan.Decides)
		})
	}
}

func TestPlan_AScheduledRunAlwaysOpensAChapter(t *testing.T) {
	t.Parallel()

	question := "Send me the loads delivered today"

	t.Run("automatic chapters on", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)
		earlier := f.turn(
			conversation.AssistantTurnOriginPerson,
			conversation.AssistantTurnStatusCompleted,
			"a",
		)
		f.chapter(earlier.ID, "Late loads", conversation.ChapterTitleModel, sequence(0))
		turn := f.turn(
			conversation.AssistantTurnOriginScheduled,
			conversation.AssistantTurnStatusRunning,
			question,
		)

		plan, err := f.svc.Plan(t.Context(), f.request(turn.ID, assistantports.ChapterPhaseOpened))
		require.NoError(t, err)

		assert.True(t, plan.Opens, "a scheduled run opens a chapter even after others")
		assert.False(t, plan.Decides)
		assert.Equal(t, "Send me the loads delivered today", plan.Fallback)
		assert.Equal(t, conversation.ChapterTitleSchedule, plan.FallbackSource)
		assert.Equal(t, conversation.ChapterTitleSchedule, plan.ModelSource)
		require.NotNil(t, plan.Request)
		assert.Equal(t, "chapter_title", plan.Request.SchemaName)
	})

	t.Run("automatic chapters off", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)
		f.control.AutoChaptersOff = true
		turn := f.turn(
			conversation.AssistantTurnOriginScheduled,
			conversation.AssistantTurnStatusRunning,
			question,
		)

		plan, err := f.svc.Plan(t.Context(), f.request(turn.ID, assistantports.ChapterPhaseOpened))
		require.NoError(t, err)

		assert.False(t, plan.Skipped())
		assert.True(t, plan.Opens)
		assert.Equal(t, conversation.ChapterTitleSchedule, plan.FallbackSource)
		assert.Nil(t, plan.Request, "no model is asked while automatic chapters are off")
		assert.False(t, plan.NeedsModel())
	})
}

func TestPlan_OpenedSkipsWhatCannotOpenAChapter(t *testing.T) {
	t.Parallel()

	t.Run("a turn that is gone", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)

		plan, err := f.svc.Plan(
			t.Context(),
			f.request("atrn_gone", assistantports.ChapterPhaseOpened),
		)
		require.NoError(t, err)
		assert.Equal(t, skipNoTurn, plan.Skip)
	})

	t.Run("a turn that already opens one", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)
		turn := f.turn(
			conversation.AssistantTurnOriginPerson,
			conversation.AssistantTurnStatusRunning,
			"Late loads",
		)
		f.chapter(turn.ID, "Late loads", conversation.ChapterTitleQuestion, nil)

		plan, err := f.svc.Plan(t.Context(), f.request(turn.ID, assistantports.ChapterPhaseOpened))
		require.NoError(t, err)
		assert.Equal(t, skipAlreadyOpen, plan.Skip)
	})

	t.Run("a blank question", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)
		turn := f.turn(
			conversation.AssistantTurnOriginPerson,
			conversation.AssistantTurnStatusRunning,
			"  \n ",
		)

		plan, err := f.svc.Plan(t.Context(), f.request(turn.ID, assistantports.ChapterPhaseOpened))
		require.NoError(t, err)
		assert.Equal(t, skipNoQuestion, plan.Skip)
	})

	t.Run("a conversation with as many chapters as it may hold", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)
		for idx := range conversation.MaxChaptersPerThread {
			earlier := f.turn(
				conversation.AssistantTurnOriginPerson,
				conversation.AssistantTurnStatusCompleted,
				"a",
			)
			f.chapter(earlier.ID, "Chapter", conversation.ChapterTitleModel, sequence(int64(idx)))
		}
		turn := f.turn(
			conversation.AssistantTurnOriginPerson,
			conversation.AssistantTurnStatusRunning,
			"One more",
		)

		plan, err := f.svc.Plan(t.Context(), f.request(turn.ID, assistantports.ChapterPhaseOpened))
		require.NoError(t, err)
		assert.Equal(t, skipLimit, plan.Skip)
	})
}

func TestPlan_AChapterWhoseTurnSavedNothingIsRemoved(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	turn := f.turn(
		conversation.AssistantTurnOriginPerson,
		conversation.AssistantTurnStatusStopped,
		"Late loads",
	)
	chapter := f.chapter(turn.ID, "Late loads", conversation.ChapterTitleQuestion, nil)

	plan, err := f.svc.Plan(t.Context(), f.request(turn.ID, assistantports.ChapterPhaseAnswered))
	require.NoError(t, err)

	assert.False(t, plan.Skipped())
	assert.True(t, plan.Remove)
	assert.Equal(t, chapter.ID, plan.ChapterID)
	assert.Nil(t, plan.Request)
}

func TestPlan_ARefusedQuestionsChapterIsRemoved(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	turn := f.turn(
		conversation.AssistantTurnOriginPerson,
		conversation.AssistantTurnStatusRefused,
		"Tell me a joke",
	)
	chapter := f.chapter(turn.ID, "Tell me a joke", conversation.ChapterTitleModel, sequence(3))

	plan, err := f.svc.Plan(t.Context(), f.request(turn.ID, assistantports.ChapterPhaseAnswered))
	require.NoError(t, err)

	assert.True(t, plan.Remove)
	assert.Equal(t, chapter.ID, plan.ChapterID)
}

func TestPlan_AChapterThePersonNamedIsNeverRemovedOrRenamed(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)
		turn := f.turn(
			conversation.AssistantTurnOriginPerson,
			conversation.AssistantTurnStatusRefused,
			"Late loads",
		)
		f.chapter(turn.ID, "My loads", conversation.ChapterTitlePerson, nil)

		plan, err := f.svc.Plan(
			t.Context(),
			f.request(turn.ID, assistantports.ChapterPhaseAnswered),
		)
		require.NoError(t, err)

		assert.Equal(t, skipNamed, plan.Skip)
		assert.False(t, plan.Remove)
	})

	t.Run("answered", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)
		turn := f.turn(
			conversation.AssistantTurnOriginPerson,
			conversation.AssistantTurnStatusCompleted,
			"Late loads",
		)
		f.chapter(turn.ID, "My loads", conversation.ChapterTitlePerson, sequence(0))

		plan, err := f.svc.Plan(
			t.Context(),
			f.request(turn.ID, assistantports.ChapterPhaseAnswered),
		)
		require.NoError(t, err)

		assert.Equal(t, skipNamed, plan.Skip)
		assert.Nil(t, plan.Request)
		assert.Empty(t, f.conversations.listed, "nothing is read for a title that stays")
	})
}

func TestPlan_AnAnsweredQuestionsChapterWaitsToBeNamedUntilAnotherOpens(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	turn := f.turn(
		conversation.AssistantTurnOriginPerson,
		conversation.AssistantTurnStatusCompleted,
		"which loads are late?",
	)
	f.chapter(turn.ID, "Which loads are late?", conversation.ChapterTitleQuestion, sequence(0))

	plan, err := f.svc.Plan(t.Context(), f.request(turn.ID, assistantports.ChapterPhaseAnswered))
	require.NoError(t, err)

	assert.Equal(t, skipNamed, plan.Skip)
	assert.Nil(t, plan.Request)
}

func TestPlan_RequestedNamesAChapterFromItsQuestionAndReply(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	question := "which loads are late on the Dallas lane?"
	turn := f.turn(
		conversation.AssistantTurnOriginPerson,
		conversation.AssistantTurnStatusCompleted,
		question,
	)
	chapter := f.chapter(
		turn.ID,
		"Which loads are late on the Dallas lane?",
		conversation.ChapterTitleQuestion,
		sequence(0),
	)
	f.conversations.byTurn[turn.ID] = []conversation.Message{
		{Role: conversation.RoleUser, Content: question},
		{Role: conversation.RoleAssistant, Content: ""},
		{Role: conversation.RoleTool, Content: "{\"rows\":3}"},
		{Role: conversation.RoleAssistant, Content: "Three loads are late on the Dallas lane."},
	}

	plan, err := f.svc.Plan(t.Context(), f.request(turn.ID, assistantports.ChapterPhaseRequested))
	require.NoError(t, err)

	assert.False(t, plan.Skipped())
	assert.Equal(t, chapter.ID, plan.ChapterID)
	assert.False(t, plan.Opens)
	assert.False(t, plan.Decides)
	assert.False(t, plan.Remove)
	assert.Equal(t, conversation.ChapterTitleModel, plan.ModelSource)
	assert.Equal(t, []string{question, "Three loads are late on the Dallas lane."}, plan.Evidence)

	req := plan.Request
	require.NotNil(t, req)
	assert.Equal(t, "chapter_title", req.SchemaName)
	assert.Equal(t, []string{
		"The message that opens the chapter",
		"The start of the agent's reply",
	}, sectionTitles(req))
	assert.Equal(t, question, req.Context.Sections[0].Content)
	assert.Equal(t, "Three loads are late on the Dallas lane.", req.Context.Sections[1].Content)

	require.Len(t, f.conversations.listed, 1)
	assert.Equal(t, turn.ID, f.conversations.listed[0].TurnID)
	assert.Equal(t, conversation.ModelHiddenKinds(), f.conversations.listed[0].ExcludeKinds)
}

func TestPlan_AModelTitleIsRenamedOnlyAfterAShortQuestion(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		question string
		renamed  bool
	}{
		"a short question":         {question: "and Acme?", renamed: true},
		"a question of five words": {question: "what about the Acme invoices", renamed: false},
		"a question of many words": {question: strings.Repeat("word ", 12), renamed: false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			f := newFixture(t)
			turn := f.turn(
				conversation.AssistantTurnOriginPerson,
				conversation.AssistantTurnStatusCompleted,
				tc.question,
			)
			f.chapter(turn.ID, "Acme", conversation.ChapterTitleModel, sequence(0))
			f.conversations.byTurn[turn.ID] = []conversation.Message{
				{Role: conversation.RoleUser, Content: tc.question},
				{Role: conversation.RoleAssistant, Content: "Acme has two open invoices."},
			}

			plan, err := f.svc.Plan(
				t.Context(),
				f.request(turn.ID, assistantports.ChapterPhaseAnswered),
			)
			require.NoError(t, err)

			assert.Equal(t, tc.renamed, plan.Request != nil)
		})
	}
}

func TestPlan_AnUnfinishedReplyKeepsTheTitle(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	turn := f.turn(
		conversation.AssistantTurnOriginPerson,
		conversation.AssistantTurnStatusFailed,
		"Late loads",
	)
	f.chapter(turn.ID, "Late loads", conversation.ChapterTitleQuestion, sequence(0))

	plan, err := f.svc.Plan(t.Context(), f.request(turn.ID, assistantports.ChapterPhaseAnswered))
	require.NoError(t, err)

	assert.Equal(t, skipNamed, plan.Skip)
}

func TestPlan_AnsweredWithoutAChapterSkips(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	turn := f.turn(
		conversation.AssistantTurnOriginPerson,
		conversation.AssistantTurnStatusCompleted,
		"Late loads",
	)

	plan, err := f.svc.Plan(t.Context(), f.request(turn.ID, assistantports.ChapterPhaseAnswered))
	require.NoError(t, err)

	assert.Equal(t, skipNoChapter, plan.Skip)
}

func TestPlan_RequestedNamesAChapterOpenedFromItsQuestion(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	question := "which loads are late?"
	turn := f.turn(
		conversation.AssistantTurnOriginPerson,
		conversation.AssistantTurnStatusCompleted,
		question,
	)
	chapter := f.chapter(
		turn.ID,
		"Which loads are late?",
		conversation.ChapterTitleQuestion,
		sequence(0),
	)
	f.conversations.byTurn[turn.ID] = []conversation.Message{
		{Role: conversation.RoleUser, Content: question},
		{Role: conversation.RoleAssistant, Content: "Three are late."},
	}

	plan, err := f.svc.Plan(t.Context(), f.request(turn.ID, assistantports.ChapterPhaseRequested))
	require.NoError(t, err)

	assert.Equal(t, chapter.ID, plan.ChapterID)
	require.NotNil(t, plan.Request)
	assert.Equal(t, "chapter_title", plan.Request.SchemaName)

	f.control.AutoChaptersOff = true
	off, err := f.svc.Plan(t.Context(), f.request(turn.ID, assistantports.ChapterPhaseRequested))
	require.NoError(t, err)
	assert.Equal(t, skipNamed, off.Skip)
}

func TestPlan_RefusesAnUnknownPhase(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	turn := f.turn(
		conversation.AssistantTurnOriginPerson,
		conversation.AssistantTurnStatusCompleted,
		"Late loads",
	)

	_, err := f.svc.Plan(t.Context(), f.request(turn.ID, assistantports.ChapterPhase("Whenever")))

	require.Error(t, err)
}
