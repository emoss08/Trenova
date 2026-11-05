package scimjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/jobs/scimjobs/scimcontract"
	"github.com/emoss08/trenova/internal/modules/identity/ports/scimports"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	reconcileDebounce       = 5 * time.Second
	pagesPerPass            = 50
	passesPerRun            = 20
	reconcileActivityPeriod = 5 * time.Minute
)

var reconcileOptions = workflow.ActivityOptions{
	StartToCloseTimeout: reconcileActivityPeriod,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 2,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    3,
	},
}

var sweepOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 2 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    10 * time.Second,
		BackoffCoefficient: 2,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    3,
	},
}

type ReconcileSummary struct {
	Reconciled int `json:"reconciled"`
	Failed     int `json:"failed"`
	Passes     int `json:"passes"`
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        scimcontract.ReconcileWorkflowName,
			Fn:          ReconcileSCIMDirectoryGrantsWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Bring the roles a SCIM directory's groups grant up to date",
		},
		{
			Name:      SweepWorkflowName,
			Fn:        SweepPendingSCIMGrantsWorkflow,
			TaskQueue: temporaltype.TaskQueueSystem.String(),
			Description: "Start a reconcile for every SCIM directory whose group roles " +
				"still wait to be brought up to date",
		},
	}
}

func ReconcileSCIMDirectoryGrantsWorkflow(
	ctx workflow.Context,
	payload *scimcontract.ReconcilePayload,
) (*ReconcileSummary, error) {
	var a *Activities
	signals := workflow.GetSignalChannel(ctx, scimcontract.ReconcileSignalName)
	actx := workflow.WithActivityOptions(ctx, reconcileOptions)
	summary := new(ReconcileSummary)

	for {
		if err := workflow.Sleep(ctx, reconcileDebounce); err != nil {
			return summary, err
		}
		drainSignals(signals)

		more := false
		for range pagesPerPass {
			var result scimports.ReconcileResult
			if err := workflow.ExecuteActivity(actx, a.ReconcilePendingActivity, payload).
				Get(actx, &result); err != nil {
				return summary, err
			}
			summary.Reconciled += result.Reconciled
			summary.Failed += result.Failed
			more = result.More
			if !more {
				break
			}
		}
		summary.Passes++

		if !drainSignals(signals) && !more {
			return summary, nil
		}

		payload.Iterations++
		if payload.Iterations >= passesPerRun {
			return summary, workflow.NewContinueAsNewError(
				ctx,
				ReconcileSCIMDirectoryGrantsWorkflow,
				&scimcontract.ReconcilePayload{
					OrganizationID: payload.OrganizationID,
					BusinessUnitID: payload.BusinessUnitID,
					DirectoryID:    payload.DirectoryID,
				},
			)
		}
	}
}

func drainSignals(signals workflow.ReceiveChannel) bool {
	received := false
	for {
		var signal scimcontract.ReconcileSignal
		if !signals.ReceiveAsync(&signal) {
			return received
		}
		received = true
	}
}

func SweepPendingSCIMGrantsWorkflow(ctx workflow.Context) (int, error) {
	var a *Activities
	ctx = workflow.WithActivityOptions(ctx, sweepOptions)

	var requested int
	err := workflow.ExecuteActivity(ctx, a.RequestPendingReconcilesActivity).Get(ctx, &requested)

	return requested, err
}
