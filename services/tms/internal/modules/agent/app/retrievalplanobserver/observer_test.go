package retrievalplanobserver_test

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/retrievalplanobserver"
	"github.com/emoss08/trenova/internal/modules/agent/app/retrievalservice"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type started struct {
	options  client.StartWorkflowOptions
	workflow any
	args     []any
}

type fakeStarter struct {
	kernelports.WorkflowStarter
	started []started
	err     error
}

func (f *fakeStarter) StartWorkflow(
	_ context.Context,
	options client.StartWorkflowOptions,
	workflow any,
	args ...any,
) (client.WorkflowRun, error) {
	f.started = append(f.started, started{options: options, workflow: workflow, args: args})
	return nil, f.err
}

func newObserver(starter *fakeStarter) *retrievalplanobserver.Observer {
	result := retrievalplanobserver.New(retrievalplanobserver.Params{
		Workflows: starter,
		Logger:    zap.NewNop(),
	})

	observer, ok := result.Observer.(*retrievalplanobserver.Observer)
	if !ok {
		panic("the observer is not the retrieval plan observer")
	}

	return observer
}

func state(features ...platformcatalog.FeatureKey) *managedfeatures.State {
	return &managedfeatures.State{ID: "current", Features: features}
}

func TestPlanChanged_TrenovaAIAddedStartsOneDelayedSweep(t *testing.T) {
	t.Parallel()

	starter := &fakeStarter{}
	err := newObserver(starter).PlanChanged(t.Context(), &managedfeatures.Replacement{
		Previous: state(platformcatalog.FeatureAccounting),
		Current:  state(platformcatalog.FeatureAccounting, platformcatalog.FeatureAIDesk),
	})
	require.NoError(t, err)

	require.Len(t, starter.started, 1)
	run := starter.started[0]
	assert.Equal(t, retrievalservice.SweepWorkflowName, run.workflow)
	assert.Empty(t, run.args, "the sweep starts from the first organization")
	assert.Equal(t, retrievalplanobserver.SweepWorkflowID, run.options.ID)
	assert.Equal(t, temporaltype.TaskQueueSystem.String(), run.options.TaskQueue)
	assert.Equal(
		t,
		enums.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
		run.options.WorkflowIDConflictPolicy,
		"a second change while a sweep is waiting or running joins it",
	)
	assert.Greater(t, run.options.StartDelay.Seconds(), 30.0,
		"the worker's feature cache holds the old plan for up to 30 seconds")
}

func TestPlanChanged_TheStartingPlanWithTrenovaAIStartsASweep(t *testing.T) {
	t.Parallel()

	starter := &fakeStarter{}
	err := newObserver(starter).PlanChanged(t.Context(), &managedfeatures.Replacement{
		Current: state(platformcatalog.FeatureAIDesk),
	})
	require.NoError(t, err)
	assert.Len(t, starter.started, 1)
}

func TestPlanChanged_StartsNothingWhen(t *testing.T) {
	t.Parallel()

	held := state(platformcatalog.FeatureAIDesk)
	tests := []struct {
		name        string
		replacement *managedfeatures.Replacement
	}{
		{name: "nothing was replaced", replacement: nil},
		{
			name:        "the same state was kept",
			replacement: &managedfeatures.Replacement{Previous: held, Current: held},
		},
		{
			name: "Trenova AI was already on",
			replacement: &managedfeatures.Replacement{
				Previous: state(platformcatalog.FeatureAIDesk),
				Current:  state(platformcatalog.FeatureAIDesk, platformcatalog.FeatureAccounting),
			},
		},
		{
			name: "Trenova AI was removed",
			replacement: &managedfeatures.Replacement{
				Previous: state(platformcatalog.FeatureAIDesk),
				Current:  state(),
			},
		},
		{
			name: "another add-on was added",
			replacement: &managedfeatures.Replacement{
				Previous: state(),
				Current:  state(platformcatalog.FeatureAccounting),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			starter := &fakeStarter{}
			require.NoError(t, newObserver(starter).PlanChanged(t.Context(), tt.replacement))
			assert.Empty(t, starter.started)
		})
	}
}

func TestPlanChanged_AStartThatFailsIsReported(t *testing.T) {
	t.Parallel()

	starter := &fakeStarter{err: errors.New("temporal is unreachable")}
	err := newObserver(starter).PlanChanged(t.Context(), &managedfeatures.Replacement{
		Previous: state(),
		Current:  state(platformcatalog.FeatureAIDesk),
	})
	require.Error(t, err, "the receiver logs it; the hourly sweep is the safety net")
}
