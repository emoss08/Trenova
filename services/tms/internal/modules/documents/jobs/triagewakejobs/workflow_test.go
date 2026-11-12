package triagewakejobs

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

type sweepWorker struct {
	calls []int64
	woken int
}

func (w *sweepWorker) WakeDue(_ context.Context, now int64) (int, error) {
	w.calls = append(w.calls, now)

	return w.woken, nil
}

func (w *sweepWorker) RecomputeDueFor(
	context.Context,
	pagination.TenantInfo,
	[]inboundmessage.Classification,
) (int64, error) {
	return 0, nil
}

// The minute sweep is a single activity that wakes whatever is due and
// reports how many it brought back.
func TestWakeDueTriageWorkflowWakesWhatIsDue(t *testing.T) {
	t.Parallel()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	worker := &sweepWorker{woken: 3}
	env.RegisterActivity(NewActivities(ActivitiesParams{Worker: worker}))
	env.RegisterWorkflow(WakeDueTriageWorkflow)

	env.ExecuteWorkflow(WakeDueTriageWorkflow)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	var woken int
	require.NoError(t, env.GetWorkflowResult(&woken))
	assert.Equal(t, 3, woken)
	assert.Len(t, worker.calls, 1)
}
