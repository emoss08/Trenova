package iftajobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var backfillRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    3,
	MaximumInterval:    30 * time.Second,
}

var backfillActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 15 * time.Minute,
	HeartbeatTimeout:    2 * time.Minute,
	RetryPolicy:         backfillRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        BackfillJurisdictionMilesWorkflowName,
			Fn:          BackfillJurisdictionMilesWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Attribute completed moves without a jurisdiction breakdown for IFTA",
		},
	}
}

func BackfillJurisdictionMilesWorkflow(
	ctx workflow.Context,
	input BackfillInput,
) (*BackfillResult, error) {
	ctx = workflow.WithActivityOptions(ctx, backfillActivityOptions)
	logger := workflow.GetLogger(ctx)
	maxMoves := input.EffectiveMaxMoves()

	var a *Activities
	result := new(BackfillResult)
	for result.Processed+result.Failed < maxMoves {
		page := new(ListMovesMissingBreakdownResult)
		if err := workflow.ExecuteActivity(
			ctx,
			a.ListMovesMissingBreakdownActivity,
			ListMovesMissingBreakdownInput{
				TenantInfo: input.TenantInfo,
				Start:      input.Start,
				End:        input.End,
				Limit:      ListMovesPageSize,
				Offset:     result.Failed,
			},
		).Get(ctx, page); err != nil {
			logger.Error("Jurisdiction backfill could not list moves", "error", err)
			return nil, err
		}
		if len(page.MoveIDs) == 0 {
			break
		}

		remaining := maxMoves - result.Processed - result.Failed
		ids := page.MoveIDs
		if len(ids) > remaining {
			ids = ids[:remaining]
		}
		for _, batch := range chunkMoveIDs(ids, AttributeBatchSize) {
			batchResult := new(AttributeMovesResult)
			if err := workflow.ExecuteActivity(
				ctx,
				a.AttributeMovesActivity,
				AttributeMovesInput{
					TenantInfo: input.TenantInfo,
					MoveIDs:    batch,
				},
			).Get(ctx, batchResult); err != nil {
				logger.Error("Jurisdiction backfill batch failed", "error", err)
				return nil, err
			}
			result.Processed += batchResult.Processed
			result.Failed += batchResult.Failed
			result.Miles = result.Miles.Add(batchResult.Miles)
		}

		if len(page.MoveIDs) < ListMovesPageSize {
			break
		}
	}

	logger.Info("Jurisdiction backfill workflow completed",
		"processed", result.Processed,
		"failed", result.Failed,
		"miles", result.Miles.String(),
	)
	return result, nil
}
