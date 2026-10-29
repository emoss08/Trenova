package factoringjobs

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/jobs/factoringjobs/factoringcontract"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringremittanceports"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	remittanceStageDescription = "Read a factor's PDF remittance once its text has been extracted"
	remittanceStagePoll        = 15 * time.Second
	remittanceStageDeadline    = 15 * time.Minute
)

var remittanceStageOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 5 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    10 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    4,
	},
}

type RemittanceStageResult struct {
	Status  string `json:"status"`
	Waiting bool   `json:"waiting"`
	Polls   int    `json:"polls"`
}

func FactoringRemittanceStageWorkflow(
	ctx workflow.Context,
	payload *factoringcontract.RemittanceStagePayload,
) (*RemittanceStageResult, error) {
	var a *Activities
	actx := workflow.WithActivityOptions(ctx, remittanceStageOptions)
	deadline := workflow.Now(ctx).Add(remittanceStageDeadline)
	result := &RemittanceStageResult{}

	for {
		step := *payload
		step.GiveUp = !workflow.Now(ctx).Add(remittanceStagePoll).Before(deadline)
		var staged factoringremittanceports.StagePendingResult
		if err := workflow.ExecuteActivity(actx, a.StageFactoringRemittanceActivity, &step).
			Get(actx, &staged); err != nil {
			return result, err
		}
		result.Polls++
		result.Waiting = staged.Waiting
		if staged.Remittance != nil {
			result.Status = staged.Remittance.Status.String()
		}
		if !staged.Waiting || step.GiveUp {
			return result, nil
		}
		if err := workflow.Sleep(ctx, remittanceStagePoll); err != nil {
			return result, err
		}
	}
}

func (a *Activities) StageFactoringRemittanceActivity(
	ctx context.Context,
	payload *factoringcontract.RemittanceStagePayload,
) (*factoringremittanceports.StagePendingResult, error) {
	result, err := a.remittances.StagePending(
		ctx,
		&factoringremittanceports.RemittanceVersionRequest{
			TenantInfo: payload.TenantInfo(),
			ID:         payload.RemittanceID,
			Version:    payload.Version,
		},
		payload.GiveUp,
	)
	if err != nil {
		return nil, err
	}
	if result.Remittance != nil {
		result.Remittance.Lines = nil
		result.Remittance.FactoringCompany = nil
	}
	return result, nil
}
