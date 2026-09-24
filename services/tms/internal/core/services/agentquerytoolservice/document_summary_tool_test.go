package agentquerytoolservice

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubDocumentRepo struct {
	repositories.DocumentRepository

	doc *document.Document
}

func (s *stubDocumentRepo) GetByID(
	_ context.Context,
	req repositories.GetDocumentByIDRequest,
) (*document.Document, error) {
	if s.doc == nil || s.doc.ID != req.ID {
		return nil, errors.New("not found")
	}

	return s.doc, nil
}

type stubContentReader struct {
	content *documentcontent.Content
	err     error
}

func (s *stubContentReader) GetContent(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) (*documentcontent.Content, error) {
	return s.content, s.err
}

func summaryDocument() *document.Document {
	return &document.Document{
		ID:           pulid.MustNew("doc_"),
		OriginalName: "tender.pdf",
		FileType:     "application/pdf",
		ResourceType: "shipment",
		ResourceID:   pulid.MustNew("shp_").String(),
	}
}

func TestGetDocumentSummary_ReadsTextAndFieldsInWindows(t *testing.T) {
	t.Parallel()

	doc := summaryDocument()
	text := strings.Repeat("a", maxDocumentSummaryRunes) + "TAIL OF THE DOCUMENT"
	tool := newGetDocumentSummaryTool(
		&stubDocumentRepo{doc: doc},
		&stubContentReader{content: &documentcontent.Content{
			Status:                   documentcontent.StatusExtracted,
			PageCount:                3,
			DetectedDocumentKind:     "tender",
			ClassificationConfidence: 0.91,
			ContentText:              text,
			StructuredData:           map[string]any{"bol": "BOL-1"},
		}},
		&fakePermissions{allowed: true},
		nil,
	)

	result, err := tool.Query(
		t.Context(),
		testParams(map[string]any{"documentId": doc.ID.String()}),
	)
	require.NoError(t, err)
	summary := result.(*documentSummary)
	assert.Equal(t, "tender.pdf", summary.FileName)
	assert.Equal(t, "shipment "+doc.ResourceID, summary.AttachedTo)
	assert.Equal(t, "Extracted", summary.Reading)
	assert.Equal(t, 3, summary.PageCount)
	assert.Equal(t, "tender", summary.LooksLike)
	assert.Equal(t, map[string]any{"bol": "BOL-1"}, summary.Fields)
	assert.True(t, summary.TextTruncated)
	assert.NotContains(t, summary.Text, "TAIL")
	assert.Contains(t, summary.Note, "offset 12000")

	result, err = tool.Query(t.Context(), testParams(map[string]any{
		"documentId": doc.ID.String(),
		"offset":     maxDocumentSummaryRunes,
	}))
	require.NoError(t, err)
	summary = result.(*documentSummary)
	assert.Equal(t, "TAIL OF THE DOCUMENT", summary.Text)
	assert.False(t, summary.TextTruncated)
	assert.Equal(t, maxDocumentSummaryRunes, summary.TextOffset)
}

// A document whose reading has not happened is still answered: the model is
// told to come back, rather than handed an error it would report as "the
// document is empty".
func TestGetDocumentSummary_SaysWhenNothingHasBeenReadYet(t *testing.T) {
	t.Parallel()

	doc := summaryDocument()
	tool := newGetDocumentSummaryTool(
		&stubDocumentRepo{doc: doc},
		&stubContentReader{err: errortypes.NewNotFoundError("Document content not found")},
		&fakePermissions{allowed: true},
		nil,
	)

	result, err := tool.Query(
		t.Context(),
		testParams(map[string]any{"documentId": doc.ID.String()}),
	)
	require.NoError(t, err)
	summary := result.(*documentSummary)
	assert.Equal(t, "Pending", summary.Reading)
	assert.Contains(t, summary.Note, "ask again shortly")

	pending := newGetDocumentSummaryTool(
		&stubDocumentRepo{doc: doc},
		&stubContentReader{content: &documentcontent.Content{
			Status: documentcontent.StatusExtracting,
		}},
		&fakePermissions{allowed: true},
		nil,
	)
	result, err = pending.Query(
		t.Context(),
		testParams(map[string]any{"documentId": doc.ID.String()}),
	)
	require.NoError(t, err)
	assert.Contains(t, result.(*documentSummary).Note, "still running")
}

func TestGetDocumentSummary_IsGatedOnDocumentsAndRefusesANonID(t *testing.T) {
	t.Parallel()

	tool := newGetDocumentSummaryTool(
		&stubDocumentRepo{},
		&stubContentReader{},
		&fakePermissions{allowed: true},
		nil,
	)
	assert.Equal(t, permission.ResourceDocument, tool.Policy().Resource)
	assert.Equal(t, "get_document_summary", tool.Name())

	_, err := tool.Query(t.Context(), testParams(map[string]any{"documentId": "tender.pdf"}))
	require.Error(t, err)
}

func TestGetDocumentSummary_SurfacesADatabaseError(t *testing.T) {
	t.Parallel()

	doc := summaryDocument()
	tool := newGetDocumentSummaryTool(
		&stubDocumentRepo{doc: doc},
		&stubContentReader{err: errors.New("connection reset by peer")},
		&fakePermissions{allowed: true},
		nil,
	)

	_, err := tool.Query(
		t.Context(),
		testParams(map[string]any{"documentId": doc.ID.String()}),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection reset by peer")
}

func TestGetDocumentSummary_RefusesADocumentOnARecordTheCallerCannotRead(t *testing.T) {
	t.Parallel()

	doc := summaryDocument()
	doc.ResourceType = "worker"
	doc.ResourceID = pulid.MustNew("wrk_").String()
	permissions := &fakePermissions{readable: map[string]*serviceports.ResourcePermissionDetail{
		permission.ResourceDocument.String(): {
			Resource:   permission.ResourceDocument.String(),
			Operations: []permission.Operation{permission.OpRead},
		},
	}}
	tool := newGetDocumentSummaryTool(
		&stubDocumentRepo{doc: doc},
		&stubContentReader{content: &documentcontent.Content{
			Status:      documentcontent.StatusIndexed,
			ContentText: "Medical examiner's certificate",
		}},
		permissions,
		nil,
	)

	_, err := tool.Query(
		t.Context(),
		testParams(map[string]any{"documentId": doc.ID.String()}),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "worker")
	require.NotNil(t, permissions.captured)
	assert.Equal(t, permission.ResourceWorker.String(), permissions.captured.Resource)
	require.NotNil(t, permissions.captured.ResourceID)
	assert.Equal(t, doc.ResourceID, permissions.captured.ResourceID.String())
}

func TestGetDocumentSummary_ReadsOnePage(t *testing.T) {
	t.Parallel()

	doc := summaryDocument()
	tool := newGetDocumentSummaryTool(
		&stubDocumentRepo{doc: doc},
		&stubContentReader{content: &documentcontent.Content{
			Status:      documentcontent.StatusIndexed,
			PageCount:   2,
			ContentText: "first page text second page text",
			Pages: []*documentcontent.Page{
				{PageNumber: 1, ExtractedText: "first page text"},
				{PageNumber: 2, ExtractedText: "second page text"},
			},
		}},
		&fakePermissions{allowed: true},
		nil,
	)

	result, err := tool.Query(t.Context(), testParams(map[string]any{
		"documentId": doc.ID.String(),
		"page":       2,
	}))
	require.NoError(t, err)
	summary := result.(*documentSummary)
	assert.Equal(t, 2, summary.Page)
	assert.Equal(t, "second page text", summary.Text)

	_, err = tool.Query(t.Context(), testParams(map[string]any{
		"documentId": doc.ID.String(),
		"page":       7,
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "page 7")
}
