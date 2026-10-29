package documentaiextractionjobs

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentanalysis"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentaiextractionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/documentcontrol"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/shared/sliceutils"
)

const aiFieldRate = "rate"

func mergeAIAnalysis(
	fallback *documentanalysis.DocumentIntelligenceAnalysis,
	aiExtract *documentaiextractionports.AIExtractResult,
) (merged *documentanalysis.DocumentIntelligenceAnalysis, accepted bool, rejectionReason string) {
	normalized := normalizeAIExtractResult(aiExtract)
	rejectionReason = validateAIExtract(normalized)
	if rejectionReason != "" {
		return fallback, false, rejectionReason
	}

	merged = analysisFromAIExtract(normalized)
	merged.ClassifierSource = fallback.ClassifierSource
	merged.ProviderFingerprint = fallback.ProviderFingerprint
	merged.ClassificationReason = fallback.ClassificationReason
	merged.RawExcerpt = fallback.RawExcerpt
	return merged, true, ""
}

func analysisFromAIExtract(
	aiExtract *documentaiextractionports.AIExtractResult,
) *documentanalysis.DocumentIntelligenceAnalysis {
	aiExtract = normalizeAIExtractResult(aiExtract)
	if aiExtract == nil {
		return &documentanalysis.DocumentIntelligenceAnalysis{
			Kind:          documentanalysis.KindRateConfirmation,
			MissingFields: []string{},
			Signals:       []string{},
			Fields:        map[string]*documentanalysis.ReviewField{},
			Stops:         []*documentanalysis.IntelligenceStop{},
			Conflicts:     []*documentanalysis.ReviewConflict{},
		}
	}

	analysis := &documentanalysis.DocumentIntelligenceAnalysis{
		Kind:              documentanalysis.KindRateConfirmation,
		OverallConfidence: documentanalysis.ClampConfidence(aiExtract.OverallConfidence),
		ReviewStatus:      NormalizeAIReviewStatus(aiExtract.ReviewStatus),
		ClassifierSource:  "ai",
		MissingFields:     sliceutils.DedupeStrings(aiExtract.MissingFields),
		Signals:           sliceutils.DedupeStrings(aiExtract.Signals),
		Fields:            make(map[string]*documentanalysis.ReviewField, len(aiExtract.Fields)),
		Stops:             make([]*documentanalysis.IntelligenceStop, 0, len(aiExtract.Stops)),
		Conflicts:         make([]*documentanalysis.ReviewConflict, 0, len(aiExtract.Conflicts)),
	}
	for key, field := range aiExtract.Fields {
		analysis.Fields[key] = &documentanalysis.ReviewField{
			Label:           field.Label,
			Value:           field.Value,
			Confidence:      documentanalysis.ClampConfidence(field.Confidence),
			Excerpt:         field.EvidenceExcerpt,
			EvidenceExcerpt: field.EvidenceExcerpt,
			PageNumber:      field.PageNumber,
			ReviewRequired:  field.ReviewRequired,
			Conflict:        field.Conflict,
			Source:          normalizeAISource(field.Source),
		}
	}

	for _, stop := range aiExtract.Stops {
		analysis.Stops = append(analysis.Stops, &documentanalysis.IntelligenceStop{
			Sequence:            stop.Sequence,
			Role:                stop.Role,
			Name:                stop.Name,
			AddressLine1:        stop.AddressLine1,
			AddressLine2:        stop.AddressLine2,
			City:                stop.City,
			State:               stop.State,
			PostalCode:          stop.PostalCode,
			Date:                stop.Date,
			TimeWindow:          stop.TimeWindow,
			AppointmentRequired: stop.AppointmentRequired,
			PageNumber:          stop.PageNumber,
			EvidenceExcerpt:     stop.EvidenceExcerpt,
			Confidence:          documentanalysis.ClampConfidence(stop.Confidence),
			ReviewRequired:      stop.ReviewRequired,
			Source:              normalizeAISource(stop.Source),
		})
	}

	for _, conflict := range aiExtract.Conflicts {
		analysis.Conflicts = append(analysis.Conflicts, &documentanalysis.ReviewConflict{
			Key:             conflict.Key,
			Label:           conflict.Label,
			Values:          conflict.Values,
			PageNumbers:     conflict.PageNumbers,
			EvidenceExcerpt: conflict.EvidenceExcerpt,
			Source:          normalizeAISource(conflict.Source),
		})
	}

	return analysis
}

func normalizeAIExtractResult(
	result *documentaiextractionports.AIExtractResult,
) *documentaiextractionports.AIExtractResult {
	if result == nil {
		return nil
	}

	normalized := &documentaiextractionports.AIExtractResult{
		DocumentKind:      NormalizeRoutedKind(result.DocumentKind),
		OverallConfidence: result.OverallConfidence,
		ReviewStatus:      NormalizeAIReviewStatus(result.ReviewStatus),
		MissingFields:     append([]string{}, result.MissingFields...),
		Signals:           append([]string{}, result.Signals...),
		Fields: make(
			map[string]documentaiextractionports.AIDocumentField,
			len(result.Fields)+3,
		),
		Stops: make([]*documentaiextractionports.AIDocumentStop, 0, len(result.Stops)),
		Conflicts: append(
			[]*documentaiextractionports.AIDocumentConflict{},
			result.Conflicts...),
	}

	for key, field := range result.Fields {
		canonicalKey := normalizeAIFieldKey(key)
		if canonicalKey == "" {
			canonicalKey = normalizeAIFieldKey(field.Label)
		}
		if canonicalKey == "" {
			continue
		}

		field.Label = strings.TrimSpace(field.Label)
		field.Value = strings.TrimSpace(field.Value)
		field.Source = normalizeAISource(field.Source)
		if existing, ok := normalized.Fields[canonicalKey]; !ok ||
			field.PageNumber > 0 && existing.PageNumber <= 0 {
			normalized.Fields[canonicalKey] = field
		}
	}

	for _, stop := range result.Stops {
		stop.Role = normalizeAIStopRole(stop.Role)
		stop.Name = strings.TrimSpace(stop.Name)
		stop.AddressLine1 = strings.TrimSpace(stop.AddressLine1)
		stop.AddressLine2 = strings.TrimSpace(stop.AddressLine2)
		stop.City = strings.TrimSpace(stop.City)
		stop.State = strings.TrimSpace(stop.State)
		stop.PostalCode = strings.TrimSpace(stop.PostalCode)
		stop.Date = strings.TrimSpace(stop.Date)
		stop.TimeWindow = strings.TrimSpace(stop.TimeWindow)
		stop.Source = normalizeAISource(stop.Source)
		normalized.Stops = append(normalized.Stops, stop)
	}

	ensureCanonicalAIField(
		normalized,
		aiFieldRate,
		[]string{
			aiFieldRate,
			"totalrate",
			"linehaul",
			"linehaulrate",
			"freightcharge",
			"total",
			"amountdue",
		},
	)
	ensureCanonicalAIFieldFromStop(
		normalized,
		documentanalysis.StopRoleShipper,
		"Shipper",
		documentanalysis.StopRolePickup,
	)
	ensureCanonicalAIFieldFromStop(
		normalized,
		documentanalysis.StopRoleConsignee,
		"Consignee",
		documentanalysis.StopRoleDelivery,
	)

	return normalized
}

func normalizeAIFieldKey(value string) string {
	replacer := strings.NewReplacer(" ", "", "_", "", "-", "", "/", "")
	normalized := replacer.Replace(strings.TrimSpace(strings.ToLower(value)))
	switch normalized {
	case aiFieldRate,
		"totalrate",
		"linehaul",
		"linehaulrate",
		"freightcharge",
		"total",
		"amountdue":
		return aiFieldRate
	case "shipper", "shippername", "shipfrom", "originname":
		return documentanalysis.StopRoleShipper
	case "consignee", "receiver", "receivername", "deliveryto", "shipto", "destinationname":
		return documentanalysis.StopRoleConsignee
	default:
		return normalized
	}
}

func normalizeAIStopRole(role string) string {
	switch strings.TrimSpace(strings.ToLower(role)) {
	case "pickup", "origin", "shipper":
		return documentanalysis.StopRolePickup
	case "delivery", "destination", "receiver", "consignee", "drop":
		return documentanalysis.StopRoleDelivery
	default:
		return strings.TrimSpace(strings.ToLower(role))
	}
}

func ensureCanonicalAIField(
	result *documentaiextractionports.AIExtractResult,
	key string,
	aliases []string,
) {
	if result == nil {
		return
	}
	if field, ok := result.Fields[key]; ok && strings.TrimSpace(field.Value) != "" {
		if field.PageNumber > 0 {
			return
		}
	}

	for _, alias := range aliases {
		field, ok := result.Fields[alias]
		if !ok || strings.TrimSpace(field.Value) == "" {
			continue
		}
		field.Source = normalizeAISource(field.Source)
		if strings.TrimSpace(field.Label) == "" {
			field.Label = key
		}
		result.Fields[key] = field
		return
	}
}

func ensureCanonicalAIFieldFromStop(
	result *documentaiextractionports.AIExtractResult,
	key, label, role string,
) {
	if result == nil {
		return
	}
	if field, ok := result.Fields[key]; ok && strings.TrimSpace(field.Value) != "" &&
		field.PageNumber > 0 {
		return
	}

	for _, stop := range result.Stops {
		if stop.Role != role || strings.TrimSpace(stop.Name) == "" || stop.PageNumber <= 0 {
			continue
		}
		result.Fields[key] = documentaiextractionports.AIDocumentField{
			Label:           label,
			Value:           stop.Name,
			Confidence:      documentanalysis.ClampConfidence(stop.Confidence),
			EvidenceExcerpt: stop.EvidenceExcerpt,
			PageNumber:      stop.PageNumber,
			ReviewRequired:  stop.ReviewRequired,
			Conflict:        false,
			Source:          normalizeAISource(stop.Source),
		}
		return
	}
}

func validateAIExtract(result *documentaiextractionports.AIExtractResult) string {
	if result == nil ||
		!strings.EqualFold(result.DocumentKind, documentanalysis.KindRateConfirmation) {
		return "ai_candidate_invalid_document_kind"
	}
	requiredFields := []string{
		documentanalysis.StopRoleShipper,
		documentanalysis.StopRoleConsignee,
		aiFieldRate,
	}
	for _, key := range requiredFields {
		field, ok := result.Fields[key]
		if !ok || strings.TrimSpace(field.Value) == "" || field.PageNumber <= 0 {
			return "ai_candidate_missing_required_field_" + key
		}
	}

	hasPickup := false
	hasDelivery := false
	for _, stop := range result.Stops {
		if stop.PageNumber <= 0 || strings.TrimSpace(stop.EvidenceExcerpt) == "" {
			return "ai_candidate_invalid_stop_metadata"
		}
		switch strings.ToLower(strings.TrimSpace(stop.Role)) {
		case documentanalysis.StopRolePickup:
			hasPickup = true
		case documentanalysis.StopRoleDelivery:
			hasDelivery = true
		}
	}
	if !hasPickup {
		return "ai_candidate_missing_pickup_stop"
	}
	if !hasDelivery {
		return "ai_candidate_missing_delivery_stop"
	}
	return ""
}

func normalizeAISource(source string) string {
	if strings.TrimSpace(source) == "" {
		return documentaiextractionports.AIDocumentSourceAI
	}
	return source
}

func NormalizeClassifierSource(source string) string {
	switch strings.TrimSpace(strings.ToLower(source)) {
	case "ai", "template", "hybrid", "deterministic":
		return strings.TrimSpace(strings.ToLower(source))
	default:
		return "ai"
	}
}

func NormalizeAIReviewStatus(status string) string {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "ready":
		return documentanalysis.ReviewStatusReady
	case "unavailable":
		return documentanalysis.ReviewStatusUnavailable
	default:
		return documentanalysis.ReviewStatusNeedsReview
	}
}

func NormalizeRoutedKind(kind string) string {
	switch strings.TrimSpace(strings.ToLower(kind)) {
	case "rateconfirmation", "rate_confirmation":
		return documentanalysis.KindRateConfirmation
	case "billoflading", "bill_of_lading":
		return documentanalysis.KindBillOfLading
	case "proofofdelivery", "proof_of_delivery":
		return documentanalysis.KindProofOfDelivery
	default:
		return documentanalysis.KindOther
	}
}

func ToAIFingerprintHint(
	fingerprint *documentanalysis.ProviderFingerprint,
) *documentaiextractionports.AIDocumentFingerprintHint {
	if fingerprint == nil {
		return nil
	}
	return &documentaiextractionports.AIDocumentFingerprintHint{
		Provider:   fingerprint.Provider,
		KindHint:   fingerprint.KindHint,
		Confidence: fingerprint.Confidence,
		Signals:    append([]string{}, fingerprint.Signals...),
	}
}

func InferDocumentType(kind string) (*documentanalysis.InferredDocumentType, bool) {
	switch kind {
	case documentanalysis.KindRateConfirmation:
		return &documentanalysis.InferredDocumentType{
			Code:           "RATECONF",
			Name:           "Rate Confirmation",
			Category:       documenttype.CategoryShipment,
			Classification: documenttype.ClassificationPublic,
			Color:          "#0f766e",
		}, true
	case documentanalysis.KindBillOfLading:
		return &documentanalysis.InferredDocumentType{
			Code:           "BOL",
			Name:           "Bill of Lading",
			Category:       documenttype.CategoryShipment,
			Classification: documenttype.ClassificationPublic,
			Color:          "#f59e0b",
		}, true
	case documentanalysis.KindProofOfDelivery:
		return &documentanalysis.InferredDocumentType{
			Code:           "POD",
			Name:           "Proof of Delivery",
			Category:       documenttype.CategoryShipment,
			Classification: documenttype.ClassificationPublic,
			Color:          "#8b5cf6",
		}, true
	case documentanalysis.KindInvoice:
		return &documentanalysis.InferredDocumentType{
			Code:           "INVOICE",
			Name:           "Invoice",
			Category:       documenttype.CategoryInvoice,
			Classification: documenttype.ClassificationPublic,
			Color:          "#3b82f6",
		}, true
	default:
		return nil, false
	}
}

func CanApplyKindToResource(resourceType, kind string) bool {
	switch kind {
	case documentanalysis.KindRateConfirmation,
		documentanalysis.KindBillOfLading,
		documentanalysis.KindProofOfDelivery,
		documentanalysis.KindInvoice:
		return strings.EqualFold(resourceType, "shipment")
	default:
		return true
	}
}

func CanGenerateShipmentDraft(
	control *documentcontrol.DocumentControl,
	resourceType, kind string,
) bool {
	return control != nil &&
		control.EnableShipmentDraftExtraction &&
		control.AllowsShipmentDraftResource(resourceType) &&
		kind == documentanalysis.KindRateConfirmation
}

func HasUsableShipmentDraft(intelligence *documentanalysis.DocumentIntelligenceAnalysis) bool {
	if strings.EqualFold(
		strings.TrimSpace(intelligence.ReviewStatus),
		documentanalysis.ReviewStatusReady,
	) {
		return true
	}

	if hasMeaningfulStopForRole(intelligence.Stops, documentanalysis.StopRolePickup) &&
		hasMeaningfulStopForRole(intelligence.Stops, documentanalysis.StopRoleDelivery) {
		return true
	}

	return hasMeaningfulField(intelligence.Fields, documentanalysis.StopRoleShipper) &&
		hasMeaningfulField(intelligence.Fields, documentanalysis.StopRoleConsignee) &&
		hasMeaningfulField(intelligence.Fields, aiFieldRate)
}

func hasMeaningfulStopForRole(stops []*documentanalysis.IntelligenceStop, role string) bool {
	for _, i := range stops {
		if !strings.EqualFold(strings.TrimSpace(i.Role), role) {
			continue
		}
		if hasReviewableStopData(i) {
			return true
		}
	}
	return false
}

func hasReviewableStopData(stop *documentanalysis.IntelligenceStop) bool {
	return strings.TrimSpace(stop.Name) != "" ||
		strings.TrimSpace(stop.AddressLine1) != "" ||
		strings.TrimSpace(stop.AddressLine2) != "" ||
		strings.TrimSpace(stop.City) != "" ||
		strings.TrimSpace(stop.State) != "" ||
		strings.TrimSpace(stop.PostalCode) != "" ||
		strings.TrimSpace(stop.Date) != "" ||
		strings.TrimSpace(stop.TimeWindow) != ""
}

func hasMeaningfulField(fields map[string]*documentanalysis.ReviewField, key string) bool {
	field, ok := fields[key]
	if !ok {
		return false
	}
	return strings.TrimSpace(field.Value) != ""
}

func ShipmentDraftDataFromAIExtract(
	aiExtract *documentaiextractionports.AIExtractResult,
) map[string]any {
	return analysisFromAIExtract(aiExtract).ToMap()
}
