package documentintelligencejobs

import (
	"context"

	services "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

func (a *Activities) applyParsingRules(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	fileName string,
	providerFingerprint string,
	extracted *extractionResult,
	intelligence documentIntelligenceAnalysis,
) documentIntelligenceAnalysis {
	if extracted == nil || intelligence.Kind != "RateConfirmation" || a.parsingRuleRuntime == nil {
		return intelligence
	}

	result, err := a.parsingRuleRuntime.ApplyPublished(ctx, &services.DocumentParsingRuntimeInput{
		TenantInfo:          tenantInfo,
		DocumentKind:        intelligence.Kind,
		FileName:            fileName,
		Text:                extracted.Text,
		ProviderFingerprint: providerFingerprint,
		Pages:               toParsingPages(extracted.Pages),
	}, toParsingAnalysis(intelligence))
	if err != nil {
		a.logger.Warn("failed to apply document parsing rules", zap.Error(err))
		return intelligence
	}
	if result == nil {
		return intelligence
	}
	return fromParsingAnalysis(intelligence, result)
}

func toParsingPages(pages []pageExtractionResult) []services.DocumentParsingPage {
	out := make([]services.DocumentParsingPage, 0, len(pages))
	for _, page := range pages {
		out = append(out, services.DocumentParsingPage{
			PageNumber: page.PageNumber,
			Text:       page.Text,
		})
	}
	return out
}

func toParsingAnalysis(intelligence documentIntelligenceAnalysis) *services.DocumentParsingAnalysis {
	fields := make(map[string]services.DocumentParsingField, len(intelligence.Fields))
	for key, field := range intelligence.Fields {
		fields[key] = services.DocumentParsingField{
			Key:             key,
			Label:           field.Label,
			Value:           field.Value,
			Confidence:      field.Confidence,
			PageNumber:      field.PageNumber,
			ReviewRequired:  field.ReviewRequired,
			EvidenceExcerpt: field.EvidenceExcerpt,
			Source:          field.Source,
		}
	}

	stops := make([]services.DocumentParsingStop, 0, len(intelligence.Stops))
	for _, stop := range intelligence.Stops {
		stops = append(stops, services.DocumentParsingStop{
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
			Confidence:          stop.Confidence,
			ReviewRequired:      stop.ReviewRequired,
			Source:              stop.Source,
		})
	}

	conflicts := make([]services.DocumentParsingConflict, 0, len(intelligence.Conflicts))
	for _, conflict := range intelligence.Conflicts {
		conflicts = append(conflicts, services.DocumentParsingConflict{
			Key:             conflict.Key,
			Label:           conflict.Label,
			Values:          conflict.Values,
			PageNumbers:     conflict.PageNumbers,
			EvidenceExcerpt: conflict.EvidenceExcerpt,
			Source:          conflict.Source,
		})
	}

	return &services.DocumentParsingAnalysis{
		Fields:            fields,
		Stops:             stops,
		Conflicts:         conflicts,
		MissingFields:     append([]string{}, intelligence.MissingFields...),
		Signals:           append([]string{}, intelligence.Signals...),
		ReviewStatus:      intelligence.ReviewStatus,
		OverallConfidence: intelligence.OverallConfidence,
		Metadata:          intelligence.ParsingRuleMetadata,
	}
}

func fromParsingAnalysis(
	baseline documentIntelligenceAnalysis,
	analysis *services.DocumentParsingAnalysis,
) documentIntelligenceAnalysis {
	if analysis == nil {
		return baseline
	}

	fields := make(map[string]reviewField, len(analysis.Fields))
	for key, field := range analysis.Fields {
		fields[key] = reviewField{
			Label:           field.Label,
			Value:           field.Value,
			Confidence:      field.Confidence,
			Excerpt:         field.EvidenceExcerpt,
			EvidenceExcerpt: field.EvidenceExcerpt,
			PageNumber:      field.PageNumber,
			ReviewRequired:  field.ReviewRequired,
			Conflict:        false,
			Source:          field.Source,
		}
	}

	stops := make([]intelligenceStop, 0, len(analysis.Stops))
	for _, stop := range analysis.Stops {
		stops = append(stops, intelligenceStop{
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
			Confidence:          stop.Confidence,
			ReviewRequired:      stop.ReviewRequired,
			Source:              stop.Source,
		})
	}

	conflicts := make([]reviewConflict, 0, len(analysis.Conflicts))
	for _, conflict := range analysis.Conflicts {
		conflicts = append(conflicts, reviewConflict{
			Key:             conflict.Key,
			Label:           conflict.Label,
			Values:          conflict.Values,
			PageNumbers:     conflict.PageNumbers,
			EvidenceExcerpt: conflict.EvidenceExcerpt,
			Source:          conflict.Source,
		})
	}

	baseline.Fields = fields
	baseline.Stops = stops
	baseline.Conflicts = conflicts
	baseline.MissingFields = append([]string{}, analysis.MissingFields...)
	baseline.Signals = append([]string{}, analysis.Signals...)
	baseline.ReviewStatus = analysis.ReviewStatus
	baseline.OverallConfidence = analysis.OverallConfidence
	baseline.ParsingRuleMetadata = analysis.Metadata
	return baseline
}
