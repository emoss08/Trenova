package documentintelligencejobs

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/shared/stringutils"
)

func addFieldFromPages(params *RegexValueFieldParams) {
	pageNumber, match := firstPageMatch(params.Regex, params.Pages)
	if len(match) < 2 {
		return
	}

	value := strings.TrimSpace(match[len(match)-1])
	if value == "" {
		return
	}

	params.Fields[params.Key] = &ReviewField{
		Label:           params.Label,
		Value:           value,
		Confidence:      pageAdjustedConfidence(params.Confidence, pageNumber, params.Pages),
		Excerpt:         strings.TrimSpace(match[0]),
		EvidenceExcerpt: strings.TrimSpace(match[0]),
		PageNumber:      pageNumber,
		ReviewRequired:  params.Confidence < reviewRequiredConfidenceFloor,
		Conflict:        hasConflictingMatches(params.Regex, value, params.Pages),
		Source:          "deterministic",
	}
	if params.Signals != nil && params.Signal != "" {
		*params.Signals = append(*params.Signals, params.Signal)
	}
}

func addCurrencyFieldFromPages(params *RegexValueFieldParams) {
	pageNumber, match := firstPageMatch(params.Regex, params.Pages)
	if len(match) < 2 {
		return
	}

	value := strings.TrimSpace(match[1])
	if value == "" {
		return
	}

	params.Fields[params.Key] = &ReviewField{
		Label:           params.Label,
		Value:           value,
		Confidence:      pageAdjustedConfidence(params.Confidence, pageNumber, params.Pages),
		Excerpt:         strings.TrimSpace(match[0]),
		EvidenceExcerpt: strings.TrimSpace(match[0]),
		PageNumber:      pageNumber,
		ReviewRequired:  params.Confidence < reviewRequiredConfidenceFloor,
		Conflict:        hasConflictingMatches(params.Regex, value, params.Pages),
		Source:          "deterministic",
	}
	if params.Signals != nil && params.Signal != "" {
		*params.Signals = append(*params.Signals, params.Signal)
	}
}

func addRegexValueFieldFromPages(params *RegexValueFieldParams) {
	pageNumber, match := firstPageMatch(params.Regex, params.Pages)
	if len(match) < 2 {
		return
	}

	value := strings.TrimSpace(match[1])
	if value == "" {
		return
	}

	params.Fields[params.Key] = &ReviewField{
		Label:           params.Label,
		Value:           value,
		Confidence:      pageAdjustedConfidence(params.Confidence, pageNumber, params.Pages),
		Excerpt:         strings.TrimSpace(match[0]),
		EvidenceExcerpt: strings.TrimSpace(match[0]),
		PageNumber:      pageNumber,
		ReviewRequired:  params.ReviewRequired,
		Conflict:        hasConflictingMatches(params.Regex, value, params.Pages),
		Source:          "deterministic",
	}
	if params.Signals != nil && params.Signal != "" {
		*params.Signals = append(*params.Signals, params.Signal)
	}
}

func addWeightFieldFromPages(params *AddWeightFieldParams, pages []*PageExtractionResult) {
	pageNumber, match := firstPageMatch(weightRegex, pages)
	if len(match) < 2 {
		return
	}

	params.Fields["weight"] = &ReviewField{
		Label:           "Weight",
		Value:           fmt.Sprintf("%s lbs", strings.TrimSpace(match[1])),
		Confidence:      pageAdjustedConfidence(0.8, pageNumber, pages),
		Excerpt:         strings.TrimSpace(match[0]),
		EvidenceExcerpt: strings.TrimSpace(match[0]),
		PageNumber:      pageNumber,
		ReviewRequired:  false,
		Source:          "deterministic",
	}

	if params.Signals != nil && params.Signal != "" {
		*params.Signals = append(*params.Signals, params.Signal)
	}
}

func addStopTimingField(params *AddStopTimingFieldParams) {
	if params.Stop == nil {
		return
	}

	value := strings.TrimSpace(
		strings.Join(
			stringutils.FilterEmpty([]string{params.Stop.Date, params.Stop.TimeWindow}),
			" ",
		),
	)
	if value == "" {
		return
	}

	params.Fields[params.Key] = &ReviewField{
		Label:           params.Label,
		Value:           value,
		Confidence:      clampConfidence((params.Confidence + params.Stop.Confidence) / 2),
		Excerpt:         params.Stop.EvidenceExcerpt,
		EvidenceExcerpt: params.Stop.EvidenceExcerpt,
		PageNumber:      params.Stop.PageNumber,
		ReviewRequired:  params.Stop.ReviewRequired,
		Source:          params.Stop.Source,
	}
	if params.Signals != nil {
		*params.Signals = append(*params.Signals, strings.ToLower(params.Label))
	}
}

func addFieldFromSectionLabels(params *AddFieldFromSectionLabelsParams) {
	matches := findSectionMatches(params)
	if len(matches) == 0 {
		return
	}

	selected := matches[0]
	conflict := false
	normalizedSelected := normalizeSectionValue(selected.Value)
	for _, match := range matches[1:] {
		if normalizeSectionValue(match.Value) != normalizedSelected {
			conflict = true
			break
		}
	}

	params.Fields[params.Key] = &ReviewField{
		Label: params.Label,
		Value: selected.Value,
		Confidence: pageAdjustedConfidence(
			params.Confidence,
			selected.PageNumber,
			params.Pages,
		),
		Excerpt:         selected.Excerpt,
		EvidenceExcerpt: selected.Excerpt,
		PageNumber:      selected.PageNumber,
		ReviewRequired: params.ReviewRequired || conflict ||
			normalizeSectionValue(selected.Value) == "",
		Conflict: conflict,
		Source:   "deterministic",
	}
	if params.Signals != nil && params.Signal != "" {
		*params.Signals = append(*params.Signals, params.Signal)
	}
}

func findSectionMatches(
	params *AddFieldFromSectionLabelsParams,
) []PageSectionMatch {
	matches := make([]PageSectionMatch, 0)
	for _, page := range params.Pages {
		lines := splitNormalizedLines(page.Text)
		for idx, line := range lines {
			if !matchesSectionLabel(line, params.Labels) {
				continue
			}
			block := collectSectionBlock(lines, idx)
			value := strings.TrimSpace(params.Extractor(line, block))
			if value == "" {
				continue
			}
			matches = append(matches, PageSectionMatch{
				PageNumber: page.PageNumber,
				Value:      value,
				Excerpt:    strings.Join(block, "\n"),
			})
		}
	}

	return dedupeSectionMatches(matches)
}

func matchesSectionLabel(line string, labels []string) bool {
	normalized := normalizeSectionLabel(line)
	for _, label := range labels {
		want := normalizeSectionLabel(label)
		if normalized == want || strings.HasPrefix(normalized, want+" ") {
			return true
		}
	}
	return false
}

func normalizeSectionLabel(value string) string {
	lower := strings.ToLower(strings.TrimSpace(value))
	lower = strings.TrimSuffix(lower, ":")
	lower = strings.ReplaceAll(lower, "-", " ")
	lower = strings.ReplaceAll(lower, "_", " ")
	lower = strings.Join(strings.Fields(lower), " ")
	return lower
}

func normalizeSectionValue(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}

func collectSectionBlock(lines []string, idx int) []string {
	end := min(idx+sectionBlockMaxLines, len(lines))

	block := make([]string, 0, end-idx)
	for pos := idx; pos < end; pos++ {
		line := strings.TrimSpace(lines[pos])
		if pos > idx && line == "" {
			break
		}
		if pos > idx && isLikelyBoundaryLine(line) {
			break
		}
		block = append(block, line)
	}
	return block
}

func isLikelyBoundaryLine(line string) bool {
	if line == "" {
		return false
	}
	normalized := normalizeSectionLabel(line)
	for _, boundary := range Boundaries {
		if normalized == boundary || strings.HasPrefix(normalized, boundary+" ") {
			return true
		}
	}
	return false
}

func extractEntityNameFromSection(header string, block []string) string {
	if value := extractSectionHeaderValue(header); value != "" && !looksLikeAddress(value) &&
		!cityStateZipRegex.MatchString(value) {
		return value
	}
	for _, line := range block[1:] {
		switch {
		case line == "":
			continue
		case looksLikeAddress(line):
			continue
		case cityStateZipRegex.MatchString(line):
			continue
		case dateValueRegex.MatchString(line):
			continue
		case strings.Contains(strings.ToLower(line), "signature"):
			continue
		default:
			return line
		}
	}
	if len(block) > 1 {
		return strings.TrimSpace(block[1])
	}
	return ""
}

func extractCommodityFromSection(header string, block []string) string {
	if value := extractSectionHeaderValue(header); value != "" {
		return value
	}
	for _, line := range block[1:] {
		if line == "" || looksLikeAddress(line) || cityStateZipRegex.MatchString(line) {
			continue
		}
		return line
	}
	return ""
}

func extractDeliveryFieldFromSection(header string, block []string) string {
	candidates := append([]string{header}, block[1:]...)
	for _, candidate := range candidates {
		date := firstRegexValue(dateValueRegex, candidate)
		window := firstRegexValue(timeWindowRegex, candidate)
		value := strings.TrimSpace(
			strings.Join(stringutils.FilterEmpty([]string{date, window}), " "),
		)
		if value != "" {
			return value
		}
	}
	if value := extractSectionHeaderValue(header); value != "" {
		return value
	}
	for _, line := range block[1:] {
		if line != "" {
			return line
		}
	}
	return ""
}

func extractSignatureFromSection(header string, block []string) string {
	if value := extractSectionHeaderValue(header); value != "" &&
		!dateValueRegex.MatchString(value) {
		return value
	}
	for _, line := range block[1:] {
		if line == "" || dateValueRegex.MatchString(line) ||
			strings.Contains(strings.ToLower(line), "date") {
			continue
		}
		return line
	}
	return ""
}

func extractFreeformSectionValue(header string, block []string) string {
	if value := extractSectionHeaderValue(header); value != "" {
		return value
	}
	for _, line := range block[1:] {
		if line != "" {
			return line
		}
	}
	return ""
}

func extractSectionHeaderValue(header string) string {
	parts := strings.SplitN(header, ":", 2)
	if len(parts) != 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func dedupeSectionMatches(matches []PageSectionMatch) []PageSectionMatch {
	deduped := make([]PageSectionMatch, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		key := fmt.Sprintf("%s|%d", normalizeSectionValue(match.Value), match.PageNumber)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		deduped = append(deduped, match)
	}
	return deduped
}

func collectFieldConflicts(fields map[string]*ReviewField) []*ReviewConflict {
	conflicts := make([]*ReviewConflict, 0)
	for key, field := range fields {
		if !field.Conflict {
			continue
		}
		conflicts = append(conflicts, &ReviewConflict{
			Key:             key,
			Label:           field.Label,
			Values:          []string{field.Value},
			PageNumbers:     nonZeroPageNumbers(field.PageNumber),
			EvidenceExcerpt: field.EvidenceExcerpt,
			Source:          field.Source,
		})
	}
	return conflicts
}

func collectStopConflicts(stops []*IntelligenceStop) []*ReviewConflict {
	conflicts := make([]*ReviewConflict, 0)

	for _, role := range []string{stopRolePickup, stopRoleDelivery} {
		addresses := make(map[string][]*IntelligenceStop)
		dates := make(map[string][]*IntelligenceStop)
		for _, stop := range stops {
			if stop.Role != role {
				continue
			}
			if address := strings.TrimSpace(strings.ToLower(stop.AddressLine1)); address != "" {
				addresses[address] = append(addresses[address], stop)
			}
			if date := strings.TrimSpace(strings.ToLower(stop.Date)); date != "" {
				dates[date] = append(dates[date], stop)
			}
		}

		if len(addresses) > 1 {
			conflicts = append(conflicts, &ReviewConflict{
				Key:             fmt.Sprintf("%sAddress", role),
				Label:           fmt.Sprintf("%s Address", roleLabel(role)),
				Values:          mapKeys(addresses),
				PageNumbers:     stopPages(stops, role),
				EvidenceExcerpt: firstStopExcerpt(stops, role),
				Source:          "deterministic",
			})
		}
		if len(dates) > 1 {
			conflicts = append(conflicts, &ReviewConflict{
				Key:             fmt.Sprintf("%sDate", role),
				Label:           fmt.Sprintf("%s Date", roleLabel(role)),
				Values:          mapKeys(dates),
				PageNumbers:     stopPages(stops, role),
				EvidenceExcerpt: firstStopExcerpt(stops, role),
				Source:          "deterministic",
			})
		}
	}

	return conflicts
}

func firstPageMatch(
	re *regexp.Regexp, pages []*PageExtractionResult,
) (pageNum int, groups []string) {
	for _, page := range pages {
		m := re.FindStringSubmatch(strings.ReplaceAll(page.Text, "\r", ""))
		if len(m) > 0 {
			return page.PageNumber, m
		}
	}
	return 0, nil
}

func hasConflictingMatches(re *regexp.Regexp, selected string, pages []*PageExtractionResult) bool {
	normalizedSelected := strings.TrimSpace(strings.ToLower(selected))
	for _, page := range pages {
		matches := re.FindAllStringSubmatch(strings.ReplaceAll(page.Text, "\r", ""), -1)
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}
			candidate := strings.TrimSpace(strings.ToLower(match[len(match)-1]))
			if candidate != "" && candidate != normalizedSelected {
				return true
			}
		}
	}
	return false
}

func pageAdjustedConfidence(base float64, pageNumber int, pages []*PageExtractionResult) float64 {
	if pageNumber <= 0 {
		return clampConfidence(base)
	}
	for _, page := range pages {
		if page.PageNumber != pageNumber {
			continue
		}
		if page.SourceKind == documentcontent.SourceKindOCR {
			return clampConfidence((base + page.OCRConfidence) / 2)
		}
		return clampConfidence((base + maxConfidence) / 2)
	}
	return clampConfidence(base)
}

func firstRegexValue(re *regexp.Regexp, text string) string {
	match := re.FindStringSubmatch(text)
	if len(match) == 0 {
		return ""
	}
	if len(match) == 1 {
		return strings.TrimSpace(match[0])
	}
	return strings.TrimSpace(match[1])
}

func hasStopRole(stops []*IntelligenceStop, role string) bool {
	for _, stop := range stops {
		if stop.Role == role {
			return true
		}
	}
	return false
}

func hasReviewRequiredStop(stops []*IntelligenceStop) bool {
	for _, stop := range stops {
		if stop.ReviewRequired {
			return true
		}
	}
	return false
}

func splitNormalizedLines(text string) []string {
	rawLines := strings.Split(strings.ReplaceAll(text, "\r", ""), "\n")
	lines := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			lines = append(lines, "")
			continue
		}
		lines = append(lines, trimmed)
	}
	return lines
}

func nonZeroPageNumbers(pageNumber int) []int {
	if pageNumber <= 0 {
		return []int{}
	}
	return []int{pageNumber}
}

func stopPages(stops []*IntelligenceStop, role string) []int {
	pages := make([]int, 0)
	seen := make(map[int]struct{})
	for _, stop := range stops {
		if stop.Role != role || stop.PageNumber <= 0 {
			continue
		}
		if _, ok := seen[stop.PageNumber]; ok {
			continue
		}
		seen[stop.PageNumber] = struct{}{}
		pages = append(pages, stop.PageNumber)
	}
	return pages
}

func firstStopExcerpt(stops []*IntelligenceStop, role string) string {
	for _, stop := range stops {
		if stop.Role == role && stop.EvidenceExcerpt != "" {
			return stop.EvidenceExcerpt
		}
	}
	return ""
}

func mapKeys[T any](items map[string][]T) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	return keys
}

func roleLabel(role string) string {
	if role == "" {
		return "Unknown"
	}
	switch role {
	case stopRolePickup:
		return "Pickup"
	case stopRoleDelivery:
		return "Delivery"
	default:
		return strings.ToUpper(role[:1]) + role[1:]
	}
}

func firstStopByRole(stops []*IntelligenceStop, role string) *IntelligenceStop {
	for idx := range stops {
		if stops[idx].Role == role {
			return stops[idx]
		}
	}
	return nil
}
