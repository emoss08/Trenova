package partyquerytools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/filtercatalog"
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

var carrierReference = readspec.Reference{
	Resource: permission.ResourceCarrier,
	What:     "Only this carrier's records, by the carrier's id",
	Source:   carrierSource,
}

func CarrierReadSpecs() []readspec.List {
	return []readspec.List{
		carrierIntelEventsListSpec(), capacityPostingsListSpec(), invoiceMatchesListSpec(),
	}
}

func carrierIntelEventsListSpec() readspec.List {
	return readspec.List{
		Name:         "carrier_list_intelligence_events",
		EntityPlural: "carrier intelligence events",
		Summary: "List carrier intelligence events, newest first: changes found on " +
			"carriers' authority, insurance and safety records. Each row gives the carrier, " +
			"the category, the severity, what the rule did, the summary, when it was " +
			"detected and whether anyone acknowledged or resolved it. Pass a row's id to " +
			"carrier_get_intelligence_event for its prior and current values, or to " +
			"carrier_acknowledge_intelligence_event or carrier_resolve_intelligence_event.",
		SearchTerms: []string{"carrier risk alerts", "authority or insurance problems"},
		QueryNote: "Words to find in the carrier's name, its DOT number or the event's " +
			"summary.",
		Names: map[string]string{
			"statuses":       querytool.ParamStatus,
			"severities":     querytool.ParamSeverity,
			"categories":     querytool.ParamCategory,
			"detectedAtFrom": "detectedFrom",
			"detectedAtTo":   "detectedTo",
			"subjectId":      "subject",
		},
		Unoffered: []string{
			"status", "severity", "category", "carrierId", "subjectType", "subjectId",
			fieldFieldPath, fieldRuleCode, fieldVendorChangedAt,
			"acknowledgedById", "resolvedById", "resolutionNote", "snapshotId", "updatedAt",
			"createdAt",
		},
		Notes: map[string]string{
			querytool.ParamStatus:   "Open and Acknowledged ones are not settled yet.",
			querytool.ParamCategory: "The parts of a carrier's record, such as Authority or Insurance.",
		},
		References: map[string]readspec.Reference{querytool.ParamCarrierID: carrierReference},
		Resource:   permission.ResourceCarrierIntelligence,
	}
}

func capacityPostingsListSpec() readspec.List {
	state := readspec.Reference{
		Kinds: []permission.RecordKind{permission.KindUSState},
		What:  "The state's id",
		Source: "an originStateId or destinationStateId on a row of this tool, or a " +
			"location's stateId from location_get",
	}

	return readspec.List{
		Name:         "carrier_list_capacity_postings",
		EntityPlural: "carrier capacity postings",
		Summary: "List trucks carriers have said they have available: carrier, origin, " +
			"destination, equipment, the window and the rate asked. Pass openOnly for the " +
			"offers whose window has not closed.",
		SearchTerms: []string{"carrier capacity", "available trucks", "truck postings"},
		References: map[string]readspec.Reference{
			querytool.ParamCarrierID: carrierReference,
			"originLocationId": {
				Resource: permission.ResourceLocation,
				What:     "Only postings from this location, by the location's id",
				Source:   "location_list or location_get",
			},
			"originStateId":      state,
			"destinationStateId": state,
			"equipmentTypeId": {
				Resource: permission.ResourceEquipmentType,
				What:     "Only postings of this equipment type, by its id",
				Source:   "equipment_list_types",
			},
		},
		Unoffered: []string{"updatedAt"},
		Resource:  permission.ResourceCarrierCapacityPosting,
	}
}

func invoiceMatchesListSpec() readspec.List {
	return readspec.List{
		Name:         "carrier_list_invoice_matches",
		EntityPlural: "carrier invoice matches",
		Summary: "List carrier freight invoices matched against what each load was " +
			"expected to cost, with the variance. Each has the carrier's invoice number, the " +
			"match status, how it was matched and where the invoice came from (source: EDI, " +
			"Document or Manual). The totals can be withheld at your data access. An " +
			"invoice a carrier sent over EDI is the carrier's own text.",
		References: map[string]readspec.Reference{querytool.ParamCarrierID: carrierReference},
		Resource:   permission.ResourceCarrierInvoiceMatch,
		Reads: querytool.ReadSpec{
			Reads:  agenttooltrust.ExternalReadMarked,
			Source: agent.TaintSourceEDI,
			Rationale: "Lists carrier invoices, some of which a carrier sent over EDI with " +
				"its own invoice text; nothing changes and nothing is sent.",
		},
		Shape:   matchSource,
		Amounts: true,
		Marks: func(row map[string]any) (agentrecord.RecordRef, bool) {
			id := querytool.StringOf(row[fieldEDICarrierInvoiceID])
			return agentrecord.RecordRef{EntityType: carrierInvoiceEntity, ID: id}, id != ""
		},
	}
}

func matchSource(row map[string]any, _ map[string]struct{}, _ filtercatalog.Clock) {
	switch {
	case querytool.StringOf(row[fieldEDICarrierInvoiceID]) != "":
		row[fieldMatchSource] = matchSourceEDI
	case querytool.StringOf(row[fieldDocumentExtraction]) != "":
		row[fieldMatchSource] = matchSourceDocument
	default:
		row[fieldMatchSource] = matchSourceManual
	}
}
