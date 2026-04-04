package documentparsingruleservice

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/documentparsingrule"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

type sectionOccurrence struct {
	Name       string
	PageNumber int
	Lines      []string
	Text       string
}

var cityStatePostalPattern = regexp.MustCompile(`(?i)\b([a-z .'-]+),\s*([a-z]{2})\s+(\d{5}(?:-\d{4})?)\b`)

func matchesVersion(
	set *documentparsingrule.RuleSet,
	version *documentparsingrule.RuleVersion,
	input *serviceports.DocumentParsingRuntimeInput,
) (bool, int, string) {
	if set == nil || version == nil || input == nil {
		return false, 0, ""
	}
	if !strings.EqualFold(string(set.DocumentKind), input.DocumentKind) {
		return false, 0, ""
	}

	text := strings.ToLower(input.Text)
	fileName := strings.ToLower(strings.TrimSpace(input.FileName))
	provider := strings.TrimSpace(input.ProviderFingerprint)
	score := set.Priority
	providerMatched := ""

	if len(version.MatchConfig.ProviderFingerprints) > 0 {
		match := false
		for _, candidate := range version.MatchConfig.ProviderFingerprints {
			if strings.EqualFold(strings.TrimSpace(candidate), provider) {
				match = true
				providerMatched = candidate
				score += 100
				break
			}
		}
		if !match {
			return false, 0, ""
		}
	}

	if len(version.MatchConfig.FileNameContains) > 0 {
		match := false
		for _, needle := range version.MatchConfig.FileNameContains {
			if strings.Contains(fileName, strings.ToLower(strings.TrimSpace(needle))) {
				match = true
				score += 15
			}
		}
		if !match {
			return false, 0, ""
		}
	}

	for _, needle := range version.MatchConfig.RequiresAll {
		if !strings.Contains(text, strings.ToLower(strings.TrimSpace(needle))) {
			return false, 0, ""
		}
		score += 10
	}

	if len(version.MatchConfig.RequiresAny) > 0 {
		match := false
		for _, needle := range version.MatchConfig.RequiresAny {
			if strings.Contains(text, strings.ToLower(strings.TrimSpace(needle))) {
				match = true
				score += 8
			}
		}
		if !match {
			return false, 0, ""
		}
	}

	for _, anchor := range version.MatchConfig.SectionAnchors {
		if !strings.Contains(text, strings.ToLower(strings.TrimSpace(anchor))) {
			return false, 0, ""
		}
		score += 5
	}

	return true, score, providerMatched
}

func evaluateVersion(
	set *documentparsingrule.RuleSet,
	version *documentparsingrule.RuleVersion,
	input *serviceports.DocumentParsingRuntimeInput,
) (*serviceports.DocumentParsingAnalysis, error) {
	sections := extractSections(version.RuleDocument.Sections, input.Pages, input.Text)
	fields := make(map[string]serviceports.DocumentParsingField, len(version.RuleDocument.Fields))
	stops := make([]serviceports.DocumentParsingStop, 0, len(version.RuleDocument.Stops))
	missing := make([]string, 0)
	signals := []string{
		fmt.Sprintf("rule:%s", set.Name),
		fmt.Sprintf("rule-version:%d", version.VersionNumber),
	}

	for _, rule := range version.RuleDocument.Fields {
		field, ok := evaluateFieldRule(rule, input.Pages, sections)
		if !ok {
			if rule.Required {
				missing = append(missing, rule.Label)
			}
			continue
		}
		fields[rule.Key] = field
		signals = append(signals, fmt.Sprintf("field:%s", rule.Key))
	}

	for _, rule := range version.RuleDocument.Stops {
		extracted := evaluateStopRule(rule, input.Pages, sections)
		if len(extracted) == 0 {
			if rule.Required {
				missing = append(missing, strings.Title(rule.Role)+" Stop")
			}
			continue
		}
		stops = append(stops, extracted...)
		signals = append(signals, fmt.Sprintf("stop:%s", rule.Role))
	}

	confidence := 0.0
	parts := 0.0
	for _, field := range fields {
		confidence += field.Confidence
		parts++
	}
	for _, stop := range stops {
		confidence += stop.Confidence
		parts++
	}
	if parts == 0 {
		return &serviceports.DocumentParsingAnalysis{
			Fields:            fields,
			Stops:             stops,
			Conflicts:         []serviceports.DocumentParsingConflict{},
			MissingFields:     missing,
			Signals:           dedupeStrings(signals),
			ReviewStatus:      "Unavailable",
			OverallConfidence: 0,
		}, nil
	}
	confidence = clampConfidence(confidence / parts)

	reviewStatus := "NeedsReview"
	if len(missing) == 0 && !hasReviewRequiredField(fields) && !hasReviewRequiredStop(stops) && confidence >= 0.82 {
		reviewStatus = "Ready"
	}

	return &serviceports.DocumentParsingAnalysis{
		Fields:            fields,
		Stops:             normalizeStopSequences(stops),
		Conflicts:         []serviceports.DocumentParsingConflict{},
		MissingFields:     dedupeStrings(missing),
		Signals:           dedupeStrings(signals),
		ReviewStatus:      reviewStatus,
		OverallConfidence: confidence,
	}, nil
}

func extractSections(
	definitions []documentparsingrule.SectionRule,
	pages []serviceports.DocumentParsingPage,
	fallbackText string,
) []sectionOccurrence {
	if len(pages) == 0 && strings.TrimSpace(fallbackText) != "" {
		pages = append(pages, serviceports.DocumentParsingPage{PageNumber: 1, Text: fallbackText})
	}
	sections := make([]sectionOccurrence, 0)
	allAnchors := collectAllSectionAnchors(definitions)
	for _, definition := range definitions {
		for _, page := range pages {
			lines := splitLines(page.Text)
			for idx := 0; idx < len(lines); idx++ {
				if !lineMatchesAny(lines[idx], definition.StartAnchors) {
					continue
				}
				block := []string{lines[idx]}
				for cursor := idx + 1; cursor < len(lines); cursor++ {
					line := lines[cursor]
					if len(definition.EndAnchors) > 0 && lineMatchesAny(line, definition.EndAnchors) {
						break
					}
					if definition.CaptureBlankLine && strings.TrimSpace(line) == "" {
						break
					}
					if len(allAnchors) > 0 && cursor > idx && lineMatchesAny(line, allAnchors) {
						break
					}
					block = append(block, line)
				}
				sections = append(sections, sectionOccurrence{
					Name:       definition.Name,
					PageNumber: page.PageNumber,
					Lines:      trimBlock(block),
					Text:       strings.Join(trimBlock(block), "\n"),
				})
				if !definition.AllowMultiple {
					break
				}
			}
		}
	}
	return sections
}

func collectAllSectionAnchors(definitions []documentparsingrule.SectionRule) []string {
	anchors := make([]string, 0)
	for _, definition := range definitions {
		anchors = append(anchors, definition.StartAnchors...)
	}
	return dedupeStrings(anchors)
}

func evaluateFieldRule(
	rule documentparsingrule.FieldRule,
	pages []serviceports.DocumentParsingPage,
	sections []sectionOccurrence,
) (serviceports.DocumentParsingField, bool) {
	candidates := linesForFieldRule(rule, pages, sections)
	for _, candidate := range candidates {
		if value, excerpt := findValueByAliases(candidate.Lines, rule.Aliases); value != "" {
			return serviceports.DocumentParsingField{
				Key:             rule.Key,
				Label:           rule.Label,
				Value:           normalizeValue(value, rule.Normalizer),
				Confidence:      confidenceOrDefault(rule.Confidence, 0.9),
				PageNumber:      candidate.PageNumber,
				ReviewRequired:  rule.Required && strings.TrimSpace(value) == "",
				EvidenceExcerpt: excerpt,
				Source:          "parsing_rule",
			}, true
		}
		if value, excerpt := findValueByPatterns(candidate.Text, rule.Patterns); value != "" {
			return serviceports.DocumentParsingField{
				Key:             rule.Key,
				Label:           rule.Label,
				Value:           normalizeValue(value, rule.Normalizer),
				Confidence:      confidenceOrDefault(rule.Confidence, 0.88),
				PageNumber:      candidate.PageNumber,
				ReviewRequired:  false,
				EvidenceExcerpt: excerpt,
				Source:          "parsing_rule",
			}, true
		}
	}
	return serviceports.DocumentParsingField{}, false
}

type fieldCandidate struct {
	PageNumber int
	Lines      []string
	Text       string
}

func linesForFieldRule(
	rule documentparsingrule.FieldRule,
	pages []serviceports.DocumentParsingPage,
	sections []sectionOccurrence,
) []fieldCandidate {
	candidates := make([]fieldCandidate, 0)
	if len(rule.SectionNames) > 0 {
		for _, section := range sections {
			if slices.ContainsFunc(rule.SectionNames, func(name string) bool {
				return strings.EqualFold(strings.TrimSpace(name), strings.TrimSpace(section.Name))
			}) {
				candidates = append(candidates, fieldCandidate{
					PageNumber: section.PageNumber,
					Lines:      section.Lines,
					Text:       section.Text,
				})
			}
		}
		return candidates
	}
	for _, page := range pages {
		lines := splitLines(page.Text)
		candidates = append(candidates, fieldCandidate{
			PageNumber: page.PageNumber,
			Lines:      lines,
			Text:       strings.Join(lines, "\n"),
		})
	}
	return candidates
}

func evaluateStopRule(
	rule documentparsingrule.StopRule,
	pages []serviceports.DocumentParsingPage,
	sections []sectionOccurrence,
) []serviceports.DocumentParsingStop {
	blocks := stopBlocksForRule(rule, pages, sections)
	stops := make([]serviceports.DocumentParsingStop, 0, len(blocks))
	for idx, block := range blocks {
		stop := serviceports.DocumentParsingStop{
			Sequence:        max(1, rule.SequenceStart) + idx,
			Role:            strings.ToLower(strings.TrimSpace(rule.Role)),
			PageNumber:      block.PageNumber,
			EvidenceExcerpt: block.Text,
			Source:          "parsing_rule",
		}
		confidenceSum := 0.0
		confidenceParts := 0.0
		reviewRequired := false
		for _, extractor := range rule.Extractors {
			value, found := extractStopField(extractor, block.Lines, block.Text)
			if !found {
				reviewRequired = reviewRequired || extractor.Required
				continue
			}
			assignStopValue(&stop, extractor.FieldKey, normalizeValue(value, extractor.Normalizer))
			confidenceSum += confidenceOrDefault(extractor.Confidence, 0.88)
			confidenceParts++
		}
		backfillCityStatePostal(&stop, block.Lines)
		for _, pattern := range rule.AppointmentPatterns {
			re, err := regexp.Compile(pattern)
			if err == nil && re.MatchString(block.Text) {
				stop.AppointmentRequired = true
			}
		}
		if confidenceParts == 0 {
			continue
		}
		stop.Confidence = clampConfidence(confidenceSum / confidenceParts)
		stop.ReviewRequired = reviewRequired || stop.Confidence < 0.82
		if stop.Name == "" && stop.AddressLine1 == "" && stop.City == "" && stop.Date == "" && stop.TimeWindow == "" {
			continue
		}
		stops = append(stops, stop)
	}
	return stops
}

func stopBlocksForRule(
	rule documentparsingrule.StopRule,
	pages []serviceports.DocumentParsingPage,
	sections []sectionOccurrence,
) []sectionOccurrence {
	if len(rule.SectionNames) > 0 {
		matches := make([]sectionOccurrence, 0)
		for _, section := range sections {
			if slices.ContainsFunc(rule.SectionNames, func(name string) bool {
				return strings.EqualFold(strings.TrimSpace(name), strings.TrimSpace(section.Name))
			}) {
				matches = append(matches, section)
			}
		}
		return matches
	}

	definition := documentparsingrule.SectionRule{
		Name:          rule.Role,
		StartAnchors:  rule.StartAnchors,
		EndAnchors:    rule.EndAnchors,
		AllowMultiple: rule.AllowMultiple,
	}
	return extractSections([]documentparsingrule.SectionRule{definition}, pages, "")
}

func extractStopField(
	extractor documentparsingrule.StopFieldRule,
	lines []string,
	text string,
) (string, bool) {
	if value, _ := findValueByAliases(lines, extractor.Aliases); value != "" {
		return value, true
	}
	if value, _ := findValueByPatterns(text, extractor.Patterns); value != "" {
		return value, true
	}
	return "", false
}

func findValueByAliases(lines []string, aliases []string) (string, string) {
	for _, alias := range aliases {
		trimmedAlias := strings.TrimSpace(alias)
		if trimmedAlias == "" {
			continue
		}
		pattern := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(trimmedAlias) + `\s*(?:[:#-]\s*|\s+)(.+)$`)
		for idx, line := range lines {
			matches := pattern.FindStringSubmatch(strings.TrimSpace(line))
			if len(matches) > 1 {
				value := strings.TrimSpace(matches[1])
				if value != "" {
					return value, line
				}
			}
			if strings.EqualFold(strings.TrimSpace(line), trimmedAlias) && idx+1 < len(lines) {
				value := strings.TrimSpace(lines[idx+1])
				if value != "" {
					return value, line + "\n" + lines[idx+1]
				}
			}
		}
	}
	return "", ""
}

func findValueByPatterns(text string, patterns []string) (string, string) {
	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		matches := re.FindStringSubmatch(text)
		if len(matches) > 1 {
			for _, match := range matches[1:] {
				if strings.TrimSpace(match) != "" {
					return strings.TrimSpace(match), firstNonEmpty(matches[0], text)
				}
			}
		}
	}
	return "", ""
}

func assignStopValue(stop *serviceports.DocumentParsingStop, key, value string) {
	switch key {
	case "name":
		stop.Name = value
	case "addressLine1":
		stop.AddressLine1 = value
	case "addressLine2":
		stop.AddressLine2 = value
	case "city":
		stop.City = value
	case "state":
		stop.State = strings.ToUpper(value)
	case "postalCode":
		stop.PostalCode = value
	case "date":
		stop.Date = value
	case "timeWindow":
		stop.TimeWindow = value
	}
}

func backfillCityStatePostal(stop *serviceports.DocumentParsingStop, lines []string) {
	if stop.City != "" && stop.State != "" && stop.PostalCode != "" {
		return
	}
	for _, line := range lines {
		matches := cityStatePostalPattern.FindStringSubmatch(line)
		if len(matches) != 4 {
			continue
		}
		if stop.City == "" {
			stop.City = strings.TrimSpace(matches[1])
		}
		if stop.State == "" {
			stop.State = strings.ToUpper(strings.TrimSpace(matches[2]))
		}
		if stop.PostalCode == "" {
			stop.PostalCode = strings.TrimSpace(matches[3])
		}
		return
	}
}

func normalizeStopSequences(stops []serviceports.DocumentParsingStop) []serviceports.DocumentParsingStop {
	if len(stops) == 0 {
		return []serviceports.DocumentParsingStop{}
	}
	for idx := range stops {
		if stops[idx].Sequence <= 0 {
			stops[idx].Sequence = idx + 1
		}
	}
	return stops
}

func normalizeValue(value, normalizer string) string {
	value = strings.TrimSpace(value)
	switch strings.TrimSpace(strings.ToLower(normalizer)) {
	case "currency":
		value = strings.ReplaceAll(value, ",", "")
		if n, err := strconv.ParseFloat(strings.TrimPrefix(strings.TrimSpace(value), "$"), 64); err == nil {
			return fmt.Sprintf("$%.2f", n)
		}
	case "state":
		return strings.ToUpper(value)
	case "reference":
		return strings.TrimSpace(strings.Trim(value, "#:"))
	default:
	}
	return value
}

func confidenceOrDefault(value, fallback float64) float64 {
	if value > 0 {
		return clampConfidence(value)
	}
	return fallback
}

func clampConfidence(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func hasReviewRequiredField(fields map[string]serviceports.DocumentParsingField) bool {
	for _, field := range fields {
		if field.ReviewRequired {
			return true
		}
	}
	return false
}

func hasReviewRequiredStop(stops []serviceports.DocumentParsingStop) bool {
	for _, stop := range stops {
		if stop.ReviewRequired {
			return true
		}
	}
	return false
}

func splitLines(text string) []string {
	if strings.TrimSpace(text) == "" {
		return []string{}
	}
	raw := strings.Split(strings.ReplaceAll(text, "\r", ""), "\n")
	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		lines = append(lines, strings.TrimSpace(line))
	}
	return lines
}

func trimBlock(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}

func lineMatchesAny(line string, needles []string) bool {
	line = strings.ToLower(strings.TrimSpace(line))
	for _, needle := range needles {
		if strings.Contains(line, strings.ToLower(strings.TrimSpace(needle))) {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func mergeAnalyses(
	baseline *serviceports.DocumentParsingAnalysis,
	candidate *serviceports.DocumentParsingAnalysis,
) *serviceports.DocumentParsingAnalysis {
	if candidate == nil {
		return baseline
	}
	if baseline == nil {
		return candidate
	}

	merged := &serviceports.DocumentParsingAnalysis{
		Fields:            make(map[string]serviceports.DocumentParsingField, len(baseline.Fields)+len(candidate.Fields)),
		Stops:             append([]serviceports.DocumentParsingStop{}, baseline.Stops...),
		Conflicts:         append([]serviceports.DocumentParsingConflict{}, baseline.Conflicts...),
		MissingFields:     []string{},
		Signals:           dedupeStrings(append(append([]string{}, baseline.Signals...), candidate.Signals...)),
		ReviewStatus:      baseline.ReviewStatus,
		OverallConfidence: maxFloat(baseline.OverallConfidence, candidate.OverallConfidence),
		Metadata:          candidate.Metadata,
	}

	for key, field := range baseline.Fields {
		merged.Fields[key] = field
	}
	for key, field := range candidate.Fields {
		if existing, ok := merged.Fields[key]; ok && strings.TrimSpace(existing.Value) != "" && strings.TrimSpace(field.Value) != "" && existing.Value != field.Value {
			merged.Conflicts = append(merged.Conflicts, serviceports.DocumentParsingConflict{
				Key:         key,
				Label:       field.Label,
				Values:      dedupeStrings([]string{existing.Value, field.Value}),
				PageNumbers: dedupeInts([]int{existing.PageNumber, field.PageNumber}),
				Source:      "parsing_rule_merge",
			})
		}
		if shouldReplaceField(merged.Fields[key], field) {
			merged.Fields[key] = field
		}
	}

	for _, stop := range candidate.Stops {
		replaced := false
		for idx, existing := range merged.Stops {
			if strings.EqualFold(existing.Role, stop.Role) && stopCompleteness(stop) >= stopCompleteness(existing) {
				merged.Stops[idx] = stop
				replaced = true
				break
			}
		}
		if !replaced {
			merged.Stops = append(merged.Stops, stop)
		}
	}

	merged.MissingFields = dedupeStrings(append(append([]string{}, baseline.MissingFields...), candidate.MissingFields...))
	for key, field := range merged.Fields {
		merged.MissingFields = removeValue(merged.MissingFields, field.Label)
		merged.MissingFields = removeValue(merged.MissingFields, key)
	}
	for _, stop := range merged.Stops {
		if strings.EqualFold(stop.Role, "pickup") {
			merged.MissingFields = removeValue(merged.MissingFields, "Pickup Stop")
		}
		if strings.EqualFold(stop.Role, "delivery") {
			merged.MissingFields = removeValue(merged.MissingFields, "Delivery Stop")
		}
	}

	if len(merged.Fields) == 0 && len(merged.Stops) == 0 {
		merged.ReviewStatus = "Unavailable"
		return merged
	}
	if len(merged.MissingFields) == 0 && len(merged.Conflicts) == 0 && !hasReviewRequiredField(merged.Fields) && !hasReviewRequiredStop(merged.Stops) && merged.OverallConfidence >= 0.82 {
		merged.ReviewStatus = "Ready"
	} else {
		merged.ReviewStatus = "NeedsReview"
	}

	return merged
}

func shouldReplaceField(existing, candidate serviceports.DocumentParsingField) bool {
	if strings.TrimSpace(existing.Value) == "" {
		return true
	}
	if strings.TrimSpace(candidate.Value) == "" {
		return false
	}
	if existing.ReviewRequired && !candidate.ReviewRequired {
		return true
	}
	return candidate.Confidence >= existing.Confidence
}

func stopCompleteness(stop serviceports.DocumentParsingStop) int {
	score := 0
	for _, value := range []string{stop.Name, stop.AddressLine1, stop.City, stop.State, stop.PostalCode, stop.Date, stop.TimeWindow} {
		if strings.TrimSpace(value) != "" {
			score++
		}
	}
	return score
}

func removeValue(values []string, target string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(target)) {
			continue
		}
		out = append(out, value)
	}
	return out
}

func dedupeInts(values []int) []int {
	out := make([]int, 0, len(values))
	for _, value := range values {
		if value <= 0 || slices.Contains(out, value) {
			continue
		}
		out = append(out, value)
	}
	return out
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
