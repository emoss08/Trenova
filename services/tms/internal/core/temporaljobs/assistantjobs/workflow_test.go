package assistantjobs

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentguard"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/emoss08/trenova/internal/core/services/assistantservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
	"github.com/emoss08/trenova/internal/core/temporaljobs/modelcall"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

type AssistantTurnWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env      *testsuite.TestWorkflowEnvironment
	runtime  *agentruntime.Service
	payload  *AssistantTurnPayload
	finished *FinishTurnInput
	notified []*NotifyUnseenTurnInput
}

func TestAssistantTurnWorkflow(t *testing.T) {
	suite.Run(t, new(AssistantTurnWorkflowTestSuite))
}

func (s *AssistantTurnWorkflowTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
	s.runtime = agentruntime.New(agentruntime.Params{
		Logger:      zap.NewNop(),
		Completion:  &agentruntimetest.ScriptedCompletion{},
		QueryTools:  &agentruntimetest.StubQueryRegistry{},
		ActionTools: &agentruntimetest.StubActionRegistry{},
		Permissions: &agentruntimetest.StubPermissions{},
	})
	s.payload = &AssistantTurnPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
		},
		TurnID:   pulid.MustNew("atrn_"),
		ThreadID: pulid.MustNew("athr_"),
		Content:  "How many loads are late?",
		Actor: serviceports.RequestActor{
			PrincipalType:  serviceports.PrincipalTypeUser,
			PrincipalID:    pulid.MustNew("usr_"),
			UserID:         pulid.MustNew("usr_"),
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
		},
	}
	s.finished = nil
	s.notified = nil

	s.env.RegisterActivity(&Activities{})
	s.env.RegisterActivity(&agentflow.Activities{})
	s.env.RegisterWorkflowWithOptions(
		NewWorkflows(s.runtime).AssistantTurnWorkflow,
		workflow.RegisterOptions{Name: AssistantTurnWorkflowName},
	)
}

// notifies records each time the turn tells the person their reply is ready.
// Every test gets it: a turn nobody drains is a turn nobody watched.
func (s *AssistantTurnWorkflowTestSuite) notifies(err error) {
	var a *Activities
	s.env.OnActivity(a.NotifyUnseenTurnActivity, mock.Anything, mock.Anything).Return(
		func(_ context.Context, in *NotifyUnseenTurnInput) error {
			s.notified = append(s.notified, in)

			return err
		},
	).Maybe()
}

func (s *AssistantTurnWorkflowTestSuite) AfterTest(_, _ string) {
	s.env.AssertExpectations(s.T())
}

// plan is a question the guard let through, opened as a real runtime turn.
func (s *AssistantTurnWorkflowTestSuite) plan() *assistantservice.TurnPlan {
	definition := &agentdefinition.Definition{
		Name:            "Dispatch helper",
		Instructions:    "Help dispatch.",
		AutonomyCeiling: agent.TierPropose,
		Enabled:         true,
	}
	definition.ApplyDefaults()

	plan := &assistantservice.TurnPlan{
		ThreadID:   s.payload.ThreadID,
		Definition: definition,
		Input:      s.payload.Content,
		Decision:   agentguard.Decision{Allowed: true},
	}
	plan.Turn = s.runtime.OpenTurn(s.T().Context(), plan.RunRequest(&s.payload.Actor)).State()

	return plan
}

func (s *AssistantTurnWorkflowTestSuite) prepares(plan *assistantservice.TurnPlan, err error) {
	var a *Activities
	s.env.OnActivity(a.PrepareTurnActivity, mock.Anything, mock.Anything).Return(plan, err).Once()
}

func (s *AssistantTurnWorkflowTestSuite) answers(text string) {
	var fa *agentflow.Activities
	s.env.OnActivity(fa.ModelCallActivity, mock.Anything, mock.Anything).Return(
		&agentruntime.ModelReply{Completion: &serviceports.ChatCompletionResult{
			Text:            text,
			ModelIdentifier: "test-model",
		}}, nil,
	).Once()
}

// finishes records what the turn handed over for saving and ends the turn
// the way the real activity would for it.
func (s *AssistantTurnWorkflowTestSuite) finishes(ending *TurnEnding, err error) {
	var a *Activities
	s.env.OnActivity(a.FinishTurnActivity, mock.Anything, mock.Anything).Return(
		func(_ context.Context, in *FinishTurnInput) (*TurnEnding, error) {
			s.finished = in

			return ending, err
		},
	)
}

func doneEnding() *TurnEnding {
	return &TurnEnding{
		Result: AssistantTurnResult{Status: "Completed"},
		Event:  temporaltype.StreamItem{Event: serviceports.AssistantEventDone},
	}
}

func (s *AssistantTurnWorkflowTestSuite) run() {
	s.notifies(nil)
	s.env.ExecuteWorkflow(AssistantTurnWorkflowName, s.payload)
	s.True(s.env.IsWorkflowCompleted())
}

func (s *AssistantTurnWorkflowTestSuite) TestAnswersTheQuestionAndSavesIt() {
	s.prepares(s.plan(), nil)
	s.answers("Twelve loads are late.")
	s.finishes(doneEnding(), nil)

	s.run()

	s.NoError(s.env.GetWorkflowError())
	var result AssistantTurnResult
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal("Completed", result.Status)

	s.Require().NotNil(s.finished)
	s.Nil(s.finished.Failure)
	s.Require().NotNil(s.finished.Run)
	s.Equal("Twelve loads are late.", s.finished.Run.Reply)
	s.Require().NotEmpty(s.finished.Events)
	s.Equal(serviceports.AssistantEventAccepted, s.finished.Events[0].Event,
		"the opening is kept in the turn's account")
}

// A refusal is an answer. The model is never asked, and the refusal is saved.
func (s *AssistantTurnWorkflowTestSuite) TestSavesARefusalWithoutAskingTheModel() {
	plan := s.plan()
	plan.Decision = agentguard.Decision{Allowed: false, Message: "I only help with freight."}
	plan.Turn = agentruntime.TurnState{}
	s.prepares(plan, nil)
	s.finishes(doneEnding(), nil)

	s.run()

	s.NoError(s.env.GetWorkflowError())
	s.Require().NotNil(s.finished)
	s.Nil(s.finished.Run)
	s.Equal(serviceports.AssistantEventRefused, s.finished.Events[0].Event)
}

// A question turned away for a reason written for the person keeps that
// reason, so the reader is told it rather than a generic failure.
func (s *AssistantTurnWorkflowTestSuite) TestPassesOnWhyAQuestionWasTurnedAway() {
	s.prepares(nil, temporal.NewNonRetryableApplicationError(
		"This conversation is full. Start a new one.", errTypeRejected, nil,
	))
	s.finishes(&TurnEnding{Result: AssistantTurnResult{Status: "Failed"}}, nil)

	s.run()

	s.NoError(s.env.GetWorkflowError())
	s.Require().NotNil(s.finished)
	s.Nil(s.finished.Plan)
	s.Equal("This conversation is full. Start a new one.", s.finished.Rejection)
}

// A model that fails for good still leaves a saved turn, and the saved turn
// still knows the provider refused the request rather than being unreachable.
func (s *AssistantTurnWorkflowTestSuite) TestSavesATurnTheModelCouldNotFinish() {
	s.prepares(s.plan(), nil)
	var fa *agentflow.Activities
	s.env.OnActivity(fa.ModelCallActivity, mock.Anything, mock.Anything).Return(
		nil, temporal.NewApplicationErrorWithOptions("bad request", modelcall.ErrTypeModelRejected,
			temporal.ApplicationErrorOptions{
				NonRetryable: true,
				Details:      []any{map[string]any{"status": http.StatusBadRequest}},
			}),
	).Once()
	s.finishes(&TurnEnding{Result: AssistantTurnResult{Status: "Failed"}}, nil)

	s.run()

	s.NoError(s.env.GetWorkflowError())
	s.Require().NotNil(s.finished)
	s.Require().NotNil(s.finished.Failure)
	s.False(s.finished.Failure.Stopped)
	s.Equal(http.StatusBadRequest, s.finished.Failure.Status)
}

// Stopping cancels the model call in flight, and what the turn had done by
// then is still saved: the save runs where the stop cannot reach it.
func (s *AssistantTurnWorkflowTestSuite) TestSavesAStoppedTurn() {
	s.prepares(s.plan(), nil)
	var fa *agentflow.Activities
	s.env.OnActivity(fa.ModelCallActivity, mock.Anything, mock.Anything).
		After(time.Minute).
		Return(&agentruntime.ModelReply{}, nil).
		Maybe()
	s.finishes(&TurnEnding{Result: AssistantTurnResult{Status: "Stopped"}}, nil)
	s.env.RegisterDelayedCallback(s.env.CancelWorkflow, time.Second)

	s.run()

	s.True(temporal.IsCanceledError(s.env.GetWorkflowError()),
		"a stopped turn is recorded as cancelled")
	s.Require().NotNil(s.finished)
	s.Require().NotNil(s.finished.Failure)
	s.True(s.finished.Failure.Stopped)
}

// A turn whose reader has the last event closes without waiting out the
// drain window.
func (s *AssistantTurnWorkflowTestSuite) TestClosesAsSoonAsItsReaderIsDone() {
	s.prepares(s.plan(), nil)
	s.answers("Done.")
	s.finishes(doneEnding(), nil)

	var closedAt time.Time
	s.env.RegisterDelayedCallback(func() {
		s.env.SignalWorkflow(temporaltype.SignalStreamDrained, nil)
	}, 0)
	started := s.env.Now()

	s.run()
	closedAt = s.env.Now()

	s.NoError(s.env.GetWorkflowError())
	s.Less(closedAt.Sub(started), agentflow.DrainWindow)
}

func (s *AssistantTurnWorkflowTestSuite) TestFailsLoudlyWhenTheTurnCannotBeSaved() {
	s.prepares(s.plan(), nil)
	s.answers("Done.")
	s.finishes(nil, errors.New("database is down"))
	closed := ""
	var a *Activities
	s.env.OnActivity(a.CloseTurnActivity, mock.Anything, mock.Anything, mock.Anything).Return(
		func(_ context.Context, payload *AssistantTurnPayload, message string) error {
			closed = message
			s.Equal(s.payload.TurnID, payload.TurnID)

			return nil
		},
	).Once()

	s.run()

	s.Error(s.env.GetWorkflowError())
	s.Contains(closed, "database is down",
		"a turn that could not be saved still closes its record, or the conversation stays locked")
}

func answeredEnding(reply string) *TurnEnding {
	return &TurnEnding{
		Result: AssistantTurnResult{
			Status: "Completed",
			Result: &serviceports.SendMessageResult{Reply: reply},
		},
		Event: temporaltype.StreamItem{Event: serviceports.AssistantEventDone},
	}
}

func (s *AssistantTurnWorkflowTestSuite) drainsImmediately() {
	s.env.RegisterDelayedCallback(func() {
		s.env.SignalWorkflow(temporaltype.SignalStreamDrained, nil)
	}, 0)
}

// Nobody read the end of the reply, so the person who asked is told it is
// ready, with the start of it.
func (s *AssistantTurnWorkflowTestSuite) TestTellsThePersonWhenNobodySawTheReplyEnd() {
	s.prepares(s.plan(), nil)
	s.answers("Twelve loads are late.")
	s.finishes(answeredEnding("Twelve   loads\nare late."), nil)

	s.run()

	s.NoError(s.env.GetWorkflowError())
	s.Require().Len(s.notified, 1)
	s.Equal(s.payload.TurnID, s.notified[0].Payload.TurnID)
	s.Equal(s.payload.ThreadID, s.notified[0].Payload.ThreadID)
	s.Equal(conversation.AssistantTurnStatusCompleted, s.notified[0].Status)
}

// A reader who had the last event saw the reply end. Telling them again is
// noise.
func (s *AssistantTurnWorkflowTestSuite) TestSaysNothingWhenTheReaderSawTheEnd() {
	s.prepares(s.plan(), nil)
	s.answers("Done.")
	s.finishes(answeredEnding("Done."), nil)
	s.drainsImmediately()

	s.run()

	s.NoError(s.env.GetWorkflowError())
	s.Empty(s.notified)
}

// A reply the person stopped is not news to them.
func (s *AssistantTurnWorkflowTestSuite) TestSaysNothingAboutAStoppedTurn() {
	s.prepares(s.plan(), nil)
	var fa *agentflow.Activities
	s.env.OnActivity(fa.ModelCallActivity, mock.Anything, mock.Anything).
		After(time.Minute).
		Return(&agentruntime.ModelReply{}, nil).
		Maybe()
	s.finishes(&TurnEnding{Result: AssistantTurnResult{Status: "Stopped"}}, nil)
	s.env.RegisterDelayedCallback(s.env.CancelWorkflow, time.Second)

	s.run()

	s.True(temporal.IsCanceledError(s.env.GetWorkflowError()))
	s.Empty(s.notified)
}

// A stopped turn whose save then failed ends Failed, but the person still
// stopped it themselves.
func (s *AssistantTurnWorkflowTestSuite) TestSaysNothingAboutAStoppedTurnThatCouldNotBeSaved() {
	s.prepares(s.plan(), nil)
	var fa *agentflow.Activities
	s.env.OnActivity(fa.ModelCallActivity, mock.Anything, mock.Anything).
		After(time.Minute).
		Return(&agentruntime.ModelReply{}, nil).
		Maybe()
	s.finishes(nil, errors.New("database is down"))
	var a *Activities
	s.env.OnActivity(a.CloseTurnActivity, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()
	s.env.RegisterDelayedCallback(s.env.CancelWorkflow, time.Second)

	s.run()

	s.Empty(s.notified)
}

// A caller waiting on the result has the reply the moment the turn ends.
func (s *AssistantTurnWorkflowTestSuite) TestSaysNothingToACallerWaitingOnTheResult() {
	s.payload.Request.Awaited = true
	s.prepares(s.plan(), nil)
	s.answers("Done.")
	s.finishes(answeredEnding("Done."), nil)

	s.run()

	s.NoError(s.env.GetWorkflowError())
	s.Empty(s.notified)
}

// A reply that failed with nobody watching is worth knowing about too: the
// person is waiting for something that is not coming.
func (s *AssistantTurnWorkflowTestSuite) TestTellsThePersonAReplyCouldNotFinish() {
	s.prepares(s.plan(), nil)
	s.answers("Done.")
	s.finishes(nil, errors.New("database is down"))
	var a *Activities
	s.env.OnActivity(a.CloseTurnActivity, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()

	s.run()

	s.Error(s.env.GetWorkflowError())
	s.Require().Len(s.notified, 1)
	s.Equal(conversation.AssistantTurnStatusFailed, s.notified[0].Status)
}

// A refusal is an answer, and the person is told it is there.
func (s *AssistantTurnWorkflowTestSuite) TestTellsThePersonAboutARefusal() {
	plan := s.plan()
	plan.Decision = agentguard.Decision{Allowed: false, Message: "I only help with freight."}
	plan.Turn = agentruntime.TurnState{}
	s.prepares(plan, nil)
	s.finishes(&TurnEnding{
		Result: AssistantTurnResult{
			Status:  "Refused",
			Refused: true,
			Result: &serviceports.SendMessageResult{
				Reply:   "I only help with freight.",
				Refused: true,
			},
		},
		Event: temporaltype.StreamItem{Event: serviceports.AssistantEventDone},
	}, nil)

	s.run()

	s.Require().Len(s.notified, 1)
	s.Equal(conversation.AssistantTurnStatusRefused, s.notified[0].Status)
}

// Telling the person is best effort: the reply is saved either way.
func (s *AssistantTurnWorkflowTestSuite) TestAReplyNobodyCouldBeToldAboutStillCompletes() {
	s.notifies(errors.New("notifications are down"))
	s.prepares(s.plan(), nil)
	s.answers("Done.")
	s.finishes(answeredEnding("Done."), nil)

	s.run()

	s.NoError(s.env.GetWorkflowError())
	var result AssistantTurnResult
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal("Completed", result.Status)
	s.NotEmpty(s.notified)
}

// An execution that began before the step existed replays without it.
func (s *AssistantTurnWorkflowTestSuite) TestAnExecutionFromBeforeTheChangeSaysNothing() {
	s.env.OnGetVersion(changeNotifyUnseenTurn, workflow.DefaultVersion, 1).
		Return(workflow.DefaultVersion)
	s.prepares(s.plan(), nil)
	s.answers("Done.")
	s.finishes(answeredEnding("Done."), nil)

	s.run()

	s.NoError(s.env.GetWorkflowError())
	s.Empty(s.notified)
}
