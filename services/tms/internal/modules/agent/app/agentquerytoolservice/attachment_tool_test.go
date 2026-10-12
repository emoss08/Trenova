package agentquerytoolservice

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
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

type conversationDocuments struct {
	stubDocumentRepo

	byResource []*document.Document
	asked      *documentports.GetDocumentsByResourceRequest
}

func (c *conversationDocuments) GetByResourceID(
	_ context.Context,
	req *documentports.GetDocumentsByResourceRequest,
) ([]*document.Document, error) {
	c.asked = req

	return c.byResource, nil
}

type readingContents struct {
	settleAfter int32
	calls       atomic.Int32
	content     *documentcontent.Content
}

func (r *readingContents) GetContent(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) (*documentcontent.Content, error) {
	if r.calls.Add(1) <= r.settleAfter {
		return nil, errortypes.NewNotFoundError("Document content not found")
	}

	return r.content, nil
}

func attachmentParams(threadID pulid.ID, params map[string]any) *agentports.QueryToolParams {
	out := testParams(params)
	out.Actor.PrincipalType = permissionports.PrincipalTypeUser
	out.ThreadID = threadID

	return out
}

func attachedDocument(threadID, userID pulid.ID) *document.Document {
	return &document.Document{
		ID:           pulid.MustNew("doc_"),
		OriginalName: "rate_confirmation.pdf",
		FileType:     "application/pdf",
		FileSize:     104_579,
		ResourceType: document.ConversationResourceType,
		ResourceID:   threadID.String(),
		UploadedByID: userID,
		CreatedAt:    1_791_752_593,
	}
}

func readAttachment(
	documents documentports.DocumentRepository,
	contents contentReader,
) *readAttachmentTool {
	tool := newReadAttachmentTool(documents, contents).(*readAttachmentTool)
	tool.wait = 50 * time.Millisecond
	tool.poll = 5 * time.Millisecond

	return tool
}

func TestReadAttachment_ReadsAFileThePersonAttachedHere(t *testing.T) {
	t.Parallel()

	thread := pulid.MustNew("athr_")
	params := attachmentParams(thread, nil)
	doc := attachedDocument(thread, params.Actor.UserID)
	params.Params = map[string]any{"documentId": doc.ID.String(), "page": 1}
	tool := readAttachment(&stubDocumentRepo{doc: doc}, &stubContentReader{
		content: &documentcontent.Content{
			Status:               documentcontent.StatusIndexed,
			PageCount:            4,
			DetectedDocumentKind: "RateConfirmation",
			ContentText:          "whole text",
			Pages: []*documentcontent.Page{{
				PageNumber:    1,
				ExtractedText: "SHIPPER#1:   BEX Logistics-OH Consolidation",
			}},
		},
	})

	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)

	summary := result.(*documentSummary)
	assert.Equal(t, "rate_confirmation.pdf", summary.FileName)
	assert.Equal(t, "Indexed", summary.Reading)
	assert.Equal(t, 1, summary.Page)
	assert.Equal(t, "SHIPPER#1:   BEX Logistics-OH Consolidation", summary.Text)
	assert.Empty(t, summary.Note)
}

func TestReadAttachment_RefusesAFileFromAnywhereElse(t *testing.T) {
	t.Parallel()

	thread := pulid.MustNew("athr_")
	params := attachmentParams(thread, nil)
	cases := map[string]*document.Document{
		"another conversation": attachedDocument(pulid.MustNew("athr_"), params.Actor.UserID),
		"another person":       attachedDocument(thread, pulid.MustNew("usr_")),
		"a shipment's file": {
			ID:           pulid.MustNew("doc_"),
			ResourceType: "shipment",
			ResourceID:   pulid.MustNew("shp_").String(),
			UploadedByID: params.Actor.UserID,
		},
	}

	for name, doc := range cases {
		params.Params = map[string]any{"documentId": doc.ID.String()}
		tool := readAttachment(&stubDocumentRepo{doc: doc}, &stubContentReader{
			content: &documentcontent.Content{
				Status:      documentcontent.StatusIndexed,
				ContentText: "secret",
			},
		})

		result, err := tool.Query(t.Context(), params)
		require.Error(t, err, name)
		assert.Nil(t, result, name)
		assert.Contains(t, err.Error(), "was not attached to this conversation", name)
	}
}

func TestReadAttachment_NeedsAConversation(t *testing.T) {
	t.Parallel()

	params := attachmentParams(pulid.Nil, map[string]any{})
	tool := readAttachment(&stubDocumentRepo{}, &stubContentReader{})

	_, err := tool.Query(t.Context(), params)
	require.ErrorIs(t, err, errNoConversation)
}

func TestReadAttachment_WaitsForAFileStillBeingRead(t *testing.T) {
	t.Parallel()

	thread := pulid.MustNew("athr_")
	params := attachmentParams(thread, nil)
	doc := attachedDocument(thread, params.Actor.UserID)
	params.Params = map[string]any{"documentId": doc.ID.String()}
	contents := &readingContents{settleAfter: 2, content: &documentcontent.Content{
		Status:      documentcontent.StatusIndexed,
		ContentText: "Pick Up Date:   08/02/22",
	}}
	tool := readAttachment(&stubDocumentRepo{doc: doc}, contents)

	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)

	summary := result.(*documentSummary)
	assert.Equal(t, "Pick Up Date:   08/02/22", summary.Text)
	assert.Equal(t, int32(3), contents.calls.Load())
}

func TestReadAttachment_SaysWhenAFileIsStillBeingRead(t *testing.T) {
	t.Parallel()

	thread := pulid.MustNew("athr_")
	params := attachmentParams(thread, nil)
	doc := attachedDocument(thread, params.Actor.UserID)
	params.Params = map[string]any{"documentId": doc.ID.String()}
	tool := readAttachment(&stubDocumentRepo{doc: doc}, &stubContentReader{
		content: &documentcontent.Content{Status: documentcontent.StatusExtracting},
	})

	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)

	summary := result.(*documentSummary)
	assert.Equal(t, "Extracting", summary.Reading)
	assert.Equal(t, attachmentStillReading, summary.Note)
}

func TestReadAttachment_ListsOnlyThePersonsFilesInThisConversation(t *testing.T) {
	t.Parallel()

	thread := pulid.MustNew("athr_")
	params := attachmentParams(thread, map[string]any{})
	own := attachedDocument(thread, params.Actor.UserID)
	someoneElses := attachedDocument(thread, pulid.MustNew("usr_"))
	documents := &conversationDocuments{byResource: []*document.Document{own, someoneElses, nil}}
	tool := readAttachment(documents, &stubContentReader{})

	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)

	require.NotNil(t, documents.asked)
	assert.Equal(t, thread.String(), documents.asked.ResourceID)
	assert.Equal(t, document.ConversationResourceType, documents.asked.ResourceType)
	files := result.(*conversationFiles)
	assert.Equal(t, 1, files.Total)
	require.Len(t, files.Files, 1)
	assert.Equal(t, own.ID.String(), files.Files[0].DocumentID)
	assert.Equal(t, "rate_confirmation.pdf", files.Files[0].FileName)
	assert.Equal(t, attachmentListNote, files.Note)
}

func TestReadAttachment_ListSaysWhenNothingWasAttached(t *testing.T) {
	t.Parallel()

	params := attachmentParams(pulid.MustNew("athr_"), map[string]any{})
	tool := readAttachment(&conversationDocuments{}, &stubContentReader{})

	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)

	files := result.(*conversationFiles)
	assert.Empty(t, files.Files)
	assert.Equal(t, attachmentNoneNote, files.Note)
}

func TestReadAttachment_IsThePersonsOwnRead(t *testing.T) {
	t.Parallel()

	policy := readAttachment(nil, nil).Policy()

	assert.Equal(t, permission.ResourceAssistant, policy.Resource)
	assert.Equal(t, agent.ToolScopeSelf, policy.Scope)
	assert.Equal(t, agent.ExternalReadAlways, policy.ReadsExternal)
	assert.Equal(t, agent.TaintSourceAttachment, policy.Source)
}
