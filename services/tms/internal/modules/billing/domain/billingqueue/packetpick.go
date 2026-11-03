package billingqueue

import "github.com/emoss08/trenova/shared/pulid"

func (b *BillingQueueItem) PickPacket(documentIDs []pulid.ID, picked bool) {
	if documentIDs == nil {
		documentIDs = []pulid.ID{}
	}
	b.PacketDocumentIDs = documentIDs
	b.PacketPicked = picked || len(documentIDs) > 0
}

func (b *BillingQueueItem) PickedPacket() ([]pulid.ID, bool) {
	if !b.PacketPicked {
		return nil, false
	}
	return b.PacketDocumentIDs, true
}
