package factoringjobs

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/jobs/factoringjobs/factoringcontract"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

type batchRecorder struct {
	mu      sync.Mutex
	batches []*factoringcontract.EvaluatePayload
}

func (r *batchRecorder) record(
	_ context.Context,
	payload *factoringcontract.EvaluatePayload,
) (*factoringports.EvaluateResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.batches = append(r.batches, payload)
	return &factoringports.EvaluateResult{Evaluated: len(payload.Pending.InvoiceIDs)}, nil
}

func newPayload() *factoringcontract.EvaluatePayload {
	return &factoringcontract.EvaluatePayload{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
}

func TestEvaluateWorkflowFoldsABurstIntoOneEvaluation(t *testing.T) {
	t.Parallel()
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()
	var a *Activities
	recorder := &batchRecorder{}
	env.OnActivity(a.EvaluateFactoringTargetsActivity, mock.Anything, mock.Anything).
		Return(recorder.record)

	first, second := pulid.MustNew("inv_"), pulid.MustNew("inv_")
	customer := pulid.MustNew("cus_")
	for idx, targets := range []factoringports.EvaluationTargets{
		{InvoiceIDs: []pulid.ID{first}},
		{InvoiceIDs: []pulid.ID{first, second}},
		{CustomerIDs: []pulid.ID{customer}},
	} {
		env.RegisterDelayedCallback(func() {
			env.SignalWorkflow(
				factoringcontract.EvaluateSignalName,
				factoringcontract.EvaluateSignal{Targets: targets},
			)
		}, time.Duration(idx+1)*time.Second)
	}

	env.ExecuteWorkflow(EvaluateFactoringReadinessWorkflow, newPayload())
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	require.Len(t, recorder.batches, 1)
	assert.ElementsMatch(t, []pulid.ID{first, second}, recorder.batches[0].Pending.InvoiceIDs)
	assert.Equal(t, []pulid.ID{customer}, recorder.batches[0].Pending.CustomerIDs)
}

func TestPayloadFallsBackToAFullSweepWhenTooMuchChanged(t *testing.T) {
	t.Parallel()
	payload := newPayload()
	ids := make([]pulid.ID, 0, factoringcontract.MaxPendingTargets+1)
	for range factoringcontract.MaxPendingTargets + 1 {
		ids = append(ids, pulid.MustNew("inv_"))
	}
	payload.Merge(&factoringports.EvaluationTargets{InvoiceIDs: ids})

	assert.True(t, payload.Everything)
	assert.True(t, payload.Pending.Empty())
	taken := payload.Take()
	assert.True(t, taken.Everything)
	assert.False(t, payload.HasWork())
}
