package assistantjobs

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/services/assistantservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

type CompactionWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env     *testsuite.TestWorkflowEnvironment
	payload *CompactionPayload
	ended   *CompactionEndInput
	saved   *SummarizedInput
}

func TestConversationCompactionWorkflow(t *testing.T) {
	suite.Run(t, new(CompactionWorkflowTestSuite))
}

func (s *CompactionWorkflowTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
	s.payload = &CompactionPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
		},
		TurnID:   pulid.MustNew("atrn_"),
		ThreadID: pulid.MustNew("athr_"),
		Actor: serviceports.RequestActor{
			PrincipalType: serviceports.PrincipalTypeUser,
			UserID:        pulid.MustNew("usr_"),
		},
		Auto: true,
	}
	s.ended = nil
	s.saved = nil

	s.env.RegisterActivity(&Activities{})
	s.env.RegisterActivity(&agentflow.Activities{})
	s.env.RegisterWorkflowWithOptions(
		NewWorkflows(nil).ConversationCompactionWorkflow,
		workflow.RegisterOptions{Name: CompactionWorkflowName},
	)
}

func (s *CompactionWorkflowTestSuite) AfterTest(_, _ string) {
	s.env.AssertExpectations(s.T())
}

func (s *CompactionWorkflowTestSuite) prepares(err error) {
	var a *Activities
	plan := &assistantservice.CompactionPlan{
		ThreadID: s.payload.ThreadID,
		Auto:     true,
		Through:  39,
		Before:   172_000,
		After:    31_000,
		Request:  &serviceports.ChatCompletionRequest{System: "Summarize."},
	}
	if err != nil {
		plan = nil
	}
	s.env.OnActivity(a.PrepareCompactionActivity, mock.Anything, mock.Anything).Return(plan, err).Once()
}

func (s *CompactionWorkflowTestSuite) ends() {
	var a *Activities
	s.env.OnActivity(a.EndCompactionActivity, mock.Anything, mock.Anything).Return(
		func(_ context.Context, in *CompactionEndInput) (*CompactionEnding, error) {
			s.ended = in
			ending := compactionEndingFor(in, in.Payload.Auto)

			return &ending, nil
		},
	).Once()
}

func (s *CompactionWorkflowTestSuite) run() {
	s.env.ExecuteWorkflow(CompactionWorkflowName, s.payload)
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError(), "every ending is told to the reader, not failed")
}

func (s *CompactionWorkflowTestSuite) TestSavesTheSummaryAndClosesTheTurn() {
	s.prepares(nil)
	var fa *agentflow.Activities
	s.env.OnActivity(fa.ModelCallActivity, mock.Anything, mock.Anything).Return(
		func(_ context.Context, in *agentflow.ModelCallInput) (*agentruntime.ModelReply, error) {
			s.False(in.Stream, "nobody reads the summary as it is written")
			s.Equal("Summarize.", in.Request.System)

			return &agentruntime.ModelReply{Completion: &serviceports.ChatCompletionResult{
				Text:            "- Customer 1 had 40 loads.",
				ModelIdentifier: "test-model",
			}}, nil
		},
	).Once()
	var a *Activities
	s.env.OnActivity(a.FinishCompactionActivity, mock.Anything, mock.Anything).Return(
		func(_ context.Context, in *SummarizedInput) (*CompactionEnding, error) {
			s.saved = in

			return &CompactionEnding{Event: temporaltype.StreamItem{
				Event: serviceports.AssistantEventCompactionFinished,
			}}, nil
		},
	).Once()

	s.run()

	s.Require().NotNil(s.saved)
	s.Equal(39, s.saved.Plan.Through)
	s.Equal("- Customer 1 had 40 loads.", s.saved.Reply.Completion.Text)
	s.Nil(s.ended, "a compaction that saved its summary has nothing else to close")
}

/*
Cancel stops the summarizing call and leaves the conversation as it was: no
summary is saved, the turn is closed as stopped, and a conversation that was
compacting itself stops doing so, or the next turn would start it again.
*/
func (s *CompactionWorkflowTestSuite) TestCancelLeavesTheConversationAsItWas() {
	s.prepares(nil)
	var fa *agentflow.Activities
	s.env.OnActivity(fa.ModelCallActivity, mock.Anything, mock.Anything).
		After(time.Minute).
		Return(&agentruntime.ModelReply{}, nil).
		Maybe()
	s.ends()
	s.env.RegisterDelayedCallback(s.env.CancelWorkflow, time.Second)

	s.run()

	s.Nil(s.saved, "nothing is saved once the person cancelled")
	s.Require().NotNil(s.ended)
	s.Equal(conversation.AssistantTurnStatusStopped, s.ended.Status)
	s.Equal(compactionCancelledMessage, s.ended.Message)
}

func (s *CompactionWorkflowTestSuite) TestPassesOnWhyACompactionWasTurnedAway() {
	s.prepares(temporal.NewNonRetryableApplicationError(
		"There is nothing to compact yet.", errTypeCompactionRejected, nil,
	))
	s.ends()

	s.run()

	s.Require().NotNil(s.ended)
	s.Equal(conversation.AssistantTurnStatusFailed, s.ended.Status)
	s.Equal("There is nothing to compact yet.", s.ended.Message)
}

func TestCompactionEndingFor_TellsTheReaderHowItEnded(t *testing.T) {
	t.Parallel()

	payload := &CompactionPayload{TurnID: pulid.MustNew("atrn_"), Auto: true}

	cancelled := compactionEndingFor(&CompactionEndInput{
		Payload: payload,
		Status:  conversation.AssistantTurnStatusStopped,
	}, true)
	assert.Equal(t, serviceports.AssistantEventCompactionCancelled, cancelled.Event.Event)
	data, ok := cancelled.Event.Data.(serviceports.AssistantCompactionEvent)
	assert.True(t, ok)
	assert.True(t, data.AutoCompactOff, "the reader is told the conversation stopped compacting itself")

	failed := compactionEndingFor(&CompactionEndInput{
		Payload: payload,
		Status:  conversation.AssistantTurnStatusFailed,
		Message: compactionFailedMessage,
	}, false)
	assert.Equal(t, serviceports.AssistantEventError, failed.Event.Event)
}

/*
A turn that leaves its conversation past the share cues a compaction; one that
leaves it below, fails, or answers in a conversation that does not compact
itself does not.
*/
func TestCompactionCue_IsSetOnlyByAnAnsweredTurnThatFilledTheConversation(t *testing.T) {
	t.Parallel()

	full := &conversation.ContextUsage{
		Instructions: 6000, Messages: 40_000, ToolResults: 130_000,
		Compactable: 150_000, Window: 200_000,
	}
	roomy := &conversation.ContextUsage{Instructions: 6000, Messages: 20_000, Compactable: 15_000, Window: 200_000}
	result := func(usage *conversation.ContextUsage, off bool) *serviceports.SendMessageResult {
		return &serviceports.SendMessageResult{Thread: &conversation.Thread{
			ContextUsage:   usage,
			AutoCompactOff: off,
		}}
	}

	cue := compactionCue(conversation.AssistantTurnStatusCompleted, result(full, false))
	if assert.NotNil(t, cue) {
		assert.Equal(t, full.Total(), cue.Before)
		assert.Equal(t, full.Total()-full.Frees(), cue.After)
	}

	assert.Nil(t, compactionCue(conversation.AssistantTurnStatusCompleted, result(roomy, false)))
	assert.Nil(t, compactionCue(conversation.AssistantTurnStatusCompleted, result(full, true)),
		"a conversation the person stopped compacting itself is left alone")
	assert.Nil(t, compactionCue(conversation.AssistantTurnStatusFailed, result(full, false)))
	assert.Nil(t, compactionCue(conversation.AssistantTurnStatusCompleted, result(nil, false)))
}

/*
A turn that leaves its conversation nearly full starts the conversation
compacting itself once its own record is closed; one that does not, starts
nothing.
*/
func (s *AssistantTurnWorkflowTestSuite) TestStartsCompactingAConversationTheTurnFilled() {
	s.prepares(s.plan(), nil)
	s.answers("Here are the 40 loads.")
	ending := doneEnding()
	ending.Compact = &CompactionCue{Before: 172_000, After: 31_000}
	s.finishes(ending, nil)
	started := false
	var a *Activities
	s.env.OnActivity(a.StartAutoCompactionActivity, mock.Anything, mock.Anything).Return(
		func(_ context.Context, payload *AssistantTurnPayload) (*StartedCompaction, error) {
			started = true
			s.Equal(s.payload.ThreadID, payload.ThreadID)

			return &StartedCompaction{TurnID: pulid.MustNew("atrn_")}, nil
		},
	).Once()

	s.run()

	s.NoError(s.env.GetWorkflowError())
	s.True(started)
}

func (s *AssistantTurnWorkflowTestSuite) TestLeavesARoomyConversationAlone() {
	s.prepares(s.plan(), nil)
	s.answers("Twelve loads are late.")
	s.finishes(doneEnding(), nil)

	s.run()

	s.NoError(s.env.GetWorkflowError())
	// AfterTest asserts no StartAutoCompactionActivity was expected or run.
}
