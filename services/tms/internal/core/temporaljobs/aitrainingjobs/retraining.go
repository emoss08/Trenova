package aitrainingjobs

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

const (
	ExtractionRetrainingWorkflowName = "ExtractionRetrainingWorkflow"
	errorTypeRetrainingRecorded      = "retraining_cycle_recorded"
)

var planOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 10 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:        5 * time.Second,
		BackoffCoefficient:     2.0,
		MaximumAttempts:        3,
		MaximumInterval:        time.Minute,
		NonRetryableErrorTypes: []string{errorTypeRetrainingRecorded},
	},
}

type RetrainingScheduleInput struct{}

type RetrainingPlanOutcome struct {
	CycleID    string `json:"cycleId,omitempty"`
	Status     string `json:"status,omitempty"`
	SkipReason string `json:"skipReason,omitempty"`
	Trigger    string `json:"trigger,omitempty"`
}

func retrainingWorkflowDefinition() temporaltype.WorkflowDefinition {
	return temporaltype.WorkflowDefinition{
		Name:        ExtractionRetrainingWorkflowName,
		Fn:          ExtractionRetrainingWorkflow,
		TaskQueue:   temporaltype.TaskQueueSystem.String(),
		Description: "Decide whether enough new corrections have been confirmed to retrain the document extraction model, and start its training export",
	}
}

func ExtractionRetrainingWorkflow(
	ctx workflow.Context,
	input *RetrainingScheduleInput,
) (*RetrainingPlanOutcome, error) {
	if input == nil {
		input = &RetrainingScheduleInput{}
	}

	var a *Activities
	planCtx := workflow.WithActivityOptions(ctx, planOptions)
	var outcome RetrainingPlanOutcome
	if err := workflow.ExecuteActivity(planCtx, a.PlanScheduledRetrainingActivity, input).
		Get(planCtx, &outcome); err != nil {
		return nil, err
	}

	return &outcome, nil
}

func (a *Activities) PlanScheduledRetrainingActivity(
	ctx context.Context,
	_ *RetrainingScheduleInput,
) (*RetrainingPlanOutcome, error) {
	if a.retrainer == nil {
		return &RetrainingPlanOutcome{}, nil
	}

	cycle, err := a.retrainer.Plan(ctx, &services.PlanAIRetrainingRequest{})
	if err != nil {
		if cycle != nil {
			return nil, temporal.NewNonRetryableApplicationError(
				fmt.Sprintf("retraining cycle %s: %s", cycle.ID, err.Error()),
				errorTypeRetrainingRecorded,
				err,
			)
		}
		return nil, fmt.Errorf("plan retraining: %w", err)
	}
	if cycle == nil {
		return &RetrainingPlanOutcome{}, nil
	}

	a.l.Info("retraining planned",
		zap.String("cycleId", cycle.ID.String()),
		zap.String("status", cycle.Status.String()),
		zap.String("skipReason", cycle.SkipReason.String()),
	)

	return &RetrainingPlanOutcome{
		CycleID:    cycle.ID.String(),
		Status:     cycle.Status.String(),
		SkipReason: cycle.SkipReason.String(),
		Trigger:    cycle.Trigger.String(),
	}, nil
}
