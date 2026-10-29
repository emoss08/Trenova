package documentquerytools

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubDocumentRepo struct {
	documentports.DocumentRepository

	doc *document.Document
}

func (s *stubDocumentRepo) GetByID(
	_ context.Context,
	req documentports.GetDocumentByIDRequest,
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
	tool := NewGetDocumentSummaryTool(
		&stubDocumentRepo{doc: doc},
		&stubContentReader{content: &documentcontent.Content{
			Status:                   documentcontent.StatusExtracted,
			PageCount:                3,
			DetectedDocumentKind:     "tender",
			ClassificationConfidence: 0.91,
			ContentText:              text,
			StructuredData: map[string]any{
				"intelligence": map[string]any{
					"kind": "RateConfirmation",
					"fields": map[string]any{
						"bol": map[string]any{
							"label":      "BOL",
							"value":      "BOL-1",
							"confidence": 0.9,
						},
						"pending": map[string]any{"label": "Pending", "value": ""},
					},
					"stops": []any{map[string]any{
						"sequence": 1, "role": "pickup", "name": "Los Angeles Terminal",
						"city": "Los Angeles", "state": "CA", "evidenceExcerpt": "SHIPPER#1",
					}},
					"missingFields": []any{"Rate"},
				},
				"aiDiagnostics": map[string]any{
					"fallbackAnalysis": map[string]any{"rawExcerpt": "x"},
				},
			},
		}},
		&querytooltest.FakePermissions{Allowed: true},
		nil,
	)

	result, err := tool.Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{"documentId": doc.ID.String()}),
	)
	require.NoError(t, err)
	summary := result.(*documentSummary)
	assert.Equal(t, "tender.pdf", summary.FileName)
	assert.Equal(t, "shipment "+doc.ResourceID, summary.AttachedTo)
	assert.Equal(t, "Extracted", summary.Reading)
	assert.Equal(t, 3, summary.PageCount)
	assert.Equal(t, "tender", summary.LooksLike)
	require.NotNil(t, summary.Findings)
	assert.Equal(t, []documentcontent.FoundField{
		{Key: "bol", Label: "BOL", Value: "BOL-1", Confidence: 0.9},
	}, summary.Findings.Fields)
	require.Len(t, summary.Findings.Stops, 1)
	assert.Equal(t, "Los Angeles Terminal", summary.Findings.Stops[0].Name)
	assert.Equal(t, []string{"Rate"}, summary.Findings.MissingFields)
	assert.True(t, summary.TextTruncated)
	assert.NotContains(t, summary.Text, "TAIL")
	assert.Contains(t, summary.Note, "offset "+strconv.Itoa(maxDocumentSummaryRunes))

	result, err = tool.Query(t.Context(), querytooltest.TestParams(map[string]any{
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
	tool := NewGetDocumentSummaryTool(
		&stubDocumentRepo{doc: doc},
		&stubContentReader{err: errortypes.NewNotFoundError("Document content not found")},
		&querytooltest.FakePermissions{Allowed: true},
		nil,
	)

	result, err := tool.Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{"documentId": doc.ID.String()}),
	)
	require.NoError(t, err)
	summary := result.(*documentSummary)
	assert.Equal(t, "Pending", summary.Reading)
	assert.Contains(t, summary.Note, "ask again shortly")

	pending := NewGetDocumentSummaryTool(
		&stubDocumentRepo{doc: doc},
		&stubContentReader{content: &documentcontent.Content{
			Status: documentcontent.StatusExtracting,
		}},
		&querytooltest.FakePermissions{Allowed: true},
		nil,
	)
	result, err = pending.Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{"documentId": doc.ID.String()}),
	)
	require.NoError(t, err)
	assert.Contains(t, result.(*documentSummary).Note, "still running")
}

func TestGetDocumentSummary_IsGatedOnDocumentsAndRefusesANonID(t *testing.T) {
	t.Parallel()

	tool := NewGetDocumentSummaryTool(
		&stubDocumentRepo{},
		&stubContentReader{},
		&querytooltest.FakePermissions{Allowed: true},
		nil,
	)
	assert.Equal(t, permission.ResourceDocument, tool.Policy().Resource)
	assert.Equal(t, "document_get_summary", tool.Name())

	_, err := tool.Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{"documentId": "tender.pdf"}),
	)
	require.Error(t, err)
}

func TestGetDocumentSummary_SurfacesADatabaseError(t *testing.T) {
	t.Parallel()

	doc := summaryDocument()
	tool := NewGetDocumentSummaryTool(
		&stubDocumentRepo{doc: doc},
		&stubContentReader{err: errors.New("connection reset by peer")},
		&querytooltest.FakePermissions{Allowed: true},
		nil,
	)

	_, err := tool.Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{"documentId": doc.ID.String()}),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection reset by peer")
}

func TestGetDocumentSummary_RefusesADocumentOnARecordTheCallerCannotRead(t *testing.T) {
	t.Parallel()

	doc := summaryDocument()
	doc.ResourceType = "worker"
	doc.ResourceID = pulid.MustNew("wrk_").String()
	permissions := &querytooltest.FakePermissions{
		Readable: map[string]*permissionports.ResourcePermissionDetail{
			permission.ResourceDocument.String(): {
				Resource:   permission.ResourceDocument.String(),
				Operations: []permission.Operation{permission.OpRead},
			},
		},
	}
	tool := NewGetDocumentSummaryTool(
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
		querytooltest.TestParams(map[string]any{"documentId": doc.ID.String()}),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "worker")
	require.NotNil(t, permissions.Captured)
	assert.Equal(t, permission.ResourceWorker.String(), permissions.Captured.Resource)
	require.NotNil(t, permissions.Captured.ResourceID)
	assert.Equal(t, doc.ResourceID, permissions.Captured.ResourceID.String())
}

func TestGetDocumentSummary_ReadsOnePage(t *testing.T) {
	t.Parallel()

	doc := summaryDocument()
	tool := NewGetDocumentSummaryTool(
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
		&querytooltest.FakePermissions{Allowed: true},
		nil,
	)

	result, err := tool.Query(t.Context(), querytooltest.TestParams(map[string]any{
		"documentId": doc.ID.String(),
		"page":       2,
	}))
	require.NoError(t, err)
	summary := result.(*documentSummary)
	assert.Equal(t, 2, summary.Page)
	assert.Equal(t, "second page text", summary.Text)

	_, err = tool.Query(t.Context(), querytooltest.TestParams(map[string]any{
		"documentId": doc.ID.String(),
		"page":       7,
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "page 7")
}
