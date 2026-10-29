package tendereffects

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
)

// CopyTender copies the tender and each of its offers, applying change to
// every offer copy, so a projection never moves the rows it was read from.
func CopyTender(entity *tender.Tender, change func(*tender.TenderOffer)) *tender.Tender {
	out := *entity
	out.Offers = make([]*tender.TenderOffer, 0, len(entity.Offers))
	for _, offer := range entity.Offers {
		if offer == nil {
			continue
		}
		copied := *offer
		change(&copied)
		out.Offers = append(out.Offers, &copied)
	}

	return &out
}
