package triageworkjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentuploadjobs"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentuploadjobs/documentuploadcontract"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/triageworkjobs/triageworkcontract"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const draftConcurrency = 4

var draftOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 3 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 2,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    3,
		NonRetryableErrorTypes: []string{
			temporaltype.ErrorTypeInvalidInput.String(),
			temporaltype.ErrorTypeNonRetryable.String(),
			temporaltype.ErrorTypePermissionDenied.String(),
		},
	},
}

var recordOptions = workflow.ActivityOptions{
	StartToCloseTimeout: time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    2 * time.Second,
		BackoffCoefficient: 2,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    10,
	},
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        triageworkcontract.DraftManyWorkflowName,
			Fn:          DraftManyWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Has Nova draft replies and loads for many Triage messages",
		},
		{
			Name:        triageworkcontract.SplitFilingWorkflowName,
			Fn:          SplitFilingWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Files page ranges of an email attachment as separate documents",
		},
	}
}

func DraftManyWorkflow(
	ctx workflow.Context,
	payload *triageworkcontract.DraftManyPayload,
) (*triageworkcontract.DraftManyResult, error) {
	ctx = workflow.WithActivityOptions(ctx, draftOptions)

	var a *Activities
	result := new(triageworkcontract.DraftManyResult)
	for start := 0; start < len(payload.MessageIDs); start += draftConcurrency {
		end := min(start+draftConcurrency, len(payload.MessageIDs))
		futures := make([]workflow.Future, 0, end-start)
		for _, messageID := range payload.MessageIDs[start:end] {
			futures = append(futures, workflow.ExecuteActivity(
				ctx,
				a.DraftTriageMessageActivity,
				&triageworkcontract.DraftOnePayload{
					BasePayload: payload.BasePayload,
					UserID:      payload.UserID,
					MessageID:   messageID,
				},
			))
		}

		for i, future := range futures {
			var drafted bool
			if err := future.Get(ctx, &drafted); err != nil {
				workflow.GetLogger(ctx).Warn("a Triage draft failed",
					"messageId", payload.MessageIDs[start+i].String(), "error", err)
				result.Failed++

				continue
			}
			if drafted {
				result.Drafted++
			}
		}
	}

	return result, nil
}

func SplitFilingWorkflow(
	ctx workflow.Context,
	payload *triageworkcontract.SplitFilingPayload,
) error {
	var a *Activities

	record := &triageworkcontract.RecordSplitPayload{
		SplitFilingPayload: *payload,
		Documents:          make([]triageworkcontract.SplitDocument, 0, len(payload.Ranges)),
	}
	for _, rng := range payload.Ranges {
		childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			WorkflowID: "document-upload-finalize-" + rng.SessionID.String(),
			TaskQueue:  temporaltype.UploadTaskQueue,
		})

		finalized := new(documentuploadjobs.FinalizeUploadResult)
		err := workflow.ExecuteChildWorkflow(
			childCtx,
			documentuploadjobs.FinalizeDocumentUploadWorkflow,
			&documentuploadcontract.FinalizeUploadPayload{
				BasePayload:   payload.BasePayload,
				SessionID:     rng.SessionID,
				PrincipalType: permissionports.PrincipalTypeUser,
				PrincipalID:   payload.UserID,
			},
		).Get(ctx, finalized)
		if err != nil || finalized.DocumentID == nil || finalized.DocumentID.IsNil() {
			workflow.GetLogger(ctx).Error("a split document could not be finalized",
				"attachmentId", payload.AttachmentID.String(),
				"sessionId", rng.SessionID.String(),
				"error", err)
			record.Failed++

			continue
		}

		record.Documents = append(record.Documents, triageworkcontract.SplitDocument{
			SplitRange: rng,
			DocumentID: *finalized.DocumentID,
		})
	}

	recordCtx := workflow.WithActivityOptions(ctx, recordOptions)

	return workflow.ExecuteActivity(recordCtx, a.RecordTriageSplitActivity, record).Get(recordCtx, nil)
}
