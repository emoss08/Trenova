package extractionevaljobs

import (
	"time"

	"github.com/emoss08/trenova/internal/core/domain/extractioneval"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
	"github.com/emoss08/trenova/internal/core/temporaljobs/modelcall"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const pendingPageSize = 50

var bookkeepingRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    5,
	MaximumInterval:    30 * time.Second,
	NonRetryableErrorTypes: []string{
		temporaltype.ErrorTypeInvalidInput.String(),
	},
}

var bookkeepingOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 2 * time.Minute,
	RetryPolicy:         bookkeepingRetryPolicy,
}

var evaluateOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 10 * time.Minute,
	HeartbeatTimeout:    modelcall.HeartbeatTimeout,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumAttempts:    evaluateAttempts,
		MaximumInterval:    time.Minute,
		NonRetryableErrorTypes: []string{
			temporaltype.ErrorTypeInvalidInput.String(),
		},
	},
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        ExtractionEvalRunWorkflowName,
			Fn:          ExtractionEvalRunWorkflow,
			TaskQueue:   temporaltype.DocumentIntelligenceTaskQueue,
			Description: "Run one AI provider's model over the active extraction cases within the evaluation budget and score each field",
		},
	}
}

func evaluationPriority(orgID pulid.ID) temporal.Priority {
	return temporal.Priority{
		PriorityKey: agentflow.PriorityEvaluation,
		FairnessKey: orgID.String(),
	}
}

func withPriority(options workflow.ActivityOptions, orgID pulid.ID) workflow.ActivityOptions {
	options.Priority = evaluationPriority(orgID)
	return options
}

func ExtractionEvalRunWorkflow(
	ctx workflow.Context,
	payload *RunPayload,
) (*RunOutcome, error) {
	input := RunInput{
		OrganizationID: payload.OrganizationID,
		BusinessUnitID: payload.BusinessUnitID,
		RunID:          payload.RunID,
	}
	bookCtx := workflow.WithActivityOptions(ctx, withPriority(bookkeepingOptions, payload.OrganizationID))
	evalCtx := workflow.WithActivityOptions(ctx, withPriority(evaluateOptions, payload.OrganizationID))

	var a *Activities
	if payload.AfterOrdinal == 0 {
		if err := workflow.ExecuteActivity(bookCtx, a.BeginExtractionEvalRunActivity, &input).Get(bookCtx, nil); err != nil {
			return nil, failRun(bookCtx, &input, "The evaluation could not be started", err)
		}
	}

	after := payload.AfterOrdinal
	evaluated := 0
	for {
		var page []PendingCase
		if err := workflow.ExecuteActivity(bookCtx, a.ListPendingExtractionEvalCasesActivity, &ListPendingInput{
			RunInput:     input,
			AfterOrdinal: after,
			Limit:        pendingPageSize,
		}).Get(bookCtx, &page); err != nil {
			return nil, failRun(bookCtx, &input, "The evaluation's cases could not be read", err)
		}
		if len(page) == 0 {
			break
		}

		for _, pending := range page {
			if evaluated >= casesPerExecution {
				return nil, workflow.NewContinueAsNewError(ctx, ExtractionEvalRunWorkflow, &RunPayload{
					OrganizationID: payload.OrganizationID,
					BusinessUnitID: payload.BusinessUnitID,
					RunID:          payload.RunID,
					AfterOrdinal:   after,
					Evaluated:      payload.Evaluated + evaluated,
				})
			}

			var decision ContinueDecision
			if err := workflow.ExecuteActivity(bookCtx, a.CheckExtractionEvalBudgetActivity, &input).
				Get(bookCtx, &decision); err != nil {
				return nil, failRun(bookCtx, &input, "The evaluation budget could not be checked", err)
			}
			if decision.Stop {
				return finishRun(bookCtx, &input, decision.Status, decision.Reason)
			}

			if err := workflow.ExecuteActivity(evalCtx, a.EvaluateExtractionEvalCaseActivity, &EvaluateInput{
				RunInput: input,
				ResultID: pending.ResultID,
			}).Get(evalCtx, nil); err != nil {
				return nil, failRun(bookCtx, &input, "A case could not be evaluated or saved", err)
			}

			after = pending.Ordinal
			evaluated++
		}
	}

	return finishRun(bookCtx, &input, extractioneval.RunStatusCompleted.String(), "")
}

func finishRun(
	ctx workflow.Context,
	input *RunInput,
	status, reason string,
) (*RunOutcome, error) {
	var a *Activities
	var outcome RunOutcome
	if err := workflow.ExecuteActivity(ctx, a.FinishExtractionEvalRunActivity, &FinishInput{
		RunInput: *input,
		Status:   status,
		Reason:   reason,
	}).Get(ctx, &outcome); err != nil {
		return nil, failRun(ctx, input, "The evaluation's results could not be totalled", err)
	}

	return &outcome, nil
}

func failRun(ctx workflow.Context, input *RunInput, message string, cause error) error {
	var a *Activities
	if err := workflow.ExecuteActivity(ctx, a.FailExtractionEvalRunActivity, &FailInput{
		RunInput: *input,
		Message:  message,
	}).Get(ctx, nil); err != nil {
		workflow.GetLogger(ctx).Error("could not mark the extraction evaluation failed",
			"runId", input.RunID.String(),
			"error", err,
		)
	}

	return cause
}
