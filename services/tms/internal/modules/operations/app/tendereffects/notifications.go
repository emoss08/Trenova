package tendereffects

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	notificationSource       = "tender_service"
	NotificationPriorityHigh = notification.PriorityHigh
	NotificationPriorityMed  = notification.PriorityMedium

	dispatchConsoleLink = "/dispatch/console"
)

type DispatchNotification struct {
	EventType     string
	Priority      notification.Priority
	Title         string
	Message       string
	CorrelationID string
	Link          string
}

func DispatchConsoleMoveLink(moveID pulid.ID) string {
	if moveID.IsNil() {
		return dispatchConsoleLink
	}
	return dispatchConsoleLink + "?move=" + moveID.String()
}

func DispatchConsoleOfferLink(offer *tender.TenderOffer) string {
	if offer == nil || offer.Tender == nil {
		return dispatchConsoleLink
	}
	return DispatchConsoleMoveLink(offer.Tender.ShipmentMoveID)
}
