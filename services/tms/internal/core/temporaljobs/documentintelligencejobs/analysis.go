package documentintelligencejobs

import (
	"strings"

	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/stringutils"
)

func analyzeDocument(
	classification *ClassificationResult,
	extracted *ExtractionResult,
) *DocumentIntelligenceAnalysis {
	if extracted == nil {
		extracted = &ExtractionResult{}
	}

	text := extracted.Text
	analysis := &DocumentIntelligenceAnalysis{
		Kind:                 classification.Kind,
		ReviewStatus:         reviewStatusNeedsReview,
		MissingFields:        []string{},
		Signals:              append([]string{}, classification.Signals...),
		ClassifierSource:     classification.Source,
		ProviderFingerprint:  classification.ProviderFingerprint,
		ClassificationReason: classification.Reason,
		Conflicts:            []*ReviewConflict{},
		Fields:               make(map[string]*ReviewField),
		Stops:                []*IntelligenceStop{},
		RawExcerpt:           stringutils.TruncateAndTrim(strings.ReplaceAll(stringutils.Truncate(text, rawExcerptPreTruncateLen), "\r", ""), rawExcerptMaxLen),
	}

	required := requiredFieldsForKind(classification.Kind)

	switch classification.Kind {
	case kindRateConfirmation:
		analyzeRateConfirmation(analysis, extracted)
	case kindBillOfLading:
		analyzeBillOfLading(analysis, extracted)
	case kindProofOfDelivery:
		analyzeProofOfDelivery(analysis, extracted)
	case kindInvoice:
		analyzeInvoice(analysis, extracted)
	}

	finalizeAnalysis(analysis, classification, required)
	return analysis
}

func finalizeAnalysis(
	analysis *DocumentIntelligenceAnalysis,
	classification *ClassificationResult,
	required []struct {
		key   string
		label string
	},
) {
	totalConfidence := classification.Confidence
	fieldCount := 1.0
	for _, field := range analysis.Fields {
		totalConfidence += field.Confidence
		fieldCount++
	}
	analysis.OverallConfidence = clampConfidence(totalConfidence / fieldCount)

	for _, field := range required {
		if _, ok := analysis.Fields[field.key]; !ok {
			analysis.MissingFields = append(analysis.MissingFields, field.label)
		}
	}
	if classification.Kind == kindRateConfirmation {
		if !hasStopRole(analysis.Stops, stopRolePickup) {
			analysis.MissingFields = sliceutils.AppendIfMissing(analysis.MissingFields, "Pickup Stop")
		}
		if !hasStopRole(analysis.Stops, stopRoleDelivery) {
			analysis.MissingFields = sliceutils.AppendIfMissing(analysis.MissingFields, "Delivery Stop")
		}
	}

	if len(analysis.Fields) == 0 {
		analysis.ReviewStatus = reviewStatusUnavailable
		return
	}

	analysis.ReviewStatus = resolveReviewStatus(analysis, classification.Kind)
}

func resolveReviewStatus(
	analysis *DocumentIntelligenceAnalysis,
	kind string,
) string {
	noConflicts := len(analysis.Conflicts) == 0
	noMissing := len(analysis.MissingFields) == 0
	highConfidence := analysis.OverallConfidence >= reviewReadyConfidenceThreshold
	baseReady := highConfidence && noMissing && noConflicts

	switch kind {
	case kindRateConfirmation:
		if baseReady &&
			hasStopRole(analysis.Stops, stopRolePickup) &&
			hasStopRole(analysis.Stops, stopRoleDelivery) &&
			!hasReviewRequiredStop(analysis.Stops) {
			return reviewStatusReady
		}
	case kindBillOfLading:
		if baseReady {
			return reviewStatusReady
		}
	case kindProofOfDelivery:
		if baseReady &&
			!analysis.Fields["deliveryWindow"].ReviewRequired &&
			!analysis.Fields["signature"].ReviewRequired {
			return reviewStatusReady
		}
	default:
		if highConfidence && len(analysis.MissingFields) <= 1 && noConflicts {
			return reviewStatusReady
		}
	}

	return reviewStatusNeedsReview
}

func analyzeRateConfirmation(analysis *DocumentIntelligenceAnalysis, extracted *ExtractionResult) {
	analysis.Stops = extractRateConfirmationStops(extracted.Pages)
	addFieldFromPages(&RegexValueFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "shipper", Label: "Shipper", Regex: shipperRegex,
		Pages: extracted.Pages, Confidence: 0.9,
	})
	addFieldFromPages(&RegexValueFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "consignee", Label: "Consignee", Regex: consigneeRegex,
		Pages: extracted.Pages, Confidence: 0.9,
	})
	addStopTimingField(&AddStopTimingFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "pickupWindow", Label: "Pickup Window",
		Stop: firstStopByRole(analysis.Stops, stopRolePickup), Confidence: 0.88,
	})
	addStopTimingField(&AddStopTimingFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "deliveryWindow", Label: "Delivery Window",
		Stop: firstStopByRole(analysis.Stops, stopRoleDelivery), Confidence: 0.88,
	})
	addFieldFromPages(&RegexValueFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "referenceNumber", Label: "Reference Number", Regex: referenceRegex,
		Pages: extracted.Pages, Confidence: 0.8,
	})
	addFieldFromPages(&RegexValueFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "commodity", Label: "Commodity", Regex: commodityRegex,
		Pages: extracted.Pages, Confidence: 0.78,
	})
	addFieldFromPages(&RegexValueFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "instructions", Label: "Instructions", Regex: instructionsRegex,
		Pages: extracted.Pages, Confidence: 0.72,
	})
	addCurrencyFieldFromPages(&RegexValueFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "rate", Label: "Rate", Regex: rateRegex,
		Pages: extracted.Pages, Confidence: 0.92, Signal: "rate amount",
	})
	addRegexValueFieldFromPages(&RegexValueFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "equipmentType", Label: "Equipment Type", Regex: equipmentRegex,
		Pages: extracted.Pages, Confidence: 0.82,
		Signal: "equipment type", ReviewRequired: false,
	})
	addWeightFieldFromPages(&AddWeightFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Text: "weight", Signal: "weight",
	}, extracted.Pages)
	analysis.Conflicts = append(analysis.Conflicts, collectFieldConflicts(analysis.Fields)...)
	analysis.Conflicts = append(analysis.Conflicts, collectStopConflicts(analysis.Stops)...)
}

func analyzeBillOfLading(analysis *DocumentIntelligenceAnalysis, extracted *ExtractionResult) {
	addFieldFromSectionLabels(&AddFieldFromSectionLabelsParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "shipper", Label: "Shipper", Pages: extracted.Pages,
		Labels:     []string{"ship from", "shipper", "shipper name", "shipper information"},
		Confidence: 0.93, Signal: "shipper",
		ReviewRequired: false, Extractor: extractEntityNameFromSection,
	})
	addFieldFromSectionLabels(&AddFieldFromSectionLabelsParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "consignee", Label: "Consignee", Pages: extracted.Pages,
		Labels:     []string{"ship to", "consignee", "receiver", "delivery to"},
		Confidence: 0.93, Signal: "consignee",
		ReviewRequired: false, Extractor: extractEntityNameFromSection,
	})
	addFieldFromSectionLabels(&AddFieldFromSectionLabelsParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "commodity", Label: "Commodity", Pages: extracted.Pages,
		Labels:     []string{"commodity", "description", "product", "articles"},
		Confidence: 0.86, Signal: "commodity",
		ReviewRequired: false, Extractor: extractCommodityFromSection,
	})
	addRegexValueFieldFromPages(&RegexValueFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "referenceNumber", Label: "BOL / Reference Number",
		Regex: bolReferenceRegex, Pages: extracted.Pages,
		Confidence: 0.85, Signal: "reference number", ReviewRequired: false,
	})
	addRegexValueFieldFromPages(&RegexValueFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "pieceCount", Label: "Pieces / Packages",
		Regex: pieceCountRegex, Pages: extracted.Pages,
		Confidence: 0.76, Signal: "piece count", ReviewRequired: true,
	})
	addWeightFieldFromPages(&AddWeightFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Text: "weight", Signal: "weight",
	}, extracted.Pages)
	analysis.Conflicts = append(analysis.Conflicts, collectFieldConflicts(analysis.Fields)...)
}

func analyzeProofOfDelivery(analysis *DocumentIntelligenceAnalysis, extracted *ExtractionResult) {
	addFieldFromSectionLabels(&AddFieldFromSectionLabelsParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "consignee", Label: "Consignee", Pages: extracted.Pages,
		Labels: []string{
			"consignee", "receiver name", "delivery to", "delivered to", "received by",
		},
		Confidence: 0.91, Signal: "consignee",
		ReviewRequired: false, Extractor: extractEntityNameFromSection,
	})
	addFieldFromSectionLabels(&AddFieldFromSectionLabelsParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "deliveryWindow", Label: "Delivery", Pages: extracted.Pages,
		Labels: []string{
			"delivery date", "delivered on", "received on", "date delivered",
		},
		Confidence: 0.89, Signal: "delivery",
		ReviewRequired: false, Extractor: extractDeliveryFieldFromSection,
	})
	if _, ok := analysis.Fields["deliveryWindow"]; !ok {
		addFieldFromPages(&RegexValueFieldParams{
			Fields: analysis.Fields, Signals: &analysis.Signals,
			Key: "deliveryWindow", Label: "Delivery", Regex: deliveryRegex,
			Pages: extracted.Pages, Confidence: 0.86,
		})
	}
	addFieldFromSectionLabels(&AddFieldFromSectionLabelsParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "signature", Label: "Signature", Pages: extracted.Pages,
		Labels: []string{
			"receiver signature", "consignee signature", "signature",
			"received by", "signed by",
		},
		Confidence: 0.82, Signal: "signature",
		ReviewRequired: false, Extractor: extractSignatureFromSection,
	})
	addRegexValueFieldFromPages(&RegexValueFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "referenceNumber", Label: "Reference Number",
		Regex: podReferenceRegex, Pages: extracted.Pages,
		Confidence: 0.82, Signal: "reference number", ReviewRequired: false,
	})
	addFieldFromSectionLabels(&AddFieldFromSectionLabelsParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "receiptNotes", Label: "Receipt Notes", Pages: extracted.Pages,
		Labels: []string{
			"remarks", "exceptions", "received in good order", "delivery status",
		},
		Confidence: 0.72, Signal: "receipt notes",
		ReviewRequired: true, Extractor: extractFreeformSectionValue,
	})
	analysis.Conflicts = append(analysis.Conflicts, collectFieldConflicts(analysis.Fields)...)
}

func analyzeInvoice(analysis *DocumentIntelligenceAnalysis, extracted *ExtractionResult) {
	addFieldFromPages(&RegexValueFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "referenceNumber", Label: "Invoice Number", Regex: referenceRegex,
		Pages: extracted.Pages, Confidence: 0.84,
	})
	addFieldFromPages(&RegexValueFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "invoiceDate", Label: "Invoice Date", Regex: invoiceDateRegex,
		Pages: extracted.Pages, Confidence: 0.88,
	})
	addFieldFromPages(&RegexValueFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "dueDate", Label: "Due Date", Regex: dueDateRegex,
		Pages: extracted.Pages, Confidence: 0.88,
	})
	addFieldFromPages(&RegexValueFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "shipper", Label: "Bill To / Shipper", Regex: shipperRegex,
		Pages: extracted.Pages, Confidence: 0.72,
	})
	addCurrencyFieldFromPages(&RegexValueFieldParams{
		Fields: analysis.Fields, Signals: &analysis.Signals,
		Key: "totalDue", Label: "Total Due", Regex: totalDueRegex,
		Pages: extracted.Pages, Confidence: 0.93, Signal: "total due",
	})
}

func requiredFieldsForKind(kind string) []struct {
	key   string
	label string
} {
	switch kind {
	case kindRateConfirmation:
		return []struct {
			key   string
			label string
		}{
			{key: "shipper", label: "Shipper"},
			{key: "consignee", label: "Consignee"},
			{key: "pickupWindow", label: "Pickup Window"},
			{key: "deliveryWindow", label: "Delivery Window"},
			{key: "rate", label: "Rate"},
		}
	case kindBillOfLading:
		return []struct {
			key   string
			label string
		}{
			{key: "shipper", label: "Shipper"},
			{key: "consignee", label: "Consignee"},
			{key: "commodity", label: "Commodity"},
			{key: "referenceNumber", label: "Reference Number"},
		}
	case kindProofOfDelivery:
		return []struct {
			key   string
			label string
		}{
			{key: "consignee", label: "Consignee"},
			{key: "deliveryWindow", label: "Delivery"},
			{key: "signature", label: "Signature"},
		}
	case kindInvoice:
		return []struct {
			key   string
			label string
		}{
			{key: "referenceNumber", label: "Invoice Number"},
			{key: "invoiceDate", label: "Invoice Date"},
			{key: "dueDate", label: "Due Date"},
			{key: "totalDue", label: "Total Due"},
		}
	default:
		return nil
	}
}
