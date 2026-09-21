package agentquerytoolservice

import (
	"context"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

// maxDocumentSummaryRunes bounds the text one call returns. A long contract
// is read in the pieces the model asks for; the whole of it in one result
// would crowd out the question.
const maxDocumentSummaryRunes = 12_000

// contentReader is the one question the summary tool asks of document
// intelligence: what has it made of this file.
type contentReader interface {
	GetContent(
		ctx context.Context,
		documentID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*documentcontent.Content, error)
}

// getDocumentSummaryTool reads a file the person attached, or any document
// they may read: its text, what it looks like, and the fields document
// intelligence pulled from it.
type getDocumentSummaryTool struct {
	documents repositories.DocumentRepository
	contents  contentReader
}

func newGetDocumentSummaryTool(
	documents repositories.DocumentRepository,
	contents contentReader,
) serviceports.AgentQueryTool {
	return &getDocumentSummaryTool{documents: documents, contents: contents}
}

func (t *getDocumentSummaryTool) Name() string { return "get_document_summary" }

func (t *getDocumentSummaryTool) Description() string {
	return "Read a document by id: what kind of document it looks like, how many pages, " +
		"the text extracted from it, and the structured fields document intelligence " +
		"found. Use it for a file the person attached to their message, or for a " +
		"document a record names. For a rate confirmation or tender, follow with " +
		"get_shipment_draft for the shipment it describes. A document whose reading " +
		"is still pending has no text yet; say so and ask again shortly."
}

func (t *getDocumentSummaryTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"documentId": map[string]any{
				"type":        "string",
				"description": "The document's id, from the attachments on the message or a record.",
			},
			"offset": map[string]any{
				"type": "integer",
				"description": "Where in the text to start, in characters, for a long document " +
					"read in pieces. Defaults to the beginning.",
				"minimum": 0,
			},
		},
		"required":             []string{"documentId"},
		"additionalProperties": false,
	}
}

func (t *getDocumentSummaryTool) PermissionResource() permission.Resource {
	return permission.ResourceDocument
}

// documentSummary is what the model reads. The text is a window, and the
// window says where it ends, so a model that needs the rest asks for it.
type documentSummary struct {
	DocumentID     string         `json:"documentId"`
	FileName       string         `json:"fileName"`
	ContentType    string         `json:"contentType,omitempty"`
	AttachedTo     string         `json:"attachedTo,omitempty"`
	PageCount      int            `json:"pageCount"`
	Reading        string         `json:"reading"`
	LooksLike      string         `json:"looksLike,omitempty"`
	Confidence     float64        `json:"confidence,omitempty"`
	Language       string         `json:"language,omitempty"`
	Fields         map[string]any `json:"fields,omitempty"`
	Text           string         `json:"text,omitempty"`
	TextOffset     int            `json:"textOffset"`
	TextTotal      int            `json:"textTotal"`
	TextTruncated  bool           `json:"textTruncated"`
	FailureMessage string         `json:"failureMessage,omitempty"`
	Note           string         `json:"note,omitempty"`
}

func (t *getDocumentSummaryTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	documentID, err := requirePulid(params.Params, "documentId")
	if err != nil {
		return nil, err
	}
	offset := optionalInt(params.Params, "offset", 0)
	if offset < 0 {
		offset = 0
	}

	tenant := tenantOf(params)
	doc, err := t.documents.GetByID(ctx, repositories.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, err
	}

	summary := &documentSummary{
		DocumentID:  doc.ID.String(),
		FileName:    doc.OriginalName,
		ContentType: doc.FileType,
		Reading:     string(documentcontent.StatusPending),
	}
	if doc.ResourceType != "" && doc.ResourceID != "" {
		summary.AttachedTo = doc.ResourceType + " " + doc.ResourceID
	}

	content, err := t.contents.GetContent(ctx, documentID, tenant)
	if err != nil || content == nil {
		summary.Note = "Nothing has been read from this document yet; ask again shortly."

		return summary, nil
	}

	summary.Reading = string(content.Status)
	summary.PageCount = content.PageCount
	summary.LooksLike = content.DetectedDocumentKind
	summary.Confidence = content.ClassificationConfidence
	summary.Language = content.DetectedLanguage
	summary.FailureMessage = content.FailureMessage
	if len(content.StructuredData) > 0 {
		summary.Fields = content.StructuredData
	}

	text := strings.TrimSpace(content.ContentText)
	runes := []rune(text)
	summary.TextTotal = len(runes)
	if offset > len(runes) {
		offset = len(runes)
	}
	summary.TextOffset = offset
	window := string(runes[offset:])
	summary.Text = stringutils.TruncateRunes(window, maxDocumentSummaryRunes)
	summary.TextTruncated = len([]rune(window)) > maxDocumentSummaryRunes
	if summary.TextTruncated {
		summary.Note = "The text continues; call again with offset " +
			strconv.Itoa(offset+maxDocumentSummaryRunes) + " for the next part."
	}

	switch content.Status {
	case documentcontent.StatusPending, documentcontent.StatusExtracting:
		summary.Note = "Extraction is still running; ask again shortly."
	case documentcontent.StatusFailed:
		summary.Note = "The document could not be read. Ask the person what it says."
	}

	return summary, nil
}
