package aitrainingservice

import (
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/aitraining"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/decimalutils"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/stringutils"
)

type replyBuilder struct {
	keys      map[string]struct{}
	pageLimit int
}

func newReplyBuilder(contract services.ExtractionContract) replyBuilder {
	keys := contract.FieldKeys()
	set := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		set[key] = struct{}{}
	}

	return replyBuilder{keys: set, pageLimit: contract.PageLimit()}
}

type replyInput struct {
	primary    *aitraining.ExampleSnapshot
	supplement *aitraining.ExampleSnapshot
	confidence float64
	kind       string
	pages      []aitraining.ExamplePage
}

func (b replyBuilder) build(in *replyInput) *services.AIExtractResult {
	visible := b.visiblePages(in.pages)
	result := &services.AIExtractResult{
		DocumentKind:      stringutils.FirstNonEmpty(in.kind, aitraining.DefaultTargetKind),
		OverallConfidence: aitraining.TargetOverallScore,
		ReviewStatus:      aitraining.TargetReviewStatus,
		MissingFields:     []string{},
		Signals:           []string{},
		Fields:            map[string]services.AIDocumentField{},
		Stops:             []*services.AIDocumentStop{},
		Conflicts:         []*services.AIDocumentConflict{},
	}
	if in.primary == nil {
		return result
	}

	b.addFields(result, in.primary.Fields, in.confidence, visible)
	if in.supplement != nil {
		b.addFields(result, in.supplement.Fields, aitraining.TargetUnverifiedScore, visible)
	}

	roleIndex := map[string]int{}
	for i := range in.primary.Stops {
		stop := &in.primary.Stops[i]
		position := roleIndex[stop.Role]
		roleIndex[stop.Role]++
		timeWindow := stop.TimeWindow
		if timeWindow == "" && in.supplement != nil {
			timeWindow = supplementalTimeWindow(in.supplement, stop.Role, position)
		}
		page, excerpt := locateEvidence(stop.Name, visible)
		result.Stops = append(result.Stops, &services.AIDocumentStop{
			Sequence:            i + 1,
			Role:                stop.Role,
			Name:                stop.Name,
			AddressLine1:        stop.AddressLine1,
			AddressLine2:        stop.AddressLine2,
			City:                stop.City,
			State:               stop.State,
			PostalCode:          stop.PostalCode,
			Date:                stop.Date,
			TimeWindow:          timeWindow,
			AppointmentRequired: stop.AppointmentRequired,
			PageNumber:          page,
			EvidenceExcerpt:     excerpt,
			Confidence:          in.confidence,
			Source:              aitraining.TargetSource,
		})
	}

	return result
}

func (b replyBuilder) addFields(
	result *services.AIExtractResult,
	fields map[string]string,
	confidence float64,
	visible []aitraining.ExamplePage,
) {
	for key, value := range fields {
		if _, known := b.keys[key]; !known || strings.TrimSpace(value) == "" {
			continue
		}
		if _, exists := result.Fields[key]; exists {
			continue
		}
		page, excerpt := locateEvidence(value, visible)
		result.Fields[key] = services.AIDocumentField{
			Label:             stringutils.HumanizeCamelCase(key),
			Value:             value,
			Confidence:        confidence,
			EvidenceExcerpt:   excerpt,
			PageNumber:        page,
			Source:            aitraining.TargetSource,
			AlternativeValues: []string{},
		}
	}
}

func (b replyBuilder) visiblePages(pages []aitraining.ExamplePage) []aitraining.ExamplePage {
	out := make([]aitraining.ExamplePage, 0, len(pages))
	for _, page := range pages {
		text := page.Text
		if len(text) > b.pageLimit {
			text = strings.ToValidUTF8(text[:b.pageLimit], "")
		}
		out = append(out, aitraining.ExamplePage{Number: page.Number, Text: text})
	}

	return out
}

func supplementalTimeWindow(supplement *aitraining.ExampleSnapshot, role string, position int) string {
	seen := 0
	for i := range supplement.Stops {
		candidate := &supplement.Stops[i]
		if candidate.Role != role {
			continue
		}
		if seen == position {
			return candidate.TimeWindow
		}
		seen++
	}

	return ""
}

func locateEvidence(value string, pages []aitraining.ExamplePage) (int, string) {
	for _, needle := range evidenceNeedles(value) {
		if page, excerpt, found := findEvidence(needle, pages); found {
			return page, excerpt
		}
	}

	return 0, ""
}

func evidenceNeedles(value string) []string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" {
		return nil
	}
	needles := []string{trimmed}
	amount, err := decimalutils.ParseMoneyText(trimmed)
	if err != nil || !amount.Valid {
		return needles
	}
	whole := amount.Decimal.Truncate(0)
	grouped := intutils.FormatWithCommas(whole.IntPart())
	if cents := amount.Decimal.Sub(whole); !cents.IsZero() {
		grouped += "." + cents.Abs().StringFixed(2)[2:]
	}

	return append(needles, grouped)
}

func findEvidence(needle string, pages []aitraining.ExamplePage) (int, string, bool) {
	for _, page := range pages {
		haystack := strings.ToLower(page.Text)
		index := strings.Index(haystack, needle)
		if index < 0 || len(haystack) != len(page.Text) {
			continue
		}
		runes := []rune(page.Text)
		start := len([]rune(page.Text[:index]))
		end := start + len([]rune(page.Text[index:index+len(needle)]))
		from := max(0, start-aitraining.EvidenceContextRunes)
		to := min(len(runes), end+aitraining.EvidenceContextRunes)

		return page.Number, stringutils.CollapseWhitespace(string(runes[from:to])), true
	}

	return 0, "", false
}
