package agentquerytoolservice

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	toolOrderList                 = "order_list"
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
	customerReference = idReference{
		resource: permission.ResourceCustomer,
		what:     "The customer",
		source:   toolCustomerList,
	}
	shipmentReference = idReference{
		resource: permission.ResourceShipment,
		what:     "The shipment",
		source:   shipmentListName,
	}
)

// operationsReadSpecs are the order, recurring shipment, service failure,
// detention and tender lists, each read through the operation the Desk or
// the record's page sends.
func operationsReadSpecs() []graphQLListSpec {
	return []graphQLListSpec{
		{
			name:         toolOrderList,
			entityPlural: pluralOrders,
			summary:      ordersSummary,
			searchTerms:  []string{"po number", "order number"},
			resource:     permission.ResourceOrder,
			notes:        map[string]string{paramStatus: noteOrderStatus},
			references:   map[string]idReference{paramCustomerID: customerReference},
		},
		{
			name:         toolListRecurringShipments,
			entityPlural: pluralRecurringShipments,
			summary:      recurringShipmentsSummary,
			searchTerms:  []string{"recurring shipments", "when a series generates next"},
			resource:     permission.ResourceRecurringShipment,
			notes:        map[string]string{paramStatus: noteSeriesStatus},
			references: map[string]idReference{
				paramCustomerID: customerReference,
				fieldOriginLocationID: {
					resource: permission.ResourceLocation,
					what:     "The lane's origin",
					source:   toolLocationList,
				},
				fieldDestinationLocationID: {
					resource: permission.ResourceLocation,
					what:     "The lane's destination",
					source:   toolLocationList,
				},
				fieldSourceShipmentID: {
					resource: permission.ResourceShipment,
					what:     "The shipment the series copies",
					source:   shipmentListName,
				},
				fieldLastGeneratedShipmentID: {
					resource: permission.ResourceShipment,
					what:     "The shipment the series last made",
					source:   shipmentListName,
				},
			},
		},
		{
			name:         toolListServiceFailures,
			entityPlural: pluralServiceFailures,
			summary:      serviceFailuresSummary,
			resource:     permission.ResourceServiceFailure,
			references: map[string]idReference{
				paramShipmentID: shipmentReference,
				fieldReasonCodeID: {
					resource: permission.ResourceServiceFailureReasonCode,
					what:     "The reason code",
					source:   toolListServiceFailureReasons,
				},
			},
		},
		{
			name:         toolListServiceFailureReasons,
			entityPlural: pluralServiceFailureReasons,
			summary:      serviceFailureReasonsSummary,
			resource:     permission.ResourceServiceFailureReasonCode,
			notes:        map[string]string{fieldActive: noteReasonCodeActive},
			defaults:     map[string]any{fieldActive: true},
		},
		{
			name:         toolListDetentionOccurrences,
			entityPlural: pluralDetentionOccurrences,
			summary:      detentionOccurrencesSummary,
			resource:     permission.ResourceDetention,
			notes: map[string]string{
				fieldIncludeAwaitingApproval: noteAwaitingApproval,
			},
			defaults:   map[string]any{fieldIncludeAwaitingApproval: true},
			references: map[string]idReference{paramShipmentID: shipmentReference},
		},
		{
			name:         toolListShipmentTenders,
			entityPlural: pluralTenders,
			summary:      shipmentTendersSummary,
			resource:     permission.ResourceTender,
			references: map[string]idReference{
				paramShipmentID: shipmentReference,
				paramShipmentMoveID: {
					resource: permission.ResourceShipmentMove,
					what:     "The move to narrow to",
					source:   "shipment_get (its moves) or dispatch_get_board",
				},
			},
			shape: shapeTender,
			reads: readSpec{
				rationale: "Reads a shipment's tenders and their offers; a carrier's own " +
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
func shapeTender(row map[string]any, _ map[string]struct{}, _ clock) {
	entity := &tender.Tender{Status: tender.Status(stringOf(row[paramStatus]))}
	if accepted := pulid.ID(stringOf(row[fieldAcceptedOfferID])); accepted.IsNotNil() {
		entity.AcceptedOfferID = &accepted
	}
	raw, _ := row[fieldOffers].([]any)
	offers := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if offer, ok := item.(map[string]any); ok {
			offers = append(offers, offer)
			entity.Offers = append(entity.Offers, &tender.TenderOffer{
				ID:     pulid.ID(stringOf(offer[recordIDField])),
				Status: tender.OfferStatus(stringOf(offer[paramStatus])),
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

func stringOf(value any) string {
	text, _ := value.(string)

	return text
}

// orderGetSpec reads one order, with every leg it groups, through the Desk's
// order read.
func orderGetSpec() getSpec {
	return getSpec{
		name:        toolOrderGet,
		entity:      "order",
		summary:     orderGetSummary,
		searchTerms: []string{"charges", "invoiced", "legs"},
		resource:    permission.ResourceOrder,
		paramName:   "orderId",
		idSource: "from order_list, the orderId on shipment_get or invoice_get, or the " +
			"page you are on",
	}
}

// serviceFailureGetSpec reads one service failure through the Desk's card.
func serviceFailureGetSpec() getSpec {
	return getSpec{
		name:      toolGetServiceFailure,
		entity:    "service failure",
		summary:   serviceFailureGetSummary,
		resource:  permission.ResourceServiceFailure,
		paramName: "serviceFailureId",
		idSource:  "from exception_list_service_failures or the page you are on",
	}
}

// detentionOccurrenceGetSpec reads one detention occurrence through the
// Desk's detention card, priced as of now while its clock runs.
func detentionOccurrenceGetSpec() getSpec {
	return getSpec{
		name:      toolGetDetentionOccurrence,
		entity:    "detention occurrence",
		summary:   detentionOccurrenceGetSummary,
		resource:  permission.ResourceDetention,
		kinds:     []permission.RecordKind{permission.KindDetentionOccurrence},
		paramName: "occurrenceId",
		idSource:  "from detention_list_occurrences, the page you are on, or this run's subject",
	}
}
