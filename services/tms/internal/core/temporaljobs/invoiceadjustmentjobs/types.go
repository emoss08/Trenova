package invoiceadjustmentjobs

import (
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
)

const InvoiceAdjustmentBatchWorkflowName = "InvoiceAdjustmentBatchWorkflow"

type BatchWorkflowPayload struct {
	BatchID         pulid.ID   `json:"batchId"`
	ItemIDs         []pulid.ID `json:"itemIds"`
	OrganizationID  pulid.ID   `json:"organizationId"`
	BusinessUnitID  pulid.ID   `json:"businessUnitId"`
	UserID          pulid.ID   `json:"userId"`
	PrincipalType   servicesports.PrincipalType `json:"principalType"`
	PrincipalID     pulid.ID   `json:"principalId"`
	APIKeyID        pulid.ID   `json:"apiKeyId"`
	WorkflowStarted int64      `json:"workflowStarted"`
}

type ProcessBatchItemPayload struct {
	BatchID        pulid.ID `json:"batchId"`
	ItemID         pulid.ID `json:"itemId"`
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	UserID         pulid.ID `json:"userId"`
	PrincipalType  servicesports.PrincipalType `json:"principalType"`
	PrincipalID    pulid.ID `json:"principalId"`
	APIKeyID       pulid.ID `json:"apiKeyId"`
}

type ProcessBatchItemResult struct {
	ItemID        pulid.ID `json:"itemId"`
	AdjustmentID  pulid.ID `json:"adjustmentId"`
	FinalStatus   string   `json:"finalStatus"`
	ErrorMessage  string   `json:"errorMessage"`
	ProcessedAt   int64    `json:"processedAt"`
}
