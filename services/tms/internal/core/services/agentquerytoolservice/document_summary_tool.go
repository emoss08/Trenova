package agentquerytoolservice

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
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
	access    fieldAccess
}

func newGetDocumentSummaryTool(
	documents repositories.DocumentRepository,
	contents contentReader,
	permissions serviceports.PermissionEngine,
	threads repositories.ThreadOwnerRepository,
) serviceports.AgentQueryTool {
	return &getDocumentSummaryTool{
		documents: documents,
		contents:  contents,
		access:    newFieldAccess(permissions).withThreads(threads),
	}
}

func (t *getDocumentSummaryTool) Name() string { return "get_document_summary" }

func (t *getDocumentSummaryTool) Description() string {
	return "Read a document by id: what kind of document it looks like, how many pages, " +
		"the text extracted from it, and the structured fields document intelligence " +
		"found. Use it for a file the person attached to their message, for a " +
		"document a record names, or for a page search_documents pointed at. For a " +
		"rate confirmation or tender, follow with " +
		"get_shipment_draft for the shipment it describes. A document whose reading " +
		"is still pending has no text yet; say so and ask again shortly."
}

func (t *getDocumentSummaryTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"documentId": map[string]any{
				"type": "string",
				"description": "The document's id, from the attachments on the message, a " +
					"documentId in get_inbound_message or get_worker_credential, or this " +
					"run's subject.",
			},
			"page": map[string]any{
				"type": "integer",
				"description": "Optional: read only this page, for example the page " +
					"search_documents found. Leave it out to read the whole text.",
				"minimum": 1,
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

func (t *getDocumentSummaryTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceDocument,
		reads:    agent.ExternalReadAlways,
		source:   agent.TaintSourceDocument,
		rationale: "Reads text extracted from a document someone outside sent; nothing changes " +
			"and nothing is sent.",
	})
}

// documentSummary is what the model reads. The text is a window, and the
// window says where it ends, so a model that needs the rest asks for it.
type documentSummary struct {
	DocumentID     string         `json:"documentId"`
	FileName       string         `json:"fileName"`
	ContentType    string         `json:"contentType,omitempty"`
	AttachedTo     string         `json:"attachedTo,omitempty"`
	PageCount      int            `json:"pageCount"`
	Page           int            `json:"page,omitempty"`
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
	offset := max(optionalInt(params.Params, "offset", 0), 0)
	page := optionalInt(params.Params, "page", 0)
	if page < 0 {
		return nil, fmt.Errorf("page %d is not a page number; pages start at 1", page)
	}

	tenant := tenantOf(params)
	doc, err := t.documents.GetByID(ctx, repositories.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, err
	}

	readable, err := t.access.readableDocuments(ctx, params, []*document.Document{doc},
		func(resource permission.Resource, recordID string) bool {
			return t.access.mayReadRecord(ctx, params, resource, recordID)
		})
	if err != nil {
		return nil, err
	}
	if !readable[doc.ID] {
		return nil, fmt.Errorf(
			"document %s is attached to a %s you may not read",
			doc.ID, stringutils.HumanizeSnakeCase(doc.OwnerResource().String()),
		)
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
	switch {
	case err != nil && !isNotFound(err):
		return nil, fmt.Errorf("read what was extracted from document %s: %w", doc.ID, err)
	case err != nil || content == nil:
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
	if page > 0 {
		pageText, found := pageTextOf(content, page)
		if !found {
			return nil, fmt.Errorf(
				"page %d is not in document %s, which has %d pages",
				page, doc.ID, max(content.PageCount, len(content.Pages)),
			)
		}
		summary.Page = page
		text = pageText
	}

	runes := []rune(text)
	summary.TextTotal = len(runes)
	offset = min(offset, len(runes))
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
