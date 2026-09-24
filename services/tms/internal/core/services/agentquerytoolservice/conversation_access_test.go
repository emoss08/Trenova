package agentquerytoolservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeThreadOwners struct {
	owners   map[pulid.ID]pulid.ID
	err      error
	requests []repositories.ThreadOwnersRequest
}

func (f *fakeThreadOwners) ThreadOwners(
	_ context.Context,
	req repositories.ThreadOwnersRequest,
) (map[pulid.ID]pulid.ID, error) {
	f.requests = append(f.requests, req)
	if f.err != nil {
		return nil, f.err
	}

	owners := make(map[pulid.ID]pulid.ID, len(req.ThreadIDs))
	for _, id := range req.ThreadIDs {
		if owner, ok := f.owners[id]; ok {
			owners[id] = owner
		}
	}

	return owners, nil
}

func conversationDocument(threadID pulid.ID) *document.Document {
	return &document.Document{
		ID:           pulid.MustNew("doc_"),
		OriginalName: "rate-confirmation.pdf",
		FileType:     "application/pdf",
		ResourceType: document.ConversationResourceType,
		ResourceID:   threadID.String(),
	}
}

func personParams(principal serviceports.PrincipalType) serviceports.QueryToolParams {
	params := testParams(map[string]any{})
	params.Actor.PrincipalType = principal

	return params
}

func conversationSummaryTool(
	doc *document.Document,
	threads repositories.ThreadOwnerRepository,
) serviceports.AgentQueryTool {
	return newGetDocumentSummaryTool(
		&stubDocumentRepo{doc: doc},
		&stubContentReader{content: &documentcontent.Content{
			Status:      documentcontent.StatusIndexed,
			ContentText: "Rate confirmation for load 4471",
		}},
		&fakePermissions{allowed: true},
		threads,
	)
}

func TestGetDocumentSummary_ConversationFileIsReadOnlyByItsOwner(t *testing.T) {
	t.Parallel()

	threadID := pulid.MustNew("athr_")
	doc := conversationDocument(threadID)

	owner := personParams(serviceports.PrincipalTypeUser)
	owner.Params = map[string]any{"documentId": doc.ID.String()}
	threads := &fakeThreadOwners{owners: map[pulid.ID]pulid.ID{threadID: owner.Actor.UserID}}

	result, err := conversationSummaryTool(doc, threads).Query(t.Context(), owner)
	require.NoError(t, err)
	assert.Equal(t, "rate-confirmation.pdf", result.(*documentSummary).FileName)
	require.Len(t, threads.requests, 1)
	assert.Equal(t, []pulid.ID{threadID}, threads.requests[0].ThreadIDs)
	assert.Equal(t, owner.OrganizationID, threads.requests[0].TenantInfo.OrgID)
	assert.Equal(t, owner.BusinessUnitID, threads.requests[0].TenantInfo.BuID)

	colleague := personParams(serviceports.PrincipalTypeUser)
	colleague.OrganizationID = owner.OrganizationID
	colleague.BusinessUnitID = owner.BusinessUnitID
	colleague.Params = map[string]any{"documentId": doc.ID.String()}

	_, err = conversationSummaryTool(doc, threads).Query(t.Context(), colleague)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "may not read")
}

func TestGetDocumentSummary_ConversationFileIsNeverReadWithoutAPerson(t *testing.T) {
	t.Parallel()

	for _, principal := range []serviceports.PrincipalType{
		serviceports.PrincipalTypeAgent,
		serviceports.PrincipalTypeAPIKey,
		"",
	} {
		t.Run(string(principal), func(t *testing.T) {
			t.Parallel()

			threadID := pulid.MustNew("athr_")
			doc := conversationDocument(threadID)
			params := personParams(principal)
			params.Params = map[string]any{"documentId": doc.ID.String()}
			threads := &fakeThreadOwners{
				owners: map[pulid.ID]pulid.ID{threadID: params.Actor.UserID},
			}

			_, err := conversationSummaryTool(doc, threads).Query(t.Context(), params)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "may not read")
			assert.Empty(t, threads.requests)
		})
	}
}

func TestGetDocumentSummary_ConversationOwnerLookupFailureRefuses(t *testing.T) {
	t.Parallel()

	doc := conversationDocument(pulid.MustNew("athr_"))
	params := personParams(serviceports.PrincipalTypeUser)
	params.Params = map[string]any{"documentId": doc.ID.String()}

	_, err := conversationSummaryTool(doc, &fakeThreadOwners{err: errors.New("pool closed")}).
		Query(t.Context(), params)
	require.ErrorContains(t, err, "pool closed")
}

func TestGetDocumentSummary_ConversationFileWithoutOwnerLookupRefuses(t *testing.T) {
	t.Parallel()

	doc := conversationDocument(pulid.MustNew("athr_"))
	params := personParams(serviceports.PrincipalTypeUser)
	params.Params = map[string]any{"documentId": doc.ID.String()}

	_, err := conversationSummaryTool(doc, nil).Query(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "may not read")
}

func TestRetrievalAccess_ChecksConversationOwnersInOneBatch(t *testing.T) {
	t.Parallel()

	params := personParams(serviceports.PrincipalTypeUser)
	mineID := pulid.MustNew("athr_")
	theirsID := pulid.MustNew("athr_")
	mine := conversationDocument(mineID)
	mineAgain := conversationDocument(mineID)
	theirs := conversationDocument(theirsID)
	orphanID := pulid.MustNew("athr_")
	orphan := conversationDocument(orphanID)
	malformed := conversationDocument(mineID)
	malformed.ResourceID = "not-a-thread"
	shipment := summaryDocument()
	threads := &fakeThreadOwners{owners: map[pulid.ID]pulid.ID{
		mineID:   params.Actor.UserID,
		theirsID: pulid.MustNew("usr_"),
	}}
	access := newFieldAccess(&fakePermissions{allowed: true}).
		withThreads(threads).
		forRetrieval(params)

	readable, err := access.ReadableDocuments(t.Context(), []*document.Document{
		mine, mineAgain, theirs, orphan, malformed, shipment, nil,
	})
	require.NoError(t, err)
	assert.Equal(t, map[pulid.ID]bool{
		mine.ID:      true,
		mineAgain.ID: true,
		shipment.ID:  true,
	}, readable)
	require.Len(t, threads.requests, 1)
	assert.ElementsMatch(t, []pulid.ID{mineID, theirsID, orphanID},
		threads.requests[0].ThreadIDs)
}

func TestRetrievalAccess_BackgroundRunSeesNoConversationFiles(t *testing.T) {
	t.Parallel()

	params := personParams(serviceports.PrincipalTypeAgent)
	threadID := pulid.MustNew("athr_")
	conversation := conversationDocument(threadID)
	shipment := summaryDocument()
	threads := &fakeThreadOwners{owners: map[pulid.ID]pulid.ID{threadID: params.Actor.UserID}}
	access := newFieldAccess(&fakePermissions{allowed: true}).
		withThreads(threads).
		forRetrieval(params)

	readable, err := access.ReadableDocuments(t.Context(), []*document.Document{
		conversation, shipment,
	})
	require.NoError(t, err)
	assert.False(t, readable[conversation.ID])
	assert.Empty(t, threads.requests)
}

func TestSearchDocuments_HandsTheSearcherConversationOwnership(t *testing.T) {
	t.Parallel()

	params := personParams(serviceports.PrincipalTypeUser)
	params.Params = map[string]any{"query": "rate confirmation"}
	mineID := pulid.MustNew("athr_")
	theirsID := pulid.MustNew("athr_")
	mine := conversationDocument(mineID)
	theirs := conversationDocument(theirsID)
	threads := &fakeThreadOwners{owners: map[pulid.ID]pulid.ID{
		mineID:   params.Actor.UserID,
		theirsID: pulid.MustNew("usr_"),
	}}
	searcher := &fakeRetrievalSearcher{documents: &serviceports.DocumentSearchResult{}}

	_, err := newSearchDocumentsTool(searcher, &fakePermissions{allowed: true}, threads).
		Query(t.Context(), params)
	require.NoError(t, err)
	require.Len(t, searcher.requests, 1)

	readable, err := searcher.requests[0].Access.ReadableDocuments(
		t.Context(),
		[]*document.Document{mine, theirs},
	)
	require.NoError(t, err)
	assert.Equal(t, map[pulid.ID]bool{mine.ID: true}, readable)
}
