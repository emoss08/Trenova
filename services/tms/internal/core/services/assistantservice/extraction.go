package assistantservice

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strings"
	"unicode"

	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/typeutils"
	"go.uber.org/zap"
)

// lowConfidence is where a reading needs a person to look: the same floor
// document intelligence uses to mark a field for review.
const lowConfidence = 0.8

// extractionPage is what the extraction shows of one page: its number and
// where each line of its text sits.
type extractionPage struct {
	Number int
	Lines  []documentcontent.LayoutLine
}

// extraction reads a shipment draft as what document intelligence read off
// the page: the file, the pages, and each field with its confidence and the
// box it was found in. It is nil when the draft holds nothing read yet, which
// leaves the result to be shown as a record card.
func (r *artifactRecorder) extraction(
	observation services.ToolObservation,
) *assistantartifact.Artifact {
	document, ok := toJSONDocument(observation.Data)
	if !ok {
		return nil
	}
	draft := document.fields
	documentID, err := pulid.Parse(typeutils.StringOfTrimmed(draft["documentId"]))
	if err != nil {
		return nil
	}

	fileName := ""
	var pages []extractionPage
	if r.svc != nil && r.svc.documents != nil {
		if doc, docErr := r.svc.documents.GetByID(r.ctx, repositories.GetDocumentByIDRequest{
			ID:         documentID,
			TenantInfo: r.tenant,
		}); docErr == nil && doc != nil {
			fileName = cmp.Or(doc.OriginalName, doc.FileName)
		}
	}
	if r.svc != nil && r.svc.contents != nil {
		content, contentErr := r.svc.contents.GetContent(r.ctx, documentID, r.tenant)
		if contentErr != nil {
			r.logger.Debug("extraction pages could not be read", zap.Error(contentErr))
		} else {
			for _, page := range content.Pages {
				pages = append(pages, extractionPage{
					Number: page.PageNumber,
					Lines:  documentcontent.LinesOf(page.Metadata),
				})
			}
		}
	}

	return extractionArtifact(observation.Call.ID, draft, fileName, pages)
}

// extractionArtifact is the mapping itself, apart from the reads, so it can
// be checked against a draft and pages as they are stored.
func extractionArtifact(
	callID string,
	draft map[string]any,
	fileName string,
	pages []extractionPage,
) *assistantartifact.Artifact {
	fields := extractionFields(draft, pages)
	if len(fields) == 0 {
		return nil
	}

	slices.SortFunc(pages, func(a, b extractionPage) int { return cmp.Compare(a.Number, b.Number) })
	pageCount := len(pages)
	for _, field := range fields {
		if page, _ := field["page"].(int); page > pageCount {
			pageCount = page
		}
	}
	needs := 0
	for _, field := range fields {
		if field["needsLook"] == true {
			needs++
		}
	}

	documentKind := typeutils.StringOfTrimmed(draft["documentKind"])
	kindLabel := stringutils.CapitalizeFirst(strings.ToLower(splitCamel(documentKind)))
	if kindLabel == "" {
		kindLabel = "Document"
	}
	title := kindLabel
	if fileName != "" {
		title += " · " + fileName
	}

	payload := map[string]any{
		"documentId":   typeutils.StringOfTrimmed(draft["documentId"]),
		"fileName":     fileName,
		"documentKind": documentKind,
		"kindLabel":    kindLabel,
		"confidence":   numberOf(draft, "confidence"),
		"reviewStatus": typeutils.StringOfTrimmed(draft["reviewStatus"]),
		"pageCount":    max(pageCount, 1),
		"fields":       fields,
		"needsLook":    needs,
		"missing":      stringsOf(draft["missingFields"]),
		"tool":         toolGetShipmentDraft,
	}
	if attached := typeutils.StringOfTrimmed(draft["attachedShipmentId"]); attached != "" {
		payload["attachedShipmentId"] = attached
	}

	return &assistantartifact.Artifact{
		Kind:             assistantartifact.KindExtraction,
		Status:           assistantartifact.StatusReady,
		Title:            artifactTitle(title),
		Payload:          payload,
		SourceToolCallID: callID,
	}
}

// extractionFields lists the draft's fields in a reading order (the order
// they sit on the page, then by name), followed by its stops.
func extractionFields(draft map[string]any, pages []extractionPage) []map[string]any {
	raw, _ := draft["fields"].(map[string]any)
	out := make([]map[string]any, 0, len(raw)+4)
	for key, value := range raw {
		field, isObject := value.(map[string]any)
		if !isObject {
			continue
		}
		text := displayValue(field["value"])
		if text == "" {
			continue
		}
		label := typeutils.StringOfTrimmed(field["label"])
		if label == "" {
			label = stringutils.CapitalizeFirst(strings.ToLower(splitCamel(key)))
		}
		out = append(out, extractionField(
			key, label, text, numberOf(field, "confidence"),
			int(numberOf(field, "pageNumber")),
			typeutils.BoolOf(field["reviewRequired"]) || typeutils.BoolOf(field["conflict"]),
			pages,
		))
	}

	stops, _ := draft["stops"].([]any)
	for idx, value := range stops {
		stop, isObject := value.(map[string]any)
		if !isObject {
			continue
		}
		place := strings.Join(nonEmpty(
			typeutils.StringOfTrimmed(stop["name"]),
			typeutils.StringOfTrimmed(stop["city"])+cityStateGap(stop)+
				typeutils.StringOfTrimmed(stop["state"]),
			typeutils.StringOfTrimmed(stop["date"]),
		), " · ")
		if place == "" {
			continue
		}
		role := stringutils.CapitalizeFirst(typeutils.StringOfTrimmed(stop["role"]))
		if role == "" {
			role = "Stop"
		}
		out = append(out, extractionField(
			fmt.Sprintf("stop%d", idx+1), role, place, numberOf(stop, "confidence"),
			int(numberOf(stop, "pageNumber")),
			typeutils.BoolOf(stop["reviewRequired"]),
			pages,
		))
	}

	slices.SortStableFunc(out, func(a, b map[string]any) int {
		return cmp.Or(
			cmp.Compare(sortPage(a), sortPage(b)),
			cmp.Compare(sortTop(a), sortTop(b)),
			cmp.Compare(a["key"].(string), b["key"].(string)),
		)
	})

	return out
}

func extractionField(
	key, label, value string,
	confidence float64,
	page int,
	review bool,
	pages []extractionPage,
) map[string]any {
	confidence = math.Round(math.Max(0, math.Min(1, confidence))*100) / 100
	field := map[string]any{
		"key":        key,
		"label":      label,
		"value":      value,
		"confidence": confidence,
		"needsLook":  review || confidence < lowConfidence,
	}
	if found, box := locate(value, page, pages); found > 0 {
		field["page"] = found
		field["box"] = box
	} else if page > 0 {
		field["page"] = page
	}

	return field
}

// locate finds the line a value was read from, on the page the reading names
// or on any page when it names none, and the part of that line the value
// covers. It returns the page and the box as fractions of the page.
func locate(value string, page int, pages []extractionPage) (int, map[string]float64) {
	needle := comparable(value)
	if needle == "" {
		return 0, nil
	}
	// A long value (an address) is looked for by its start, which is the
	// part most likely to sit on one line.
	if runes := []rune(needle); len(runes) > 24 {
		needle = string(runes[:24])
	}
	for _, candidate := range pages {
		if page > 0 && candidate.Number != page {
			continue
		}
		for _, line := range candidate.Lines {
			hay := comparable(line.Text)
			at := strings.Index(hay, needle)
			if at < 0 || hay == "" {
				continue
			}
			share := float64(len([]rune(hay)))
			start := float64(len([]rune(hay[:at]))) / share
			span := float64(len([]rune(needle))) / share

			return candidate.Number, map[string]float64{
				"x": round4(line.X + line.W*start),
				"y": round4(line.Y),
				"w": round4(math.Max(line.W*span, 0.02)),
				"h": round4(math.Max(line.H, 0.01)),
			}
		}
	}

	return 0, nil
}

// comparable is text as a value and a line are compared: lower case, letters
// and digits only, so "$1,359.56" on the page matches "1359.56" in the draft.
func comparable(text string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(text) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}

	return b.String()
}

func displayValue(value any) string {
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

func nonEmpty(parts ...string) []string {
	return slices.DeleteFunc(parts, func(part string) bool { return strings.TrimSpace(part) == "" })
}

func cityStateGap(stop map[string]any) string {
	if typeutils.StringOfTrimmed(stop["city"]) != "" && typeutils.StringOfTrimmed(stop["state"]) != "" {
		return ", "
	}

	return ""
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
