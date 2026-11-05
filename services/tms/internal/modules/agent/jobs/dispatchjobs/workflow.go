package dispatchjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/jobs/dispatchjobs/dispatchcoveragecontract"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var dispatchRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    3,
	MaximumInterval:    30 * time.Second,
}

const (
	// coverageDebounce waits for a burst of new and uncovered loads to land,
	// an EDI batch or a morning's tender acceptances, before planning once.
	coverageDebounce         = 2 * time.Minute
	coverageIterationsPerRun = 25
)

var coverageActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 5 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    3,
	},
}

var dispatchActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 15 * time.Minute,
	HeartbeatTimeout:    time.Minute,
	RetryPolicy:         dispatchRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        HorizonPlanSweepWorkflowName,
			Fn:          HorizonPlanSweepWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Re-plan coverage for organizations that plan on a schedule",
		},
		{
			Name:        dispatchcoveragecontract.RefreshWorkflowName,
			Fn:          DispatchCoverageRefreshWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Plan an organization's uncovered moves again after its board changed",
		},
	}
}

func HorizonPlanSweepWorkflow(ctx workflow.Context) (*HorizonPlanSweepResult, error) {
	ctx = workflow.WithActivityOptions(ctx, dispatchActivityOptions)

	var a *Activities
	result := new(HorizonPlanSweepResult)
	if err := workflow.ExecuteActivity(
		ctx,
		a.HorizonPlanSweepActivity,
	).Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).Error("Horizon plan sweep workflow failed", "error", err)
		return nil, err
	}

	workflow.GetLogger(ctx).Info("Horizon plan sweep workflow completed",
		"tenantsScanned", result.TenantsScanned,
		"tenantsPlanned", result.TenantsPlanned,
		"tenantsFailed", result.TenantsFailed,
		"movesPlanned", result.MovesPlanned,
		"movesUncovered", result.MovesUncovered,
		"toursBuilt", result.ToursBuilt,
		"chainedMoves", result.ChainedMoves,
	)

	return result, nil
}

// DispatchCoverageRefreshWorkflow plans one organization's coverage again
// after loads were created or uncovered. Signals close together collapse into
// one plan: it sleeps for the debounce, drains what arrived, plans, and stops
// once nothing more came in while it planned.
func DispatchCoverageRefreshWorkflow(
	ctx workflow.Context,
	payload *dispatchcoveragecontract.RefreshPayload,
) (*CoverageRefreshResult, error) {
	var a *Activities
	signals := workflow.GetSignalChannel(ctx, dispatchcoveragecontract.RefreshSignalName)
	actx := workflow.WithActivityOptions(ctx, coverageActivityOptions)
	result := &CoverageRefreshResult{}

	for {
		if err := workflow.Sleep(ctx, coverageDebounce); err != nil {
			return result, err
		}
		drainCoverageSignals(signals)

		if err := workflow.ExecuteActivity(actx, a.RefreshCoverageActivity, payload).
			Get(actx, result); err != nil {
			return result, err
		}

		if !drainCoverageSignals(signals) {
			return result, nil
		}
		payload.Iterations++
		if payload.Iterations >= coverageIterationsPerRun {
			return result, workflow.NewContinueAsNewError(
				ctx,
				DispatchCoverageRefreshWorkflow,
				&dispatchcoveragecontract.RefreshPayload{
					OrganizationID: payload.OrganizationID,
					BusinessUnitID: payload.BusinessUnitID,
				},
			)
		}
	}
}

func drainCoverageSignals(signals workflow.ReceiveChannel) bool {
	received := false
	for {
		var signal dispatchcoveragecontract.RefreshSignal
		if !signals.ReceiveAsync(&signal) {
			return received
		}
		received = true
	}
}
