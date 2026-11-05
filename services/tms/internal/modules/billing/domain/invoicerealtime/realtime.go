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
	ID             pulid.ID `json:"id"`
	NotifiedUserID pulid.ID `json:"notifiedUserId,omitempty"`
}

func NewRealtimeSnapshot(entity *invoice.Invoice) *RealtimeSnapshot {
	if entity == nil {
		return nil
	}
	return &RealtimeSnapshot{ID: entity.ID}
}
