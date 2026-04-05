package documentintelligencejobs

import (
	"regexp"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/shared/stringutils"
)

func extractRateConfirmationStops(pages []*PageExtractionResult) []*IntelligenceStop {
	stops := make([]*IntelligenceStop, 0, 2)

	for _, page := range pages {
		lines := splitNormalizedLines(page.Text)
		for idx, line := range lines {
			role, ok := detectStopRole(line)
			if !ok {
				continue
			}

			stop := &IntelligenceStop{
				Sequence:        len(stops) + 1,
				Role:            role,
				PageNumber:      page.PageNumber,
				EvidenceExcerpt: collectStopExcerpt(lines, idx),
				Confidence:      baseStopConfidence(page),
				ReviewRequired:  false,
				Source:          "deterministic",
			}

			block := collectRateConfirmationStopBlock(lines, idx)
			if len(block) > 0 {
				stop.EvidenceExcerpt = strings.Join(block, "\n")
			}
			populateRateConfirmationStop(stop, block)
			if !hasMeaningfulStopData(stop) {
				continue
			}

			if stop.AddressLine1 == "" || stop.Date == "" {
				stop.ReviewRequired = true
				stop.Confidence = clampConfidence(stop.Confidence - stopMissingAddressPenalty)
			}
			if stop.City == "" || stop.State == "" {
				stop.ReviewRequired = true
				stop.Confidence = clampConfidence(stop.Confidence - stopMissingCityStatePenalty)
			}

			stops = append(stops, stop)
		}
	}

	return stops
}

func detectStopRole(line string) (string, bool) {
	lower := strings.ToLower(strings.TrimSpace(line))
	if isStopMetadataLine(lower) {
		return "", false
	}
	switch {
	case stopSectionRegex.MatchString(lower):
		switch {
		case strings.HasPrefix(lower, "shipper"),
			strings.HasPrefix(lower, "pickup"),
			strings.HasPrefix(lower, "origin"):
			return stopRolePickup, true
		case strings.HasPrefix(lower, "receiver"),
			strings.HasPrefix(lower, "consignee"),
			strings.HasPrefix(lower, "delivery"),
			strings.HasPrefix(lower, "drop"),
			strings.HasPrefix(lower, "destination"):
			return stopRoleDelivery, true
		}
		return "", false
	case strings.HasPrefix(lower, "pickup"):
		if strings.HasPrefix(lower, "pickup date") || strings.HasPrefix(lower, "pickup window") {
			return "", false
		}
		return stopRolePickup, true
	case strings.HasPrefix(lower, "delivery"), strings.HasPrefix(lower, "drop"):
		if strings.HasPrefix(lower, "delivery date") ||
			strings.HasPrefix(lower, "delivery window") {
			return "", false
		}
		return stopRoleDelivery, true
	default:
		return "", false
	}
}

func collectRateConfirmationStopBlock(lines []string, idx int) []string {
	end := min(idx+stopBlockMaxLines, len(lines))

	block := make([]string, 0, end-idx)
	blankRun := 0
	for pos := idx; pos < end; pos++ {
		line := strings.TrimSpace(lines[pos])

		if pos > idx {
			if _, nextStop := detectStopRole(line); nextStop {
				break
			}
			if strings.HasPrefix(line, "--- PAGE ") {
				break
			}
		}

		if line == "" {
			blankRun++
			if blankRun > stopBlockMaxBlankRun && hasStopSignal(block) {
				break
			}
			continue
		}

		blankRun = 0
		block = append(block, line)
	}

	return block
}

func populateRateConfirmationStop(stop *IntelligenceStop, block []string) {
	if stop == nil || len(block) == 0 {
		return
	}

	header := block[0]
	if labelValue := extractLabelValue(header); labelValue != "" &&
		!isStopMetadataLine(strings.ToLower(labelValue)) {
		switch {
		case looksLikeAddress(labelValue):
			stop.AddressLine1 = labelValue
		case dateLabelRegex.MatchString(header):
			if stop.Date == "" {
				stop.Date = firstRegexValue(dateValueRegex, labelValue)
			}
			if stop.TimeWindow == "" {
				stop.TimeWindow = firstRegexValue(timeWindowRegex, labelValue)
			}
		default:
			stop.Name = labelValue
		}
	}

	if stop.Date == "" {
		stop.Date = findLastRegexValue(dateValueRegex, block)
	}
	if stop.TimeWindow == "" {
		stop.TimeWindow = findLastStopTimeValue(block)
	}

	cityIdx, city, state, postalCode := findLastCityStateZip(block)
	if cityIdx >= 0 {
		stop.City = city
		stop.State = state
		stop.PostalCode = postalCode
		if stop.AddressLine1 == "" {
			stop.AddressLine1, stop.AddressLine2 = extractAddressBeforeCity(block, cityIdx)
		}
		if stop.Name == "" {
			stop.Name = extractStopNameBeforeIndex(block, cityIdx)
		}
	}

	if stop.AddressLine1 == "" {
		stop.AddressLine1 = findLastAddressLine(block)
	}
	if stop.Name == "" {
		stop.Name = extractStopNameBeforeIndex(block, len(block))
	}

	stop.Name = sanitizeStopName(stop.Name)
	stop.AddressLine1 = strings.TrimSpace(stop.AddressLine1)
	stop.AddressLine2 = strings.TrimSpace(stop.AddressLine2)
	stop.Date = strings.TrimSpace(stop.Date)
	stop.TimeWindow = strings.TrimSpace(stop.TimeWindow)
	stop.AppointmentRequired = strings.Contains(
		strings.ToLower(strings.Join(block, "\n")),
		"appointment",
	) ||
		strings.Contains(strings.ToLower(stop.TimeWindow), "appt")
}

func hasStopSignal(block []string) bool {
	for _, line := range block {
		if looksLikeAddress(line) || cityStateZipRegex.MatchString(line) ||
			dateValueRegex.MatchString(line) ||
			timeWindowRegex.MatchString(line) {
			return true
		}
	}
	return false
}

func findLastRegexValue(re *regexp.Regexp, block []string) string {
	for idx := len(block) - 1; idx >= 0; idx-- {
		if value := firstRegexValue(re, block[idx]); value != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func findLastStopTimeValue(block []string) string {
	if value := findLastRegexValue(timeWindowRegex, block); value != "" {
		return value
	}
	return findLastRegexValue(appointmentRegex, block)
}

func findLastCityStateZip(
	block []string,
) (foundIdx int, foundCity, foundState, foundZip string) {
	for i := len(block) - 1; i >= 0; i-- {
		if !cityStateZipRegex.MatchString(block[i]) {
			continue
		}
		city, state, postalCode := extractCityStateZip(block[i])
		if city != "" && state != "" {
			return i, city, state, postalCode
		}
	}
	return -1, "", "", ""
}

func extractAddressBeforeCity(
	block []string, cityIdx int,
) (addr1, addr2 string) { //nolint:unparam // addr2 reserved for future suite/apt lines
	if cityIdx <= 0 || cityIdx > len(block) {
		return "", ""
	}

	prevIdx, prevLine := previousMeaningfulStopLine(block, cityIdx-1)
	if prevIdx < 0 {
		return "", ""
	}

	if looksLikeAddress(prevLine) {
		return prevLine, ""
	}
	if isStreetFragment(prevLine) {
		numberIdx, numberLine := previousMeaningfulStopLine(block, prevIdx-1)
		if numberIdx >= 0 && isNumericAddressPrefix(numberLine) {
			return strings.TrimSpace(numberLine + " " + prevLine), ""
		}
	}

	return "", ""
}

func extractStopNameBeforeIndex(block []string, limit int) string {
	if limit > len(block) {
		limit = len(block)
	}
	for idx := limit - 1; idx >= 0; idx-- {
		line := strings.TrimSpace(block[idx])
		if !isUsableStopName(line) {
			continue
		}
		return line
	}
	return ""
}

func previousMeaningfulStopLine(
	block []string, start int,
) (foundIdx int, foundLine string) {
	for i := start; i >= 0; i-- {
		l := strings.TrimSpace(block[i])
		if l == "" || isStopMetadataLine(strings.ToLower(l)) ||
			phoneLineRegex.MatchString(l) {
			continue
		}
		return i, l
	}
	return -1, ""
}

func findLastAddressLine(block []string) string {
	for idx := len(block) - 1; idx >= 0; idx-- {
		line := strings.TrimSpace(block[idx])
		if line == "" || isStopMetadataLine(strings.ToLower(line)) {
			continue
		}
		if looksLikeAddress(line) {
			return line
		}
		if isStreetFragment(line) {
			if prevIdx, prevLine := previousMeaningfulStopLine(block, idx-1); prevIdx >= 0 &&
				isNumericAddressPrefix(prevLine) {
				return strings.TrimSpace(prevLine + " " + line)
			}
		}
	}
	return ""
}

func sanitizeStopName(name string) string {
	trimmed := strings.TrimSpace(name)
	if !isUsableStopName(trimmed) {
		return ""
	}
	return trimmed
}

func hasMeaningfulStopData(stop *IntelligenceStop) bool {
	return strings.TrimSpace(stop.Name) != "" ||
		strings.TrimSpace(stop.AddressLine1) != "" ||
		(strings.TrimSpace(stop.City) != "" && strings.TrimSpace(stop.State) != "") ||
		strings.TrimSpace(stop.Date) != "" ||
		strings.TrimSpace(stop.TimeWindow) != ""
}

func isUsableStopName(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	lower := strings.ToLower(trimmed)
	if isStopMetadataLine(lower) || phoneLineRegex.MatchString(trimmed) ||
		looksLikeAddress(trimmed) ||
		cityStateZipRegex.MatchString(trimmed) ||
		dateValueRegex.MatchString(trimmed) ||
		isStreetFragment(trimmed) ||
		isNumericAddressPrefix(trimmed) {
		return false
	}
	return true
}

func isStopMetadataLine(lower string) bool {
	normalized := normalizeSectionLabel(lower)
	if normalized == "" {
		return false
	}

	switch {
	case normalized == "shipper instructions",
		normalized == "receiver instructions",
		normalized == "address",
		normalized == "phone",
		normalized == "ref #",
		normalized == "ref",
		normalized == "commodity",
		normalized == "est wgt",
		normalized == "units",
		normalized == "count",
		normalized == "pallets",
		normalized == "temp",
		normalized == "driver name",
		normalized == "trailer #",
		normalized == "tractor #",
		normalized == "pickup#",
		normalized == "delivery#",
		normalized == "appointment#",
		normalized == "pick up date",
		normalized == "pick up time",
		normalized == "pickup date",
		normalized == "pickup time",
		normalized == "delivery date",
		normalized == "delivery time":
		return true
	case strings.HasPrefix(normalized, "please "),
		strings.HasPrefix(normalized, "scheduled "),
		strings.HasPrefix(normalized, "page "),
		strings.HasPrefix(normalized, "this load was booked"),
		strings.HasPrefix(normalized, "thank you"),
		normalized == "loose(s)":
		return true
	default:
		return false
	}
}

func isStreetFragment(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || looksLikeAddress(trimmed) || cityStateZipRegex.MatchString(trimmed) {
		return false
	}
	if isNumericAddressPrefix(trimmed) || phoneLineRegex.MatchString(trimmed) ||
		dateValueRegex.MatchString(trimmed) {
		return false
	}
	lower := strings.ToLower(trimmed)
	if isStopMetadataLine(lower) {
		return false
	}

	return stringutils.ContainsAny(lower,
		"street", "st", "road", "rd", "drive", "dr",
		"avenue", "ave", "boulevard", "blvd", "lane", "ln",
		"court", "ct", "circle", "cir", "way",
		"parkway", "pkwy", "highway", "hwy", "suite", "ste",
	)
}

func isNumericAddressPrefix(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	for _, r := range trimmed {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func collectStopExcerpt(lines []string, idx int) string {
	end := min(idx+stopExcerptLines, len(lines))
	chunk := make([]string, 0, end-idx)
	for _, line := range lines[idx:end] {
		if strings.TrimSpace(line) == "" {
			break
		}
		chunk = append(chunk, line)
	}
	return strings.Join(chunk, "\n")
}

func extractLabelValue(line string) string {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func looksLikeAddress(value string) bool {
	return addressLineRegex.MatchString(strings.TrimSpace(value))
}

func extractCityStateZip(line string) (city, state, postalCode string) {
	match := cityStateZipRegex.FindStringSubmatch(strings.TrimSpace(line))
	if len(match) != 4 {
		return "", "", ""
	}
	return strings.TrimSpace(
			match[1],
		), strings.ToUpper(
			strings.TrimSpace(match[2]),
		), strings.TrimSpace(
			match[3],
		)
}

func baseStopConfidence(page *PageExtractionResult) float64 {
	if page.SourceKind == documentcontent.SourceKindOCR {
		return clampConfidence((ocrBaseStopConfidence + page.OCRConfidence) / 2)
	}
	return nativeBaseStopConfidence
}
