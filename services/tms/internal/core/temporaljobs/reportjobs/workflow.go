package reportjobs

import (
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	ErrTypeReportValidation    = "REPORT_VALIDATION"
	ErrTypeReportAuthorization = "REPORT_AUTHORIZATION"
	ErrTypeReportTooExpensive  = "REPORT_TOO_EXPENSIVE"
)

var prepareActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 1 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    2 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumAttempts:    3,
		MaximumInterval:    15 * time.Second,
		NonRetryableErrorTypes: []string{
			ErrTypeReportValidation,
			ErrTypeReportAuthorization,
		},
	},
}

var finalizeActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 2 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    2 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumAttempts:    10,
		MaximumInterval:    30 * time.Second,
	},
}

var deliverActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 2 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    2 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumAttempts:    deliverMaxAttempts,
		MaximumInterval:    30 * time.Second,
	},
}

var cleanupActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 10 * time.Minute,
	HeartbeatTimeout:    time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumAttempts:    3,
		MaximumInterval:    time.Minute,
	},
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        "RunReportWorkflow",
			Fn:          RunReportWorkflow,
			TaskQueue:   temporaltype.ReportTaskQueue,
			Description: "Compile, execute, render, and deliver a report run",
		},
		{
			Name:        "CleanupExpiredReportRunsWorkflow",
			Fn:          CleanupExpiredReportRunsWorkflow,
			TaskQueue:   temporaltype.ReportTaskQueue,
			Description: "Delete expired report artifacts and mark runs expired",
		},
		{
			Name:        "DispatchDueReportSchedulesWorkflow",
			Fn:          DispatchDueReportSchedulesWorkflow,
			TaskQueue:   temporaltype.ReportTaskQueue,
			Description: "Dispatch report runs for due user schedules",
		},
		{
			Name:        "ReconcileZombieReportRunsWorkflow",
			Fn:          ReconcileZombieReportRunsWorkflow,
			TaskQueue:   temporaltype.ReportTaskQueue,
			Description: "Fail report runs whose workflows are no longer running",
		},
	}
}

func RunReportWorkflow(
	ctx workflow.Context,
	payload *RunReportPayload,
) (*RunReportResult, error) {
	var a *Activities
	startedAt := workflow.Now(ctx)

	var prepared *PreparedRun
	prepareCtx := workflow.WithActivityOptions(ctx, prepareActivityOptions)
	if err := workflow.ExecuteActivity(
		prepareCtx, a.PrepareRunActivity, payload,
	).Get(prepareCtx, &prepared); err != nil {
		return finalizeFailure(ctx, payload, startedAt, err)
	}

	maxRunDuration := 30 * time.Minute
	if prepared.MaxRunSeconds > 0 {
		maxRunDuration = time.Duration(prepared.MaxRunSeconds) * time.Second
	}

	executeOptions := workflow.ActivityOptions{
		StartToCloseTimeout: maxRunDuration,
		HeartbeatTimeout:    90 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    5 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    2,
			MaximumInterval:    time.Minute,
			NonRetryableErrorTypes: []string{
				ErrTypeReportValidation,
				ErrTypeReportAuthorization,
				ErrTypeReportTooExpensive,
			},
		},
	}

	var execResult *ExecuteResult
	executeCtx := workflow.WithActivityOptions(ctx, executeOptions)
	if err := workflow.ExecuteActivity(
		executeCtx, a.ExecuteAndRenderActivity, prepared,
	).Get(executeCtx, &execResult); err != nil {
		return finalizeFailure(ctx, payload, startedAt, err)
	}

	finalize := &FinalizePayload{
		RunID:             payload.RunID,
		OrganizationID:    payload.OrganizationID,
		BusinessUnitID:    payload.BusinessUnitID,
		Status:            report.RunStatusSucceeded,
		ArtifactKey:       execResult.ArtifactKey,
		CacheHit:          execResult.CacheHit,
		ArtifactExpiresAt: execResult.ArtifactExpiresAt,
		RowCount:          execResult.RowCount,
		ByteSize:          execResult.ByteSize,
		Truncated:         execResult.Truncated,
		DurationMs:        workflow.Now(ctx).Sub(startedAt).Milliseconds(),
	}

	finalizeCtx := workflow.WithActivityOptions(ctx, finalizeActivityOptions)
	if err := workflow.ExecuteActivity(
		finalizeCtx, a.FinalizeRunActivity, finalize,
	).Get(finalizeCtx, nil); err != nil {
		return nil, err
	}

	// Delivery is best-effort: a failed email or notification must never fail
	// the run itself — the artifact is finalized and downloadable either way.
	if !prepared.ScheduleID.IsNil() {
		deliverCtx := workflow.WithActivityOptions(ctx, deliverActivityOptions)
		if err := workflow.ExecuteActivity(
			deliverCtx, a.DeliverScheduledRunActivity,
			&DeliverRunPayload{
				RunID:          payload.RunID,
				OrganizationID: payload.OrganizationID,
				BusinessUnitID: payload.BusinessUnitID,
			},
		).Get(deliverCtx, nil); err != nil {
			workflow.GetLogger(ctx).Error("scheduled report delivery failed",
				"runId", payload.RunID.String(), "error", err)
		}
	}

	return &RunReportResult{
		RunID:     payload.RunID,
		Status:    report.RunStatusSucceeded,
		RowCount:  execResult.RowCount,
		ByteSize:  execResult.ByteSize,
		Truncated: execResult.Truncated,
	}, nil
}

// finalizeFailure always records a terminal state for the run — including on
// workflow cancellation — so a run can never be left stuck in queued/running.
func finalizeFailure(
	ctx workflow.Context,
	payload *RunReportPayload,
	startedAt time.Time,
	cause error,
) (*RunReportResult, error) {
	status := report.RunStatusFailed
	if temporal.IsCanceledError(cause) || errors.Is(ctx.Err(), workflow.ErrCanceled) {
		status = report.RunStatusCanceled
	}

	finalize := &FinalizePayload{
		RunID:          payload.RunID,
		OrganizationID: payload.OrganizationID,
		BusinessUnitID: payload.BusinessUnitID,
		Status:         status,
		Error:          runErrorFrom(cause),
		DurationMs:     workflow.Now(ctx).Sub(startedAt).Milliseconds(),
	}

	// Finalization must run even when the workflow itself was canceled.
	finalizeCtx, cancel := workflow.NewDisconnectedContext(ctx)
	defer cancel()
	finalizeCtx = workflow.WithActivityOptions(finalizeCtx, finalizeActivityOptions)

	if err := workflow.ExecuteActivity(
		finalizeCtx, (*Activities)(nil).FinalizeRunActivity, finalize,
	).Get(finalizeCtx, nil); err != nil {
		return nil, err
	}

	return &RunReportResult{RunID: payload.RunID, Status: status}, cause
}

func runErrorFrom(err error) *report.RunError {
	var appErr *temporal.ApplicationError
	if errors.As(err, &appErr) {
		return &report.RunError{
			Code:    appErr.Type(),
			Message: appErr.Message(),
		}
	}
	if temporal.IsCanceledError(err) {
		return &report.RunError{Code: "CANCELED", Message: "The run was canceled"}
	}

	var timeoutErr *temporal.TimeoutError
	if errors.As(err, &timeoutErr) {
		return &report.RunError{
			Code:    "TIMEOUT",
			Message: "The report exceeded its maximum run duration — narrow your filters or date range",
		}
	}

	return &report.RunError{
		Code:    "INTERNAL",
		Message: "Report generation failed",
		Detail:  err.Error(),
	}
}

func CleanupExpiredReportRunsWorkflow(
	ctx workflow.Context,
) (*CleanupExpiredResult, error) {
	ctx = workflow.WithActivityOptions(ctx, cleanupActivityOptions)

	var a *Activities
	var result *CleanupExpiredResult
	if err := workflow.ExecuteActivity(
		ctx, a.CleanupExpiredArtifactsActivity,
	).Get(ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func DispatchDueReportSchedulesWorkflow(
	ctx workflow.Context,
) (*DispatchDueSchedulesResult, error) {
	ctx = workflow.WithActivityOptions(ctx, cleanupActivityOptions)

	var a *Activities
	var result *DispatchDueSchedulesResult
	if err := workflow.ExecuteActivity(
		ctx, a.DispatchDueSchedulesActivity,
	).Get(ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func ReconcileZombieReportRunsWorkflow(
	ctx workflow.Context,
) (*ReconcileZombiesResult, error) {
	ctx = workflow.WithActivityOptions(ctx, cleanupActivityOptions)

	var a *Activities
	var result *ReconcileZombiesResult
	if err := workflow.ExecuteActivity(
		ctx, a.ReconcileZombieRunsActivity,
	).Get(ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}
