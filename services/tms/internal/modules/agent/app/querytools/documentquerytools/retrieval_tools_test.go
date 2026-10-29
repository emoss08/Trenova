package documentquerytools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/airetrieval"
	"github.com/emoss08/trenova/internal/modules/agent/ports/retrievalports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	searcher := &querytooltest.FakeRetrievalSearcher{
		Documents: &retrievalports.DocumentSearchResult{
			Hits: []retrievalports.DocumentSearchHit{
				{
					Document:      first,
					Page:          2,
					Snippet:       "Lumper fee reimbursed with receipt.",
					ShowsFileName: true,
					Match:         retrievalports.RetrievalMatchBoth,
				},
				{Document: second, Match: retrievalports.RetrievalMatchMeaning},
			},
			Semantics: retrievalports.RetrievalSemantics{Used: true},
		},
	}
	tool := NewSearchDocumentsTool(searcher, &querytooltest.FakePermissions{Allowed: true}, nil)

	result, err := tool.Query(t.Context(), querytooltest.TestParams(map[string]any{
		"query": "lumper fee",
		"limit": 50,
	}))
	require.NoError(t, err)

	outcome := result.(*RetrievalOutcome)
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

	assert.Equal(t, []agentrecord.RecordRef{
		{EntityType: agent.TaintEntityDocument, ID: first.ID.String()},
		{EntityType: agent.TaintEntityDocument, ID: second.ID.String()},
	}, outcome.TaintedRecords())

	require.Len(t, searcher.Requests, 1)
	assert.Equal(t, documentSearchMaxLimit, searcher.Requests[0].Limit)
	assert.NotNil(t, searcher.Requests[0].Access)
	assert.Equal(t, "lumper fee", searcher.Requests[0].Query)
}

func TestSearchDocumentsSaysWhenMeaningWasNotSearched(t *testing.T) {
	t.Parallel()

	searcher := &querytooltest.FakeRetrievalSearcher{
		Documents: &retrievalports.DocumentSearchResult{
			Hits: []retrievalports.DocumentSearchHit{},
			Semantics: retrievalports.RetrievalSemantics{
				Reason: airetrieval.UnavailableReasonNoProvider,
			},
		},
	}
	tool := NewSearchDocumentsTool(searcher, &querytooltest.FakePermissions{Allowed: true}, nil)

	result, err := tool.Query(
		t.Context(),
		querytooltest.TestParams(map[string]any{"query": "bol 88213"}),
	)
	require.NoError(t, err)
	outcome := result.(*RetrievalOutcome)
	assert.Equal(t, "words only: meaning search is not set up", outcome.SearchedBy)
	assert.Zero(t, outcome.Count)
	assert.NotEmpty(t, outcome.Note)
	assert.Empty(t, outcome.TaintedRecords())
	assert.Equal(t, documentSearchDefaultLimit, searcher.Requests[0].Limit)
}

func TestSearchDocumentsNeedsAQuery(t *testing.T) {
	t.Parallel()

	searcher := &querytooltest.FakeRetrievalSearcher{}
	tool := NewSearchDocumentsTool(searcher, &querytooltest.FakePermissions{Allowed: true}, nil)
	_, err := tool.Query(t.Context(), querytooltest.TestParams(map[string]any{"query": "  "}))
	require.ErrorIs(t, err, errSearchQueryRequired)
	assert.Empty(t, searcher.Requests)
}
