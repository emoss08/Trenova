package billingcontract

import (
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

const SendInvoiceEDIWorkflowName = "SendInvoiceEDIWorkflow"

type SendInvoiceEDIPayload struct {
	temporaltype.BasePayload
	InvoiceID     pulid.ID                  `json:"invoiceId"`
	Force         bool                      `json:"force"`
	PrincipalType kernelports.PrincipalType `json:"principalType"`
	PrincipalID   pulid.ID                  `json:"principalId"`
	APIKeyID      pulid.ID                  `json:"apiKeyId"`
}

const AutoPostInvoiceWorkflowName = "AutoPostInvoiceWorkflow"

const SendInvoiceEmailWorkflowName = "SendInvoiceEmailWorkflow"

const GenerateInvoicePDFWorkflowName = "GenerateInvoicePDFWorkflow"

type AutoPostInvoicePayload struct {
	temporaltype.BasePayload
	InvoiceID     pulid.ID                  `json:"invoiceId"`
	PrincipalType kernelports.PrincipalType `json:"principalType"`
	PrincipalID   pulid.ID                  `json:"principalId"`
	APIKeyID      pulid.ID                  `json:"apiKeyId"`
}

// AutoPostFailurePayload reports why an automatic post gave up, so the person
// who approved the invoice learns it is still a draft.
type AutoPostFailurePayload struct {
	AutoPostInvoicePayload
	Reason string `json:"reason"`
}

type SendInvoiceEmailPayload struct {
	temporaltype.BasePayload
	InvoiceID     pulid.ID                  `json:"invoiceId"`
	BaseURL       string                    `json:"baseUrl"`
	PrincipalType kernelports.PrincipalType `json:"principalType"`
	PrincipalID   pulid.ID                  `json:"principalId"`
	APIKeyID      pulid.ID                  `json:"apiKeyId"`
}

type GenerateInvoicePDFPayload struct {
	temporaltype.BasePayload
	InvoiceID     pulid.ID                  `json:"invoiceId"`
	BaseURL       string                    `json:"baseUrl"`
	PrincipalType kernelports.PrincipalType `json:"principalType"`
	PrincipalID   pulid.ID                  `json:"principalId"`
	APIKeyID      pulid.ID                  `json:"apiKeyId"`
}
