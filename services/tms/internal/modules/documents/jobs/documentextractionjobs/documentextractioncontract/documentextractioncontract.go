package documentextractioncontract

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

const (
	ProcessWorkflowName   = "ProcessDocumentExtractionWorkflow"
	ReconcileWorkflowName = "ReconcileDocumentExtractionWorkflow"
)

type ProcessDocumentExtractionPayload struct {
	temporaltype.BasePayload

	DocumentID pulid.ID `json:"documentId"`
}

func WorkflowID(documentID pulid.ID) string {
	return "document-intelligence-" + documentID.String()
}

func Start(
	ctx context.Context,
	starter kernelports.WorkflowStarter,
	payload *ProcessDocumentExtractionPayload,
) error {
	_, err := starter.StartWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       WorkflowID(payload.DocumentID),
			TaskQueue:                                temporaltype.DocumentExtractionTaskQueue,
			WorkflowExecutionErrorWhenAlreadyStarted: true,
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
			WorkflowIDConflictPolicy:                 enums.WORKFLOW_ID_CONFLICT_POLICY_FAIL,
			StaticSummary:                            "Extracting content for document " + payload.DocumentID.String(),
		},
		ProcessWorkflowName,
		payload,
	)

	return err
}
