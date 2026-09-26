package aitrainingjobs

import (
	"errors"
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var bookkeepingOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 2 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:        time.Second,
		BackoffCoefficient:     2.0,
		MaximumAttempts:        5,
		MaximumInterval:        30 * time.Second,
		NonRetryableErrorTypes: []string{errorTypeExportInactive},
	},
}

var organizationOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 2 * time.Hour,
	HeartbeatTimeout:    2 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:        5 * time.Second,
		BackoffCoefficient:     2.0,
		MaximumAttempts:        3,
		MaximumInterval:        time.Minute,
		NonRetryableErrorTypes: []string{errorTypeExportInactive},
	},
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        AITrainingExportWorkflowName,
			Fn:          AITrainingExportWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Write an anonymized model-training export from the AI corrections of every organization that consented",
		},
	}
}

func AITrainingExportWorkflow(ctx workflow.Context, payload *ExportPayload) (*ExportOutcome, error) {
	bookCtx := workflow.WithActivityOptions(ctx, bookkeepingOptions)
	orgCtx := workflow.WithActivityOptions(ctx, organizationOptions)
	input := ExportInput{ExportID: payload.ExportID}

	var a *Activities
	if payload.NextOrdinal == 0 {
		if err := workflow.ExecuteActivity(bookCtx, a.BeginAITrainingExportActivity, &input).
			Get(bookCtx, nil); err != nil {
			if inactive(err) {
				return finish(bookCtx, &input)
			}
			return nil, fail(bookCtx, &input, "The export could not be started", err)
		}
	}

	ordinal := max(payload.NextOrdinal, 1)
	afterOrg, afterBU := payload.AfterOrganizationID, payload.AfterBusinessUnitID
	processed := 0
	for {
		var page []ConsentingOrganization
		if err := workflow.ExecuteActivity(bookCtx, a.ListAITrainingOrganizationsActivity, &ListOrganizationsInput{
			AfterOrganizationID: afterOrg,
			AfterBusinessUnitID: afterBU,
			Limit:               organizationPageSize,
		}).Get(bookCtx, &page); err != nil {
			return nil, fail(bookCtx, &input, "The consenting organizations could not be read", err)
		}
		if len(page) == 0 {
			break
		}

		for _, organization := range page {
			if processed >= organizationsPerExecution {
				return nil, workflow.NewContinueAsNewError(ctx, AITrainingExportWorkflow, &ExportPayload{
					ExportID:            payload.ExportID,
					AfterOrganizationID: afterOrg,
					AfterBusinessUnitID: afterBU,
					NextOrdinal:         ordinal,
				})
			}

			var outcome ExportOrganizationOutcome
			if err := workflow.ExecuteActivity(orgCtx, a.ExportAITrainingOrganizationActivity, &ExportOrganizationInput{
				ExportID:     payload.ExportID,
				Ordinal:      ordinal,
				Organization: organization,
			}).Get(orgCtx, &outcome); err != nil {
				if inactive(err) {
					return finish(bookCtx, &input)
				}
				return nil, fail(bookCtx, &input, "An organization's examples could not be exported", err)
			}

			afterOrg, afterBU = organization.OrganizationID, organization.BusinessUnitID
			ordinal++
			processed++
		}
		if len(page) < organizationPageSize {
			break
		}
	}

	return finish(bookCtx, &input)
}

func finish(ctx workflow.Context, input *ExportInput) (*ExportOutcome, error) {
	var a *Activities
	var outcome ExportOutcome
	if err := workflow.ExecuteActivity(ctx, a.FinishAITrainingExportActivity, input).Get(ctx, &outcome); err != nil {
		return nil, fail(ctx, input, "The export's manifest could not be written", err)
	}

	return &outcome, nil
}

func fail(ctx workflow.Context, input *ExportInput, message string, cause error) error {
	var a *Activities
	if err := workflow.ExecuteActivity(ctx, a.FailAITrainingExportActivity, &FailInput{
		ExportID: input.ExportID,
		Message:  message,
	}).Get(ctx, nil); err != nil {
		workflow.GetLogger(ctx).Error("could not mark the training export failed",
			"exportId", input.ExportID.String(),
			"error", err,
		)
	}

	return cause
}

func inactive(err error) bool {
	var appErr *temporal.ApplicationError
	return errors.As(err, &appErr) && appErr.Type() == errorTypeExportInactive
}
