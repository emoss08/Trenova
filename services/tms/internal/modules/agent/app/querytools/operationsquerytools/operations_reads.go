package operationsquerytools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	toolOrderGet                  = "order_get"
	toolListRecurringShipments    = "shipment_list_recurring"
	toolListServiceFailures       = "exception_list_service_failures"
	toolListServiceFailureReasons = "exception_list_service_failure_reason_codes"
	toolGetServiceFailure         = "exception_get_service_failure"
	toolGetDetentionOccurrence    = "detention_get_occurrence"
	toolListDetentionOccurrences  = "detention_list_occurrences"
	toolListShipmentTenders       = "dispatch_list_tenders"

	fieldReasonCodeID            = "reasonCodeId"
	fieldSourceShipmentID        = "sourceShipmentId"
	fieldLastGeneratedShipmentID = "lastGeneratedShipmentId"
	fieldOriginLocationID        = "originLocationId"
	fieldDestinationLocationID   = "destinationLocationId"
	fieldIncludeAwaitingApproval = "includeAwaitingApproval"
	fieldOffers                  = "offers"
	fieldAcceptedOfferID         = "acceptedOfferId"
	fieldLive                    = "live"
	fieldAnswerable              = "answerable"
	fieldBackOutAllowed          = "backOutAllowed"
	fieldCounterNeedsApproval    = "counterNeedsApproval"

	labelOrder                  = "order"
	searchTermCharges           = "charges"
	searchTermLegs              = "legs"
	pluralOrders                = "orders"
	pluralRecurringShipments    = "recurring shipments"
	pluralServiceFailures       = "service failures"
	pluralServiceFailureReasons = "reason codes"
	pluralDetentionOccurrences  = "detention occurrences"
	pluralTenders               = "tenders"

	noteOrderStatus      = "Billed means invoiced; Closed means nothing more will be added."
	noteSeriesStatus     = "Paused generates nothing until resumed; Expired has ended."
	noteReasonCodeActive = "Defaults to true: only an active code can resolve a failure."
	noteAwaitingApproval = "Defaults to true, so the charges holding an invoice are listed " +
		"beside the running clocks."

	ordersSummary = "List customer orders, the commercial record that groups one or more " +
		"shipments and their charges under a customer's order number, PO or BOL. " +
		"order_get opens one with its shipments and charges."
	recurringShipmentsSummary = "List recurring shipments: the series that copy one saved " +
		"shipment on a schedule for a customer's standing lane, with when each next " +
		"generates. Use the id with shipment_update_recurring, shipment_set_recurring_status " +
		"or shipment_generate_recurring."
	serviceFailuresSummary = "List the service failures that happened: late and missed " +
		"pickups and deliveries, with how late, which stop, and whether anyone reviewed or " +
		"resolved them. Each was detected by the service check or recorded by a person. " +
		"Filter on status Open for what still needs attention, on detectedAt for a period " +
		"such as last week, or give a shipmentId. Resolving one takes its id and version, " +
		"which are in the row."
	serviceFailureReasonsSummary = "The reason codes a service failure can carry, such as " +
		"Weather, Shipper delay or Equipment. Call this before " +
		"exception_resolve_service_failure, which needs a reason code id, and pick the code " +
		"whose category matches what actually happened; do not resolve with a code that " +
		"merely sounds close."
	detentionOccurrencesSummary = "List every open detention occurrence: a truck at a stop " +
		"past its free time, with minutes until free time ends, notice status and the " +
		"amount at risk. The notice status says whether the customer notice is due or " +
		"overdue. Urgency Lost means the notice window closed without a notice and the " +
		"charge may not be collectable; NoticeOverdue and NoticeDueSoon say what to send. " +
		"detention_send_notice sends it. The list also carries every stopped charge " +
		"waiting on approval, as AwaitingApproval: its shipment cannot be invoiced until " +
		"someone approves or waives it, which detention_approve and detention_waive " +
		"propose. A running clock is priced as of now, so billableMinutes and amountAtRisk " +
		"are what it would bill if the truck left this minute. Narrow to one load with " +
		"shipmentId or proNumber rather than paging through urgencies."
	shipmentTendersSummary = "List a shipment's tenders, newest first, with every offer in " +
		"rank order. Each tender shows its mode and status; each offer the carrier, the " +
		"rate, how it was sent, its status, any counter the carrier made and when it lapses " +
		"or was answered. Use it to see who holds an offer before recording a carrier's " +
		"answer, a decision on its counter or the carrier backing out with " +
		"dispatch_record_tender_response, or to find a live tender to withdraw with " +
		"dispatch_cancel_tender. Narrow to one move with shipmentMoveId."
	orderGetSummary = "Retrieve one customer order by id with the shipments it groups and " +
		"its order-level charges, including which charges are already invoiced. Use " +
		"order_list first when you have an order number, PO or BOL rather than an id."
	serviceFailureGetSummary = "Retrieve one service failure by id: the late or missed " +
		"stop, how late, the reason code, the notes, and who reviewed, resolved or voided " +
		"it. Use exception_list_service_failures first when you do not have an id."
	detentionOccurrenceGetSummary = "Retrieve one detention occurrence by id: the stop, the " +
		"clock, free time, billable minutes, amounts, notice status, and the evidence and " +
		"notices on file. A clock still running also carries currentlyBillable, priced as " +
		"of now; quote that rather than the stored amounts, which date from the last " +
		"recalculation. Use detention_list_occurrences first when you do not have an id."
)

var (
	shipmentReference = readspec.Reference{
		Resource: permission.ResourceShipment,
		What:     "The shipment",
		Source:   querytool.ShipmentListName,
	}
)

// ReadSpecs are the order, recurring shipment, service failure,
// detention and tender lists, each read through the operation the Desk or
// the record's page sends.
func ReadSpecs() []readspec.List {
	return []readspec.List{
		{
			Name:         querytool.ToolOrderList,
			EntityPlural: pluralOrders,
			Summary:      ordersSummary,
			SearchTerms:  []string{"po number", "order number"},
			Resource:     permission.ResourceOrder,
			Notes:        map[string]string{querytool.ParamStatus: noteOrderStatus},
			References: map[string]readspec.Reference{
				querytool.ParamCustomerID: readspec.CustomerReference,
			},
		},
		{
			Name:         toolListRecurringShipments,
			EntityPlural: pluralRecurringShipments,
			Summary:      recurringShipmentsSummary,
			SearchTerms:  []string{"recurring shipments", "when a series generates next"},
			Resource:     permission.ResourceRecurringShipment,
			Notes:        map[string]string{querytool.ParamStatus: noteSeriesStatus},
			References: map[string]readspec.Reference{
				querytool.ParamCustomerID: readspec.CustomerReference,
				fieldOriginLocationID: {
					Resource: permission.ResourceLocation,
					What:     "The lane's origin",
					Source:   querytool.ToolLocationList,
				},
				fieldDestinationLocationID: {
					Resource: permission.ResourceLocation,
					What:     "The lane's destination",
					Source:   querytool.ToolLocationList,
				},
				fieldSourceShipmentID: {
					Resource: permission.ResourceShipment,
					What:     "The shipment the series copies",
					Source:   querytool.ShipmentListName,
				},
				fieldLastGeneratedShipmentID: {
					Resource: permission.ResourceShipment,
					What:     "The shipment the series last made",
					Source:   querytool.ShipmentListName,
				},
			},
		},
		{
			Name:         toolListServiceFailures,
			EntityPlural: pluralServiceFailures,
			Summary:      serviceFailuresSummary,
			Resource:     permission.ResourceServiceFailure,
			References: map[string]readspec.Reference{
				querytool.ParamShipmentID: shipmentReference,
				fieldReasonCodeID: {
					Resource: permission.ResourceServiceFailureReasonCode,
					What:     "The reason code",
					Source:   toolListServiceFailureReasons,
				},
			},
		},
		{
			Name:         toolListServiceFailureReasons,
			EntityPlural: pluralServiceFailureReasons,
			Summary:      serviceFailureReasonsSummary,
			Resource:     permission.ResourceServiceFailureReasonCode,
			Notes:        map[string]string{querytool.FieldActive: noteReasonCodeActive},
			Defaults:     map[string]any{querytool.FieldActive: true},
		},
		{
			Name:         toolListDetentionOccurrences,
			EntityPlural: pluralDetentionOccurrences,
			Summary:      detentionOccurrencesSummary,
			Resource:     permission.ResourceDetention,
			Notes: map[string]string{
				fieldIncludeAwaitingApproval: noteAwaitingApproval,
			},
			Defaults:   map[string]any{fieldIncludeAwaitingApproval: true},
			References: map[string]readspec.Reference{querytool.ParamShipmentID: shipmentReference},
		},
		{
			Name:         toolListShipmentTenders,
			EntityPlural: pluralTenders,
			Summary:      shipmentTendersSummary,
			Resource:     permission.ResourceTender,
			References: map[string]readspec.Reference{
				querytool.ParamShipmentID: shipmentReference,
				paramShipmentMoveID: {
					Resource: permission.ResourceShipmentMove,
					What:     "The move to narrow to",
					Source:   "shipment_get (its moves) or dispatch_get_board",
				},
			},
			Shape: shapeTender,
			Reads: querytool.ReadSpec{
				Rationale: "Reads a shipment's tenders and their offers; a carrier's own " +
					"words, such as a decline reason, are left out, so nothing written " +
					"outside the organization is read.",
			},
		},
	}
}

// shapeTender adds what the tender rules say about a tender row: whether it
// is live, which offers dispatch_record_tender_response can still answer,
// which accepted offer the carrier may back out of, and a counter's approval
// only while the counter stands.
func shapeTender(row map[string]any, _ map[string]struct{}, _ filtercatalog.Clock) {
	entity := &tender.Tender{Status: tender.Status(querytool.StringOf(row[querytool.ParamStatus]))}
	if accepted := pulid.ID(querytool.StringOf(row[fieldAcceptedOfferID])); accepted.IsNotNil() {
		entity.AcceptedOfferID = &accepted
	}
	raw, _ := row[fieldOffers].([]any)
	offers := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if offer, ok := item.(map[string]any); ok {
			offers = append(offers, offer)
			entity.Offers = append(entity.Offers, &tender.TenderOffer{
				ID:     pulid.ID(querytool.StringOf(offer[querytool.RecordIDField])),
				Status: tender.OfferStatus(querytool.StringOf(offer[querytool.ParamStatus])),
			})
		}
	}

	row[fieldLive] = entity.IsLive()
	backOut, canBackOut := entity.CanBounce()
	for idx, offer := range offers {
		status := entity.Offers[idx].Status
		offer[fieldAnswerable] = entity.Status == tender.StatusActive &&
			status.IsAwaitingAnswer()
		offer[fieldBackOutAllowed] = canBackOut && backOut.ID == entity.Offers[idx].ID
		if status != tender.OfferStatusCountered {
			delete(offer, fieldCounterNeedsApproval)
		}
	}
}

// OrderGetSpec reads one order, with every leg it groups, through the Desk's
// order read.
func OrderGetSpec() readspec.Get {
	return readspec.Get{
		Name:        toolOrderGet,
		Entity:      labelOrder,
		Summary:     orderGetSummary,
		SearchTerms: []string{searchTermCharges, "invoiced", searchTermLegs},
		Resource:    permission.ResourceOrder,
		ParamName:   "orderId",
		IDSource: "from order_list, the orderId on shipment_get or invoice_get, or the " +
			"page you are on",
	}
}

// ServiceFailureGetSpec reads one service failure through the Desk's card.
func ServiceFailureGetSpec() readspec.Get {
	return readspec.Get{
		Name:      toolGetServiceFailure,
		Entity:    "service failure",
		Summary:   serviceFailureGetSummary,
		Resource:  permission.ResourceServiceFailure,
		ParamName: "serviceFailureId",
		IDSource:  "from exception_list_service_failures or the page you are on",
	}
}

// DetentionOccurrenceGetSpec reads one detention occurrence through the
// Desk's detention card, priced as of now while its clock runs.
func DetentionOccurrenceGetSpec() readspec.Get {
	return readspec.Get{
		Name:      toolGetDetentionOccurrence,
		Entity:    "detention occurrence",
		Summary:   detentionOccurrenceGetSummary,
		Resource:  permission.ResourceDetention,
		Kinds:     []permission.RecordKind{permission.KindDetentionOccurrence},
		ParamName: "occurrenceId",
		IDSource:  "from detention_list_occurrences, the page you are on, or this run's subject",
	}
}
