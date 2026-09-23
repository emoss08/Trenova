package assistantjobs

import (
	"context"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/assistantservice"
	"github.com/emoss08/trenova/internal/core/services/assistantturnservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/zap"
)

type turnRecords struct {
	mu        sync.Mutex
	turn      *conversation.AssistantTurn
	completed []repositories.CompleteAssistantTurnRequest
}

func (r *turnRecords) Start(
	_ context.Context,
	turn *conversation.AssistantTurn,
) (*conversation.AssistantTurn, error) {
	return turn, nil
}

func (r *turnRecords) GetByID(
	context.Context,
	repositories.GetAssistantTurnRequest,
) (*conversation.AssistantTurn, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.turn, nil
}

func (r *turnRecords) Active(
	context.Context,
	repositories.ActiveAssistantTurnRequest,
) (*conversation.AssistantTurn, error) {
	return nil, nil
}

func (r *turnRecords) ListLive(
	context.Context,
	repositories.ListLiveAssistantTurnsRequest,
) ([]*repositories.LiveAssistantTurn, error) {
	return nil, nil
}

func (r *turnRecords) Complete(
	_ context.Context,
	req repositories.CompleteAssistantTurnRequest,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.completed = append(r.completed, req)
	r.turn.Status = req.Status

	return nil
}

func (r *turnRecords) MarkWorkflow(context.Context, pulid.ID, pagination.TenantInfo, string) error {
	return nil
}

// ledger keeps steps the way the real one does: a key is claimed once, and a
// later claim of it is told how it went.
type ledger struct {
	mu    sync.Mutex
	steps map[string]serviceports.RunStep
}

func newLedger(steps ...serviceports.RunStep) *ledger {
	l := &ledger{steps: map[string]serviceports.RunStep{}}
	for _, step := range steps {
		l.steps[step.Key] = step
	}

	return l
}

func (l *ledger) Claim(
	_ context.Context,
	_ pagination.TenantInfo,
	step serviceports.RunStep,
) (serviceports.StepVerdict, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if held, ok := l.steps[step.Key]; ok {
		if held.Status == serviceports.RunStepCompleted {
			return serviceports.StepVerdict{State: serviceports.StepCompleted}, nil
		}

		return serviceports.StepVerdict{State: serviceports.StepUnknown}, nil
	}
	step.Status = serviceports.RunStepStarted
	l.steps[step.Key] = step

	return serviceports.StepVerdict{State: serviceports.StepFresh}, nil
}

func (l *ledger) Settle(
	_ context.Context,
	_ pagination.TenantInfo,
	step serviceports.RunStep,
) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.steps[step.Key] = step

	return nil
}

func (l *ledger) Record(context.Context, pagination.TenantInfo, serviceports.RunStep) error {
	return nil
}

func (l *ledger) Loaded(
	context.Context,
	pagination.TenantInfo,
	serviceports.RunStepOwner,
) ([]serviceports.RunStep, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	steps := make([]serviceports.RunStep, 0, len(l.steps))
	for _, step := range l.steps {
		steps = append(steps, step)
	}

	return steps, nil
}

func finishActivities(turn *conversation.AssistantTurn, steps *ledger) (*Activities, *turnRecords) {
	records := &turnRecords{turn: turn}

	return NewActivities(ActivitiesParams{
		Logger: zap.NewNop(),
		Turns: assistantturnservice.New(
			assistantturnservice.Params{Logger: zap.NewNop(), Turns: records},
		),
		TurnRepo: records,
		Steps:    steps,
	}), records
}

func finishInput(turn *conversation.AssistantTurn) *FinishTurnInput {
	return &FinishTurnInput{
		Payload: &AssistantTurnPayload{
			BasePayload: temporaltype.BasePayload{
				OrganizationID: turn.OrganizationID,
				BusinessUnitID: turn.BusinessUnitID,
			},
			TurnID:   turn.ID,
			ThreadID: turn.ThreadID,
			Actor:    serviceports.RequestActor{UserID: pulid.MustNew("usr_")},
		},
		Rejection: "This conversation is full. Start a new one.",
	}
}

func runningTurn() *conversation.AssistantTurn {
	return &conversation.AssistantTurn{
		ID:             pulid.MustNew("atrn_"),
		ThreadID:       pulid.MustNew("athr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Status:         conversation.AssistantTurnStatusRunning,
		WorkflowID:     "assistant-turn:atrn_1",
	}
}

func finish(t *testing.T, a *Activities, in *FinishTurnInput) *TurnEnding {
	t.Helper()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestActivityEnvironment()
	env.RegisterActivity(a)

	value, err := env.ExecuteActivity(a.FinishTurnActivity, in)
	require.NoError(t, err)

	var ending TurnEnding
	require.NoError(t, value.Get(&ending))

	return &ending
}

func TestFinishTurn_TellsTheReaderWhyTheQuestionWasTurnedAway(t *testing.T) {
	t.Parallel()

	turn := runningTurn()
	steps := newLedger()
	a, records := finishActivities(turn, steps)

	ending := finish(t, a, finishInput(turn))

	assert.Equal(t, serviceports.AssistantEventError, ending.Event.Event)
	assert.Equal(t, "This conversation is full. Start a new one.", ending.Result.Message)
	require.Len(t, records.completed, 1)
	assert.Equal(t, conversation.AssistantTurnStatusFailed, records.completed[0].Status)

	saved := steps.steps[turnFinishKey+":1"]
	assert.Equal(
		t,
		serviceports.RunStepCompleted,
		saved.Status,
		"the save is settled in the ledger",
	)
}

// A retry of a save that already landed must not append the turn twice. It
// sends the reader to the conversation instead.
func TestFinishTurn_NeverSavesATurnTwice(t *testing.T) {
	t.Parallel()

	turn := runningTurn()
	turn.Status = conversation.AssistantTurnStatusCompleted
	steps := newLedger(serviceports.RunStep{
		Kind:   serviceports.RunStepCompletion,
		Key:    turnFinishKey + ":1",
		Status: serviceports.RunStepCompleted,
	})
	a, records := finishActivities(turn, steps)

	ending := finish(t, a, finishInput(turn))

	assert.Empty(t, records.completed, "nothing is saved or closed a second time")
	assert.Equal(t, serviceports.AssistantEventDone, ending.Event.Event)
	assert.Equal(t, true, ending.Event.Data.(map[string]any)["replay"])
}

// A save that began and was lost may have landed, so it is not repeated
// either.
func TestFinishTurn_DoesNotRepeatASaveThatMayHaveLanded(t *testing.T) {
	t.Parallel()

	turn := runningTurn()
	steps := newLedger(serviceports.RunStep{
		Kind:   serviceports.RunStepCompletion,
		Key:    turnFinishKey + ":1",
		Status: serviceports.RunStepStarted,
	})
	a, records := finishActivities(turn, steps)

	ending := finish(t, a, finishInput(turn))

	/*
		The attempt that began the save was lost before closing the record.
		Nothing is saved again, but the record is closed: left Running it held
		the conversation's one live slot, and refused every later question.
	*/
	require.Len(t, records.completed, 1)
	assert.Equal(t, conversation.AssistantTurnStatusCompleted, records.completed[0].Status)
	assert.Equal(t, serviceports.AssistantEventDone, ending.Event.Event)
}

// A save that failed cleanly appended nothing, so a later attempt saves.
func TestFinishTurn_SavesAfterAnAttemptThatFailedCleanly(t *testing.T) {
	t.Parallel()

	turn := runningTurn()
	steps := newLedger(serviceports.RunStep{
		Kind:   serviceports.RunStepCompletion,
		Key:    turnFinishKey + ":0",
		Status: serviceports.RunStepFailed,
	})
	a, records := finishActivities(turn, steps)

	finish(t, a, finishInput(turn))

	require.Len(t, records.completed, 1)
}

func closeTurn(t *testing.T, a *Activities, turn *conversation.AssistantTurn) {
	t.Helper()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestActivityEnvironment()
	env.RegisterActivity(a)

	payload := finishInput(turn).Payload
	_, err := env.ExecuteActivity(a.CloseTurnActivity, payload, "This reply could not be saved: db")
	require.NoError(t, err)
}

func TestCloseTurn_ClosesARecordTheSaveCouldNotClose(t *testing.T) {
	t.Parallel()

	turn := runningTurn()
	a, records := finishActivities(turn, newLedger())

	closeTurn(t, a, turn)

	require.Len(t, records.completed, 1)
	assert.Equal(t, conversation.AssistantTurnStatusFailed, records.completed[0].Status)
	assert.Contains(t, records.completed[0].Error, "could not be saved")
}

func TestCloseTurn_LeavesARecordThatAlreadyEnded(t *testing.T) {
	t.Parallel()

	turn := runningTurn()
	turn.Status = conversation.AssistantTurnStatusCompleted
	a, records := finishActivities(turn, newLedger())

	closeTurn(t, a, turn)

	assert.Empty(t, records.completed)
}

type resumedFollowUps struct {
	mu      sync.Mutex
	resumed []serviceports.ResumeFollowUpsRequest
}

func (r *resumedFollowUps) ResumeFollowUps(
	_ context.Context,
	req serviceports.ResumeFollowUpsRequest,
) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.resumed = append(r.resumed, req)
}

// A decision made while this turn held the conversation found it busy, and
// its report was dropped. The turn resumes it once its record is closed and
// the conversation is free again.
func TestFinishTurn_ResumesTheFollowUpsItKeptOut(t *testing.T) {
	t.Parallel()

	turn := runningTurn()
	turn.Status = conversation.AssistantTurnStatusCompleted
	steps := newLedger(serviceports.RunStep{
		Kind:   serviceports.RunStepCompletion,
		Key:    turnFinishKey + ":1",
		Status: serviceports.RunStepCompleted,
	})
	a, _ := finishActivities(turn, steps)
	followUps := &resumedFollowUps{}
	a.followUps = followUps
	in := finishInput(turn)
	in.Rejection = ""
	in.Plan = &assistantservice.TurnPlan{ThreadID: turn.ThreadID}

	finish(t, a, in)

	require.Len(t, followUps.resumed, 1)
	assert.Equal(t, turn.ThreadID, followUps.resumed[0].ThreadID)
	assert.Equal(t, turn.OrganizationID, followUps.resumed[0].TenantInfo.OrgID)
}

// A turn that never got as far as a plan saved nothing, a follow-up that
// could not be prepared among them. Resuming from it would start the same
// failing follow-up again, for ever.
func TestFinishTurn_DoesNotResumeFromATurnThatNeverStarted(t *testing.T) {
	t.Parallel()

	turn := runningTurn()
	a, records := finishActivities(turn, newLedger())
	followUps := &resumedFollowUps{}
	a.followUps = followUps

	finish(t, a, finishInput(turn))

	require.Len(t, records.completed, 1)
	assert.Empty(t, followUps.resumed)
}
