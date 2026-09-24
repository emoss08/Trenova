package aidocumentservice

import (
	"strings"

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
	result.Stops = parsed.Stops
	result.Conflicts = parsed.Conflicts

	for i := range parsed.Fields {
		field := &parsed.Fields[i]
		key := strings.TrimSpace(field.Key)
		if key == "" {
			continue
		}
		result.Fields[key] = serviceports.AIDocumentField{
			Label:             field.Label,
			Value:             field.Value,
			Confidence:        field.Confidence,
			EvidenceExcerpt:   field.EvidenceExcerpt,
			PageNumber:        field.PageNumber,
			ReviewRequired:    field.ReviewRequired,
			Conflict:          field.Conflict,
			Source:            field.Source,
			AlternativeValues: field.AlternativeValues,
		}
	}

	return result
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
