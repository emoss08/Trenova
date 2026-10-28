package chapterjobs

import (
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/chapterjobs/chaptercontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func newEnv(t *testing.T) *testsuite.TestWorkflowEnvironment {
	t.Helper()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	return env
}

func tenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.ID("org_a"), BuID: pulid.ID("bu_a")}
}

func input() *chaptercontract.ThreadChaptersInput {
	return &chaptercontract.ThreadChaptersInput{TenantInfo: tenant(), ThreadID: pulid.ID("athr_a")}
}

func turnRequest(turnID pulid.ID) agentports.ChapterTurnRequest {
	return agentports.ChapterTurnRequest{
		TenantInfo: tenant(),
		ThreadID:   pulid.ID("athr_a"),
		UserID:     pulid.ID("usr_a"),
		TurnID:     turnID,
		Phase:      agentports.ChapterPhaseOpened,
	}
}

func modelPlan(req agentports.ChapterTurnRequest) *agentports.ChapterPlan {
	return &agentports.ChapterPlan{
		Turn:           req,
		Opens:          true,
		Fallback:       "Why is load 1200 late?",
		FallbackSource: conversation.ChapterTitleQuestion,
		ModelSource:    conversation.ChapterTitleModel,
		Request: &agentports.StructuredCompletionRequest{
			TenantInfo: tenant(),
			SchemaName: "chapter_title",
		},
	}
}

func cue(env *testsuite.TestWorkflowEnvironment, req agentports.ChapterTurnRequest, after time.Duration) {
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(chaptercontract.TurnSignalName, req)
	}, after)
}

func TestThreadChaptersWorkflow_AsksTheModelAndKeepsWhatItSaid(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	var a *Activities
	req := turnRequest(pulid.ID("atrn_a"))
	cue(env, req, 0)

	env.OnActivity(a.PlanChapterActivity, mock.Anything, mock.MatchedBy(
		func(got *agentports.ChapterTurnRequest) bool { return *got == req },
	)).Return(modelPlan(req), nil).Once()
	env.OnActivity(a.ChapterModelActivity, mock.Anything, mock.MatchedBy(
		func(plan *agentports.ChapterPlan) bool {
			return plan.Turn.TurnID == req.TurnID && plan.Request != nil &&
				plan.Request.SchemaName == "chapter_title"
		},
	)).Return(&agentports.StructuredCompletionResult{Text: `{"title":"Late load 1200"}`}, nil).Once()
	env.OnActivity(a.ApplyChapterActivity, mock.Anything, mock.MatchedBy(
		func(in *ApplyInput) bool {
			return in.Plan != nil && in.Plan.Opens && in.Plan.Turn.TurnID == req.TurnID &&
				in.Result != nil && in.Result.Text == `{"title":"Late load 1200"}`
		},
	)).Return(&agentports.ChapterOutcome{ChapterID: pulid.ID("achp_a"), Opened: true}, nil).Once()

	env.ExecuteWorkflow(ThreadChaptersWorkflow, input())

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result ChaptersResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, ChaptersResult{Turns: 1, Opened: 1}, result)
	env.AssertExpectations(t)
}

func TestThreadChaptersWorkflow_ASkippedTurnAsksNothingAndChangesNothing(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	var a *Activities
	req := turnRequest(pulid.ID("atrn_a"))
	cue(env, req, 0)

	env.OnActivity(a.PlanChapterActivity, mock.Anything, mock.Anything).
		Return(&agentports.ChapterPlan{Turn: req, Skip: "the turn continues its chapter"}, nil).Once()
	env.OnActivity(a.ChapterModelActivity, mock.Anything, mock.Anything).Never()
	env.OnActivity(a.ApplyChapterActivity, mock.Anything, mock.Anything).Never()

	env.ExecuteWorkflow(ThreadChaptersWorkflow, input())

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result ChaptersResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, ChaptersResult{Turns: 1}, result)
	env.AssertActivityNumberOfCalls(t, "ChapterModelActivity", 0)
	env.AssertActivityNumberOfCalls(t, "ApplyChapterActivity", 0)
	env.AssertExpectations(t)
}

func TestThreadChaptersWorkflow_APlanWithoutAQuestionForTheModelGoesStraightToApply(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	var a *Activities
	req := turnRequest(pulid.ID("atrn_a"))
	req.Phase = agentports.ChapterPhaseAnswered
	cue(env, req, 0)

	plan := &agentports.ChapterPlan{Turn: req, ChapterID: pulid.ID("achp_a"), Remove: true}
	env.OnActivity(a.PlanChapterActivity, mock.Anything, mock.Anything).Return(plan, nil).Once()
	env.OnActivity(a.ChapterModelActivity, mock.Anything, mock.Anything).Never()
	env.OnActivity(a.ApplyChapterActivity, mock.Anything, mock.MatchedBy(
		func(in *ApplyInput) bool { return in.Plan.Remove && in.Result == nil },
	)).Return(&agentports.ChapterOutcome{ChapterID: pulid.ID("achp_a"), Removed: true}, nil).Once()

	env.ExecuteWorkflow(ThreadChaptersWorkflow, input())

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result ChaptersResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, ChaptersResult{Turns: 1, Removed: 1}, result)
	env.AssertExpectations(t)
}

/*
A model that does not answer must not cost the person their chapter: the
first question still opens one, named from its own words.
*/
func TestThreadChaptersWorkflow_AFailedModelCallStillKeepsTheChapter(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	var a *Activities
	req := turnRequest(pulid.ID("atrn_a"))
	cue(env, req, 0)

	env.OnActivity(a.PlanChapterActivity, mock.Anything, mock.Anything).Return(modelPlan(req), nil).Once()
	env.OnActivity(a.ChapterModelActivity, mock.Anything, mock.Anything).
		Return(nil, temporal.NewNonRetryableApplicationError("no provider", "ModelRefused", errors.New("no provider")))
	env.OnActivity(a.ApplyChapterActivity, mock.Anything, mock.MatchedBy(
		func(in *ApplyInput) bool {
			return in.Plan != nil && in.Plan.Turn.TurnID == req.TurnID && in.Result == nil
		},
	)).Return(&agentports.ChapterOutcome{ChapterID: pulid.ID("achp_a"), Opened: true, Fallback: true}, nil).Once()

	env.ExecuteWorkflow(ThreadChaptersWorkflow, input())

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result ChaptersResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, ChaptersResult{Turns: 1, Opened: 1, Fallbacks: 1}, result)
	env.AssertExpectations(t)
}

func TestThreadChaptersWorkflow_AFailedPlanIsCountedAndTheNextTurnStillRuns(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	var a *Activities
	first := turnRequest(pulid.ID("atrn_a"))
	second := turnRequest(pulid.ID("atrn_b"))
	cue(env, first, 0)
	cue(env, second, time.Minute)

	env.OnActivity(a.PlanChapterActivity, mock.Anything, mock.MatchedBy(
		func(got *agentports.ChapterTurnRequest) bool { return got.TurnID == first.TurnID },
	)).Return(nil, temporal.NewNonRetryableApplicationError("down", "Down", errors.New("down")))
	env.OnActivity(a.PlanChapterActivity, mock.Anything, mock.MatchedBy(
		func(got *agentports.ChapterTurnRequest) bool { return got.TurnID == second.TurnID },
	)).Return(&agentports.ChapterPlan{Turn: second, Skip: "the turn continues its chapter"}, nil).Once()

	env.ExecuteWorkflow(ThreadChaptersWorkflow, input())

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result ChaptersResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, ChaptersResult{Turns: 2, Failed: 1}, result)
}

func TestThreadChaptersWorkflow_EndsWhenTheConversationGoesQuiet(t *testing.T) {
	t.Parallel()

	env := newEnv(t)

	started := env.Now()
	env.ExecuteWorkflow(ThreadChaptersWorkflow, input())

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	assert.GreaterOrEqual(t, env.Now().Sub(started), chaptercontract.IdleWait)

	var result ChaptersResult
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, ChaptersResult{}, result)
}

func TestThreadChaptersWorkflow_ContinuesAsNewAfterItsTurnsPerRun(t *testing.T) {
	t.Parallel()

	env := newEnv(t)
	var a *Activities
	for idx := range chaptercontract.TurnsPerRun {
		cue(env, turnRequest(pulid.MustNew("atrn_")), time.Duration(idx)*time.Second)
	}

	env.OnActivity(a.PlanChapterActivity, mock.Anything, mock.Anything).
		Return(&agentports.ChapterPlan{Skip: "the turn continues its chapter"}, nil)

	env.ExecuteWorkflow(ThreadChaptersWorkflow, input())

	require.True(t, env.IsWorkflowCompleted())
	assert.True(t, workflow.IsContinueAsNewError(env.GetWorkflowError()),
		"a long conversation starts a fresh history rather than growing one forever")
	env.AssertActivityNumberOfCalls(t, "PlanChapterActivity", chaptercontract.TurnsPerRun)
}
