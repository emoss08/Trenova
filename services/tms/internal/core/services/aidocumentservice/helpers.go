package aidocumentservice

import (
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentaiextractionports"
)

func convertExtractResponse(parsed *extractResponse) *documentaiextractionports.AIExtractResult {
	result := &documentaiextractionports.AIExtractResult{
		MissingFields: []string{},
		Signals:       []string{},
		Fields:        map[string]documentaiextractionports.AIDocumentField{},
		Stops:         []*documentaiextractionports.AIDocumentStop{},
		Conflicts:     []*documentaiextractionports.AIDocumentConflict{},
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
		result.Fields[key] = documentaiextractionports.AIDocumentField{
			Label:           aicorrection.FieldLabel(key),
			Value:           field.Value,
			Confidence:      field.Confidence,
			EvidenceExcerpt: field.EvidenceExcerpt,
			PageNumber:      field.PageNumber,
			ReviewRequired:  field.ReviewRequired,
			Conflict:        conflict,
			Source:          documentaiextractionports.AIDocumentSourceAI,
		}
	}

	return result
}

func derivedStops(stops []*documentaiextractionports.AIDocumentStop) []*documentaiextractionports.AIDocumentStop {
	derived := make([]*documentaiextractionports.AIDocumentStop, 0, len(stops))
	for _, stop := range stops {
		if stop == nil {
			continue
		}
		stop.Sequence = len(derived) + 1
		stop.Source = documentaiextractionports.AIDocumentSourceAI
		derived = append(derived, stop)
	}

	return derived
}

func derivedConflicts(
	conflicts []*documentaiextractionports.AIDocumentConflict,
) []*documentaiextractionports.AIDocumentConflict {
	derived := make([]*documentaiextractionports.AIDocumentConflict, 0, len(conflicts))
	for _, conflict := range conflicts {
		if conflict == nil {
			continue
		}
		conflict.Key = aicorrection.CanonicalFieldKey(conflict.Key)
		conflict.Label = aicorrection.FieldLabel(conflict.Key)
		conflict.Source = documentaiextractionports.AIDocumentSourceAI
		derived = append(derived, conflict)
	}

	return derived
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
