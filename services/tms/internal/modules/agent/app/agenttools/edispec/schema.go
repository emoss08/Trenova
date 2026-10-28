package edispec

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediconnectiondraft"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediconnectiondraftports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/jsonschemautils"
)

const (
	ParamDocumentID     = "documentId"
	ParamPartnerID      = "ediPartnerId"
	ParamCustomerID     = "customerId"
	ParamValues         = "values"
	ParamRequirements   = "requirements"
	ParamMappings       = "mappings"
	ParamGaps           = "gaps"
	ParamResolutions    = "resolutions"
	ParamDocumentDigest = "documentDigest"

	maxValueRunes   = 500
	maxQuoteRunes   = 400
	maxSectionRunes = 120
	maxCodeRunes    = 20
	maxCodes        = 50
	maxNoteRunes    = 300
	maxDigestRunes  = 80
	maxSegmentRunes = 3
	maxElement      = 99
	maxPage         = 10000

	keyCitation   = "citation"
	keyConfidence = "confidence"
)

var (
	fieldSource = agenttoolschema.Source("edi.connectionDraftField", ediconnectiondraft.FieldKeys())
	setSource   = agenttoolschema.Source(
		"edi.connectionDraftTransactionSet",
		[]edidocument.TransactionSet{
			edidocument.TransactionSet204,
			edidocument.TransactionSet210,
			edidocument.TransactionSet214,
			edidocument.TransactionSet990,
		},
	)
	directionSource = agenttoolschema.Source("edi.documentDirection", []string{
		string(edidocument.DocumentDirectionInbound),
		string(edidocument.DocumentDirectionOutbound),
	})
	usageSource = agenttoolschema.Source(
		"edi.requirementUsage",
		[]ediconnectiondraftports.RequirementUsage{
			ediconnectiondraftports.RequirementUsageMandatory,
			ediconnectiondraftports.RequirementUsageOptional,
			ediconnectiondraftports.RequirementUsageConditional,
			ediconnectiondraftports.RequirementUsageNotUsed,
		},
	)
	entitySource = agenttoolschema.Source("edi.mappingEntityType", []edi.MappingEntityType{
		edi.MappingEntityTypeCustomer,
		edi.MappingEntityTypeServiceType,
		edi.MappingEntityTypeShipmentType,
		edi.MappingEntityTypeFormulaTemplate,
		edi.MappingEntityTypeLocation,
		edi.MappingEntityTypeCommodity,
		edi.MappingEntityTypeAccessorialCharge,
		edi.MappingEntityTypeServiceFailureReasonCode,
	})
	gapSource = agenttoolschema.Source("edi.specGapKind", []ediconnectiondraftports.GapKind{
		ediconnectiondraftports.GapKindUnsupported,
		ediconnectiondraftports.GapKindAmbiguous,
	})
	resolutionSource = agenttoolschema.Source(
		"edi.draftResolutionSource",
		[]ediconnectiondraftports.ResolutionSource{
			ediconnectiondraftports.ResolutionSourceCandidate,
			ediconnectiondraftports.ResolutionSourcePerson,
		},
	)
)

func described(property map[string]any, description string) map[string]any {
	property[toolschema.KeyDescription] = description
	return property
}

func integer(minimum, maximum int, description string) map[string]any {
	property := jsonschemautils.Integer(description)
	property[toolschema.KeyMinimum] = minimum
	property[toolschema.KeyMaximum] = maximum
	return property
}

func citationSchema() map[string]any {
	return described(jsonschemautils.Object(map[string]any{
		"page": integer(1, maxPage,
			"The page the quote is on, counting from 1 as agent_read_attachment numbers them."),
		"quote": described(jsonschemautils.String(maxQuoteRunes),
			"The exact words on that page that state the value. Copy them; never paraphrase."),
		"section": described(jsonschemautils.String(maxSectionRunes),
			"The guide's own section number or heading, when it has one."),
	}, "page", "quote"), "Where the document says it.")
}

func confidenceSchema() map[string]any {
	return described(jsonschemautils.Number(0, 1),
		"How sure you are that the quote means this value, from 0 to 1. Below 0.8 is held for "+
			"the person to decide.")
}

func ParamSchema(withDigest bool) map[string]any {
	properties := map[string]any{
		ParamDocumentID: agenttoolschema.RecordIDText(permission.ResourceDocument,
			"The partner's implementation guide the person attached to this conversation, from "+
				"agent_read_attachment."),
		ParamPartnerID: agenttoolschema.KindID("The partner to update when the person names one, "+
			"from edi_list_partners. Leave it out and the draft matches one by code, ISA ID or "+
			"AS2 ID, or plans a new partner.", permission.KindEDIPartner),
		ParamCustomerID: agenttoolschema.RecordIDText(permission.ResourceCustomer,
			"The customer this partner's documents are for, by id from customer_list, only when "+
				"the person names which customer. Leave it out otherwise, and never pass a name "+
				"or guess one from the partner's."),
		ParamValues: jsonschemautils.DescribedArray(
			"Every setting the guide states, each with the quote that states it.",
			jsonschemautils.Object(map[string]any{
				"field": agenttoolschema.Enum("The setting the value fills. The our, local "+
					"and carrier settings are this company's, the carrier the guide is written "+
					"for; the partner settings are the trading partner's.", fieldSource),
				"value": described(jsonschemautils.String(maxValueRunes),
					"The value as the guide gives it."),
				keyConfidence: confidenceSchema(),
				keyCitation:   citationSchema(),
			}, "field", "value", keyConfidence, keyCitation),
			ediconnectiondraft.MaxValues,
		),
		ParamRequirements: jsonschemautils.DescribedArray(
			"What the guide requires of a segment or element of a transaction set.",
			jsonschemautils.Object(map[string]any{
				"transactionSet": agenttoolschema.Enum("", setSource),
				"direction": agenttoolschema.Enum(
					"Inbound is what the partner sends you; Outbound is what you send them.",
					directionSource,
				),
				"segment": described(jsonschemautils.String(maxSegmentRunes),
					"The segment, such as B2, N1 or AT7."),
				"elementPosition": integer(
					1,
					maxElement,
					"The element's position, such as 4 for B2-04. Leave it out for the segment itself.",
				),
				"usage": agenttoolschema.Enum("", usageSource),
				"codes": jsonschemautils.DescribedArray("The only codes the guide allows there.",
					jsonschemautils.String(maxCodeRunes), maxCodes),
				keyCitation: citationSchema(),
			}, "transactionSet", "direction", "segment", "usage", keyCitation),
			ediconnectiondraft.MaxRequirements,
		),
		ParamMappings: jsonschemautils.DescribedArray(
			"A code the partner uses and the Trenova record it means.",
			jsonschemautils.Object(map[string]any{
				"entityType": agenttoolschema.Enum("", entitySource),
				"sourceValue": described(jsonschemautils.String(maxSectionRunes),
					"The partner's code as the guide lists it."),
				"sourceLabel": described(jsonschemautils.String(maxSectionRunes),
					"The guide's meaning for the code."),
				"targetId": agenttoolschema.IDText(
					"The Trenova record it maps to, of the entityType's kind: from customer_list, " +
						"masterdata_list_service_types, masterdata_list_shipment_types, formula_list_templates, " +
						"location_list, masterdata_list_commodities, masterdata_list_accessorial_charges or " +
						"exception_list_service_failure_reason_codes. Never guess one.",
				),
				keyConfidence: confidenceSchema(),
				keyCitation:   citationSchema(),
			}, "entityType", "sourceValue", "targetId", keyConfidence, keyCitation),
			ediconnectiondraft.MaxMappings,
		),
		ParamGaps: jsonschemautils.DescribedArray(
			"What the guide asks for that Trenova does not do, or leaves unclear.",
			jsonschemautils.Object(map[string]any{
				"kind": agenttoolschema.Enum("", gapSource),
				"summary": described(jsonschemautils.String(ediconnectiondraft.MaxSummaryRunes),
					"What the person needs to know, in a sentence."),
				keyCitation: citationSchema(),
			}, "kind", "summary", keyCitation),
			ediconnectiondraft.MaxGaps,
		),
		ParamResolutions: jsonschemautils.DescribedArray(
			"The person's answer to a setting the last draft held for them.",
			jsonschemautils.Object(map[string]any{
				"field": agenttoolschema.Enum("", fieldSource),
				"value": described(jsonschemautils.String(maxValueRunes), "The value they chose."),
				"source": agenttoolschema.Enum(
					"Candidate when they picked one of the draft's values; Person when they gave it.",
					resolutionSource,
				),
				"note": described(jsonschemautils.String(maxNoteRunes),
					"What they said, in their words."),
			}, "field", "value", "source"),
			ediconnectiondraft.MaxResolutions,
		),
	}
	required := []string{ParamDocumentID, ParamValues}
	if withDigest {
		properties[ParamDocumentDigest] = described(jsonschemautils.String(maxDigestRunes),
			"The documentDigest edi_draft_connection returned for this guide.")
		required = append(required, ParamDocumentDigest)
	}
	return jsonschemautils.Object(properties, required...)
}
