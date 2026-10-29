package agentjobs

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentqualityports"
	"go.temporal.io/sdk/workflow"
)

const (
	CaptureEvalCaseCandidatesWorkflowName = "CaptureEvalCaseCandidatesWorkflow"
	CaptureEvalCaseCandidatesScheduleID   = "agent-eval-case-capture"
	PurgeEvalCasesWorkflowName            = "PurgeEvalCasesWorkflow"
	PurgeEvalCasesScheduleID              = "agent-eval-case-retention"

	evalCaseCaptureEvery    = 2 * time.Hour // must stay under evalCaseCaptureLookback
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
) (*agentqualityports.CaptureEvalCaseCandidatesResult, error) {
	var a *Activities
	total := &agentqualityports.CaptureEvalCaseCandidatesResult{}

	since := workflow.Now(ctx).Add(-evalCaseCaptureLookback).Unix()
	captureCtx := workflow.WithActivityOptions(ctx, evalCaseSweepOptions)
	for range evalCaseCaptureRounds {
		var round agentqualityports.CaptureEvalCaseCandidatesResult
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

func PurgeEvalCasesWorkflow(ctx workflow.Context) (*agentqualityports.PurgeEvalCasesResult, error) {
	var a *Activities
	result := &agentqualityports.PurgeEvalCasesResult{}

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
) (*agentqualityports.CaptureEvalCaseCandidatesResult, error) {
	if a.caseService == nil {
		return &agentqualityports.CaptureEvalCaseCandidatesResult{}, nil
	}

	return a.caseService.CaptureCandidates(ctx, agentqualityports.CaptureEvalCaseCandidatesRequest{
		Since: input.Since,
		Limit: input.Limit,
	})
}

func (a *Activities) PurgeEvalCasesActivity(
	ctx context.Context,
	input *PurgeEvalCasesInput,
) (*agentqualityports.PurgeEvalCasesResult, error) {
	if a.caseService == nil {
		return &agentqualityports.PurgeEvalCasesResult{}, nil
	}

	return a.caseService.Purge(ctx, agentqualityports.PurgeEvalCasesRequest{Now: input.Now})
}
