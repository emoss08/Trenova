package invoicerealtime

import (
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	RealtimeActionUpdated      = "updated"
	RealtimeActionPDFGenerated = "pdf.generated"
	RealtimeActionSendUpdated  = "send.updated"
	RealtimeActionEDIUpdated   = "edi.updated"
	RealtimeActionPostFailed   = "post.failed"
)

type RealtimeSnapshot struct {
	ID                 pulid.ID              `json:"id"`
	Number             string                `json:"number"`
	CustomerID         pulid.ID              `json:"customerId"`
	BillingQueueItemID pulid.ID              `json:"billingQueueItemId"`
	Status             invoice.Status        `json:"status"`
	SendStatus         invoice.SendStatus    `json:"sendStatus"`
	LastSendError      string                `json:"lastSendError,omitempty"`
	PDFDocumentID      pulid.ID              `json:"pdfDocumentId,omitempty"`
	EDISendStatus      invoice.EDISendStatus `json:"ediSendStatus"`
	PostError          string                `json:"postError,omitempty"`
	NotifiedUserID     pulid.ID              `json:"notifiedUserId,omitempty"`
	UpdatedAt          int64                 `json:"updatedAt"`
}

func NewRealtimeSnapshot(entity *invoice.Invoice) *RealtimeSnapshot {
	if entity == nil {
		return nil
	}
	return &RealtimeSnapshot{
		ID:                 entity.ID,
		Number:             entity.Number,
		CustomerID:         entity.CustomerID,
		BillingQueueItemID: entity.BillingQueueItemID,
		Status:             entity.Status,
		SendStatus:         entity.SendStatus,
		LastSendError:      entity.LastSendError,
		PDFDocumentID:      entity.PDFDocumentID,
		EDISendStatus:      entity.EDISendStatus,
		UpdatedAt:          entity.UpdatedAt,
	}
}
