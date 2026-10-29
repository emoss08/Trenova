package masterdatatools

import (
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/carrierpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/shared/money"
	"github.com/shopspring/decimal"
)

// tenderPreview is a tender as creating it would start it: the tender, with
// each offered rate as a line, and the offer each carrier would be sent.
func TenderPreview(
	summary string,
	planned *tenderservice.TenderPreview,
) (*agentpreview.ToolPreview, error) {
	entity := planned.Tender
	created, err := toolpreview.Create(
		toolpreview.Record{Resource: permission.ResourceTender, Label: tenderLabel(planned)},
		entity,
		toolpreview.Only(
			"mode",
			"status",
			changepreview.PreviewFieldShipmentMoveID,
			"routingGuideId",
		),
		toolpreview.WithRefs(map[string]permission.Resource{
			changepreview.PreviewFieldShipmentMoveID: permission.ResourceShipmentMove,
			"routingGuideId":                         permission.ResourceRoutingGuide,
		}),
	)
	if err != nil {
		return nil, err
	}

	lines := make([]agentpreview.MoneyLine, 0, len(entity.Offers))
	for _, offer := range entity.Offers {
		lines = append(lines, agentpreview.MoneyLine{
			Label: fmt.Sprintf(
				"%d. %s (%s)",
				offer.Rank,
				offerCarrierName(planned, offer),
				carrierpreview.RateMethodWords(offer.RateMethod),
			),
			After: changepreview.KnownAmount(offer.Rate),
		})
	}
	block := toolpreview.MoneyBlock(money.DefaultCurrencyCode, lines...)
	block.TotalBefore = decimal.NullDecimal{}
	block.TotalAfter = decimal.NullDecimal{}
	block.Delta = decimal.NullDecimal{}
	toolpreview.AttachMoney(created, block)

	changes := make([]*agentpreview.RecordChange, 0, len(entity.Offers)+1)
	changes = append(changes, created)
	for index, offer := range entity.Offers {
		changes = append(changes, tenderOfferSend(planned, offer, index))
	}

	return toolpreview.Build(summary, changes...), nil
}

// tenderOfferSend is the offer one carrier would receive: by email to the
// address on file or the one the call names, or over the carrier's EDI link.
func tenderOfferSend(
	planned *tenderservice.TenderPreview,
	offer *tender.TenderOffer,
	index int,
) *agentpreview.RecordChange {
	message := &agentpreview.MessagePreview{
		Subject: "Load tender offer",
		Body: fmt.Sprintf("%s %s offered, to be accepted within %s.",
			money.FormatMinor(money.MinorUnits(offer.Rate), money.DefaultCurrencyCode),
			carrierpreview.RateMethodWords(offer.RateMethod),
			time.Duration(offer.OfferTTLSeconds)*time.Second),
	}
	if sequential(planned.Tender.Mode) && index > 0 {
		message.Body += " Sent only if every carrier before it declines or lets the offer lapse."
	}

	switch offer.Channel {
	case tender.ChannelEDI:
		message.Channel = agentpreview.MessageChannelEDI
		message.To = []string{offerCarrierName(planned, offer) + " (EDI 204)"}
	case tender.ChannelEmail:
		message.Channel = agentpreview.MessageChannelEmail
		message.To = []string{offer.RecipientEmail}
	default:
		message.Channel = agentpreview.MessageChannelEmail
		message.To = []string{offer.RecipientEmail}
	}

	return toolpreview.Send(toolpreview.Record{
		Resource: permission.ResourceCarrier,
		ID:       offer.CarrierID,
		Label:    offerCarrierName(planned, offer),
	}, message)
}

func sequential(mode tender.Mode) bool {
	return mode == tender.ModeWaterfall || mode == tender.ModeSpotSequential
}

func offerCarrierName(planned *tenderservice.TenderPreview, offer *tender.TenderOffer) string {
	if name := strings.TrimSpace(planned.Carriers[offer.CarrierID]); name != "" {
		return name
	}

	return changepreview.LabelCarrier
}

func tenderLabel(planned *tenderservice.TenderPreview) string {
	if planned.Guide != nil {
		return "Routing guide tender: " + planned.Guide.Name
	}

	return string(planned.Tender.Mode) + " tender"
}

func ScreeningSentence(lead string, entries []tenderservice.GuideEntryScreeningResult) string {
	if len(entries) == 0 {
		return ""
	}

	parts := make([]string, 0, len(entries))
	for _, entry := range entries {
		parts = append(parts, entry.CarrierName+" ("+strings.Join(entry.Reasons, "; ")+")")
	}

	return " " + lead + ": " + strings.Join(parts, ", ") + "."
}
