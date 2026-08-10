package tenderjobs

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

const (
	TenderSweepWorkflowName = "TenderSweepWorkflow"

	// sweepGraceSeconds keeps the sweep clear of healthy workflows: an offer
	// must be overdue by this margin before the sweep intervenes.
	sweepGraceSeconds = 120

	sweepBatchLimit = 200
)

type TenderSweepResult struct {
	Checked   int `json:"checked"`
	Nudged    int `json:"nudged"`
	Recovered int `json:"recovered"`
	Failed    int `json:"failed"`
}

func TenderSweepWorkflow(ctx workflow.Context) (*TenderSweepResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    3,
		},
	})

	var a *Activities
	result := new(TenderSweepResult)
	if err := workflow.ExecuteActivity(ctx, a.SweepStalledTendersActivity).
		Get(ctx, result); err != nil {
		return nil, err
	}

	return result, nil
}

// SweepStalledTendersActivity is the safety net under the workflow-per-tender
// model: a live tender whose current offer is overdue gets its workflow
// nudged, and a tender whose workflow is gone gets it restarted — the
// workflow rebuilds its position from the rows.
func (a *Activities) SweepStalledTendersActivity(
	ctx context.Context,
) (*TenderSweepResult, error) {
	result := new(TenderSweepResult)

	overdueBy := timeutils.NowUnix() - sweepGraceSeconds
	tenders, err := a.repo.ListForSweep(ctx, repositories.ListTendersForSweepRequest{
		ExpiredBy: overdueBy,
		Limit:     sweepBatchLimit,
	})
	if err != nil {
		return nil, err
	}

	for _, entity := range tenders {
		result.Checked++
		if entity.WorkflowID == "" {
			result.Failed++
			a.l.Error("live tender has no workflow id; cannot recover",
				zap.String("tenderId", entity.ID.String()))
			continue
		}

		signalErr := a.workflows.SignalWorkflow(
			ctx, entity.WorkflowID, "", TenderSignalName,
			Signal{Kind: SignalKindSweepNudge},
		)
		if signalErr == nil {
			result.Nudged++
			continue
		}

		a.l.Warn("tender workflow unreachable; restarting it",
			zap.Error(signalErr),
			zap.String("tenderId", entity.ID.String()),
			zap.String("workflowId", entity.WorkflowID))

		_, startErr := a.workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
			ID:        entity.WorkflowID,
			TaskQueue: temporaltype.TaskQueueDispatch.String(),
		}, CarrierTenderWorkflowName, WorkflowInput{
			TenderID:       entity.ID,
			OrganizationID: entity.OrganizationID,
			BusinessUnitID: entity.BusinessUnitID,
			Mode:           entity.Mode,
		})
		if startErr != nil {
			result.Failed++
			a.l.Error("failed to restart tender workflow",
				zap.Error(startErr), zap.String("tenderId", entity.ID.String()))
			continue
		}
		result.Recovered++
	}

	return result, nil
}
