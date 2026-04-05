package documentintelligencejobs

import (
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	services "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/floatutils"
)

func mergeAIAnalysis(
	fallback *DocumentIntelligenceAnalysis,
	aiExtract *services.AIExtractResult,
) (merged *DocumentIntelligenceAnalysis, accepted bool, rejectionReason string) {
	normalized := normalizeAIExtractResult(aiExtract)
	rejectionReason = validateAIExtract(normalized)
	if rejectionReason != "" {
		return fallback, false, rejectionReason
	}

	merged = analysisFromAIExtract(normalized)
	merged.ClassifierSource = fallback.ClassifierSource
	merged.ProviderFingerprint = fallback.ProviderFingerprint
	merged.ClassificationReason = fallback.ClassificationReason
	merged.ParsingRuleMetadata = fallback.ParsingRuleMetadata
	merged.RawExcerpt = fallback.RawExcerpt
	return merged, true, ""
}

func analysisFromAIExtract(aiExtract *services.AIExtractResult) *DocumentIntelligenceAnalysis {
	aiExtract = normalizeAIExtractResult(aiExtract)
	if aiExtract == nil {
		return &DocumentIntelligenceAnalysis{
			Kind:          kindRateConfirmation,
			MissingFields: []string{},
			Signals:       []string{},
			Fields:        map[string]*ReviewField{},
			Stops:         []*IntelligenceStop{},
			Conflicts:     []*ReviewConflict{},
		}
	}

	analysis := &DocumentIntelligenceAnalysis{
		Kind:              kindRateConfirmation,
		OverallConfidence: clampConfidence(aiExtract.OverallConfidence),
		ReviewStatus:      normalizeAIReviewStatus(aiExtract.ReviewStatus),
		ClassifierSource:  "ai",
		MissingFields:     dedupeStrings(aiExtract.MissingFields),
		Signals:           dedupeStrings(aiExtract.Signals),
		Fields:            make(map[string]*ReviewField, len(aiExtract.Fields)),
		Stops:             make([]*IntelligenceStop, 0, len(aiExtract.Stops)),
		Conflicts:         make([]*ReviewConflict, 0, len(aiExtract.Conflicts)),
	}
	for key, field := range aiExtract.Fields {
		analysis.Fields[key] = &ReviewField{
			Label:           field.Label,
			Value:           field.Value,
			Confidence:      clampConfidence(field.Confidence),
			Excerpt:         field.EvidenceExcerpt,
			EvidenceExcerpt: field.EvidenceExcerpt,
			PageNumber:      field.PageNumber,
			ReviewRequired:  field.ReviewRequired,
			Conflict:        field.Conflict,
			Source:          normalizeAISource(field.Source),
		}
	}

	for _, stop := range aiExtract.Stops {
		analysis.Stops = append(analysis.Stops, &IntelligenceStop{
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
			Confidence:          clampConfidence(stop.Confidence),
			ReviewRequired:      stop.ReviewRequired,
			Source:              normalizeAISource(stop.Source),
		})
	}

	for _, conflict := range aiExtract.Conflicts {
		analysis.Conflicts = append(analysis.Conflicts, &ReviewConflict{
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

func normalizeAIExtractResult(result *services.AIExtractResult) *services.AIExtractResult {
	if result == nil {
		return nil
	}

	normalized := &services.AIExtractResult{
		DocumentKind:      normalizeRoutedKind(result.DocumentKind),
		OverallConfidence: result.OverallConfidence,
		ReviewStatus:      normalizeAIReviewStatus(result.ReviewStatus),
		MissingFields:     append([]string{}, result.MissingFields...),
		Signals:           append([]string{}, result.Signals...),
		Fields:            make(map[string]services.AIDocumentField, len(result.Fields)+3),
		Stops:             make([]*services.AIDocumentStop, 0, len(result.Stops)),
		Conflicts:         append([]*services.AIDocumentConflict{}, result.Conflicts...),
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
		"rate",
		[]string{
			"rate",
			"totalrate",
			"linehaul",
			"linehaulrate",
			"freightcharge",
			"total",
			"amountdue",
		},
	)
	ensureCanonicalAIFieldFromStop(normalized, stopRoleShipper, "Shipper", stopRolePickup)
	ensureCanonicalAIFieldFromStop(normalized, stopRoleConsignee, "Consignee", stopRoleDelivery)

	return normalized
}

func normalizeAIFieldKey(value string) string {
	replacer := strings.NewReplacer(" ", "", "_", "", "-", "", "/", "")
	normalized := replacer.Replace(strings.TrimSpace(strings.ToLower(value)))
	switch normalized {
	case "rate", "totalrate", "linehaul", "linehaulrate", "freightcharge", "total", "amountdue":
		return "rate"
	case "shipper", "shippername", "shipfrom", "originname":
		return stopRoleShipper
	case "consignee", "receiver", "receivername", "deliveryto", "shipto", "destinationname":
		return stopRoleConsignee
	default:
		return normalized
	}
}

func normalizeAIStopRole(role string) string {
	switch strings.TrimSpace(strings.ToLower(role)) {
	case "pickup", "origin", "shipper":
		return stopRolePickup
	case "delivery", "destination", "receiver", "consignee", "drop":
		return stopRoleDelivery
	default:
		return strings.TrimSpace(strings.ToLower(role))
	}
}

func ensureCanonicalAIField(result *services.AIExtractResult, key string, aliases []string) {
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

func ensureCanonicalAIFieldFromStop(result *services.AIExtractResult, key, label, role string) {
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
		result.Fields[key] = services.AIDocumentField{
			Label:           label,
			Value:           stop.Name,
			Confidence:      clampConfidence(stop.Confidence),
			EvidenceExcerpt: stop.EvidenceExcerpt,
			PageNumber:      stop.PageNumber,
			ReviewRequired:  stop.ReviewRequired,
			Conflict:        false,
			Source:          normalizeAISource(stop.Source),
		}
		return
	}
}

func validateAIExtract(result *services.AIExtractResult) string {
	if result == nil || !strings.EqualFold(result.DocumentKind, kindRateConfirmation) {
		return "ai_candidate_invalid_document_kind"
	}
	requiredFields := []string{stopRoleShipper, stopRoleConsignee, "rate"}
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
		case stopRolePickup:
			hasPickup = true
		case stopRoleDelivery:
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
		return "ai"
	}
	return source
}

func normalizeClassifierSource(source string) string {
	switch strings.TrimSpace(strings.ToLower(source)) {
	case "ai", "template", "hybrid", "deterministic":
		return strings.TrimSpace(strings.ToLower(source))
	default:
		return "ai"
	}
}

func normalizeAIReviewStatus(status string) string {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "ready":
		return reviewStatusReady
	case "unavailable":
		return reviewStatusUnavailable
	default:
		return reviewStatusNeedsReview
	}
}

func normalizeRoutedKind(kind string) string {
	switch strings.TrimSpace(strings.ToLower(kind)) {
	case "rateconfirmation", "rate_confirmation":
		return kindRateConfirmation
	case "billoflading", "bill_of_lading":
		return kindBillOfLading
	case "proofofdelivery", "proof_of_delivery":
		return kindProofOfDelivery
	default:
		return kindOther
	}
}

func toAIFingerprintHint(fingerprint *ProviderFingerprint) *services.AIDocumentFingerprintHint {
	if fingerprint == nil {
		return nil
	}
	return &services.AIDocumentFingerprintHint{
		Provider:   fingerprint.Provider,
		KindHint:   fingerprint.KindHint,
		Confidence: fingerprint.Confidence,
		Signals:    append([]string{}, fingerprint.Signals...),
	}
}

func clampConfidence(value float64) float64 {
	return floatutils.Clamp(value, 0, maxConfidence)
}

func dedupeStrings(items []string) []string {
	if len(items) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func inferDocumentType(kind string) (*InferredDocumentType, bool) {
	switch kind {
	case kindRateConfirmation:
		return &InferredDocumentType{
			Code:           "RATECONF",
			Name:           "Rate Confirmation",
			Category:       documenttype.CategoryShipment,
			Classification: documenttype.ClassificationPublic,
			Color:          "#0f766e",
		}, true
	case kindBillOfLading:
		return &InferredDocumentType{
			Code:           "BOL",
			Name:           "Bill of Lading",
			Category:       documenttype.CategoryShipment,
			Classification: documenttype.ClassificationPublic,
			Color:          "#f59e0b",
		}, true
	case kindProofOfDelivery:
		return &InferredDocumentType{
			Code:           "POD",
			Name:           "Proof of Delivery",
			Category:       documenttype.CategoryShipment,
			Classification: documenttype.ClassificationPublic,
			Color:          "#8b5cf6",
		}, true
	case kindInvoice:
		return &InferredDocumentType{
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

func canApplyKindToResource(resourceType, kind string) bool {
	switch kind {
	case kindRateConfirmation, kindBillOfLading, kindProofOfDelivery, kindInvoice:
		return strings.EqualFold(resourceType, "shipment")
	default:
		return true
	}
}

func canGenerateShipmentDraft(
	control *tenant.DocumentControl,
	resourceType, kind string,
) bool {
	return control != nil &&
		control.EnableShipmentDraftExtraction &&
		control.AllowsShipmentDraftResource(resourceType) &&
		kind == kindRateConfirmation
}

func hasUsableShipmentDraft(intelligence *DocumentIntelligenceAnalysis) bool {
	if strings.EqualFold(strings.TrimSpace(intelligence.ReviewStatus), reviewStatusReady) {
		return true
	}

	if hasMeaningfulStopForRole(intelligence.Stops, stopRolePickup) &&
		hasMeaningfulStopForRole(intelligence.Stops, stopRoleDelivery) {
		return true
	}

	return hasMeaningfulField(intelligence.Fields, stopRoleShipper) &&
		hasMeaningfulField(intelligence.Fields, stopRoleConsignee) &&
		hasMeaningfulField(intelligence.Fields, "rate")
}

func hasMeaningfulStopForRole(stops []*IntelligenceStop, role string) bool {
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

func hasReviewableStopData(stop *IntelligenceStop) bool {
	return strings.TrimSpace(stop.Name) != "" ||
		strings.TrimSpace(stop.AddressLine1) != "" ||
		strings.TrimSpace(stop.AddressLine2) != "" ||
		strings.TrimSpace(stop.City) != "" ||
		strings.TrimSpace(stop.State) != "" ||
		strings.TrimSpace(stop.PostalCode) != "" ||
		strings.TrimSpace(stop.Date) != "" ||
		strings.TrimSpace(stop.TimeWindow) != ""
}

func hasMeaningfulField(fields map[string]*ReviewField, key string) bool {
	field, ok := fields[key]
	if !ok {
		return false
	}
	return strings.TrimSpace(field.Value) != ""
}

