package tendertools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/carrierpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmenttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

var tenderOfferLabels = map[string]string{
	"responseSource":   "Answered through",
	fieldDeclineReason: "Decline reason",
	fieldCounterRate:   "Counter rate",
	fieldCounterNote:   "Counter note",
	"revokeReason":     "Backed out because",
}

func (t *CancelTenderTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	summary := "Would withdraw this tender. Reason: " + request.Reason
	plan, err := t.tenders.PreviewCancel(ctx, request)
	if err != nil {
		if changepreview.IsRefusal(err) || errortypes.IsNotFoundError(err) {
			return changepreview.WarnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}

	tenderChange, err := tenderStateChange(plan.Before, plan.After,
		changepreview.FieldStatus, "cancellationReason", "canceledAt")
	if err != nil {
		return nil, err
	}
	tenderChange.Operation = agentpreview.PreviewOperationArchive

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
		plan.Before.Mode,
		changepreview.CountOf(withdrawn, "offer"),
		changepreview.CountOf(skipped, "carrier"),
		request.Reason,
	)
	if plan.Before.Status == tender.StatusNeedsReview {
		summary += " A carrier's acceptance is waiting on review in this tender."
	}

	return toolpreview.Build(
		summary,
		append([]*agentpreview.RecordChange{tenderChange}, offers...)...), nil
}

func (t *RecordTenderResponseTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	if bounce, err := t.bounceRequest(&params); bounce != nil || err != nil {
		if err != nil {
			return nil, err
		}
		return t.previewBounce(ctx, bounce)
	}

	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf("Would record the carrier's %s.", responseWord(request.Action))
	plan, err := t.tenders.PreviewResponse(ctx, request)
	if err != nil {
		if changepreview.IsRefusal(err) || errortypes.IsNotFoundError(err) {
			return changepreview.WarnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}

	changes := make([]*agentpreview.RecordChange, 0, len(plan.Tender.Offers)+2)
	if plan.Tender.Status != plan.TenderAfter.Status {
		tenderChange, tErr := tenderStateChange(plan.Tender, plan.TenderAfter,
			changepreview.FieldStatus, "acceptedOfferId", "acceptedAt", "exhaustedAt")
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

	carrier := shipmenttools.CarrierNameOfOffer(plan.Offer)
	offered := rateWords(plan.Offer.RateMethod, plan.Offer.Rate)
	decidingCounter := plan.Offer.Status == tender.OfferStatusCountered

	//nolint:exhaustive // a decline falls through to the shared summary below
	switch request.Action {
	case tender.ResponseActionCounter:
		return toolpreview.Build(fmt.Sprintf(
			"Would record that %s countered at %s against %s offered. The offer holds for a "+
				"person to take or refuse the counter before its decision deadline; if the "+
				"organization takes counters within the margin floor on its own and this one "+
				"clears it, the carrier is assigned at the counter rate at once.",
			carrier, rateWords(plan.Offer.RateMethod, request.CounterRate), offered,
		), changes...), nil
	case tender.ResponseActionAccept:
		attachOfferPay(offers, plan.Offer, carrier)
		what := fmt.Sprintf("Would record that %s accepted the offer at %s.", carrier, offered)
		if decidingCounter {
			what = fmt.Sprintf(
				"Would take %s's counter at %s instead of the %s offered.",
				carrier, rateWords(plan.Offer.RateMethod, plan.Offer.AgreedRate()), offered,
			)
			if plan.Offer.CounterNeedsApproval {
				what += " The counter is below the margin floor, so taking it needs the " +
					"tender approve permission."
			}
		}
		preview := toolpreview.Build(what+" The tender closes, the carrier is assigned to "+
			"the move at that rate, and Trenova then issues the executed rate confirmation "+
			"and emails it to the carrier's rate confirmation contacts.", changes...)
		preview.Partial = true

		return preview, nil
	}

	summary = fmt.Sprintf("Would record that %s declined the offer at %s.", carrier, offered)
	if decidingCounter {
		summary = fmt.Sprintf(
			"Would refuse %s's counter at %s.",
			carrier, rateWords(plan.Offer.RateMethod, plan.Offer.AgreedRate()),
		)
	}
	switch {
	case plan.NextOffer != nil:
		changes = append(changes, tenderOfferSendChange(plan.NextOffer))
		summary += fmt.Sprintf(" The tender moves on and offers the load to %s.",
			shipmenttools.CarrierNameOfOffer(plan.NextOffer))
	case plan.Exhausted:
		summary += " Nobody is left to ask, so the tender ends without a carrier."
	default:
		summary += " The other carriers' offers stay open."
	}

	return toolpreview.Build(summary, changes...), nil
}

// previewBounce is a carrier backing out of its acceptance: the acceptance
// revoked, the carrier off the move with its rate confirmation voided, and the
// tender live again for the carriers it had not reached.
func (t *RecordTenderResponseTool) previewBounce(
	ctx context.Context,
	request *tenderservice.BounceTenderRequest,
) (*agentpreview.ToolPreview, error) {
	summary := "Would record that the carrier backed out of its acceptance. Reason: " +
		request.Reason
	plan, err := t.tenders.PreviewBounce(ctx, request)
	if err != nil {
		if changepreview.IsRefusal(err) || errortypes.IsNotFoundError(err) {
			return changepreview.WarnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}

	tenderChange, err := tenderStateChange(plan.Before, plan.After,
		changepreview.FieldStatus, "acceptedOfferId", "bounceCount")
	if err != nil {
		return nil, err
	}
	offers, err := tenderOfferChanges(plan.Before, plan.After)
	if err != nil {
		return nil, err
	}
	changes := append([]*agentpreview.RecordChange{tenderChange}, offers...)

	carrier := shipmenttools.CarrierNameOfOffer(plan.Revoked)
	summary = fmt.Sprintf("Would record that %s backed out of the tender it accepted.", carrier)
	if plan.Unassigns {
		summary += " The carrier comes off the move and its rate confirmation is voided."
	}
	switch {
	case plan.WillExhaust:
		summary += " Nobody is left to ask, so the tender ends without a carrier and the " +
			"organization's escalation policy decides what happens next."
	case plan.Before.Mode.IsSequential():
		changes = append(changes, tenderOfferSendChange(plan.Reoffers[0]))
		summary += fmt.Sprintf(" The tender goes on to the %s it had not reached, starting "+
			"with %s.", changepreview.CountOf(len(plan.Reoffers), "carrier"),
			shipmenttools.CarrierNameOfOffer(plan.Reoffers[0]))
	default:
		for _, offer := range plan.Reoffers {
			changes = append(changes, tenderOfferSendChange(offer))
		}
		summary += fmt.Sprintf(" The load is offered again to the %s whose offers the "+
			"acceptance cut short.", changepreview.CountOf(len(plan.Reoffers), "carrier"))
	}
	summary += " Reason: " + request.Reason

	preview := toolpreview.Build(summary, changes...)
	preview.Partial = plan.Unassigns

	return preview, nil
}

func rateWords(method shipment.CarrierRateMethod, rate decimal.Decimal) string {
	return money.FormatMinor(money.MinorUnits(rate), money.DefaultCurrencyCode) + " " +
		carrierpreview.RateMethodWords(method)
}

// tenderStateChange is the tender before and after, limited to the fields
// the write moves; times it stamps are shown but left out of the digest.
func tenderStateChange(
	before, after *tender.Tender,
	fields ...string,
) (*agentpreview.RecordChange, error) {
	return toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceTender,
		ID:       before.ID,
		Label:    string(before.Mode) + " tender",
		Version:  changepreview.PreviewVersion(before.Version),
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
func tenderOfferChanges(before, after *tender.Tender) ([]*agentpreview.RecordChange, error) {
	previous := make(map[pulid.ID]*tender.TenderOffer, len(before.Offers))
	for _, offer := range before.Offers {
		if offer != nil {
			previous[offer.ID] = offer
		}
	}

	changes := make([]*agentpreview.RecordChange, 0, len(after.Offers))
	for _, offer := range after.Offers {
		prior, ok := previous[offer.ID]
		if !ok || prior.Status == offer.Status {
			continue
		}
		change, err := toolpreview.Changed(
			toolpreview.Record{
				Resource: permission.ResourceTender,
				ID:       offer.ID,
				Label: fmt.Sprintf(
					"Offer %d to %s",
					offer.Rank,
					shipmenttools.CarrierNameOfOffer(prior),
				),
				Version: changepreview.PreviewVersion(prior.Version),
			},
			prior,
			offer,
			toolpreview.Only(
				changepreview.FieldStatus,
				"responseSource",
				fieldDeclineReason,
				fieldCounterRate,
				fieldCounterNote,
				"revokeReason",
				"respondedAt",
			),
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
func tenderOfferSendChange(offer *tender.TenderOffer) *agentpreview.RecordChange {
	message := &agentpreview.MessagePreview{
		Channel: agentpreview.MessageChannelEmail,
		To:      []string{offer.RecipientEmail},
		Subject: "Load tender offer",
		Body: money.FormatMinor(money.MinorUnits(offer.Rate), money.DefaultCurrencyCode) +
			" " + carrierpreview.RateMethodWords(offer.RateMethod) + " offered.",
	}
	if offer.Channel == tender.ChannelEDI {
		message.Channel = agentpreview.MessageChannelEDI
		message.To = []string{shipmenttools.CarrierNameOfOffer(offer) + " (EDI 204)"}
	}

	return toolpreview.Send(toolpreview.Record{
		Resource: permission.ResourceCarrier,
		ID:       offer.CarrierID,
		Label:    shipmenttools.CarrierNameOfOffer(offer),
	}, message)
}

// attachOfferPay puts a flat offer's rate on the answered offer's change as
// the pay the acceptance commits. A per-mile rate is not a total, so it is
// named in the summary instead.
func attachOfferPay(
	changes []*agentpreview.RecordChange,
	offer *tender.TenderOffer,
	carrier string,
) {
	if offer.RateMethod == shipment.CarrierRateMethodPerMile {
		return
	}
	for _, change := range changes {
		if change.EntityID != offer.ID {
			continue
		}
		toolpreview.AttachMoney(change, toolpreview.MoneyBlock(
			money.DefaultCurrencyCode,
			agentpreview.MoneyLine{
				Label: "Carrier pay to " + carrier,
				After: changepreview.KnownAmount(offer.AgreedRate()),
			},
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
	switch action {
	case tender.ResponseActionAccept:
		return "acceptance"
	case tender.ResponseActionCounter:
		return "counter-offer"
	case tender.ResponseActionDecline:
		return "decline"
	default:
		return "decline"
	}
}
