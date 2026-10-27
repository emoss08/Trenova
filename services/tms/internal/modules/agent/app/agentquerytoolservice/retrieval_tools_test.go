package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/airetrieval"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRetrievalSearcher struct {
	documents *agentports.DocumentSearchResult
	messages  *agentports.InboundMessageSearchResult
	requests  []agentports.RetrievalSearchRequest
}

func (f *fakeRetrievalSearcher) SearchDocuments(
	_ context.Context,
	req *agentports.RetrievalSearchRequest,
) (*agentports.DocumentSearchResult, error) {
	f.requests = append(f.requests, *req)

	return f.documents, nil
}

func (f *fakeRetrievalSearcher) SearchInboundMessages(
	_ context.Context,
	req *agentports.RetrievalSearchRequest,
) (*agentports.InboundMessageSearchResult, error) {
	f.requests = append(f.requests, *req)

	return f.messages, nil
}

func TestSearchDocumentsReturnsRowsAndMarksEveryDocument(t *testing.T) {
	t.Parallel()

	first := &document.Document{
		ID:           pulid.MustNew("doc_"),
		OriginalName: "acme-ratecon.pdf",
		DetectedKind: "RateConfirmation",
		ResourceType: "shipment",
		ResourceID:   "shp_1",
	}
	second := &document.Document{ID: pulid.MustNew("doc_"), OriginalName: "private.pdf"}
	searcher := &fakeRetrievalSearcher{documents: &agentports.DocumentSearchResult{
		Hits: []agentports.DocumentSearchHit{
			{
				Document:      first,
				Page:          2,
				Snippet:       "Lumper fee reimbursed with receipt.",
				ShowsFileName: true,
				Match:         agentports.RetrievalMatchBoth,
			},
			{Document: second, Match: agentports.RetrievalMatchMeaning},
		},
		Semantics: agentports.RetrievalSemantics{Used: true},
	}}
	tool := newSearchDocumentsTool(searcher, &fakePermissions{allowed: true}, nil)

	result, err := tool.Query(t.Context(), testParams(map[string]any{
		"query": "lumper fee",
		"limit": 50,
	}))
	require.NoError(t, err)

	outcome := result.(*retrievalOutcome)
	assert.Equal(t, "words and meaning", outcome.SearchedBy)
	rows := outcome.Items.([]documentSearchRow)
	require.Len(t, rows, 2)
	assert.Equal(t, documentSearchRow{
		DocumentID: first.ID.String(),
		FileName:   "acme-ratecon.pdf",
		LooksLike:  "RateConfirmation",
		Page:       2,
		Snippet:    "Lumper fee reimbursed with receipt.",
		AttachedTo: "shipment shp_1",
		Match:      "both",
	}, rows[0])
	assert.Empty(t, rows[1].FileName, "a file name above the caller's ceiling is left out")
	assert.Equal(t, "meaning", rows[1].Match)

	assert.Equal(t, []agent.RecordRef{
		{EntityType: agent.TaintEntityDocument, ID: first.ID.String()},
		{EntityType: agent.TaintEntityDocument, ID: second.ID.String()},
	}, outcome.TaintedRecords())

	require.Len(t, searcher.requests, 1)
	assert.Equal(t, documentSearchMaxLimit, searcher.requests[0].Limit)
	assert.NotNil(t, searcher.requests[0].Access)
	assert.Equal(t, "lumper fee", searcher.requests[0].Query)
}

func TestSearchDocumentsSaysWhenMeaningWasNotSearched(t *testing.T) {
	t.Parallel()

	searcher := &fakeRetrievalSearcher{documents: &agentports.DocumentSearchResult{
		Hits: []agentports.DocumentSearchHit{},
		Semantics: agentports.RetrievalSemantics{
			Reason: airetrieval.UnavailableReasonNoProvider,
		},
	}}
	tool := newSearchDocumentsTool(searcher, &fakePermissions{allowed: true}, nil)

	result, err := tool.Query(t.Context(), testParams(map[string]any{"query": "bol 88213"}))
	require.NoError(t, err)
	outcome := result.(*retrievalOutcome)
	assert.Equal(t, "words only: meaning search is not set up", outcome.SearchedBy)
	assert.Zero(t, outcome.Count)
	assert.NotEmpty(t, outcome.Note)
	assert.Empty(t, outcome.TaintedRecords())
	assert.Equal(t, documentSearchDefaultLimit, searcher.requests[0].Limit)
}

func TestSearchDocumentsNeedsAQuery(t *testing.T) {
	t.Parallel()

	searcher := &fakeRetrievalSearcher{}
	tool := newSearchDocumentsTool(searcher, &fakePermissions{allowed: true}, nil)
	_, err := tool.Query(t.Context(), testParams(map[string]any{"query": "  "}))
	require.ErrorIs(t, err, errSearchQueryRequired)
	assert.Empty(t, searcher.requests)
}

func TestSearchToolPoliciesReadOutsideTextAlways(t *testing.T) {
	t.Parallel()

	documents := newSearchDocumentsTool(nil, nil, nil).Policy()
	assert.Equal(t, permission.ResourceDocument, documents.Resource)
	assert.Equal(t, agent.ExternalReadAlways, documents.ReadsExternal)
	assert.Equal(t, agent.TaintSourceDocument, documents.Source)

	inbox, err := provideListInboundMessagesTool(&keyedReader{}, nil, nil)
	require.NoError(t, err)
	messages := inbox.Policy()
	assert.Equal(t, permission.ResourceInboundMessage, messages.Resource)
	assert.Equal(t, agent.ExternalReadAlways, messages.ReadsExternal)
	assert.Equal(t, agent.TaintSourceInboundMessage, messages.Source)
}

func TestRetrievalAccessChecksEachRecordOnceAndHonoursTheCeiling(t *testing.T) {
	t.Parallel()

	permissions := &countingPermissions{fakePermissions: fakePermissions{
		readable: map[string]*permissionports.ResourcePermissionDetail{
			permission.ResourceDocument.String(): {
				Resource:       permission.ResourceDocument.String(),
				Operations:     []permission.Operation{permission.OpRead},
				MaxSensitivity: permission.SensitivityInternal,
			},
			permission.ResourceShipment.String(): {
				Resource:       permission.ResourceShipment.String(),
				Operations:     []permission.Operation{permission.OpRead},
				MaxSensitivity: permission.SensitivityInternal,
			},
			permission.ResourceWorker.String(): {
				Resource:       permission.ResourceWorker.String(),
				Operations:     []permission.Operation{permission.OpRead},
				MaxSensitivity: permission.SensitivityInternal,
			},
		},
	}}
	access := newFieldAccess(permissions).forRetrieval(testParams(nil))
	shipment := pulid.MustNew("shp_").String()

	assert.True(t, access.MayReadRecord(t.Context(), permission.ResourceShipment, shipment))
	assert.True(t, access.MayReadRecord(t.Context(), permission.ResourceShipment, shipment))
	assert.False(t, access.MayReadRecord(t.Context(), permission.ResourceCustomer, "cus_1"))
	require.Len(t, permissions.requests, 3, "one resource check and one record check, remembered")
	assert.Nil(t, permissions.requests[0].ResourceID)
	require.NotNil(t, permissions.requests[1].ResourceID)
	assert.Equal(t, shipment, permissions.requests[1].ResourceID.String())
	assert.Equal(t, permission.ResourceCustomer.String(), permissions.requests[2].Resource)

	assert.True(t, access.ShowsRecordText(t.Context(), permission.ResourceShipment))
	assert.False(t, access.ShowsRecordText(t.Context(), permission.ResourceWorker),
		"a worker's records are Restricted and the ceiling is Internal")
	assert.False(t, access.ShowsRecordText(t.Context(), permission.ResourceUser))
	assert.True(t, access.ShowsField(t.Context(), permission.ResourceDocument, "contentText"))
}

type countingPermissions struct {
	fakePermissions

	requests []*permissionports.PermissionCheckRequest
}

func (c *countingPermissions) Check(
	ctx context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	c.requests = append(c.requests, req)

	return c.fakePermissions.Check(ctx, req)
}
