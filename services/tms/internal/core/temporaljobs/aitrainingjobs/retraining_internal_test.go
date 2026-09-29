package aitrainingjobs

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aitraining"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/zap"
)

type planningRetrainer struct {
	services.AIRetrainingService
	cycle *aitraining.RetrainingCycle
	err   error
	calls []services.PlanAIRetrainingRequest
}

func (f *planningRetrainer) Plan(
	_ context.Context,
	req *services.PlanAIRetrainingRequest,
) (*aitraining.RetrainingCycle, error) {
	f.calls = append(f.calls, *req)
	return f.cycle, f.err
}

func TestPlanScheduledRetrainingActivity(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("airc_")
	errDatabase := errors.New("database unavailable")

	t.Run("no retrainer wired", func(t *testing.T) {
		t.Parallel()
		a := &Activities{l: zap.NewNop()}
		outcome, err := a.PlanScheduledRetrainingActivity(t.Context(), &RetrainingScheduleInput{})
		require.NoError(t, err)
		assert.Equal(t, &RetrainingPlanOutcome{}, outcome)
	})

	t.Run("schedule turned off", func(t *testing.T) {
		t.Parallel()
		retrainer := &planningRetrainer{}
		a := &Activities{l: zap.NewNop(), retrainer: retrainer}
		outcome, err := a.PlanScheduledRetrainingActivity(t.Context(), &RetrainingScheduleInput{})
		require.NoError(t, err)
		assert.Empty(t, outcome.CycleID)
		require.Len(t, retrainer.calls, 1)
		assert.False(t, retrainer.calls[0].Manual, "the schedule never plans as an operator")
	})

	t.Run("a started cycle", func(t *testing.T) {
		t.Parallel()
		a := &Activities{l: zap.NewNop(), retrainer: &planningRetrainer{cycle: &aitraining.RetrainingCycle{
			ID:      id,
			Status:  aitraining.RetrainingStatusExporting,
			Trigger: aitraining.RetrainingTriggerDrift,
		}}}
		outcome, err := a.PlanScheduledRetrainingActivity(t.Context(), &RetrainingScheduleInput{})
		require.NoError(t, err)
		assert.Equal(t, &RetrainingPlanOutcome{
			CycleID: id.String(),
			Status:  "Exporting",
			Trigger: "Drift",
		}, outcome)
	})

	t.Run("a failure before anything is recorded is retried", func(t *testing.T) {
		t.Parallel()
		a := &Activities{l: zap.NewNop(), retrainer: &planningRetrainer{err: errDatabase}}
		_, err := a.PlanScheduledRetrainingActivity(t.Context(), &RetrainingScheduleInput{})
		require.ErrorIs(t, err, errDatabase)
		var applicationErr *temporal.ApplicationError
		assert.False(t, errors.As(err, &applicationErr))
	})

	t.Run("a failure after the cycle is recorded is not retried", func(t *testing.T) {
		t.Parallel()
		a := &Activities{l: zap.NewNop(), retrainer: &planningRetrainer{
			cycle: &aitraining.RetrainingCycle{ID: id, Status: aitraining.RetrainingStatusFailed},
			err:   errDatabase,
		}}
		_, err := a.PlanScheduledRetrainingActivity(t.Context(), &RetrainingScheduleInput{})
		var applicationErr *temporal.ApplicationError
		require.ErrorAs(t, err, &applicationErr)
		assert.True(t, applicationErr.NonRetryable())
		assert.Equal(t, errorTypeRetrainingRecorded, applicationErr.Type())
		assert.Contains(t, applicationErr.Error(), id.String())
	})
}

func TestExtractionRetrainingWorkflow(t *testing.T) {
	t.Parallel()

	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.RegisterActivity(&Activities{})

	var a *Activities
	env.OnActivity(a.PlanScheduledRetrainingActivity, mock.Anything, mock.Anything).
		Return(&RetrainingPlanOutcome{CycleID: "airc_1", Status: "Skipped", SkipReason: "TooSoon"}, nil).
		Once()

	env.ExecuteWorkflow(ExtractionRetrainingWorkflow, &RetrainingScheduleInput{})
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var outcome RetrainingPlanOutcome
	require.NoError(t, env.GetWorkflowResult(&outcome))
	assert.Equal(t, "TooSoon", outcome.SkipReason)
	env.AssertExpectations(t)
}

func TestRetrainingSchedule(t *testing.T) {
	t.Parallel()

	schedules := NewScheduleProvider().GetSchedules()
	require.Len(t, schedules, 1)
	assert.Equal(t, RetrainingScheduleID, schedules[0].ID)

	registered := false
	for _, definition := range RegisterWorkflows() {
		if definition.Name == ExtractionRetrainingWorkflowName {
			registered = true
		}
	}
	assert.True(t, registered, "the scheduled workflow is registered on the worker")
}
