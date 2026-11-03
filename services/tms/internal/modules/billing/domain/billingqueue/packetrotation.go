package billingqueue

import (
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingvocab"
	"github.com/emoss08/trenova/shared/pulid"
)

type PacketPageRotation struct {
	DocumentID pulid.ID `json:"documentId"`
	Page       int      `json:"page"`
	Rotation   int      `json:"rotation"`
}

func (b *BillingQueueItem) PageRotationsFor(documentID pulid.ID) []billingvocab.PageRotation {
	out := make([]billingvocab.PageRotation, 0)
	for _, turn := range b.PacketPageRotations {
		if turn.DocumentID == documentID {
			out = append(out, billingvocab.PageRotation{Page: turn.Page, Rotation: turn.Rotation})
		}
	}
	return out
}
