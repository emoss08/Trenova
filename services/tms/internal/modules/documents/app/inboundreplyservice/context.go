package inboundreplyservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/app/triagenova"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
)

const (
	minutesPerHour      = 60
	halfHourMinutes     = 30
	etaTimeLayout       = "15:04"
	etaDayLayout        = "Mon Jan 2, 15:04"
	statusDelivered     = "delivered"
	statusOnSchedule    = "on schedule"
	shipmentStatusDone  = "Completed"
	shipmentStatusReady = "ReadyToInvoice"
	shipmentInvoiced    = "Invoiced"
)

type messageContext struct {
	message     *inboundmessage.InboundMessage
	shipment    *inboundmessageports.ShipmentPeek
	reply       inboundmessageports.ReplyContext
	stops       *inboundmessageports.StopTimes
	location    *time.Location
	orgName     string
	attachments []triage.ReplyAttachment
	evidence    bool
}

func (s *Service) messageContext(
	ctx context.Context,
	message *inboundmessage.InboundMessage,
) *messageContext {
	tenantInfo := tenantOf(message)
	out := &messageContext{message: message, location: time.UTC}
	s.loadOrganization(ctx, tenantInfo, out)

	if message.MatchedShipmentID.IsNotNil() && s.peeks != nil {
		peeks, err := s.peeks.ShipmentPeeks(ctx, tenantInfo, []pulid.ID{message.MatchedShipmentID})
		if err != nil {
			s.l.Warn("could not read the matched shipment for a reply",
				zap.String("messageId", message.ID.String()), zap.Error(err))
		} else {
			out.shipment = peeks[message.MatchedShipmentID]
		}
	}

	out.reply = replyContextFor(message, out.shipment, out.location, s.now())

	return out
}

func (s *Service) loadOrganization(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	out *messageContext,
) {
	org, location, err := triagenova.Organization(ctx, s.organizations, tenantInfo)
	if err != nil {
		s.l.Warn("could not read the organization for a reply", zap.Error(err))

		return
	}
	out.orgName = org.Name
	out.location = location
}

func replyContextFor(
	message *inboundmessage.InboundMessage,
	shipment *inboundmessageports.ShipmentPeek,
	location *time.Location,
	now int64,
) inboundmessageports.ReplyContext {
	out := inboundmessageports.ReplyContext{
		SenderFirstName: stringutils.FirstName(message.FromName),
		SenderAddress:   message.FromAddress,
	}
	if shipment == nil {
		return out
	}

	out.ProNumber = shipment.ProNumber
	out.PONumber = shipment.PONumber
	out.CustomerName = shipment.CustomerName
	out.Status = shipmentStatusPhrase(shipment)
	out.ETA = etaPhrase(shipment, location, now)
	out.Lane = lanePhrase(shipment)

	return out
}

func shipmentStatusPhrase(shipment *inboundmessageports.ShipmentPeek) string {
	if shipment.DeliveredAt != nil || isDeliveredStatus(shipment.Status) {
		return statusDelivered
	}
	if shipment.Late && shipment.LateMinutes > 0 {
		return "running " + behindPhrase(shipment.LateMinutes) + " behind"
	}

	return statusOnSchedule
}

func isDeliveredStatus(status string) bool {
	switch status {
	case shipmentStatusDone, shipmentStatusReady, shipmentInvoiced:
		return true
	default:
		return false
	}
}

func behindPhrase(minutes int) string {
	if minutes < minutesPerHour {
		return "about " + stringutils.CountNoun(minutes, "minute", "minutes")
	}

	hours := (minutes + halfHourMinutes) / minutesPerHour
	unit := "hours"
	if hours == 1 {
		unit = "hour"
	}

	return fmt.Sprintf("about %s %s", intutils.SmallNumberWord(hours), unit)
}

func etaPhrase(
	shipment *inboundmessageports.ShipmentPeek,
	location *time.Location,
	now int64,
) string {
	at := shipment.ETA
	if shipment.DeliveredAt != nil {
		at = shipment.DeliveredAt
	}
	if at == nil || *at <= 0 {
		return ""
	}

	when := time.Unix(*at, 0).In(location)
	today := time.Unix(now, 0).In(location)
	if when.YearDay() == today.YearDay() && when.Year() == today.Year() {
		return when.Format(etaTimeLayout) + " today"
	}
	tomorrow := today.AddDate(0, 0, 1)
	if when.YearDay() == tomorrow.YearDay() && when.Year() == tomorrow.Year() {
		return when.Format(etaTimeLayout) + " tomorrow"
	}

	return when.Format(etaDayLayout)
}

func lanePhrase(shipment *inboundmessageports.ShipmentPeek) string {
	if len(shipment.Stops) == 0 {
		return ""
	}

	origin := shipment.Stops[0]
	destination := shipment.Stops[len(shipment.Stops)-1]
	from := strings.TrimSpace(origin.City)
	to := strings.TrimSpace(destination.City)
	if from == "" || to == "" {
		return ""
	}

	return from + " → " + to
}
