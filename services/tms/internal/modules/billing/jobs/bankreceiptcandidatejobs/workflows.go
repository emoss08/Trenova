package bankreceiptcandidatejobs

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/billing/jobs/bankreceiptcandidatejobs/bankreceiptcandidatecontract"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	drainDebounce         = 3 * time.Second
	drainIterationsPerRun = 25
	drainPagesPerRound    = 20
	drainPageSize         = 50
	drainRetryAfter       = time.Minute
	drainWakeLimit        = 1000
	drainActivityTimeout  = 5 * time.Minute
)

var drainActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: drainActivityTimeout,
	HeartbeatTimeout:    2 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    3,
	},
}

var sweepActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 2 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    30 * time.Second,
		MaximumAttempts:    3,
	},
}

func BankReceiptCandidateDrainWorkflow(
	ctx workflow.Context,
	payload *temporaljobs.DrainPayload,
) (*temporaljobs.DrainTotals, error) {
	var a *Activities
	actx := workflow.WithActivityOptions(ctx, drainActivityOptions)
	totals, again, err := temporaljobs.RunTenantDrain(ctx, temporaljobs.TenantDrainOptions{
		SignalName:       bankreceiptcandidatecontract.DrainSignalName,
		Debounce:         drainDebounce,
		PagesPerRound:    drainPagesPerRound,
		IterationsPerRun: drainIterationsPerRun,
		Iterations:       payload.Iterations,
		Page: func(ctx workflow.Context) (*temporaljobs.DrainPage, error) {
			var page temporaljobs.DrainPage
			err := workflow.ExecuteActivity(actx, a.DrainBankReceiptCandidatesActivity, payload).
				Get(actx, &page)

			return &page, err
		},
	})
	if err != nil || !again {
		return totals, err
	}

	return totals, workflow.NewContinueAsNewError(ctx, BankReceiptCandidateDrainWorkflow,
		&temporaljobs.DrainPayload{
			OrganizationID: payload.OrganizationID,
			BusinessUnitID: payload.BusinessUnitID,
		})
}

func BankReceiptCandidateWakeWorkflow(ctx workflow.Context) (int, error) {
	var a *Activities
	actx := workflow.WithActivityOptions(ctx, sweepActivityOptions)
	var woken int
	err := workflow.ExecuteActivity(actx, a.WakeDueBankReceiptCandidateDrainsActivity).
		Get(actx, &woken)

	return woken, err
}

func BankReceiptCandidateSweepWorkflow(ctx workflow.Context) (int64, error) {
	var a *Activities
	actx := workflow.WithActivityOptions(ctx, sweepActivityOptions)
	var queued int64
	if err := workflow.ExecuteActivity(actx, a.QueueUnappliedBankReceiptsActivity).
		Get(actx, &queued); err != nil {
		return 0, err
	}
	if err := workflow.ExecuteActivity(actx, a.WakeDueBankReceiptCandidateDrainsActivity).
		Get(actx, nil); err != nil {
		return queued, err
	}

	return queued, nil
}
