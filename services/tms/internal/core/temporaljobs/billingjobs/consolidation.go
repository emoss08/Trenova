package billingjobs

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

const ConsolidatedInvoiceRunWorkflowName = "ConsolidatedInvoiceRunWorkflow"

// ConsolidatedInvoiceRunResult is what one sweep produced.
type ConsolidatedInvoiceRunResult struct {
	SchedulesDue    int   `json:"schedulesDue"`
	RunsBuilt       int   `json:"runsBuilt"`
	InvoicesCreated int   `json:"invoicesCreated"`
	GroupsSkipped   int   `json:"groupsSkipped"`
	Failed          int   `json:"failed"`
	CompletedAt     int64 `json:"completedAt"`
}

// A sweep that fails is retried, but not forever: a misconfigured schedule fails
// the same way every time, and the next hourly tick will pick the period up
// anyway because the billing watermark has not moved.
var consolidatedInvoiceRunRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    10 * time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    3,
	MaximumInterval:    2 * time.Minute,
	NonRetryableErrorTypes: []string{
		temporaltype.ErrorTypeInvalidInput.String(),
		temporaltype.ErrorTypeNonRetryable.String(),
		temporaltype.ErrorTypePermissionDenied.String(),
	},
}

var consolidatedInvoiceRunActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 30 * time.Minute,
	HeartbeatTimeout:    time.Minute,
	RetryPolicy:         consolidatedInvoiceRunRetryPolicy,
}

// ConsolidatedInvoiceRunWorkflow bills every closed, unbilled period.
func ConsolidatedInvoiceRunWorkflow(
	ctx workflow.Context,
) (*ConsolidatedInvoiceRunResult, error) {
	ctx = workflow.WithActivityOptions(ctx, consolidatedInvoiceRunActivityOptions)

	var a *Activities
	result := new(ConsolidatedInvoiceRunResult)
	if err := workflow.ExecuteActivity(
		ctx,
		a.SweepConsolidatedInvoiceRunsActivity,
	).Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).Error("Consolidated invoice run sweep failed", "error", err)
		return nil, err
	}

	return result, nil
}

// SweepConsolidatedInvoiceRunsActivity asks every statement customer which of
// their periods have closed, and bills them.
func (a *Activities) SweepConsolidatedInvoiceRunsActivity(
	ctx context.Context,
) (*ConsolidatedInvoiceRunResult, error) {
	if a.invoiceRunSweeper == nil {
		return &ConsolidatedInvoiceRunResult{CompletedAt: timeutils.NowUnix()}, nil
	}

	swept, err := a.invoiceRunSweeper.SweepDueSchedules(ctx, consolidationSystemActor())
	if err != nil {
		a.logger.Error("failed to sweep consolidated invoice runs", zap.Error(err))
		return nil, err
	}

	return &ConsolidatedInvoiceRunResult{
		SchedulesDue:    swept.SchedulesDue,
		RunsBuilt:       swept.RunsBuilt,
		InvoicesCreated: swept.InvoicesCreated,
		GroupsSkipped:   swept.GroupsSkipped,
		Failed:          swept.Failed,
		CompletedAt:     timeutils.NowUnix(),
	}, nil
}

func consolidationSystemActor() *services.RequestActor {
	return &services.RequestActor{
		PrincipalType: services.PrincipalTypeSystem,
		PrincipalID:   services.SystemPrincipalID,
		UserID:        services.SystemPrincipalID,
	}
}
