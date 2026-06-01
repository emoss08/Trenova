package billingjobs

import (
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	AutoPostInvoiceWorkflowName    = "AutoPostInvoiceWorkflow"
	SendInvoiceEmailWorkflowName   = "SendInvoiceEmailWorkflow"
	GenerateInvoicePDFWorkflowName = "GenerateInvoicePDFWorkflow"
)

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

type SendInvoiceEmailPayload struct {
	temporaltype.BasePayload
	InvoiceID     pulid.ID               `json:"invoiceId"`
	BaseURL       string                 `json:"baseUrl"`
	PrincipalType services.PrincipalType `json:"principalType"`
	PrincipalID   pulid.ID               `json:"principalId"`
	APIKeyID      pulid.ID               `json:"apiKeyId"`
}

type SendInvoiceEmailResult struct {
	InvoiceID   pulid.ID `json:"invoiceId"`
	SendStatus  string   `json:"sendStatus"`
	Attempts    int      `json:"attempts"`
	CompletedAt int64    `json:"completedAt"`
}

type GenerateInvoicePDFPayload struct {
	temporaltype.BasePayload
	InvoiceID     pulid.ID               `json:"invoiceId"`
	BaseURL       string                 `json:"baseUrl"`
	PrincipalType services.PrincipalType `json:"principalType"`
	PrincipalID   pulid.ID               `json:"principalId"`
	APIKeyID      pulid.ID               `json:"apiKeyId"`
}

type PrepareInvoicePDFUploadResult struct {
	InvoiceID pulid.ID `json:"invoiceId"`
	SessionID pulid.ID `json:"sessionId"`
}

type GenerateInvoicePDFResult struct {
	InvoiceID   pulid.ID `json:"invoiceId"`
	DocumentID  pulid.ID `json:"documentId"`
	CompletedAt int64    `json:"completedAt"`
}
