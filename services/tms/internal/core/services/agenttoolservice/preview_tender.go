package agenttoolservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/tenderservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/shared/money"
	"github.com/shopspring/decimal"
)

var (
	_ serviceports.ToolPreviewer = (*tenderToRoutingGuideTool)(nil)
	_ serviceports.ToolPreviewer = (*tenderToCarriersTool)(nil)
)

func (t *tenderToRoutingGuideTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolPreview, error) {
	request, err := t.request(params)
	if err != nil {
		return nil, err
	}

	planned, err := t.tenders.PreviewWaterfall(ctx, request)
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf(
		"Would offer the move down routing guide %s to %s, one at a time in this order.",
		planned.Guide.Name,
		countOf(len(planned.Tender.Offers), "carrier"),
	)
	if screening := planned.Screening; screening != nil {
		summary += screeningSentence("Skipped as ineligible", screening.Skipped)
		summary += screeningSentence("Offered despite a warning", screening.Warned)
	}

	return tenderPreview(summary, planned)
}

func (t *tenderToCarriersTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolPreview, error) {
	request, err := t.request(params)
	if err != nil {
		return nil, err
	}

	planned, err := t.tenders.PreviewSpot(ctx, request)
	if err != nil {
		return nil, err
	}

	order := "all at once"
	if request.Mode == tender.ModeSpotSequential {
		order = "one at a time in this order"
	}
	summary := fmt.Sprintf("Would offer the move to %s, %s.",
		countOf(len(planned.Tender.Offers), "carrier"), order)
	if len(planned.Warnings) > 0 {
		summary += " Eligibility warnings: " + strings.Join(planned.Warnings, "; ") + "."
	}

	preview, err := tenderPreview(summary, planned)
	if err != nil {
		return nil, err
	}
	if planned.RefusalError != nil {
		toolpreview.Warn(preview, agent.PreviewWarningWouldFail, planned.RefusalError.Error())
	}

	return preview, nil
}

// tenderPreview is a tender as creating it would start it: the tender, with
// each offered rate as a line, and the offer each carrier would be sent.
func tenderPreview(
	summary string,
	planned *tenderservice.TenderPreview,
) (*agent.ToolPreview, error) {
	entity := planned.Tender
	created, err := toolpreview.Create(
		toolpreview.Record{Resource: permission.ResourceTender, Label: tenderLabel(planned)},
		entity,
		toolpreview.Only("mode", "status", "shipmentMoveId", "routingGuideId"),
		toolpreview.WithRefs(map[string]permission.Resource{
			"shipmentMoveId": permission.ResourceShipmentMove,
			"routingGuideId": permission.ResourceRoutingGuide,
		}),
	)
	if err != nil {
		return nil, err
	}

	lines := make([]agent.MoneyLine, 0, len(entity.Offers))
	for _, offer := range entity.Offers {
		lines = append(lines, agent.MoneyLine{
			Label: fmt.Sprintf("%d. %s (%s)",
				offer.Rank, offerCarrierName(planned, offer), rateMethodWords(offer.RateMethod)),
			After: knownAmount(offer.Rate),
		})
	}
	block := toolpreview.MoneyBlock(money.DefaultCurrencyCode, lines...)
	block.TotalBefore = decimal.NullDecimal{}
	block.TotalAfter = decimal.NullDecimal{}
	block.Delta = decimal.NullDecimal{}
	toolpreview.AttachMoney(created, block)

	changes := make([]*agent.RecordChange, 0, len(entity.Offers)+1)
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
) *agent.RecordChange {
	message := &agent.MessagePreview{
		Subject: "Load tender offer",
		Body: fmt.Sprintf("%s %s offered, to be accepted within %s.",
			money.FormatMinor(money.MinorUnits(offer.Rate), money.DefaultCurrencyCode),
			rateMethodWords(offer.RateMethod),
			time.Duration(offer.OfferTTLSeconds)*time.Second),
	}
	if sequential(planned.Tender.Mode) && index > 0 {
		message.Body += " Sent only if every carrier before it declines or lets the offer lapse."
	}

	switch offer.Channel {
	case tender.ChannelEDI:
		message.Channel = agent.MessageChannelEDI
		message.To = []string{offerCarrierName(planned, offer) + " (EDI 204)"}
	case tender.ChannelEmail:
		message.Channel = agent.MessageChannelEmail
		message.To = []string{offer.RecipientEmail}
	default:
		message.Channel = agent.MessageChannelEmail
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

	return "Carrier"
}

func tenderLabel(planned *tenderservice.TenderPreview) string {
	if planned.Guide != nil {
		return "Routing guide tender: " + planned.Guide.Name
	}

	return string(planned.Tender.Mode) + " tender"
}

func rateMethodWords(method shipment.CarrierRateMethod) string {
	if method == shipment.CarrierRateMethodPerMile {
		return "per mile"
	}

	return "flat"
}

func screeningSentence(lead string, entries []tenderservice.GuideEntryScreeningResult) string {
	if len(entries) == 0 {
		return ""
	}

	parts := make([]string, 0, len(entries))
	for _, entry := range entries {
		parts = append(parts, entry.CarrierName+" ("+strings.Join(entry.Reasons, "; ")+")")
	}

	return " " + lead + ": " + strings.Join(parts, ", ") + "."
}
