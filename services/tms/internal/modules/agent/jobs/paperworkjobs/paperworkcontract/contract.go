package paperworkcontract

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentflow/agentflowcontract"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const ReadWorkflowName = "PaperworkReadWorkflow"

func WorkflowID(documentID pulid.ID) string {
	return "paperwork-read:" + documentID.String()
}

type ReadInput struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	DocumentID pulid.ID              `json:"documentId"`
	UserID     pulid.ID              `json:"userId"`
}

type RequesterParams struct {
	fx.In

	Workflows kernelports.WorkflowStarter `optional:"true"`
	Logger    *zap.Logger
}

type Requester struct {
	workflows kernelports.WorkflowStarter
	l         *zap.Logger
}

var _ documentports.PaperworkReadRequester = (*Requester)(nil)

func NewRequester(p RequesterParams) documentports.PaperworkReadRequester {
	return &Requester{workflows: p.Workflows, l: p.Logger.Named("job.paperwork-read")}
}

// RequestRead starts Nova reading the document once its upload commits. A
// document already being read is not read twice.
func (r *Requester) RequestRead(ctx context.Context, doc *document.Document, userID pulid.ID) {
	if r.workflows == nil || doc == nil ||
		doc.ProcessingProfile != document.ProcessingProfileShipmentPaperwork {
		return
	}

	tenant := pagination.TenantInfo{OrgID: doc.OrganizationID, BuID: doc.BusinessUnitID}
	kernelports.AfterCommit(ctx, func(committed context.Context) {
		if _, err := r.workflows.StartWorkflow(
			committed,
			client.StartWorkflowOptions{
				ID:            WorkflowID(doc.ID),
				TaskQueue:     temporaltype.TaskQueueAgentChat.String(),
				StaticSummary: "Nova reads a shipment document",
				Priority: temporal.Priority{
					PriorityKey: agentflowcontract.PriorityOneShot,
					FairnessKey: tenant.OrgID.String(),
				},
			},
			ReadWorkflowName,
			&ReadInput{TenantInfo: tenant, DocumentID: doc.ID, UserID: userID},
		); err != nil {
			r.l.Warn("could not start reading a shipment document",
				zap.String("documentId", doc.ID.String()), zap.Error(err))
		}
	})
}
