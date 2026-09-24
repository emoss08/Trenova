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
		ResourceType: "assistant_thread",
		ResourceID:   "athr_1",
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
	)

	result, err := tool.Query(
		t.Context(),
		testParams(map[string]any{"documentId": doc.ID.String()}),
	)
	require.NoError(t, err)
	summary := result.(*documentSummary)
	assert.Equal(t, "tender.pdf", summary.FileName)
	assert.Equal(t, "assistant_thread athr_1", summary.AttachedTo)
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
		&stubContentReader{err: errors.New("no row")},
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

	tool := newGetDocumentSummaryTool(&stubDocumentRepo{}, &stubContentReader{})
	assert.Equal(t, permission.ResourceDocument, tool.Policy().Resource)
	assert.Equal(t, "get_document_summary", tool.Name())

	_, err := tool.Query(t.Context(), testParams(map[string]any{"documentId": "tender.pdf"}))
	require.Error(t, err)
}
