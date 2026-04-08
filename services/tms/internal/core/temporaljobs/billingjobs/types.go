package billingjobs

import (
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

const AutoPostInvoiceWorkflowName = "AutoPostInvoiceWorkflow"

type AutoPostInvoicePayload struct {
	temporaltype.BasePayload
	InvoiceID     pulid.ID               `json:"invoiceId"`
	PrincipalType services.PrincipalType `json:"principalType"`
	PrincipalID   pulid.ID               `json:"principalId"`
	APIKeyID      pulid.ID               `json:"apiKeyId"`
}

type AutoPostInvoiceResult struct {
	InvoiceID     pulid.ID `json:"invoiceId"`
	PostedAt      int64    `json:"postedAt"`
	CompletedAt   int64    `json:"completedAt"`
	AlreadyPosted bool     `json:"alreadyPosted"`
}
