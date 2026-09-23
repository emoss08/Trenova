package assistantservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubDocuments struct {
	repositories.DocumentRepository

	docs map[pulid.ID]*document.Document
}

func (s *stubDocuments) GetByID(
	_ context.Context,
	req repositories.GetDocumentByIDRequest,
) (*document.Document, error) {
	if doc, ok := s.docs[req.ID]; ok {
		return doc, nil
	}

	return nil, errors.New("not found")
}

type stubContents struct {
	serviceports.DocumentContentService

	content *documentcontent.Content
	err     error
}

func (s *stubContents) GetContent(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) (*documentcontent.Content, error) {
	return s.content, s.err
}

func ownDocument(thread *conversation.Thread, actor *serviceports.RequestActor) *document.Document {
	return &document.Document{
		ID:           pulid.MustNew("doc_"),
		OriginalName: "rate-con.pdf",
		FileType:     "application/pdf",
		FileSize:     1024,
		ResourceType: AttachmentResourceType,
		ResourceID:   thread.ID.String(),
		UploadedByID: actor.UserID,
	}
}

func TestSendMessageStream_StoresAttachmentsAndMentionsOnTheUserTurn(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		textTurn("The rate confirmation is for Acme."),
	}}
	svc, conversations := newConversationService(completion, testDefinition())
	actor := testActor()
	doc := ownDocument(conversations.thread, actor)
	svc.documents = &stubDocuments{docs: map[pulid.ID]*document.Document{doc.ID: doc}}
	svc.contents = &stubContents{content: &documentcontent.Content{
		Status:               documentcontent.StatusExtracted,
		PageCount:            2,
		DetectedDocumentKind: "rate_confirmation",
		ContentText:          "RATE CONFIRMATION Acme Foods $1,200",
	}}

	customer := pulid.MustNew("cust_")
	result, err := svc.sendMessage(t.Context(), &serviceports.SendMessageRequest{
		ThreadID:              conversations.thread.ID,
		Content:               "Does this match what we quoted?",
		TenantInfo:            actor.TenantInfo(),
		AttachmentDocumentIDs: []pulid.ID{doc.ID, doc.ID},
		Mentions: []agent.EntityRef{
			{Type: "customer", ID: customer.String(), Label: " Acme Foods "},
		},
	}, actor, nil)
	require.NoError(t, err)
	assert.False(t, result.Refused)

	require.NotEmpty(t, conversations.appended)
	user := conversations.appended[0]
	require.Len(t, user.Attachments, 1, "a repeated id is one attachment")
	assert.Equal(t, doc.ID, user.Attachments[0].DocumentID)
	assert.Equal(t, "rate-con.pdf", user.Attachments[0].FileName)
	assert.Equal(t, "application/pdf", user.Attachments[0].ContentType)
	require.Len(t, user.Mentions, 1)
	assert.Equal(
		t,
		agent.EntityRef{Type: "customer", ID: customer.String(), Label: "Acme Foods"},
		user.Mentions[0],
	)
	for _, message := range conversations.appended[1:] {
		assert.Nil(t, message.Attachments)
		assert.Nil(t, message.Mentions)
	}

	require.NotNil(t, completion.LastReq)
	assert.Contains(t, completion.LastReq.System, "<attachments>")
	assert.Contains(t, completion.LastReq.System, "file: rate-con.pdf (application/pdf, 2 pages)")
	assert.Contains(t, completion.LastReq.System, "looks like: rate_confirmation")
	assert.Contains(t, completion.LastReq.System, "excerpt: RATE CONFIRMATION Acme Foods")
	assert.Contains(t, completion.LastReq.System, "<mentioned_records>")
	assert.Contains(t, completion.LastReq.System, "- customer "+customer.String()+": Acme Foods")
}

// A document id is a capability to read the document. The only ones a
// message may carry are the person's own uploads to this thread; another
// user's file, a shipment's file, or a file on another thread is refused
// before anything is read, so an attachment cannot be used to read across.
func TestSendMessageStream_RefusesAnAttachmentThatIsNotTheirs(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{textTurn("ok")}}
	svc, conversations := newConversationService(completion, testDefinition())
	actor := testActor()

	someoneElses := ownDocument(conversations.thread, actor)
	someoneElses.UploadedByID = pulid.MustNew("usr_")
	otherThread := ownDocument(conversations.thread, actor)
	otherThread.ResourceID = pulid.MustNew("athr_").String()
	shipmentFile := ownDocument(conversations.thread, actor)
	shipmentFile.ResourceType = "shipment"
	svc.documents = &stubDocuments{docs: map[pulid.ID]*document.Document{
		someoneElses.ID: someoneElses,
		otherThread.ID:  otherThread,
		shipmentFile.ID: shipmentFile,
	}}

	for name, id := range map[string]pulid.ID{
		"someone else's upload": someoneElses.ID,
		"another thread's file": otherThread.ID,
		"a shipment's file":     shipmentFile.ID,
		"an unknown document":   pulid.MustNew("doc_"),
	} {
		_, err := svc.sendMessage(t.Context(), &serviceports.SendMessageRequest{
			ThreadID:              conversations.thread.ID,
			Content:               "Read this",
			TenantInfo:            actor.TenantInfo(),
			AttachmentDocumentIDs: []pulid.ID{id},
		}, actor, nil)
		var multiErr *errortypes.MultiError
		require.ErrorAs(t, err, &multiErr, name)
		assert.Equal(t, "attachmentDocumentIds[0]", multiErr.Errors[0].Field, name)
	}
	assert.Nil(t, completion.LastReq, "nothing reached the model")
	assert.Zero(t, conversations.appendCalls, "nothing was saved")
}

func TestSendMessageStream_BoundsAttachmentsAndMentions(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{textTurn("ok")}}
	svc, conversations := newConversationService(completion, testDefinition())
	actor := testActor()
	svc.documents = &stubDocuments{}

	tooManyFiles := make([]pulid.ID, MaxAttachments+1)
	for i := range tooManyFiles {
		tooManyFiles[i] = pulid.MustNew("doc_")
	}
	_, err := svc.sendMessage(t.Context(), &serviceports.SendMessageRequest{
		ThreadID:              conversations.thread.ID,
		Content:               "Read these",
		TenantInfo:            actor.TenantInfo(),
		AttachmentDocumentIDs: tooManyFiles,
	}, actor, nil)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Equal(t, "attachmentDocumentIds", multiErr.Errors[0].Field)

	_, err = svc.sendMessage(t.Context(), &serviceports.SendMessageRequest{
		ThreadID:   conversations.thread.ID,
		Content:    "Who is this",
		TenantInfo: actor.TenantInfo(),
		Mentions:   []agent.EntityRef{{Type: "user_secret", ID: "not-an-id"}},
	}, actor, nil)
	require.ErrorAs(t, err, &multiErr)
	fields := make(map[string]bool, len(multiErr.Errors))
	for _, fieldErr := range multiErr.Errors {
		fields[fieldErr.Field] = true
	}
	assert.True(t, fields["mentions[0].type"])
	assert.True(t, fields["mentions[0].id"])
}

// A file whose extraction has not finished is still an attachment: the model
// is told it is pending rather than shown nothing, so it can say so and ask
// again with the tool instead of answering as if there were no file.
func TestSendMessageStream_NamesAPendingAttachment(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{textTurn("ok")}}
	svc, conversations := newConversationService(completion, testDefinition())
	actor := testActor()
	doc := ownDocument(conversations.thread, actor)
	svc.documents = &stubDocuments{docs: map[pulid.ID]*document.Document{doc.ID: doc}}
	svc.contents = &stubContents{err: errors.New("no content yet")}

	_, err := svc.sendMessage(t.Context(), &serviceports.SendMessageRequest{
		ThreadID:              conversations.thread.ID,
		Content:               "What is this?",
		TenantInfo:            actor.TenantInfo(),
		AttachmentDocumentIDs: []pulid.ID{doc.ID},
	}, actor, nil)
	require.NoError(t, err)

	assert.Contains(t, completion.LastReq.System, "id: "+doc.ID.String())
	assert.Contains(t, completion.LastReq.System, "reading: Pending")
}

func TestSendMessageStream_RendersTheTableView(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{textTurn("ok")}}
	svc, conversations := newConversationService(completion, testDefinition())
	actor := testActor()

	rows := 17
	_, err := svc.sendMessage(t.Context(), &serviceports.SendMessageRequest{
		ThreadID:   conversations.thread.ID,
		Content:    "Why are these late?",
		TenantInfo: actor.TenantInfo(),
		Page: &agent.PageContext{
			Path: "/shipments",
			View: &agent.PageView{
				Resource: "shipment",
				Sort: []domaintypes.SortField{
					{Field: "createdAt", Direction: dbtype.SortDirectionDesc},
				},
				RowCount: &rows,
			},
		},
	}, actor, nil)
	require.NoError(t, err)

	assert.Contains(t, completion.LastReq.System, "<page_view>\nresource: shipment")
	assert.Contains(t, completion.LastReq.System, "rows matching: 17")
	require.NotNil(t, conversations.appended[0].PageContext.View)
	assert.Equal(t, 17, *conversations.appended[0].PageContext.View.RowCount)
}

type stubDefinitionList struct {
	repositories.AgentDefinitionRepository

	definitions []*agentdefinition.Definition
	last        *repositories.ListAgentDefinitionRequest
}

func (s *stubDefinitionList) List(
	_ context.Context,
	req *repositories.ListAgentDefinitionRequest,
) (*pagination.ListResult[*agentdefinition.Definition], error) {
	s.last = req

	return &pagination.ListResult[*agentdefinition.Definition]{Items: s.definitions}, nil
}

func (s *stubDefinitionList) GetByID(
	_ context.Context,
	req repositories.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	for _, definition := range s.definitions {
		if definition.ID == req.ID {
			return definition, nil
		}
	}

	return nil, errors.New("not found")
}

type creatingConversations struct {
	stubConversations

	created *conversation.Thread
}

func (c *creatingConversations) CreateThread(
	_ context.Context,
	thread *conversation.Thread,
) (*conversation.Thread, error) {
	thread.ID = pulid.MustNew("athr_")
	c.created = thread
	c.thread = thread

	return thread, nil
}

func chatDefinition(name string, template agentdefinition.Template) *agentdefinition.Definition {
	d := testDefinition()
	d.ID = pulid.MustNew("agdef_")
	d.Name = name
	d.Template = template
	d.TriggerMode = agentdefinition.TriggerChat

	return d
}

// A quick question goes to the general assistant when the organization has
// one enabled for chat, on a thread that begins hidden; the thread is named
// on the stream before the answer so it can be kept whatever happens next.
func TestAsk_AnswersOnAHiddenThreadWithTheGeneralAssistant(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		textTurn("Twelve loads are late."),
	}}
	svc := newService(completion, &stubQueryRegistry{}, &stubActionRegistry{})
	dispatch := chatDefinition("Dispatch", agentdefinition.TemplateDispatchAssistant)
	general := chatDefinition("Assistant", agentdefinition.TemplateGeneralAssistant)
	definitions := &stubDefinitionList{
		definitions: []*agentdefinition.Definition{dispatch, general},
	}
	conversations := &creatingConversations{}
	svc.definitions = definitions
	svc.conversations = conversations
	actor := testActor()

	var events []string
	var announced *conversation.Thread
	result, err := svc.ask(t.Context(), &serviceports.AskRequest{
		Content:    "How many loads are late today?",
		TenantInfo: actor.TenantInfo(),
		Page:       &agent.PageContext{Path: "/dispatch"},
	}, actor, func(event serviceports.StreamEvent) {
		events = append(events, event.Event)
		if event.Event == serviceports.AssistantEventThread {
			announced = event.Data.(*conversation.Thread)
		}
	})
	require.NoError(t, err)

	assert.True(t, definitions.last.EnabledOnly)
	assert.True(t, definitions.last.ChatOnly)
	require.NotNil(t, conversations.created)
	assert.Equal(
		t,
		general.ID,
		conversations.created.AgentDefinitionID,
		"the general assistant answers",
	)
	assert.Equal(t, conversation.ThreadOriginAsk, conversations.created.Origin)
	assert.Equal(t, "How many loads are late today?", conversations.created.Title)
	require.NotNil(t, announced)
	assert.Equal(t, conversations.created.ID, announced.ID)
	assert.Equal(t, serviceports.AssistantEventThread, events[0], "the thread is announced first")
	assert.Equal(t, "Twelve loads are late.", result.Reply)
	assert.Equal(t, "/dispatch", conversations.appended[0].PageContext.Path)
}

func TestAsk_FallsBackToAnyChatAgentAndRefusesWithNone(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{textTurn("ok")}}
	svc := newService(completion, &stubQueryRegistry{}, &stubActionRegistry{})
	dispatch := chatDefinition("Dispatch", agentdefinition.TemplateDispatchAssistant)
	definitions := &stubDefinitionList{definitions: []*agentdefinition.Definition{dispatch}}
	conversations := &creatingConversations{}
	svc.definitions = definitions
	svc.conversations = conversations
	actor := testActor()

	_, err := svc.ask(t.Context(), &serviceports.AskRequest{
		Content:    "Anything late?",
		TenantInfo: actor.TenantInfo(),
	}, actor, nil)
	require.NoError(t, err)
	assert.Equal(t, dispatch.ID, conversations.created.AgentDefinitionID)

	definitions.definitions = nil
	_, err = svc.ask(t.Context(), &serviceports.AskRequest{
		Content:    "Anything late?",
		TenantInfo: actor.TenantInfo(),
	}, actor, nil)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))

	_, err = svc.ask(t.Context(), &serviceports.AskRequest{
		Content:    "   ",
		TenantInfo: actor.TenantInfo(),
	}, actor, nil)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
}
