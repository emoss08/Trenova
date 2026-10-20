package agentquerytoolservice

import (
	"fmt"
	"maps"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	maxDocumentSummaryRunes = 7_000
	paramOffset             = "offset"
)

type documentSummary struct {
	DocumentID     string                    `json:"documentId"`
	FileName       string                    `json:"fileName"`
	ContentType    string                    `json:"contentType,omitempty"`
	AttachedTo     string                    `json:"attachedTo,omitempty"`
	PageCount      int                       `json:"pageCount"`
	Page           int                       `json:"page,omitempty"`
	Reading        string                    `json:"reading"`
	LooksLike      string                    `json:"looksLike,omitempty"`
	Confidence     float64                   `json:"confidence,omitempty"`
	Language       string                    `json:"language,omitempty"`
	Findings       *documentcontent.Findings `json:"findings,omitempty"`
	Text           string                    `json:"text,omitempty"`
	TextOffset     int                       `json:"textOffset"`
	TextTotal      int                       `json:"textTotal"`
	TextTruncated  bool                      `json:"textTruncated"`
	FailureMessage string                    `json:"failureMessage,omitempty"`
	// StillBeingRead says the reading has not finished: the text is not in,
	// or the findings are the rule-based ones until a model's reading lands.
	StillBeingRead bool   `json:"stillBeingRead,omitempty"`
	Note           string `json:"note,omitempty"`
}

type textWindow struct {
	page   int
	offset int
	limit  int
}

func textWindowOf(params map[string]any) (textWindow, error) {
	window := textWindow{
		page:   optionalInt(params, paramPage, 0),
		offset: max(optionalInt(params, paramOffset, 0), 0),
		limit:  readLimit(params, maxDocumentSummaryRunes, maxDocumentSummaryRunes),
	}
	if window.page < 0 {
		return textWindow{}, fmt.Errorf(
			"page %d is not a page number; pages start at 1", window.page,
		)
	}

	return window, nil
}

func withTextWindow(properties map[string]any) map[string]any {
	merged := make(map[string]any, len(properties)+3)
	maps.Copy(merged, properties)
	merged[paramPage] = map[string]any{
		toolschema.KeyType: toolschema.TypeInteger,
		toolschema.KeyDescription: "Optional: read only this page, for example the page " +
			"document_list found. Leave it out to read the whole text.",
		toolschema.KeyMinimum: 1,
	}
	merged[paramOffset] = map[string]any{
		toolschema.KeyType: toolschema.TypeInteger,
		toolschema.KeyDescription: "Where in the text to start, in characters, for a " +
			"long document read in pieces. Defaults to the beginning.",
		toolschema.KeyMinimum: 0,
	}
	merged[paramLimit] = toolschema.Clamped(map[string]any{
		toolschema.KeyType:    toolschema.TypeInteger,
		toolschema.KeyMinimum: 1,
		toolschema.KeyMaximum: maxDocumentSummaryRunes,
		toolschema.KeyDescription: fmt.Sprintf(
			"How many characters of text to return from the offset: %d unless you ask, "+
				"at most %d.", maxDocumentSummaryRunes, maxDocumentSummaryRunes),
	})

	return merged
}

func summarizeDocument(
	doc *document.Document,
	content *documentcontent.Content,
	window textWindow,
) (*documentSummary, error) {
	summary := &documentSummary{
		DocumentID:  doc.ID.String(),
		FileName:    doc.OriginalName,
		ContentType: doc.FileType,
		Reading:     string(documentcontent.StatusPending),
	}
	if doc.ResourceType != "" && doc.ResourceID != "" {
		summary.AttachedTo = doc.ResourceType + " " + doc.ResourceID
	}
	if content == nil {
		summary.Note = "Nothing has been read from this document yet; ask again shortly."

		return summary, nil
	}

	summary.Reading = string(content.Status)
	summary.PageCount = content.PageCount
	summary.LooksLike = content.DetectedDocumentKind
	summary.Confidence = content.ClassificationConfidence
	summary.Language = content.DetectedLanguage
	summary.FailureMessage = content.FailureMessage
	summary.Findings = content.Findings()

	text := content.PagedText()
	if window.page > 0 {
		pageText, found := pageTextOf(content, window.page)
		if !found {
			return nil, fmt.Errorf(
				"page %d is not in document %s, which has %d pages",
				window.page, doc.ID, max(content.PageCount, len(content.Pages)),
			)
		}
		summary.Page = window.page
		text = pageText
	}

	runes := []rune(text)
	summary.TextTotal = len(runes)
	offset := min(window.offset, len(runes))
	summary.TextOffset = offset
	rest := string(runes[offset:])
	summary.Text = stringutils.TruncateRunes(rest, window.limit)
	summary.TextTruncated = len(runes)-offset > window.limit
	if summary.TextTruncated {
		summary.Note = "The text continues; call again with offset " +
			strconv.Itoa(offset+window.limit) + " for the next part."
	}

	switch content.Status {
	case documentcontent.StatusPending, documentcontent.StatusExtracting:
		summary.Note = "Extraction is still running; ask again shortly."
	case documentcontent.StatusFailed:
		summary.Note = "The document could not be read. Ask the person what it says."
	case documentcontent.StatusExtracted, documentcontent.StatusIndexed:
	}

	return summary, nil
}

func pageTextOf(content *documentcontent.Content, page int) (string, bool) {
	for _, candidate := range content.Pages {
		if candidate.PageNumber == page {
			return strings.TrimSpace(candidate.ExtractedText), true
		}
	}
	if len(content.Pages) == 0 && page == 1 {
		return strings.TrimSpace(content.ContentText), true
	}

	return "", false
}

func isNotFound(err error) bool {
	return errortypes.IsNotFoundError(err) || dberror.IsNotFoundError(err)
}
