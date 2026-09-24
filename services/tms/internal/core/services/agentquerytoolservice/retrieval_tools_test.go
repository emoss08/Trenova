package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRetrievalSearcher struct {
	documents *serviceports.DocumentSearchResult
	messages  *serviceports.InboundMessageSearchResult
	requests  []serviceports.RetrievalSearchRequest
}

func (f *fakeRetrievalSearcher) SearchDocuments(
	_ context.Context,
	req serviceports.RetrievalSearchRequest,
) (*serviceports.DocumentSearchResult, error) {
	f.requests = append(f.requests, req)

	return f.documents, nil
}

func (f *fakeRetrievalSearcher) SearchInboundMessages(
	_ context.Context,
	req serviceports.RetrievalSearchRequest,
) (*serviceports.InboundMessageSearchResult, error) {
	f.requests = append(f.requests, req)

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
	searcher := &fakeRetrievalSearcher{documents: &serviceports.DocumentSearchResult{
		Hits: []serviceports.DocumentSearchHit{
			{
				Document:      first,
				Page:          2,
				Snippet:       "Lumper fee reimbursed with receipt.",
				ShowsFileName: true,
				Match:         serviceports.RetrievalMatchBoth,
			},
			{Document: second, Match: serviceports.RetrievalMatchMeaning},
		},
		Semantics: serviceports.RetrievalSemantics{Used: true},
	}}
	tool := newSearchDocumentsTool(searcher, &fakePermissions{allowed: true}, nil)

	result, err := tool.Query(t.Context(), testParams(map[string]any{
		"query": "lumper fee",
		"limit": 50,
	}))
	require.NoError(t, err)

	outcome := result.(retrievalOutcome)
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

	searcher := &fakeRetrievalSearcher{documents: &serviceports.DocumentSearchResult{
		Hits: []serviceports.DocumentSearchHit{},
		Semantics: serviceports.RetrievalSemantics{
			Reason: airetrieval.UnavailableReasonNoProvider,
		},
	}}
	tool := newSearchDocumentsTool(searcher, &fakePermissions{allowed: true}, nil)

	result, err := tool.Query(t.Context(), testParams(map[string]any{"query": "bol 88213"}))
	require.NoError(t, err)
	outcome := result.(retrievalOutcome)
	assert.Equal(t, "words only: meaning search is not set up", outcome.SearchedBy)
	assert.Zero(t, outcome.Count)
	assert.NotEmpty(t, outcome.Note)
	assert.Empty(t, outcome.TaintedRecords())
	assert.Equal(t, documentSearchDefaultLimit, searcher.requests[0].Limit)
}

func TestSearchToolsNeedAQuery(t *testing.T) {
	t.Parallel()

	searcher := &fakeRetrievalSearcher{}
	for _, tool := range []serviceports.AgentQueryTool{
		newSearchDocumentsTool(searcher, &fakePermissions{allowed: true}, nil),
		newSearchInboundMessagesTool(searcher, &fakePermissions{allowed: true}),
	} {
		_, err := tool.Query(t.Context(), testParams(map[string]any{"query": "  "}))
		require.ErrorIs(t, err, errSearchQueryRequired, tool.Name())
	}
	assert.Empty(t, searcher.requests)
}

func TestSearchInboundMessagesMarksEveryMessage(t *testing.T) {
	t.Parallel()

	message := &inboundmessage.InboundMessage{
		ID:             pulid.MustNew("imsg_"),
		Status:         inboundmessage.StatusInReview,
		Classification: inboundmessage.ClassificationDetentionDispute,
		ReceivedAt:     1_790_000_000,
	}
	searcher := &fakeRetrievalSearcher{messages: &serviceports.InboundMessageSearchResult{
		Hits: []serviceports.InboundMessageSearchHit{{
			Message: message,
			Subject: "Detention charges on invoice 7781",
			From:    "ap@globex.test",
			Snippet: "We dispute the detention.",
			Match:   serviceports.RetrievalMatchWords,
		}},
	}}
	tool := newSearchInboundMessagesTool(searcher, &fakePermissions{allowed: true})

	result, err := tool.Query(t.Context(), testParams(map[string]any{"query": "detention dispute"}))
	require.NoError(t, err)

	outcome := result.(retrievalOutcome)
	rows := outcome.Items.([]inboundSearchRow)
	require.Len(t, rows, 1)
	assert.Equal(t, message.ID.String(), rows[0].ID)
	assert.Equal(t, "Detention charges on invoice 7781", rows[0].Subject)
	assert.True(t, rows[0].NeedsReview)
	assert.Equal(t, "DetentionDispute", rows[0].Classification)
	assert.Equal(t, "words", rows[0].Match)
	assert.Equal(t, []agent.RecordRef{
		{EntityType: agent.TaintEntityInboundMessage, ID: message.ID.String()},
	}, outcome.TaintedRecords())
}

func TestSearchToolPoliciesReadOutsideTextAlways(t *testing.T) {
	t.Parallel()

	documents := newSearchDocumentsTool(nil, nil, nil).Policy()
	assert.Equal(t, permission.ResourceDocument, documents.Resource)
	assert.Equal(t, agent.ExternalReadAlways, documents.ReadsExternal)
	assert.Equal(t, agent.TaintSourceDocument, documents.Source)

	messages := newSearchInboundMessagesTool(nil, nil).Policy()
	assert.Equal(t, permission.ResourceInboundMessage, messages.Resource)
	assert.Equal(t, agent.ExternalReadAlways, messages.ReadsExternal)
	assert.Equal(t, agent.TaintSourceInboundMessage, messages.Source)
}

func TestRetrievalAccessChecksEachRecordOnceAndHonoursTheCeiling(t *testing.T) {
	t.Parallel()

	permissions := &countingPermissions{fakePermissions: fakePermissions{
		readable: map[string]*serviceports.ResourcePermissionDetail{
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

	requests []*serviceports.PermissionCheckRequest
}

func (c *countingPermissions) Check(
	ctx context.Context,
	req *serviceports.PermissionCheckRequest,
) (*serviceports.PermissionCheckResult, error) {
	c.requests = append(c.requests, req)

	return c.fakePermissions.Check(ctx, req)
}
