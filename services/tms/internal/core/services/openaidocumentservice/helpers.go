package openaidocumentservice

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/stringutils"
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

	result.DocumentKind = parsed.DocumentKind
	result.OverallConfidence = parsed.OverallConfidence
	result.ReviewStatus = parsed.ReviewStatus
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

func extractResponseText(envelope *responsesEnvelope) string {
	if envelope == nil {
		return ""
	}
	if strings.TrimSpace(envelope.OutputText) != "" {
		return envelope.OutputText
	}
	for _, output := range envelope.Output {
		for _, content := range output.Content {
			if strings.TrimSpace(content.Text) != "" {
				return content.Text
			}
		}
	}
	return ""
}

func errorCode(envelope *responsesEnvelope) string {
	if envelope == nil || envelope.Error == nil {
		return ""
	}
	return strings.TrimSpace(envelope.Error.Code)
}

func errorMessage(envelope *responsesEnvelope) string {
	if envelope == nil || envelope.Error == nil {
		return ""
	}
	return strings.TrimSpace(envelope.Error.Message)
}

func responseIncompleteReason(envelope *responsesEnvelope) string {
	if envelope == nil || envelope.IncompleteDetails == nil {
		return ""
	}
	return strings.TrimSpace(envelope.IncompleteDetails.Reason)
}

func incompleteFailureMessage(status, reason string) string {
	if strings.TrimSpace(reason) == "" {
		return ""
	}

	if strings.EqualFold(status, "incomplete") {
		return fmt.Sprintf("AI background extraction ended incomplete: %s", reason)
	}

	return fmt.Sprintf("AI background extraction ended with status %s: %s", status, reason)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func redactPrompt(systemPrompt, userPrompt string) string {
	sum := sha256.Sum256([]byte(userPrompt))
	return fmt.Sprintf(
		"system=%q user_sha256=%s user_preview=%q",
		systemPrompt,
		hex.EncodeToString(sum[:]),
		stringutils.Truncate(userPrompt, 512),
	)
}

func redactResponse(text string) string {
	sum := sha256.Sum256([]byte(text))
	return fmt.Sprintf(
		"sha256=%s preview=%q",
		hex.EncodeToString(sum[:]),
		stringutils.Truncate(text, 1024),
	)
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

func isRetryableAIError(err error) bool {
	if err == nil {
		return false
	}
	if ae, ok := errors.AsType[*apiError](err); ok {
		switch ae.StatusCode {
		case http.StatusTooManyRequests,
			http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout:
			return true
		}
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "timeout")
}
