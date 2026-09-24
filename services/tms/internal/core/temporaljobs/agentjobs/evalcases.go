package agentjobs

import (
	"context"
	"time"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"go.temporal.io/sdk/workflow"
)

const (
	CaptureEvalCaseCandidatesWorkflowName = "CaptureEvalCaseCandidatesWorkflow"
	CaptureEvalCaseCandidatesScheduleID   = "agent-eval-case-capture"
	PurgeEvalCasesWorkflowName            = "PurgeEvalCasesWorkflow"
	PurgeEvalCasesScheduleID              = "agent-eval-case-retention"

	evalCaseCaptureEvery    = 30 * time.Minute
	evalCaseCaptureLookback = 6 * time.Hour
	evalCaseCaptureBatch    = 100
	evalCaseCaptureRounds   = 20
	evalCaseSweepTimeout    = 10 * time.Minute
)

var evalCaseSweepOptions = workflow.ActivityOptions{
	StartToCloseTimeout: evalCaseSweepTimeout,
	RetryPolicy:         deterministicRetry,
}

type CaptureEvalCaseCandidatesInput struct {
	Since int64 `json:"since"`
	Limit int   `json:"limit"`
}

type PurgeEvalCasesInput struct {
	Now int64 `json:"now"`
}

func CaptureEvalCaseCandidatesWorkflow(
	ctx workflow.Context,
) (*serviceports.CaptureEvalCaseCandidatesResult, error) {
	var a *Activities
	total := &serviceports.CaptureEvalCaseCandidatesResult{}

	since := workflow.Now(ctx).Add(-evalCaseCaptureLookback).Unix()
	captureCtx := workflow.WithActivityOptions(ctx, evalCaseSweepOptions)
	for range evalCaseCaptureRounds {
		var round serviceports.CaptureEvalCaseCandidatesResult
		if err := workflow.ExecuteActivity(
			captureCtx,
			a.CaptureEvalCaseCandidatesActivity,
			&CaptureEvalCaseCandidatesInput{Since: since, Limit: evalCaseCaptureBatch},
		).Get(captureCtx, &round); err != nil {
			return nil, err
		}
		total.Scanned += round.Scanned
		total.Captured += round.Captured
		total.Duplicates += round.Duplicates
		total.Failed += round.Failed
		if round.Scanned < evalCaseCaptureBatch || round.Captured == 0 {
			break
		}
	}

	return total, nil
}

func PurgeEvalCasesWorkflow(ctx workflow.Context) (*serviceports.PurgeEvalCasesResult, error) {
	var a *Activities
	result := &serviceports.PurgeEvalCasesResult{}

	purgeCtx := workflow.WithActivityOptions(ctx, evalCaseSweepOptions)
	if err := workflow.ExecuteActivity(purgeCtx, a.PurgeEvalCasesActivity, &PurgeEvalCasesInput{
		Now: workflow.Now(ctx).Unix(),
	}).Get(purgeCtx, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (a *Activities) CaptureEvalCaseCandidatesActivity(
	ctx context.Context,
	input *CaptureEvalCaseCandidatesInput,
) (*serviceports.CaptureEvalCaseCandidatesResult, error) {
	if a.caseService == nil {
		return &serviceports.CaptureEvalCaseCandidatesResult{}, nil
	}

	return a.caseService.CaptureCandidates(ctx, serviceports.CaptureEvalCaseCandidatesRequest{
		Since: input.Since,
		Limit: input.Limit,
	})
}

func (a *Activities) PurgeEvalCasesActivity(
	ctx context.Context,
	input *PurgeEvalCasesInput,
) (*serviceports.PurgeEvalCasesResult, error) {
	if a.caseService == nil {
		return &serviceports.PurgeEvalCasesResult{}, nil
	}

	return a.caseService.Purge(ctx, serviceports.PurgeEvalCasesRequest{Now: input.Now})
}
