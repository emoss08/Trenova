package tendereffects

import (
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenteventservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
)

// HashOfferToken derives the stored lookup hash for a raw token.

func CarrierNameOf(offer *tender.TenderOffer) string {
	if offer == nil || offer.Carrier == nil {
		return "the carrier"
	}
	return offer.Carrier.Name
}

func TenderRefForOffer(offer *tender.TenderOffer) shipmenteventservice.TenderRef {
	ref := shipmenteventservice.TenderRef{
		TenderID: offer.TenderID,
		OfferID:  offer.ID,
	}
	if offer.Tender != nil {
		ref.ShipmentID = offer.Tender.ShipmentID
		ref.MoveID = offer.Tender.ShipmentMoveID
	}
	return ref
}
