package documentquerytools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/fieldgate"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/retrievalports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func conversationDocument(uploadedBy pulid.ID) *document.Document {
	return &document.Document{
		ID:           pulid.MustNew("doc_"),
		OriginalName: "rate-confirmation.pdf",
		FileType:     "application/pdf",
		ResourceType: document.ConversationResourceType,
		ResourceID:   pulid.MustNew("athr_").String(),
		UploadedByID: uploadedBy,
	}
}

func conversationSummaryTool(doc *document.Document) agenttoolports.AgentQueryTool {
	return NewGetDocumentTool(
		&stubDocumentRepo{doc: doc},
		&stubContentReader{content: &documentcontent.Content{
			Status:      documentcontent.StatusIndexed,
			ContentText: "Rate confirmation for load 4471",
		}},
		noLabels(),
		noUploader(),
		&querytooltest.FakePermissions{Allowed: true},
	)
}

func TestGetDocument_ConversationFileIsReadOnlyByItsOwner(t *testing.T) {
	t.Parallel()

	owner := querytooltest.PersonParams(permissionports.PrincipalTypeUser)
	doc := conversationDocument(owner.Actor.UserID)
	owner.Params = map[string]any{"documentId": doc.ID.String()}

	result, err := conversationSummaryTool(doc).Query(t.Context(), owner)
	require.NoError(t, err)
	assert.Equal(t, "rate-confirmation.pdf", result.(*documentSummary).FileName)

	colleague := querytooltest.PersonParams(permissionports.PrincipalTypeUser)
	colleague.OrganizationID = owner.OrganizationID
	colleague.BusinessUnitID = owner.BusinessUnitID
	colleague.Actor.OrganizationID = owner.OrganizationID
	colleague.Actor.BusinessUnitID = owner.BusinessUnitID
	colleague.Params = map[string]any{"documentId": doc.ID.String()}

	_, err = conversationSummaryTool(doc).Query(t.Context(), colleague)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "may not read")
}

func TestGetDocument_ConversationFileIsNeverReadWithoutAPerson(t *testing.T) {
	t.Parallel()

	for _, principal := range []kernelports.PrincipalType{
		permissionports.PrincipalTypeAgent,
		permissionports.PrincipalTypeAPIKey,
		"",
	} {
		t.Run(string(principal), func(t *testing.T) {
			t.Parallel()

			params := querytooltest.PersonParams(principal)
			doc := conversationDocument(params.Actor.UserID)
			params.Params = map[string]any{"documentId": doc.ID.String()}

			_, err := conversationSummaryTool(doc).Query(t.Context(), params)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "may not read")
		})
	}
}

func TestRetrievalAccess_KeepsOnlyTheConversationFilesThePersonAttached(t *testing.T) {
	t.Parallel()

	params := querytooltest.PersonParams(permissionports.PrincipalTypeUser)
	mine := conversationDocument(params.Actor.UserID)
	mineAgain := conversationDocument(params.Actor.UserID)
	theirs := conversationDocument(pulid.MustNew("usr_"))
	unattributed := conversationDocument(pulid.Nil)
	shipment := summaryDocument()
	access := fieldgate.NewAccess(&querytooltest.FakePermissions{Allowed: true}).
		ForRetrieval(params)

	readable, err := access.ReadableDocuments(t.Context(), []*document.Document{
		mine, mineAgain, theirs, unattributed, shipment, nil,
	})
	require.NoError(t, err)
	assert.Equal(t, map[pulid.ID]bool{
		mine.ID:      true,
		mineAgain.ID: true,
		shipment.ID:  true,
	}, readable)
}

func TestRetrievalAccess_BackgroundRunSeesNoConversationFiles(t *testing.T) {
	t.Parallel()

	params := querytooltest.PersonParams(permissionports.PrincipalTypeAgent)
	conversation := conversationDocument(params.Actor.UserID)
	shipment := summaryDocument()
	access := fieldgate.NewAccess(&querytooltest.FakePermissions{Allowed: true}).
		ForRetrieval(params)

	readable, err := access.ReadableDocuments(t.Context(), []*document.Document{
		conversation, shipment,
	})
	require.NoError(t, err)
	assert.False(t, readable[conversation.ID])
}

func TestSearchDocuments_HandsTheSearcherTheConversationRule(t *testing.T) {
	t.Parallel()

	params := querytooltest.PersonParams(permissionports.PrincipalTypeUser)
	params.Params = map[string]any{"query": "rate confirmation"}
	mine := conversationDocument(params.Actor.UserID)
	theirs := conversationDocument(pulid.MustNew("usr_"))
	searcher := &querytooltest.FakeRetrievalSearcher{
		Documents: &retrievalports.DocumentSearchResult{},
	}

	_, err := NewSearchDocumentsTool(searcher, &querytooltest.FakePermissions{Allowed: true}).
		Query(t.Context(), params)
	require.NoError(t, err)
	require.Len(t, searcher.Requests, 1)

	readable, err := searcher.Requests[0].Access.ReadableDocuments(
		t.Context(),
		[]*document.Document{mine, theirs},
	)
	require.NoError(t, err)
	assert.Equal(t, map[pulid.ID]bool{mine.ID: true}, readable)
}
