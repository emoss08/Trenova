package billingqueuejobs

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/jobs/billingqueuejobs/billingqueuecontract"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	// tenantDebounce gathers a burst of changes into one round of checks.
	tenantDebounce = 15 * time.Second
	// tenantPagesPerRound bounds the history one round writes; a backlog
	// past it runs the next round at once rather than after the debounce.
	tenantPagesPerRound    = 25
	tenantIterationsPerRun = 25
	// briefEvery keeps the brief, a model call, to its own pace while checks
	// run every round.
	briefEvery = time.Minute
	// drainPageSize is the items one activity checks. Each may take a Nova
	// review, so the page stays small enough to finish well inside its
	// timeout.
	drainPageSize = 20
	// recheckRetryAfter and reviewRetryAfter are how long an item whose
	// checks or review failed waits before it is due again.
	recheckRetryAfter = 5 * time.Minute
	reviewRetryAfter  = time.Hour
	recheckStaleAfter = 6 * time.Hour
	// recheckSweepItems is how many stale items one sweep marks due, the
	// same pace the sweep has always kept.
	recheckSweepItems   = 500
	recheckSweepTenants = 1000
)

var drainActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 30 * time.Minute,
	HeartbeatTimeout:    4 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    3,
	},
}

var recheckActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 2 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    3,
	},
}

var agentReviewOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 3 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    10 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    2,
	},
}

var briefActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 2 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    10 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    2,
	},
}

var recheckSweepOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 2 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    30 * time.Second,
		MaximumAttempts:    3,
	},
}

type TenantWorkflowResult struct {
	Checked int `json:"checked"`
	Failed  int `json:"failed"`
	Briefs  int `json:"briefs"`
}

type DrainRechecksResult struct {
	Checked int  `json:"checked"`
	Failed  int  `json:"failed"`
	Full    bool `json:"full"`
}

type RecheckSweepResult struct {
	Marked  int64 `json:"marked"`
	Tenants int   `json:"tenants"`
}

// BillingQueueTenantWorkflow runs one tenant's billing queue in the
// background: the checks on every item marked due, Nova's review and
// auto-approval of those items, and the queue's brief. One runs per tenant,
// however many items change, and the items themselves are read from their
// due marker, so a signal carries nothing and a lost one costs only the
// sweep's delay.
func BillingQueueTenantWorkflow(
	ctx workflow.Context,
	payload *billingqueuecontract.TenantPayload,
) (*TenantWorkflowResult, error) {
	signals := workflow.GetSignalChannel(ctx, billingqueuecontract.TenantSignalName)
	result := &TenantWorkflowResult{}
	backlog := false

	for {
		if !backlog {
			if err := workflow.Sleep(ctx, tenantDebounce); err != nil {
				return result, err
			}
		}
		if _, brief := drainTenantSignals(signals); brief {
			payload.BriefPending = true
		}

		more, err := drainRound(ctx, payload, result)
		if err != nil {
			return result, err
		}
		backlog = more
		writeBriefWhenDue(ctx, payload, result)

		received, brief := drainTenantSignals(signals)
		if brief {
			payload.BriefPending = true
		}
		if !received && !backlog {
			if !payload.BriefPending {
				return result, nil
			}
			woke, wokeForBrief := awaitBriefOrSignal(ctx, signals, payload)
			if wokeForBrief {
				payload.BriefPending = true
			}
			if !woke {
				writeBrief(ctx, payload, result)
				return result, nil
			}
		}

		payload.Iterations++
		if payload.Iterations >= tenantIterationsPerRun {
			return result, workflow.NewContinueAsNewError(
				ctx,
				BillingQueueTenantWorkflow,
				&billingqueuecontract.TenantPayload{
					OrganizationID: payload.OrganizationID,
					BusinessUnitID: payload.BusinessUnitID,
					BriefPending:   payload.BriefPending,
					LastBriefAt:    payload.LastBriefAt,
				},
			)
		}
	}
}

// drainRound checks due items page by page, up to the round's bound, and
// reports whether due items are left over.
func drainRound(
	ctx workflow.Context,
	payload *billingqueuecontract.TenantPayload,
	result *TenantWorkflowResult,
) (bool, error) {
	var a *Activities
	actx := workflow.WithActivityOptions(ctx, drainActivityOptions)

	for range tenantPagesPerRound {
		var page DrainRechecksResult
		if err := workflow.ExecuteActivity(actx, a.DrainRechecksActivity, payload).
			Get(actx, &page); err != nil {
			return false, err
		}
		result.Checked += page.Checked
		result.Failed += page.Failed
		if page.Checked > 0 {
			payload.BriefPending = true
		}
		if !page.Full {
			return false, nil
		}
	}

	return true, nil
}

func writeBriefWhenDue(
	ctx workflow.Context,
	payload *billingqueuecontract.TenantPayload,
	result *TenantWorkflowResult,
) {
	if !payload.BriefPending || briefWait(ctx, payload) > 0 {
		return
	}

	writeBrief(ctx, payload, result)
}

func writeBrief(
	ctx workflow.Context,
	payload *billingqueuecontract.TenantPayload,
	result *TenantWorkflowResult,
) {
	var a *Activities
	actx := workflow.WithActivityOptions(ctx, briefActivityOptions)
	if err := workflow.ExecuteActivity(actx, a.RefreshBriefActivity, payload).
		Get(actx, nil); err != nil {
		workflow.GetLogger(ctx).Warn("billing queue brief failed", "error", err)
	}

	payload.BriefPending = false
	payload.LastBriefAt = workflow.Now(ctx).Unix()
	result.Briefs++
}

func briefWait(ctx workflow.Context, payload *billingqueuecontract.TenantPayload) time.Duration {
	if payload.LastBriefAt == 0 {
		return 0
	}

	return time.Unix(payload.LastBriefAt, 0).Add(briefEvery).Sub(workflow.Now(ctx))
}

// awaitBriefOrSignal waits out the brief's pace. It reports whether a signal
// arrived first, and whether that signal asked for a brief.
func awaitBriefOrSignal(
	ctx workflow.Context,
	signals workflow.ReceiveChannel,
	payload *billingqueuecontract.TenantPayload,
) (bool, bool) {
	wait := briefWait(ctx, payload)
	if wait <= 0 {
		return false, false
	}

	var signal billingqueuecontract.TenantSignal
	received, _ := signals.ReceiveWithTimeout(ctx, wait, &signal)

	return received, received && signal.Brief
}

// drainTenantSignals takes every signal waiting, reporting whether there was
// one and whether any asked for a brief.
func drainTenantSignals(signals workflow.ReceiveChannel) (bool, bool) {
	received, brief := false, false
	for {
		var signal billingqueuecontract.TenantSignal
		if !signals.ReceiveAsync(&signal) {
			return received, brief
		}
		received = true
		brief = brief || signal.Brief
	}
}

// BillingQueueRecheckSweepWorkflow is the backstop behind the drains: it marks
// items whose checks are missing or stale as due, then wakes every tenant
// with due items, which also catches a tenant whose signal was lost.
func BillingQueueRecheckSweepWorkflow(ctx workflow.Context) (*RecheckSweepResult, error) {
	var a *Activities
	actx := workflow.WithActivityOptions(ctx, recheckSweepOptions)
	staleBefore := workflow.Now(ctx).Add(-recheckStaleAfter).Unix()

	var result RecheckSweepResult
	if err := workflow.ExecuteActivity(actx, a.SweepRechecksActivity, staleBefore).
		Get(actx, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
