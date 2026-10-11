package invoiceadjustmentcontract

import (
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
)

const InvoiceAdjustmentBatchWorkflowName = "InvoiceAdjustmentBatchWorkflow"

type BatchWorkflowPayload struct {
	BatchID         pulid.ID                  `json:"batchId"`
	ItemIDs         []pulid.ID                `json:"itemIds"`
	OrganizationID  pulid.ID                  `json:"organizationId"`
	BusinessUnitID  pulid.ID                  `json:"businessUnitId"`
	UserID          pulid.ID                  `json:"userId"`
	PrincipalType   kernelports.PrincipalType `json:"principalType"`
	PrincipalID     pulid.ID                  `json:"principalId"`
	APIKeyID        pulid.ID                  `json:"apiKeyId"`
	WorkflowStarted int64                     `json:"workflowStarted"`
}
