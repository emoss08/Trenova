package evalcasejobs

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentqualityports"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/fx"
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
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    30 * time.Second,
		MaximumAttempts:    3,
	},
}

type ActivitiesParams struct {
	fx.In

	CaseService agentqualityports.AgentEvalCaseService
}

type Activities struct {
	caseService agentqualityports.AgentEvalCaseService
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{caseService: p.CaseService}
}

var Module = fx.Module("agent-eval-case-jobs",
	fx.Provide(NewActivities),
)

func Workflows() []registry.WorkflowDefinition {
	return []registry.WorkflowDefinition{
		{
			Name:        CaptureEvalCaseCandidatesWorkflowName,
			Fn:          CaptureEvalCaseCandidatesWorkflow,
			Description: "Capture decided proposals as candidate evaluation cases",
		},
		{
			Name:        PurgeEvalCasesWorkflowName,
			Fn:          PurgeEvalCasesWorkflow,
			Description: "Purge expired evaluation cases and those of deleted conversations",
		},
	}
}

func Schedules() []*schedule.Schedule {
	return []*schedule.Schedule{
		{
			ID:            CaptureEvalCaseCandidatesScheduleID,
			Description:   "Capture decided proposals as candidate evaluation cases",
			Spec:          schedule.Every(evalCaseCaptureEvery),
			Workflow:      CaptureEvalCaseCandidatesWorkflow,
			TaskQueue:     temporaltype.TaskQueueAgentBackground.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": CaptureEvalCaseCandidatesScheduleID,
			},
		},
		{
			ID:            PurgeEvalCasesScheduleID,
			Description:   "Purge expired evaluation cases and those of deleted conversations",
			Spec:          schedule.Cron("45 4 * * *"),
			Workflow:      PurgeEvalCasesWorkflow,
			TaskQueue:     temporaltype.TaskQueueAgentBackground.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": PurgeEvalCasesScheduleID,
			},
		},
	}
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
	return a.caseService.CaptureCandidates(ctx, agentqualityports.CaptureEvalCaseCandidatesRequest{
		Since: input.Since,
		Limit: input.Limit,
	})
}

func (a *Activities) PurgeEvalCasesActivity(
	ctx context.Context,
	input *PurgeEvalCasesInput,
) (*agentqualityports.PurgeEvalCasesResult, error) {
	return a.caseService.Purge(ctx, agentqualityports.PurgeEvalCasesRequest{Now: input.Now})
}
