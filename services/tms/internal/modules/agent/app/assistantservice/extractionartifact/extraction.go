package extractionartifact

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/artifactpayload"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/recordviews"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aicorrection"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/textcase"
	"github.com/emoss08/trenova/shared/typeutils"
)

// lowConfidence is where a reading needs a person to look: the same floor
// document intelligence uses to mark a field for review.
const lowConfidence = 0.8

// DraftDocumentIDParam names the document in intake_get_draft's arguments
// and in the draft it returns.
const DraftDocumentIDParam = "documentId"

const (
	maxNeedleRunes = 24
	minPrefixRunes = 6
	minBoxWidth    = 0.02
	minBoxHeight   = 0.01

	reasonConflict = "conflict"
	reasonReview   = "review"
	reasonLow      = "low"

	stopPartName    = "name"
	stopPartAddress = "address"
	stopPartWindow  = "window"
)

type ExtractionPage struct {
	Number int
	Width  int
	Height int
	Lines  []documentcontent.LayoutLine
}

type ExtractionSource struct {
	FileName       string
	ContentType    string
	KindConfidence float64
	Pages          []ExtractionPage
	FieldEvidence  map[string]string
	StopEvidence   map[int]string
}

type fieldReading struct {
	key        string
	label      string
	value      string
	evidence   string
	confidence float64
	page       int
	review     bool
	conflict   bool
	readings   []string
	stop       *stopPart
}

type stopPart struct {
	sequence    int
	role        string
	part        string
	place       string
	appointment bool
}

type placement struct {
	page int
	box  map[string]float64
}

func DraftEvidence(
	data map[string]any,
) (fields map[string]string, stops map[int]string) {
	fields = map[string]string{}
	if raw, ok := data["fields"].(map[string]any); ok {
		for key, value := range raw {
			field, isObject := value.(map[string]any)
			if !isObject {
				continue
			}
			if excerpt := evidenceOf(field); excerpt != "" {
				fields[key] = excerpt
			}
		}
	}

	stops = map[int]string{}
	if raw, ok := data["stops"].([]any); ok {
		for idx, value := range raw {
			stop, isObject := value.(map[string]any)
			if !isObject {
				continue
			}
			if excerpt := evidenceOf(stop); excerpt != "" {
				stops[idx] = excerpt
			}
		}
	}

	return fields, stops
}

func evidenceOf(object map[string]any) string {
	return cmp.Or(
		typeutils.StringOfTrimmed(object["evidenceExcerpt"]),
		typeutils.StringOfTrimmed(object["excerpt"]),
	)
}

func ExtractionArtifact(
	callID string,
	draft map[string]any,
	source *ExtractionSource,
) *assistantartifact.Artifact {
	pages := slices.Clone(source.Pages)
	slices.SortFunc(pages, func(a, b ExtractionPage) int { return cmp.Compare(a.Number, b.Number) })

	fields := extractionFields(draftReadings(draft, source), pages)
	if len(fields) == 0 {
		return nil
	}

	pageCount := len(pages)
	needs := 0
	for _, field := range fields {
		if page, _ := field["page"].(int); page > pageCount {
			pageCount = page
		}
		if field["needsLook"] == true {
			needs++
		}
	}

	documentKind := typeutils.StringOfTrimmed(draft["documentKind"])
	kindLabel := textcase.CapitalizeFirst(strings.ToLower(splitCamel(documentKind)))
	if kindLabel == "" {
		kindLabel = "Document"
	}
	title := kindLabel
	if source.FileName != "" {
		title += " · " + source.FileName
	}

	missing := artifactpayload.StringsOf(draft["missingFields"])
	payload := map[string]any{
		"documentId":     typeutils.StringOfTrimmed(draft[DraftDocumentIDParam]),
		"fileName":       source.FileName,
		"contentType":    source.ContentType,
		"documentKind":   documentKind,
		"kindLabel":      kindLabel,
		"kindConfidence": roundConfidence(source.KindConfidence),
		"confidence":     recordviews.NumberOf(draft, "confidence"),
		"reviewStatus":   typeutils.StringOfTrimmed(draft["reviewStatus"]),
		"pageCount":      max(pageCount, 1),
		"pages":          pageSizes(pages),
		"fields":         fields,
		"needsLook":      needs,
		"missing":        missing,
		"missingFields":  missingFields(missing),
		"tool":           artifactpayload.ToolGetShipmentDraft,
	}
	if attached := typeutils.StringOfTrimmed(draft["attachedShipmentId"]); attached != "" {
		payload["attachedShipmentId"] = attached
	}

	return &assistantartifact.Artifact{
		Kind:             assistantartifact.KindExtraction,
		Status:           assistantartifact.StatusReady,
		Title:            assistantartifact.NormalizeTitle(title),
		Payload:          payload,
		SourceToolCallID: callID,
		LineageKey: ExtractionLineageKey(
			typeutils.StringOfTrimmed(draft[DraftDocumentIDParam]),
		),
	}
}

// ExtractionLineageKey names the extraction of one document, so the card
// shown when the file was attached and the one the agent opens by reading
// the draft are versions of one artifact.
func ExtractionLineageKey(documentID string) string {
	return assistantartifact.LineageKeyFor(
		assistantartifact.KindExtraction,
		artifactpayload.ToolGetShipmentDraft,
		map[string]any{DraftDocumentIDParam: documentID},
	)
}

func pageSizes(pages []ExtractionPage) []map[string]int {
	out := make([]map[string]int, 0, len(pages))
	for _, page := range pages {
		if page.Number < 1 {
			continue
		}
		size := map[string]int{"number": page.Number}
		if page.Width > 0 && page.Height > 0 {
			size["width"] = page.Width
			size["height"] = page.Height
		}
		out = append(out, size)
	}

	return out
}

func missingFields(labels []string) []map[string]string {
	out := make([]map[string]string, 0, len(labels))
	for _, label := range labels {
		label = strings.TrimSpace(label)
		if label == "" {
			continue
		}
		key := aicorrection.CanonicalFieldKey(label)
		if key == label {
			key = textcase.LowerCamel(label)
		}
		out = append(out, map[string]string{"key": key, "label": label})
	}

	return out
}

func draftReadings(draft map[string]any, source *ExtractionSource) []fieldReading {
	raw, _ := draft["fields"].(map[string]any)
	stops, _ := draft["stops"].([]any)
	conflicts := conflictReadings(draft["conflicts"])

	out := make([]fieldReading, 0, len(raw)+len(stops)*3)
	for key, value := range raw {
		field, isObject := value.(map[string]any)
		if !isObject {
			continue
		}
		text := DisplayValue(field["value"])
		if text == "" {
			continue
		}
		label := typeutils.StringOfTrimmed(field["label"])
		if label == "" {
			label = textcase.CapitalizeFirst(strings.ToLower(splitCamel(key)))
		}
		readings := conflicts[key]
		out = append(out, fieldReading{
			key:        key,
			label:      label,
			value:      text,
			evidence:   source.FieldEvidence[key],
			confidence: recordviews.NumberOf(field, "confidence"),
			page:       int(recordviews.NumberOf(field, "pageNumber")),
			review:     typeutils.BoolOf(field["reviewRequired"]),
			conflict:   typeutils.BoolOf(field["conflict"]) || len(readings) > 1,
			readings:   readings,
		})
	}

	for idx, value := range stops {
		stop, isObject := value.(map[string]any)
		if !isObject {
			continue
		}
		out = append(out, stopReadings(idx, stop, source.StopEvidence[idx])...)
	}

	return out
}

func stopReadings(idx int, stop map[string]any, evidence string) []fieldReading {
	sequence := int(recordviews.NumberOf(stop, "sequence"))
	if sequence < 1 {
		sequence = idx + 1
	}
	role := strings.ToLower(typeutils.StringOfTrimmed(stop["role"]))
	appointment := typeutils.BoolOf(stop["appointmentRequired"])
	place := stopPlace(stop)

	parts := []struct {
		part  string
		label string
		value string
	}{
		{stopPartName, "Name", typeutils.StringOfTrimmed(stop["name"])},
		{stopPartAddress, "Address", stopAddress(stop)},
		{stopPartWindow, "Window", strings.Join(stringutils.NonEmptyStrings(
			typeutils.StringOfTrimmed(stop["date"]),
			typeutils.StringOfTrimmed(stop["timeWindow"]),
		), " ")},
	}

	out := make([]fieldReading, 0, len(parts))
	for _, part := range parts {
		if part.value == "" {
			continue
		}
		out = append(out, fieldReading{
			key:        fmt.Sprintf("stop%d.%s", sequence, part.part),
			label:      part.label,
			value:      part.value,
			evidence:   evidence,
			confidence: recordviews.NumberOf(stop, "confidence"),
			page:       int(recordviews.NumberOf(stop, "pageNumber")),
			review:     typeutils.BoolOf(stop["reviewRequired"]),
			stop: &stopPart{
				sequence:    sequence,
				role:        role,
				part:        part.part,
				place:       place,
				appointment: appointment && part.part == stopPartWindow,
			},
		})
	}

	return out
}

func stopPlace(stop map[string]any) string {
	return strings.Join(stringutils.NonEmptyStrings(
		typeutils.StringOfTrimmed(stop["city"]),
		typeutils.StringOfTrimmed(stop["state"]),
	), ", ")
}

func stopAddress(stop map[string]any) string {
	place := stopPlace(stop)
	if postal := typeutils.StringOfTrimmed(stop["postalCode"]); postal != "" {
		place = strings.TrimSpace(place + " " + postal)
	}

	return strings.Join(stringutils.NonEmptyStrings(
		typeutils.StringOfTrimmed(stop["addressLine1"]),
		typeutils.StringOfTrimmed(stop["addressLine2"]),
		place,
	), ", ")
}

func conflictReadings(value any) map[string][]string {
	raw, _ := value.([]any)
	out := make(map[string][]string, len(raw))
	for _, item := range raw {
		conflict, isObject := item.(map[string]any)
		if !isObject {
			continue
		}
		key := typeutils.StringOfTrimmed(conflict["key"])
		values, _ := conflict["values"].([]any)
		if key == "" || len(values) == 0 {
			continue
		}
		readings := make([]string, 0, len(values))
		for _, reading := range values {
			if text := DisplayValue(reading); text != "" && !slices.Contains(readings, text) {
				readings = append(readings, text)
			}
		}
		out[key] = readings
	}

	return out
}

func extractionFields(readings []fieldReading, pages []ExtractionPage) []map[string]any {
	out := make([]map[string]any, 0, len(readings))
	for i := range readings {
		out = append(out, extractionField(&readings[i], pages))
	}

	slices.SortStableFunc(out, func(a, b map[string]any) int {
		return cmp.Or(
			cmp.Compare(sortPage(a), sortPage(b)),
			cmp.Compare(sortTop(a), sortTop(b)),
			cmp.Compare(fieldKey(a), fieldKey(b)),
		)
	})

	return out
}

func extractionField(reading *fieldReading, pages []ExtractionPage) map[string]any {
	confidence := roundConfidence(reading.confidence)
	reason := ""
	switch {
	case reading.conflict:
		reason = reasonConflict
	case reading.review:
		reason = reasonReview
	case confidence < lowConfidence:
		reason = reasonLow
	}

	field := map[string]any{
		"key":        reading.key,
		"label":      reading.label,
		"value":      reading.value,
		"confidence": confidence,
		"needsLook":  reason != "",
	}
	if reason != "" {
		field["reason"] = reason
	}
	if len(reading.readings) > 1 {
		field["readings"] = reading.readings
	}
	if reading.stop != nil {
		field["stop"] = reading.stop.sequence
		field["role"] = reading.stop.role
		field["part"] = reading.stop.part
		if reading.stop.place != "" {
			field["place"] = reading.stop.place
		}
		if reading.stop.appointment {
			field["appointment"] = true
		}
	}

	if found, ok := locate(reading, pages); ok {
		field["page"] = found.page
		field["box"] = found.box
	} else if reading.page > 0 {
		field["page"] = reading.page
	}

	return field
}

func roundConfidence(value float64) float64 {
	return math.Round(math.Max(0, math.Min(1, value))*100) / 100
}

type lineMatch struct {
	page   int
	line   *documentcontent.LayoutLine
	at     int
	span   int
	score  int
	weight int
}

func (m *lineMatch) beats(other *lineMatch) bool {
	return other == nil || m.score > other.score ||
		(m.score == other.score && m.weight > other.weight)
}

func locate(reading *fieldReading, pages []ExtractionPage) (placement, bool) {
	evidence := foldedText(reading.evidence)
	label := foldedText(reading.label)
	words := strings.Fields(reading.value)
	for n := len(words); n > 0; n-- {
		value := foldedText(strings.Join(words[:n], " "))
		if n < len(words) && utf8.RuneCountInString(value) < minPrefixRunes {
			break
		}
		if best := bestLine(reading.page, pages, value, evidence, label); best != nil {
			return placement{page: best.page, box: boxOf(best)}, true
		}
	}

	return placement{}, false
}

func bestLine(named int, pages []ExtractionPage, value, evidence, label string) *lineMatch {
	if value == "" {
		return nil
	}
	needle := value
	if utf8.RuneCountInString(needle) > maxNeedleRunes {
		needle = string([]rune(needle)[:maxNeedleRunes])
	}

	var best *lineMatch
	for pass := range 2 {
		for pi := range pages {
			page := &pages[pi]
			if named > 0 && (pass == 0) != (page.Number == named) {
				continue
			}
			for li := range page.Lines {
				match, ok := matchLine(&page.Lines[li], value, needle, evidence, label)
				if !ok {
					continue
				}
				match.page = page.Number
				if match.beats(best) {
					best = &match
				}
			}
		}
		if best != nil || named <= 0 {
			break
		}
	}

	return best
}

func matchLine(
	line *documentcontent.LayoutLine,
	value, needle, evidence, label string,
) (lineMatch, bool) {
	hay, _ := comparableIndex(line.Text)
	at := strings.Index(hay, needle)
	if at < 0 || hay == "" {
		return lineMatch{}, false
	}

	span := len(needle)
	for span < len(value) && at+span < len(hay) && hay[at+span] == value[span] {
		span++
	}

	score, weight := 1, 0
	if quoted(hay, value, evidence) {
		score += 2
		weight = len(hay)
	}
	if label != "" && strings.Contains(hay[:at], label) {
		score++
	}

	return lineMatch{line: line, at: at, span: span, score: score, weight: weight}, true
}

func quoted(hay, value, evidence string) bool {
	if evidence == "" {
		return false
	}

	return strings.Contains(evidence, hay) ||
		(len(evidence) > len(value) && strings.Contains(hay, evidence))
}

func boxOf(match *lineMatch) map[string]float64 {
	line := match.line
	_, index := comparableIndex(line.Text)
	runes := []rune(line.Text)
	first := index[match.at]
	last := index[match.at+match.span-1] + 1

	x, right := spanAcross(line, runes, first, last)

	return map[string]float64{
		"x": round4(x),
		"y": round4(line.Y),
		"w": round4(math.Max(right-x, minBoxWidth)),
		"h": round4(math.Max(line.H, minBoxHeight)),
	}
}

func spanAcross(
	line *documentcontent.LayoutLine,
	runes []rune,
	first, last int,
) (left, right float64) {
	if words := wordRanges(runes); len(words)*2 == len(line.Words) && len(words) > 0 {
		return wordEdge(line, words, first, false), wordEdge(line, words, last, true)
	}

	share := float64(len(runes))
	return line.X + line.W*float64(first)/share, line.X + line.W*float64(last)/share
}

type wordRange struct {
	start int
	end   int
}

func wordRanges(runes []rune) []wordRange {
	var out []wordRange
	start := -1
	for i, r := range runes {
		switch {
		case r == ' ' && start >= 0:
			out = append(out, wordRange{start: start, end: i})
			start = -1
		case r != ' ' && start < 0:
			start = i
		}
	}
	if start >= 0 {
		out = append(out, wordRange{start: start, end: len(runes)})
	}

	return out
}

func wordEdge(line *documentcontent.LayoutLine, words []wordRange, at int, closing bool) float64 {
	for i, word := range words {
		if at > word.end || (at == word.end && !closing) {
			continue
		}
		left, right := line.Words[i*2], line.Words[i*2+1]
		if at <= word.start {
			if closing && i > 0 {
				return line.Words[i*2-1]
			}
			return left
		}
		share := float64(at-word.start) / float64(word.end-word.start)
		return left + (right-left)*share
	}

	return line.Words[len(line.Words)-1]
}

// foldedText is text as a value and a line are compared: lower case, letters
// and digits only, so "$1,359.56" on the page matches "1359.56" in the draft.
func foldedText(text string) string {
	hay, _ := comparableIndex(text)
	return hay
}

func comparableIndex(text string) (folded string, index []int) {
	var b strings.Builder
	index = make([]int, 0, len(text))
	position := 0
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			lower := unicode.ToLower(r)
			b.WriteRune(lower)
			for range utf8.RuneLen(lower) {
				index = append(index, position)
			}
		}
		position++
	}

	return b.String(), index
}

func DisplayValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case float64:
		if typed == math.Trunc(typed) {
			return fmt.Sprintf("%.0f", typed)
		}

		return fmt.Sprintf("%.2f", typed)
	case bool:
		if typed {
			return "Yes"
		}

		return "No"
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func splitCamel(text string) string {
	var b strings.Builder
	for idx, r := range text {
		if idx > 0 && unicode.IsUpper(r) {
			b.WriteByte(' ')
		}
		b.WriteRune(r)
	}

	return b.String()
}

func fieldKey(field map[string]any) string {
	key, _ := field["key"].(string)

	return key
}

func sortPage(field map[string]any) int {
	if page, ok := field["page"].(int); ok {
		return page
	}

	return math.MaxInt32
}

func sortTop(field map[string]any) float64 {
	if box, ok := field["box"].(map[string]float64); ok {
		return box["y"]
	}

	return 2
}

func round4(value float64) float64 {
	return math.Round(value*10000) / 10000
}
