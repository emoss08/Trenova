package assistantservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentwait"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentwaitports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type draftReader struct {
	draft map[string]any
	asked []map[string]any
}

func (r *draftReader) Name() string        { return toolGetShipmentDraft }
func (r *draftReader) Description() string { return "Read a document's shipment draft." }
func (r *draftReader) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType:     toolschema.TypeObject,
		toolschema.KeyRequired: []string{draftDocumentIDParam},
	}
}

func (r *draftReader) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name: toolGetShipmentDraft, Kind: agenttooltrust.ToolKindQuery,
		Resource: permission.ResourceDocument, Operation: permission.OpRead,
	}
}

func (r *draftReader) Query(
	_ context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	r.asked = append(r.asked, params.Params)
	if r.draft == nil {
		return nil, errortypes.NewNotFoundError("Draft not found")
	}

	return r.draft, nil
}

func readDraft(documentID pulid.ID) map[string]any {
	return map[string]any{
		draftDocumentIDParam: documentID.String(),
		"status":             "Ready",
		"documentKind":       "RateConfirmation",
		"confidence":         0.96,
		"fields": map[string]any{
			"bol": map[string]any{
				"label": "BOL", "value": "123456789", "confidence": 0.94, "pageNumber": 1,
			},
		},
	}
}

func cardService(reader *draftReader) (*Service, *stubArtifactRepo) {
	repo := &stubArtifactRepo{}

	return &Service{
		logger:    zap.NewNop(),
		artifacts: repo,
		runtime: agentruntime.New(agentruntime.Params{
			Logger:      zap.NewNop(),
			QueryTools:  &stubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{reader}},
			ActionTools: &stubActionRegistry{},
			Permissions: &agenttest.StubPermissions{},
		}),
	}, repo
}

func deskThread() *conversation.Thread {
	return &conversation.Thread{ID: pulid.MustNew("athr_"), Origin: conversation.ThreadOriginDesk}
}

/*
A rate confirmation dropped into the Desk shows what was read off it without
anyone asking: the card is kept beside the conversation as the turn begins, one
per file, read as the person through intake_get_draft, and the agent is told
where it is so the reply points to it instead of publishing the reading as a
document of its own.
*/
func TestShowAttachments_ShowsWhatWasReadOffAnAttachedFile(t *testing.T) {
	t.Parallel()

	documentID := pulid.MustNew("doc_")
	reader := &draftReader{draft: readDraft(documentID)}
	svc, repo := cardService(reader)
	attachments := []agentturn.RuntimeAttachment{{DocumentID: documentID.String()}}

	svc.showAttachments(
		t.Context(),
		deskThread(),
		testActor(),
		testActor().TenantInfo(),
		attachments,
	)

	require.Len(t, repo.upserts, 1)
	card := repo.upserts[0]
	assert.Equal(t, assistantartifact.KindExtraction, card.Kind)
	assert.Equal(t, attachmentCallID+documentID.String(), card.SourceToolCallID)
	assert.Equal(t, extractionLineageKey(documentID.String()), card.LineageKey)
	assert.Equal(t, documentID.String(), card.Payload[draftDocumentIDParam])
	assert.Equal(t, []map[string]any{{draftDocumentIDParam: documentID.String()}}, reader.asked)
	assert.Equal(t, "[Rate confirmation](artifact:"+card.ID.String()+")", attachments[0].Shown)
}

func TestShowAttachments_ShowsNothingUntilSomethingIsRead(t *testing.T) {
	t.Parallel()

	documentID := pulid.MustNew("doc_")
	draft := readDraft(documentID)
	draft["fields"] = map[string]any{}
	svc, repo := cardService(&draftReader{draft: draft})
	attachments := []agentturn.RuntimeAttachment{{DocumentID: documentID.String()}}

	svc.showAttachments(
		t.Context(),
		deskThread(),
		testActor(),
		testActor().TenantInfo(),
		attachments,
	)

	assert.Empty(t, repo.upserts)
	assert.Empty(t, attachments[0].Shown)
}

func TestShowAttachments_AFileWithNoDraftShowsNoCard(t *testing.T) {
	t.Parallel()

	svc, repo := cardService(&draftReader{})
	attachments := []agentturn.RuntimeAttachment{{DocumentID: pulid.MustNew("doc_").String()}}

	svc.showAttachments(
		t.Context(),
		deskThread(),
		testActor(),
		testActor().TenantInfo(),
		attachments,
	)

	assert.Empty(t, repo.upserts)
}

// The palette answers with nothing beside it, so it keeps no card.
func TestShowAttachments_KeepsNoCardWhereTheAnswerIsReadAlone(t *testing.T) {
	t.Parallel()

	documentID := pulid.MustNew("doc_")
	svc, repo := cardService(&draftReader{draft: readDraft(documentID)})
	thread := deskThread()
	thread.Origin = conversation.ThreadOriginAsk

	svc.showAttachments(t.Context(), thread, testActor(), testActor().TenantInfo(),
		[]agentturn.RuntimeAttachment{{DocumentID: documentID.String()}})

	assert.Empty(t, repo.upserts)
}

/*
The card the attachment opened and the card the agent opens by reading the draft
are one artifact: reading a draft that has not changed points the reply at the
card already beside the conversation instead of keeping a copy.
*/
func TestObserve_ReadingTheDraftAgainPointsAtTheAttachmentsCard(t *testing.T) {
	t.Parallel()

	documentID := pulid.MustNew("doc_")
	reader := &draftReader{draft: readDraft(documentID)}
	svc, repo := cardService(reader)
	thread := deskThread()
	recorder := svc.newArtifactRecorder(
		t.Context(),
		thread,
		testActor().TenantInfo(),
		testActor(),
		nil,
	)
	shown := recorder.attachedExtraction(documentID)
	require.NotNil(t, shown)

	again, err := recorder.observe(agentruntimeports.ToolObservation{
		Call: llmports.ToolCall{
			ID: "call_draft", Name: toolGetShipmentDraft,
			Arguments: map[string]any{draftDocumentIDParam: documentID.String()},
		},
		Data: readDraft(documentID),
	})

	require.NoError(t, err)
	require.NotNil(t, again)
	assert.Equal(t, shown.ID, again.ID)
	assert.Len(t, repo.upserts, 1)
}

func TestShowReading_TellsTheAgentWhereTheCardIs(t *testing.T) {
	t.Parallel()

	documentID := pulid.MustNew("doc_")
	svc, repo := cardService(&draftReader{draft: readDraft(documentID)})

	note := svc.showReading(
		t.Context(),
		deskThread(),
		testActor(),
		testActor().TenantInfo(),
		documentID,
	)

	require.Len(t, repo.upserts, 1)
	assert.Contains(t, note, "(artifact:"+repo.upserts[0].ID.String()+")")
	assert.Contains(t, note, "do not publish the reading as a document")
	assert.Empty(
		t,
		svc.showReading(
			t.Context(),
			deskThread(),
			testActor(),
			testActor().TenantInfo(),
			pulid.Nil,
		),
	)
}

type recordingReadWaits struct {
	waited  []*agentwaitports.WaitForDocumentReadRequest
	stopped []*agentwaitports.WaitForDocumentReadRequest
}

func (r *recordingReadWaits) WaitForDocumentRead(
	_ context.Context,
	req *agentwaitports.WaitForDocumentReadRequest,
) (*agentwait.Wait, error) {
	r.waited = append(r.waited, req)

	return &agentwait.Wait{}, nil
}

func (r *recordingReadWaits) StopWaitingForDocumentRead(
	_ context.Context,
	req *agentwaitports.WaitForDocumentReadRequest,
) error {
	r.stopped = append(r.stopped, req)

	return nil
}

func readAttachmentResult(documentID pulid.ID, reading string) *agentruntimeports.ToolObservation {
	data := map[string]any{
		draftDocumentIDParam: documentID.String(),
		"fileName":           "rate-con.pdf",
		"reading":            reading,
	}
	if reading != "Indexed" {
		data["stillBeingRead"] = true
	}

	return &agentruntimeports.ToolObservation{
		Call: llmports.ToolCall{ID: "call_read", Name: agentdefinition.CoreToolReadAttachment},
		Data: data,
	}
}

/*
A file still being read parks the conversation on it, so it picks up again on
its own when the reading finishes; the person never has to ask how it is going.
*/
func TestWaitOnUnreadFile_ParksTheConversationOnAFileStillBeingRead(t *testing.T) {
	t.Parallel()

	waits := &recordingReadWaits{}
	svc := &Service{logger: zap.NewNop(), readWaits: waits}
	run := &UnreadFileRun{
		ThreadID: pulid.MustNew("athr_"), DefinitionID: pulid.MustNew("adef_"), Actor: testActor(),
	}
	documentID := pulid.MustNew("doc_")

	svc.WaitOnUnreadFile(t.Context(), run, readAttachmentResult(documentID, "Extracting"))

	require.Len(t, waits.waited, 1)
	assert.Equal(t, documentID, waits.waited[0].DocumentID)
	assert.Equal(t, run.ThreadID, waits.waited[0].ThreadID)
	assert.Equal(t, run.DefinitionID, waits.waited[0].DefinitionID)
	assert.Equal(t, "rate-con.pdf", waits.waited[0].FileName)
	assert.Empty(t, waits.stopped)
}

// The text is in but the model is still reading it: still being read.
func TestWaitOnUnreadFile_WaitsWhileTheModelIsReading(t *testing.T) {
	t.Parallel()

	waits := &recordingReadWaits{}
	svc := &Service{logger: zap.NewNop(), readWaits: waits}
	observation := readAttachmentResult(pulid.MustNew("doc_"), "Indexed")
	observation.Data.(map[string]any)["stillBeingRead"] = true

	svc.WaitOnUnreadFile(
		t.Context(),
		&UnreadFileRun{ThreadID: pulid.MustNew("athr_"), Actor: testActor()},
		observation,
	)

	assert.Len(t, waits.waited, 1)
	assert.Empty(t, waits.stopped)
}

func TestWaitOnUnreadFile_AFileReadInFullEndsTheWaitOnIt(t *testing.T) {
	t.Parallel()

	waits := &recordingReadWaits{}
	svc := &Service{logger: zap.NewNop(), readWaits: waits}
	run := &UnreadFileRun{ThreadID: pulid.MustNew("athr_"), Actor: testActor()}

	svc.WaitOnUnreadFile(t.Context(), run, readAttachmentResult(pulid.MustNew("doc_"), "Indexed"))

	assert.Empty(t, waits.waited)
	assert.Len(t, waits.stopped, 1)
}

func TestWaitOnUnreadFile_LeavesOtherWorkAlone(t *testing.T) {
	t.Parallel()

	documentID := pulid.MustNew("doc_")
	thread := pulid.MustNew("athr_")
	otherTool := readAttachmentResult(documentID, "Extracting")
	otherTool.Call.Name = toolGetShipmentDraft
	failed := readAttachmentResult(documentID, "Extracting")
	failed.Failed = true

	tests := map[string]struct {
		run         *UnreadFileRun
		observation *agentruntimeports.ToolObservation
	}{
		"a delegated task": {
			run:         &UnreadFileRun{ThreadID: thread, Actor: testActor(), Delegated: true},
			observation: readAttachmentResult(documentID, "Extracting"),
		},
		"no conversation": {
			run:         &UnreadFileRun{Actor: testActor()},
			observation: readAttachmentResult(documentID, "Extracting"),
		},
		"another tool": {
			run:         &UnreadFileRun{ThreadID: thread, Actor: testActor()},
			observation: otherTool,
		},
		"a failed call": {
			run:         &UnreadFileRun{ThreadID: thread, Actor: testActor()},
			observation: failed,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			waits := &recordingReadWaits{}
			svc := &Service{logger: zap.NewNop(), readWaits: waits}

			svc.WaitOnUnreadFile(t.Context(), tt.run, tt.observation)

			assert.Empty(t, waits.waited)
			assert.Empty(t, waits.stopped)
		})
	}
}

type extractedDocuments struct {
	documentports.DocumentRepository

	doc *document.Document
}

func (d *extractedDocuments) GetByID(
	_ context.Context,
	_ documentports.GetDocumentByIDRequest,
) (*document.Document, error) {
	return d.doc, nil
}

type attachmentConversations struct {
	assistantports.ConversationRepository

	thread  *conversation.Thread
	carries bool
	asked   []assistantports.CarriesAttachmentRequest
}

func (c *attachmentConversations) CarriesAttachment(
	_ context.Context,
	req *assistantports.CarriesAttachmentRequest,
) (bool, error) {
	c.asked = append(c.asked, *req)

	return c.carries, nil
}

func (c *attachmentConversations) GetThread(
	_ context.Context,
	req assistantports.GetThreadRequest,
) (*conversation.Thread, error) {
	if c.thread == nil || c.thread.ID != req.ID || c.thread.UserID != req.UserID {
		return nil, errortypes.NewNotFoundError("Thread not found")
	}

	return c.thread, nil
}

func extractedEvent(doc *document.Document) *agentports.AgentEvent {
	return &agentports.AgentEvent{
		Kind:       agenttrigger.EventDocumentExtracted,
		SubjectID:  doc.ID,
		TenantInfo: pagination.TenantInfo{OrgID: doc.OrganizationID, BuID: doc.BusinessUnitID},
	}
}

func attachedDocument(thread *conversation.Thread) *document.Document {
	return &document.Document{
		ID:             pulid.MustNew("doc_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		ResourceType:   document.ConversationResourceType,
		ResourceID:     thread.ID.String(),
		UploadedByID:   thread.UserID,
	}
}

/*
Reading a file takes longer than most replies, so its card usually arrives
after the reply, and the model's reading arrives after the rule-based one: each
saved extraction revises the card in the conversation the file was sent in,
read as the person who sent it.
*/
func TestAttachmentExtracted_ShowsTheCardInTheConversationTheFileWasSentIn(t *testing.T) {
	t.Parallel()

	thread := deskThread()
	thread.UserID = pulid.MustNew("usr_")
	doc := attachedDocument(thread)
	svc, repo := cardService(&draftReader{draft: readDraft(doc.ID)})
	conversations := &attachmentConversations{thread: thread, carries: true}
	svc.documents = &extractedDocuments{doc: doc}
	svc.conversations = conversations

	svc.AttachmentExtracted(t.Context(), extractedEvent(doc))

	require.Len(t, conversations.asked, 1)
	assert.Equal(t, thread.ID, conversations.asked[0].ThreadID)
	assert.Equal(t, doc.ID, conversations.asked[0].DocumentID)
	require.Len(t, repo.upserts, 1)
	assert.Equal(t, thread.ID, repo.upserts[0].ThreadID)
	assert.Equal(t, attachmentCallID+doc.ID.String(), repo.upserts[0].SourceToolCallID)
}

func TestAttachmentExtracted_AFileUploadedButNeverSentGetsNoCard(t *testing.T) {
	t.Parallel()

	thread := deskThread()
	thread.UserID = pulid.MustNew("usr_")
	doc := attachedDocument(thread)
	svc, repo := cardService(&draftReader{draft: readDraft(doc.ID)})
	svc.documents = &extractedDocuments{doc: doc}
	svc.conversations = &attachmentConversations{thread: thread}

	svc.AttachmentExtracted(t.Context(), extractedEvent(doc))

	assert.Empty(t, repo.upserts)
}

func TestAttachmentExtracted_LeavesAShipmentsDocumentAlone(t *testing.T) {
	t.Parallel()

	thread := deskThread()
	thread.UserID = pulid.MustNew("usr_")
	doc := attachedDocument(thread)
	doc.ResourceType = "shipment"
	svc, repo := cardService(&draftReader{draft: readDraft(doc.ID)})
	conversations := &attachmentConversations{thread: thread, carries: true}
	svc.documents = &extractedDocuments{doc: doc}
	svc.conversations = conversations

	svc.AttachmentExtracted(t.Context(), extractedEvent(doc))

	assert.Empty(t, conversations.asked)
	assert.Empty(t, repo.upserts)
}
