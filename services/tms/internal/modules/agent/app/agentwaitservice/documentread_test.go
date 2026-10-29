package agentwaitservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentwait"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentwaitports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentcontentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDocuments struct {
	documentports.DocumentRepository

	doc *document.Document
}

func (f *fakeDocuments) GetByID(
	_ context.Context,
	req documentports.GetDocumentByIDRequest,
) (*document.Document, error) {
	if f.doc == nil || f.doc.ID != req.ID {
		return nil, errortypes.NewNotFoundError("Document not found")
	}

	return f.doc, nil
}

type cancellingWorkflows struct {
	*fakeWorkflows

	cancelled []string
}

func (c *cancellingWorkflows) CancelWorkflow(_ context.Context, workflowID, _ string) error {
	c.cancelled = append(c.cancelled, workflowID)

	return nil
}

func beingRead() *document.Document {
	return &document.Document{
		ID:            pulid.MustNew("doc_"),
		OriginalName:  "rate-con.pdf",
		ContentStatus: document.ContentStatusExtracting,
	}
}

type fakeContents struct {
	documentcontentports.DocumentContentRepository

	content *documentcontent.Content
}

func (f *fakeContents) GetByDocumentID(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) (*documentcontent.Content, error) {
	if f.content == nil {
		return nil, errortypes.NewNotFoundError("Content not found")
	}

	return f.content, nil
}

func (h *harness) withDocument(doc *document.Document) *fakeContents {
	h.svc.facts.documents = &fakeDocuments{doc: doc}
	contents := &fakeContents{}
	h.svc.facts.contents = contents

	return contents
}

func modelReading(status documentcontent.Status, acceptance string) *documentcontent.Content {
	extracted := now - 30

	return &documentcontent.Content{
		Status:          status,
		LastExtractedAt: &extracted,
		StructuredData: map[string]any{
			"aiDiagnostics": map[string]any{"acceptanceStatus": acceptance},
		},
	}
}

func (h *harness) waitFor(
	t *testing.T,
	threadID pulid.ID,
	doc *document.Document,
) (*agentwait.Wait, error) {
	t.Helper()

	return h.svc.WaitForDocumentRead(t.Context(), &agentwaitports.WaitForDocumentReadRequest{
		Actor:        h.actor,
		DefinitionID: pulid.MustNew("agdef_"),
		ThreadID:     threadID,
		DocumentID:   doc.ID,
		FileName:     doc.OriginalName,
	})
}

/*
A file still being read parks the conversation instead of the person: the wait
watches the file, says in the composer what it is for, and plans to carry on
with what the person asked, so they never have to send their message again.
*/
func TestWaitForDocumentRead_ParksTheConversationOnTheFile(t *testing.T) {
	t.Parallel()

	h := newHarness()
	doc := beingRead()
	h.withDocument(doc)

	wait, err := h.waitFor(t, pulid.MustNew("athr_"), doc)

	require.NoError(t, err)
	assert.Equal(t, agentwait.KindDocumentRead, wait.Kind)
	assert.Equal(t, doc.ID, wait.WatchID)
	assert.Equal(t, "rate-con.pdf to finish being read", wait.Description)
	assert.Contains(t, wait.Then, "without asking them to send their message again")
	require.NotNil(t, wait.DueAt)
	assert.Equal(t, now+documentReadRecheckSeconds, *wait.DueAt)
	assert.Equal(t, now+documentReadLifetimeSeconds, wait.ExpiresAt)
	assert.Len(t, h.workflows.started, 1)
}

// Finding the file still being read a second time keeps the one wait.
func TestWaitForDocumentRead_KeepsTheWaitAlreadyOpen(t *testing.T) {
	t.Parallel()

	h := newHarness()
	doc := beingRead()
	h.withDocument(doc)
	threadID := pulid.MustNew("athr_")

	first, err := h.waitFor(t, threadID, doc)
	require.NoError(t, err)
	second, err := h.waitFor(t, threadID, doc)
	require.NoError(t, err)

	assert.Equal(t, first.ID, second.ID)
	assert.Len(t, h.waits.items, 1)
	assert.Len(t, h.workflows.started, 1)
}

func TestWaitForDocumentRead_RefusesAFileAlreadyRead(t *testing.T) {
	t.Parallel()

	h := newHarness()
	doc := beingRead()
	doc.ContentStatus = document.ContentStatusIndexed
	h.withDocument(doc)

	_, err := h.waitFor(t, pulid.MustNew("athr_"), doc)

	require.True(t, errortypes.IsBusinessError(err))
	assert.Contains(t, err.Error(), "rate-con.pdf has finished being read.")
	assert.Empty(t, h.waits.items)
}

// The model's reading of a rate confirmation is part of reading it: a file
// whose text is in while the model is still reading it is still waited on.
func TestWaitForDocumentRead_WaitsForTheModelsReading(t *testing.T) {
	t.Parallel()

	h := newHarness()
	doc := beingRead()
	doc.ContentStatus = document.ContentStatusIndexed
	h.withDocument(doc).content = modelReading(documentcontent.StatusIndexed, "pending")

	wait, err := h.waitFor(t, pulid.MustNew("athr_"), doc)

	require.NoError(t, err)
	assert.Equal(t, agentwait.KindDocumentRead, wait.Kind)
}

/*
The timer reads the file again: still being read puts the check off, and a
reading that ended, however it ended, picks the work up saying how.
*/
func TestCheck_ReadsTheFileAgain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(doc *document.Document)
		met    bool
		detail string
	}{
		{
			name:   "still being read",
			mutate: func(*document.Document) {},
		},
		{
			name: "read",
			mutate: func(doc *document.Document) {
				doc.ContentStatus = document.ContentStatusIndexed
			},
			met:    true,
			detail: "rate-con.pdf has finished being read.",
		},
		{
			name: "could not be read",
			mutate: func(doc *document.Document) {
				doc.ContentStatus = document.ContentStatusFailed
				doc.ContentError = "The file is password protected."
			},
			met:    true,
			detail: "rate-con.pdf could not be read: The file is password protected.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc := beingRead()
			tt.mutate(doc)
			wait := openWait(agentwait.KindDocumentRead, agentwait.Condition{DocumentID: doc.ID})
			h := newHarness(wait)
			h.withDocument(doc)

			check, err := h.svc.Check(t.Context(), tenantOf(wait), wait.ID, now)

			require.NoError(t, err)
			assert.Equal(t, tt.met, check.Met)
			assert.Equal(t, tt.detail, check.Detail)
			if !tt.met {
				require.NotNil(t, check.DueAt)
				assert.Equal(t, now+documentReadRecheckSeconds, *check.DueAt)
			}
		})
	}
}

func TestCheck_AFileRemovedEndsTheWaitSayingSo(t *testing.T) {
	t.Parallel()

	wait := openWait(
		agentwait.KindDocumentRead,
		agentwait.Condition{DocumentID: pulid.MustNew("doc_")},
	)
	h := newHarness(wait)
	h.withDocument(nil)

	check, err := h.svc.Check(t.Context(), tenantOf(wait), wait.ID, now)

	require.NoError(t, err)
	assert.True(t, check.Met)
	assert.Equal(t, "The file was removed before it was read.", check.Detail)
}

// Saving the file's extraction ends its wait at once, not at the next look.
func TestNotifyEvent_AnExtractionSavedEndsTheWaitOnItsFile(t *testing.T) {
	t.Parallel()

	documentID := pulid.MustNew("doc_")
	waiting := openWait(agentwait.KindDocumentRead, agentwait.Condition{DocumentID: documentID})
	other := openWait(
		agentwait.KindDocumentRead,
		agentwait.Condition{DocumentID: pulid.MustNew("doc_")},
	)
	h := newHarness(waiting, other)

	h.svc.NotifyEvent(t.Context(), &agenteventports.AgentEvent{
		Kind:       agenttrigger.EventDocumentExtracted,
		SubjectID:  documentID,
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	})

	require.Len(t, h.workflows.signals, 1)
	assert.Equal(t, waiting.WorkflowID, h.workflows.signals[0].workflowID)
	assert.Contains(t, h.workflows.signals[0].met.Detail, "Its reading finished at")
}

/*
A conversation that got the file another way (the person sent another message
and the agent read it in full) is not picked up again for nothing: reading it
cancels its wait, and a wait on the same file in another conversation stays.
*/
func TestStopWaitingForDocumentRead_CancelsOnlyThatConversationsWait(t *testing.T) {
	t.Parallel()

	documentID := pulid.MustNew("doc_")
	here := openWait(agentwait.KindDocumentRead, agentwait.Condition{DocumentID: documentID})
	elsewhere := openWait(agentwait.KindDocumentRead, agentwait.Condition{DocumentID: documentID})
	h := newHarness(here, elsewhere)
	workflows := &cancellingWorkflows{fakeWorkflows: h.workflows}
	h.svc.workflows = workflows

	err := h.svc.StopWaitingForDocumentRead(t.Context(), &agentwaitports.WaitForDocumentReadRequest{
		Actor:      h.actor,
		ThreadID:   here.ThreadID,
		DocumentID: documentID,
	})

	require.NoError(t, err)
	assert.Equal(t, agentwait.StatusCancelled, here.Status)
	assert.Equal(t, agentwait.StatusWaiting, elsewhere.Status)
	assert.Equal(t, []string{here.WorkflowID}, workflows.cancelled)
}

// The model's reading is waited for, and one Docling could not start (a server
// with extraction off) ends the wait saying why, not that the file was read.
func TestCheck_WaitsForTheModelAndSaysWhyNothingWasRead(t *testing.T) {
	t.Parallel()

	doc := beingRead()
	doc.ContentStatus = document.ContentStatusIndexed
	wait := openWait(agentwait.KindDocumentRead, agentwait.Condition{DocumentID: doc.ID})
	h := newHarness(wait)
	contents := h.withDocument(doc)

	contents.content = modelReading(documentcontent.StatusIndexed, "pending")
	check, err := h.svc.Check(t.Context(), tenantOf(wait), wait.ID, now)
	require.NoError(t, err)
	assert.False(t, check.Met)

	contents.content = modelReading(documentcontent.StatusIndexed, "accepted")
	check, err = h.svc.Check(t.Context(), tenantOf(wait), wait.ID, now)
	require.NoError(t, err)
	assert.True(t, check.Met)
	assert.Equal(t, "rate-con.pdf has finished being read.", check.Detail)

	contents.content = modelReading(documentcontent.StatusIndexed, "not_attempted")
	contents.content.FailureMessage = "Document extraction is turned off on this server."
	check, err = h.svc.Check(t.Context(), tenantOf(wait), wait.ID, now)
	require.NoError(t, err)
	assert.True(t, check.Met)
	assert.Equal(t,
		"rate-con.pdf could not be read: Document extraction is turned off on this server.",
		check.Detail)
}
