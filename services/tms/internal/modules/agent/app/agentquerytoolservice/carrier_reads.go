package agentquerytoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

const (
	carrierInvoiceEntity = "edi_carrier_invoice"
	carrierSource        = "carrier_list or carrier_get"

	fieldEDICarrierInvoiceID = "ediCarrierInvoiceId"
	fieldDocumentExtraction  = "documentAiExtractionId"
	fieldMatchSource         = "source"
	fieldFieldPath           = "fieldPath"
	fieldRuleCode            = "ruleCode"
	fieldVendorChangedAt     = "vendorChangedAt"

	matchSourceEDI      = "EDI"
	matchSourceDocument = "Document"
	matchSourceManual   = "Manual"
)

var carrierReference = idReference{
	resource: permission.ResourceCarrier,
	what:     "Only this carrier's records, by the carrier's id",
	source:   carrierSource,
}

func carrierReadSpecs() []graphQLListSpec {
	return []graphQLListSpec{
		carrierIntelEventsListSpec(), capacityPostingsListSpec(), invoiceMatchesListSpec(),
	}
}

func carrierIntelEventsListSpec() graphQLListSpec {
	return graphQLListSpec{
		name:         "carrier_list_intelligence_events",
		entityPlural: "carrier intelligence events",
		summary: "List carrier intelligence events, newest first: changes found on " +
			"carriers' authority, insurance and safety records. Each row gives the carrier, " +
			"the category, the severity, what the rule did, the summary, when it was " +
			"detected and whether anyone acknowledged or resolved it. Pass a row's id to " +
			"carrier_get_intelligence_event for its prior and current values, or to " +
			"carrier_acknowledge_intelligence_event or carrier_resolve_intelligence_event.",
		searchTerms: []string{"carrier risk alerts", "authority or insurance problems"},
		queryNote: "Words to find in the carrier's name, its DOT number or the event's " +
			"summary.",
		names: map[string]string{
			"statuses":       paramStatus,
			"severities":     paramSeverity,
			"categories":     paramCategory,
			"detectedAtFrom": "detectedFrom",
			"detectedAtTo":   "detectedTo",
			"subjectId":      "subject",
		},
		unoffered: []string{
			"status", "severity", "category", "carrierId", "subjectType", "subjectId",
			fieldFieldPath, fieldRuleCode, fieldVendorChangedAt,
			"acknowledgedById", "resolvedById", "resolutionNote", "snapshotId", "updatedAt",
			"createdAt",
		},
		notes: map[string]string{
			paramStatus:   "Open and Acknowledged ones are not settled yet.",
			paramCategory: "The parts of a carrier's record, such as Authority or Insurance.",
		},
		references: map[string]idReference{paramCarrierID: carrierReference},
		resource:   permission.ResourceCarrierIntelligence,
	}
}

func capacityPostingsListSpec() graphQLListSpec {
	state := idReference{
		kinds: []permission.RecordKind{permission.KindUSState},
		what:  "The state's id",
		source: "an originStateId or destinationStateId on a row of this tool, or a " +
			"location's stateId from location_get",
	}

	return graphQLListSpec{
		name:         "carrier_list_capacity_postings",
		entityPlural: "carrier capacity postings",
		summary: "List trucks carriers have said they have available: carrier, origin, " +
			"destination, equipment, the window and the rate asked. Pass openOnly for the " +
			"offers whose window has not closed.",
		searchTerms: []string{"carrier capacity", "available trucks", "truck postings"},
		references: map[string]idReference{
			paramCarrierID: carrierReference,
			"originLocationId": {
				resource: permission.ResourceLocation,
				what:     "Only postings from this location, by the location's id",
				source:   "location_list or location_get",
			},
			"originStateId":      state,
			"destinationStateId": state,
			"equipmentTypeId": {
				resource: permission.ResourceEquipmentType,
				what:     "Only postings of this equipment type, by its id",
				source:   "equipment_list_types",
			},
		},
		unoffered: []string{"updatedAt"},
		resource:  permission.ResourceCarrierCapacityPosting,
	}
}

func invoiceMatchesListSpec() graphQLListSpec {
	return graphQLListSpec{
		name:         "carrier_list_invoice_matches",
		entityPlural: "carrier invoice matches",
		summary: "List carrier freight invoices matched against what each load was " +
			"expected to cost, with the variance. Each has the carrier's invoice number, the " +
			"match status, how it was matched and where the invoice came from (source: EDI, " +
			"Document or Manual). The totals can be withheld at your data access. An " +
			"invoice a carrier sent over EDI is the carrier's own text.",
		references: map[string]idReference{paramCarrierID: carrierReference},
		resource:   permission.ResourceCarrierInvoiceMatch,
		reads: readSpec{
			reads:  agenttooltrust.ExternalReadMarked,
			source: agent.TaintSourceEDI,
			rationale: "Lists carrier invoices, some of which a carrier sent over EDI with " +
				"its own invoice text; nothing changes and nothing is sent.",
		},
		shape:   matchSource,
		amounts: true,
		marks: func(row map[string]any) (agentrecord.RecordRef, bool) {
			id := stringOf(row[fieldEDICarrierInvoiceID])
			return agentrecord.RecordRef{EntityType: carrierInvoiceEntity, ID: id}, id != ""
		},
	}
}

func matchSource(row map[string]any, _ map[string]struct{}, _ clock) {
	switch {
	case stringOf(row[fieldEDICarrierInvoiceID]) != "":
		row[fieldMatchSource] = matchSourceEDI
	case stringOf(row[fieldDocumentExtraction]) != "":
		row[fieldMatchSource] = matchSourceDocument
	default:
		row[fieldMatchSource] = matchSourceManual
	}
}
