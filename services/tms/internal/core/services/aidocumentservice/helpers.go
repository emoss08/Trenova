package aidocumentservice

import (
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

func convertExtractResponse(parsed *extractResponse) *serviceports.AIExtractResult {
	result := &serviceports.AIExtractResult{
		MissingFields: []string{},
		Signals:       []string{},
		Fields:        map[string]serviceports.AIDocumentField{},
		Stops:         []*serviceports.AIDocumentStop{},
		Conflicts:     []*serviceports.AIDocumentConflict{},
	}
	if parsed == nil {
		return result
	}

	result.DocumentKind = strings.TrimSpace(parsed.DocumentKind)
	result.OverallConfidence = clampAIConfidence(parsed.OverallConfidence)
	result.ReviewStatus = normalizeReviewStatus(parsed.ReviewStatus)
	result.MissingFields = parsed.MissingFields
	result.Signals = parsed.Signals
	result.Stops = derivedStops(parsed.Stops)
	result.Conflicts = derivedConflicts(parsed.Conflicts)

	conflicted := make(map[string]struct{}, len(result.Conflicts))
	for _, conflict := range result.Conflicts {
		conflicted[conflict.Key] = struct{}{}
	}

	for i := range parsed.Fields {
		field := &parsed.Fields[i]
		key := strings.TrimSpace(field.Key)
		if key == "" {
			continue
		}
		_, conflict := conflicted[aicorrection.CanonicalFieldKey(key)]
		result.Fields[key] = serviceports.AIDocumentField{
			Label:           aicorrection.FieldLabel(key),
			Value:           field.Value,
			Confidence:      field.Confidence,
			EvidenceExcerpt: field.EvidenceExcerpt,
			PageNumber:      field.PageNumber,
			ReviewRequired:  field.ReviewRequired,
			Conflict:        conflict,
			Source:          serviceports.AIDocumentSourceAI,
		}
	}

	return result
}

func derivedStops(stops []*serviceports.AIDocumentStop) []*serviceports.AIDocumentStop {
	derived := make([]*serviceports.AIDocumentStop, 0, len(stops))
	for _, stop := range stops {
		if stop == nil {
			continue
		}
		stop.Sequence = len(derived) + 1
		stop.Source = serviceports.AIDocumentSourceAI
		derived = append(derived, stop)
	}

	return derived
}

func derivedConflicts(
	conflicts []*serviceports.AIDocumentConflict,
) []*serviceports.AIDocumentConflict {
	derived := make([]*serviceports.AIDocumentConflict, 0, len(conflicts))
	for _, conflict := range conflicts {
		if conflict == nil {
			continue
		}
		conflict.Key = aicorrection.CanonicalFieldKey(conflict.Key)
		conflict.Label = aicorrection.FieldLabel(conflict.Key)
		conflict.Source = serviceports.AIDocumentSourceAI
		derived = append(derived, conflict)
	}

	return derived
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}

	return ""
}

func normalizeReviewStatus(status string) string {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "ready":
		return "Ready"
	case "needsreview", "needs_review":
		return "NeedsReview"
	case "unavailable":
		return "Unavailable"
	default:
		return "NeedsReview"
	}
}

func clampAIConfidence(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}

	return value
}
