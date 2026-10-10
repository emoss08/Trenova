package billingjobs

import (
	"github.com/emoss08/trenova/shared/pulid"
)

const ()

type AutoPostInvoiceResult struct {
	InvoiceID     pulid.ID `json:"invoiceId"`
	PostedAt      int64    `json:"postedAt"`
	CompletedAt   int64    `json:"completedAt"`
	AlreadyPosted bool     `json:"alreadyPosted"`
	Voided        bool     `json:"voided"`
}

type SendInvoiceEmailResult struct {
	InvoiceID   pulid.ID `json:"invoiceId"`
	SendStatus  string   `json:"sendStatus"`
	Attempts    int      `json:"attempts"`
	CompletedAt int64    `json:"completedAt"`
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
