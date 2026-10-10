package billingcontract

import (
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

const SendInvoiceEDIWorkflowName = "SendInvoiceEDIWorkflow"

type SendInvoiceEDIPayload struct {
	temporaltype.BasePayload
	InvoiceID     pulid.ID               `json:"invoiceId"`
	Force         bool                   `json:"force"`
	PrincipalType services.PrincipalType `json:"principalType"`
	PrincipalID   pulid.ID               `json:"principalId"`
	APIKeyID      pulid.ID               `json:"apiKeyId"`
}

const AutoPostInvoiceWorkflowName = "AutoPostInvoiceWorkflow"

const SendInvoiceEmailWorkflowName = "SendInvoiceEmailWorkflow"

const GenerateInvoicePDFWorkflowName = "GenerateInvoicePDFWorkflow"

type AutoPostInvoicePayload struct {
	temporaltype.BasePayload
	InvoiceID     pulid.ID               `json:"invoiceId"`
	PrincipalType services.PrincipalType `json:"principalType"`
	PrincipalID   pulid.ID               `json:"principalId"`
	APIKeyID      pulid.ID               `json:"apiKeyId"`
}

type SendInvoiceEmailPayload struct {
	temporaltype.BasePayload
	InvoiceID     pulid.ID               `json:"invoiceId"`
	BaseURL       string                 `json:"baseUrl"`
	PrincipalType services.PrincipalType `json:"principalType"`
	PrincipalID   pulid.ID               `json:"principalId"`
	APIKeyID      pulid.ID               `json:"apiKeyId"`
}

type GenerateInvoicePDFPayload struct {
	temporaltype.BasePayload
	InvoiceID     pulid.ID               `json:"invoiceId"`
	BaseURL       string                 `json:"baseUrl"`
	PrincipalType services.PrincipalType `json:"principalType"`
	PrincipalID   pulid.ID               `json:"principalId"`
	APIKeyID      pulid.ID               `json:"apiKeyId"`
}
