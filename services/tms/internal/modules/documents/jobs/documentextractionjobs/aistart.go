package documentextractionjobs

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentanalysis"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
)

func (a *Activities) startAIExtractionWorkflow(
	ctx context.Context,
	doc *document.Document,
	userID pulid.ID,
	extractedAt int64,
) error {
	if a.workflowStarter == nil || !a.workflowStarter.Enabled() {
		return kernelports.ErrWorkflowStarterDisabled
	}

	_, err := a.workflowStarter.StartWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID: fmt.Sprintf(
				"document-ai-extraction-%s-%d",
				doc.ID.String(),
				extractedAt,
			),
			TaskQueue:                                temporaltype.DocumentExtractionTaskQueue,
			WorkflowExecutionErrorWhenAlreadyStarted: true,
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY,
			StaticSummary: fmt.Sprintf(
				"Running AI extraction for document %s",
				doc.ID,
			),
		},
		"ProcessDocumentAIExtractionWorkflow",
		&documentanalysis.ProcessDocumentAIExtractionPayload{
			BasePayload: temporaltype.BasePayload{
				OrganizationID: doc.OrganizationID,
				BusinessUnitID: doc.BusinessUnitID,
				UserID:         userID,
			},
			DocumentID:  doc.ID,
			ExtractedAt: extractedAt,
		},
	)
	if err != nil {
		var alreadyStarted *serviceerror.WorkflowExecutionAlreadyStarted
		if errors.As(err, &alreadyStarted) {
			return nil
		}
	}

	return err
}
