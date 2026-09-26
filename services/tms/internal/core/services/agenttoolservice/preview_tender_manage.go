package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
)

var (
	_ serviceports.ToolPreviewer = (*cancelTenderTool)(nil)
	_ serviceports.ToolPreviewer = (*recordTenderResponseTool)(nil)
	_ serviceports.ToolValidator = (*cancelTenderTool)(nil)
	_ serviceports.ToolValidator = (*recordTenderResponseTool)(nil)
)

var tenderOfferLabels = map[string]string{
	"responseSource":   "Answered through",
	fieldDeclineReason: "Decline reason",
}

func (t *cancelTenderTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	summary := "Would withdraw this tender. Reason: " + request.Reason
	plan, err := t.tenders.PreviewCancel(ctx, request)
	if err != nil {
		if isRefusal(err) || errortypes.IsNotFoundError(err) {
			return warnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}

	tenderChange, err := tenderStateChange(plan.Before, plan.After,
		fieldStatus, "cancellationReason", "canceledAt")
	if err != nil {
		return nil, err
	}
	tenderChange.Operation = agent.PreviewOperationArchive

	offers, err := tenderOfferChanges(plan.Before, plan.After)
	if err != nil {
		return nil, err
	}

	withdrawn, skipped := 0, 0
	for _, offer := range plan.After.Offers {
		//nolint:exhaustive // only the two statuses a withdrawal moves an offer to are counted
		switch offer.Status {
		case tender.OfferStatusWithdrawn:
			withdrawn++
		case tender.OfferStatusSkipped:
			skipped++
		default:
		}
	}
	withdrawn -= countOffers(plan.Before, tender.OfferStatusWithdrawn)
	skipped -= countOffers(plan.Before, tender.OfferStatusSkipped)

	summary = fmt.Sprintf(
		"Would withdraw the %s tender: %s carriers hold would be withdrawn and their answer "+
			"links stop working, and %s not yet asked would be skipped. Reason: %s",
		plan.Before.Mode, countOf(withdrawn, "offer"), countOf(skipped, "carrier"), request.Reason,
	)
	if plan.Before.Status == tender.StatusNeedsReview {
		summary += " A carrier's acceptance is waiting on review in this tender."
	}

	return toolpreview.Build(
		summary,
		append([]*agent.RecordChange{tenderChange}, offers...)...), nil
}

func (t *recordTenderResponseTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf("Would record the carrier's %s.", responseWord(request.Action))
	plan, err := t.tenders.PreviewResponse(ctx, request)
	if err != nil {
		if isRefusal(err) || errortypes.IsNotFoundError(err) {
			return warnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}

	changes := make([]*agent.RecordChange, 0, len(plan.Tender.Offers)+2)
	if plan.Tender.Status != plan.TenderAfter.Status {
		tenderChange, tErr := tenderStateChange(plan.Tender, plan.TenderAfter,
			fieldStatus, "acceptedOfferId", "acceptedAt", "exhaustedAt")
		if tErr != nil {
			return nil, tErr
		}
		changes = append(changes, tenderChange)
	}
	offers, err := tenderOfferChanges(plan.Tender, plan.TenderAfter)
	if err != nil {
		return nil, err
	}
	changes = append(changes, offers...)

	carrier := carrierNameOfOffer(plan.Offer)
	rate := money.FormatMinor(money.MinorUnits(plan.Offer.Rate), money.DefaultCurrencyCode) +
		" " + rateMethodWords(plan.Offer.RateMethod)

	if request.Action == tender.ResponseActionAccept {
		attachOfferPay(offers, plan.Offer, carrier)
		preview := toolpreview.Build(fmt.Sprintf(
			"Would record that %s accepted the offer at %s. The tender closes, the carrier is "+
				"assigned to the move at that rate, and Trenova then issues the executed rate "+
				"confirmation and emails it to the carrier's rate confirmation contacts.",
			carrier, rate,
		), changes...)
		preview.Partial = true

		return preview, nil
	}

	summary = fmt.Sprintf("Would record that %s declined the offer at %s.", carrier, rate)
	switch {
	case plan.NextOffer != nil:
		changes = append(changes, tenderOfferSendChange(plan.NextOffer))
		summary += fmt.Sprintf(" The tender moves on and offers the load to %s.",
			carrierNameOfOffer(plan.NextOffer))
	case plan.Exhausted:
		summary += " Nobody is left to ask, so the tender ends without a carrier."
	default:
		summary += " The other carriers' offers stay open."
	}

	return toolpreview.Build(summary, changes...), nil
}

// tenderStateChange is the tender before and after, limited to the fields
// the write moves; times it stamps are shown but left out of the digest.
func tenderStateChange(
	before, after *tender.Tender,
	fields ...string,
) (*agent.RecordChange, error) {
	return toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceTender,
		ID:       before.ID,
		Label:    string(before.Mode) + " tender",
		Version:  previewVersion(before.Version),
	}, before, after,
		toolpreview.Only(fields...),
		toolpreview.Volatile("canceledAt", "acceptedAt", "exhaustedAt"),
		toolpreview.WithRefs(map[string]permission.Resource{
			"acceptedOfferId": permission.ResourceTender,
		}),
	)
}

// tenderOfferChanges is every offer the write moves, in rank order, each as
// its own record; an offer left where it was is left out.
func tenderOfferChanges(before, after *tender.Tender) ([]*agent.RecordChange, error) {
	previous := make(map[pulid.ID]*tender.TenderOffer, len(before.Offers))
	for _, offer := range before.Offers {
		if offer != nil {
			previous[offer.ID] = offer
		}
	}

	changes := make([]*agent.RecordChange, 0, len(after.Offers))
	for _, offer := range after.Offers {
		prior, ok := previous[offer.ID]
		if !ok || prior.Status == offer.Status {
			continue
		}
		change, err := toolpreview.Changed(toolpreview.Record{
			Resource: permission.ResourceTender,
			ID:       offer.ID,
			Label:    fmt.Sprintf("Offer %d to %s", offer.Rank, carrierNameOfOffer(prior)),
			Version:  previewVersion(prior.Version),
		}, prior, offer,
			toolpreview.Only(fieldStatus, "responseSource", fieldDeclineReason, "respondedAt"),
			toolpreview.Volatile("respondedAt"),
			toolpreview.Labels(tenderOfferLabels),
		)
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}

	return changes, nil
}

// tenderOfferSendChange is the offer a sequential tender sends next: by
// email to the address it was addressed to, or over the carrier's EDI link.
func tenderOfferSendChange(offer *tender.TenderOffer) *agent.RecordChange {
	message := &agent.MessagePreview{
		Channel: agent.MessageChannelEmail,
		To:      []string{offer.RecipientEmail},
		Subject: "Load tender offer",
		Body: money.FormatMinor(money.MinorUnits(offer.Rate), money.DefaultCurrencyCode) +
			" " + rateMethodWords(offer.RateMethod) + " offered.",
	}
	if offer.Channel == tender.ChannelEDI {
		message.Channel = agent.MessageChannelEDI
		message.To = []string{carrierNameOfOffer(offer) + " (EDI 204)"}
	}

	return toolpreview.Send(toolpreview.Record{
		Resource: permission.ResourceCarrier,
		ID:       offer.CarrierID,
		Label:    carrierNameOfOffer(offer),
	}, message)
}

// attachOfferPay puts a flat offer's rate on the answered offer's change as
// the pay the acceptance commits. A per-mile rate is not a total, so it is
// named in the summary instead.
func attachOfferPay(changes []*agent.RecordChange, offer *tender.TenderOffer, carrier string) {
	if offer.RateMethod == shipment.CarrierRateMethodPerMile {
		return
	}
	for _, change := range changes {
		if change.EntityID != offer.ID {
			continue
		}
		toolpreview.AttachMoney(change, toolpreview.MoneyBlock(money.DefaultCurrencyCode,
			agent.MoneyLine{Label: "Carrier pay to " + carrier, After: knownAmount(offer.Rate)},
		), toolpreview.SensitiveAs("rate"))

		return
	}
}

func countOffers(entity *tender.Tender, status tender.OfferStatus) int {
	n := 0
	for _, offer := range entity.Offers {
		if offer != nil && offer.Status == status {
			n++
		}
	}

	return n
}

func responseWord(action tender.ResponseAction) string {
	if action == tender.ResponseActionAccept {
		return "acceptance"
	}

	return "decline"
}
