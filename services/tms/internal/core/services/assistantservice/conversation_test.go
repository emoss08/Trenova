package assistantservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubConversations struct {
	repositories.ConversationRepository

	thread   *conversation.Thread
	appended []conversation.Message
	// appendCtxErr is the state of the context AppendTurn was given, so a test
	// can prove the save survives the request being cancelled.
	appendCtxErr error
	appendCalls  int
	// messages is what ListMessages serves; lastList is what it was asked.
	messages []conversation.Message
	lastList repositories.ListMessagesRequest
	// count is the thread's length as CountMessages reports it.
	count int
}

func (s *stubConversations) CountMessages(
	context.Context,
	repositories.CountMessagesRequest,
) (int, error) {
	return s.count, nil
}

func (s *stubConversations) GetThread(
	context.Context,
	repositories.GetThreadRequest,
) (*conversation.Thread, error) {
	return s.thread, nil
}

func (s *stubConversations) ListMessages(
	_ context.Context,
	req repositories.ListMessagesRequest,
) ([]conversation.Message, error) {
	s.lastList = req

	return s.messages, nil
}

func (s *stubConversations) AppendTurn(
	ctx context.Context,
	req repositories.AppendTurnRequest,
) ([]conversation.Message, error) {
	s.appended = req.Messages
	s.appendCtxErr = ctx.Err()
	s.appendCalls++
	return req.Messages, nil
}

func (s *stubConversations) UpdateThread(
	_ context.Context,
	thread *conversation.Thread,
) (*conversation.Thread, error) {
	return thread, nil
}

type stubDefinitions struct {
	repositories.AgentDefinitionRepository

	definition *agentdefinition.Definition
}

func (s *stubDefinitions) GetByID(
	context.Context,
	repositories.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	return s.definition, nil
}

func newConversationService(
	completion *scriptedCompletion,
	definition *agentdefinition.Definition,
) (*Service, *stubConversations) {
	svc := newService(completion, &stubQueryRegistry{}, &stubActionRegistry{})
	conversations := &stubConversations{thread: &conversation.Thread{
		ID:                pulid.MustNew("athr_"),
		AgentDefinitionID: pulid.MustNew("agdef_"),
		Title:             "Dispatch",
	}}
	svc.conversations = conversations
	svc.definitions = &stubDefinitions{definition: definition}

	return svc, conversations
}

func TestSendMessageStream_StoresThePageContextOnTheUserTurn(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		textTurn("It is at the Los Angeles terminal."),
	}}
	svc, conversations := newConversationService(completion, testDefinition())
	actor := testActor()

	result, err := svc.sendMessage(t.Context(), &serviceports.SendMessageRequest{
		ThreadID: conversations.thread.ID,
		Content:  "Where is this shipment?",
		Page: &agent.PageContext{
			Path:       " /shipments?panelEntityId=shp_1 ",
			EntityType: "shipment",
			EntityID:   "shp_1",
			Title:      "Shipments | Acme",
		},
		TenantInfo: actor.TenantInfo(),
	}, actor, nil)
	require.NoError(t, err)
	assert.False(t, result.Refused)

	require.NotEmpty(t, conversations.appended)
	user := conversations.appended[0]
	assert.Equal(t, conversation.RoleUser, user.Role)
	require.NotNil(t, user.PageContext, "the user turn carries the page it was asked from")
	assert.Equal(t, "/shipments?panelEntityId=shp_1", user.PageContext.Path)
	assert.Equal(t, "shipment", user.PageContext.EntityType)
	assert.Equal(t, "shp_1", user.PageContext.EntityID)

	for _, message := range conversations.appended[1:] {
		assert.Nil(t, message.PageContext, "only the user turn records the page")
	}

	require.NotNil(t, completion.LastReq, "the chat model was called")
	assert.Contains(
		t,
		completion.LastReq.System,
		"record: shipment shp_1",
		"the page reaches the model as fenced runtime context",
	)
}

func TestSendMessageStream_RejectsAPageContextItCouldNotTrust(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		textTurn("never reached"),
	}}
	svc, conversations := newConversationService(completion, testDefinition())
	actor := testActor()

	_, err := svc.sendMessage(t.Context(), &serviceports.SendMessageRequest{
		ThreadID: conversations.thread.ID,
		Content:  "Where is this?",
		Page: &agent.PageContext{
			Path:       "https://evil.example/steal",
			EntityType: "user_credential",
			EntityID:   "1",
		},
		TenantInfo: actor.TenantInfo(),
	}, actor, nil)

	require.Error(t, err)
	assert.True(t, errortypes.IsMultiError(err), "field errors name the offending context fields")
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	fields := make(map[string]bool, len(multiErr.Errors))
	for _, fieldErr := range multiErr.Errors {
		fields[fieldErr.Field] = true
	}
	assert.True(t, fields["context.path"])
	assert.True(t, fields["context.entityType"])
	assert.Zero(t, completion.CallCount, "a rejected request never reaches the model")
	assert.Empty(t, conversations.appended, "nothing is saved for a rejected request")
}

func numberedMessages(from, count int) []conversation.Message {
	messages := make([]conversation.Message, 0, count)
	for i := range count {
		messages = append(messages, conversation.Message{
			ID:       pulid.MustNew("amsg_"),
			Sequence: from + i,
			Role:     conversation.RoleUser,
			Content:  "m",
		})
	}

	return messages
}

// A page is the newest N of what lies below the cursor, and one row past it
// says whether there is a page above without counting what is left.
func TestListMessages_PagesFromTheEndAndSaysWhetherMoreExist(t *testing.T) {
	t.Parallel()

	svc, conversations := newConversationService(&scriptedCompletion{}, testDefinition())
	conversations.messages = numberedMessages(10, 3)
	conversations.count = 13
	before := 13

	page, err := svc.ListMessages(t.Context(), serviceports.ListThreadMessagesRequest{
		Thread:         repositories.GetThreadRequest{ID: conversations.thread.ID},
		Limit:          2,
		BeforeSequence: &before,
	})
	require.NoError(t, err)

	assert.Equal(t, 3, conversations.lastList.Limit, "one more than the page")
	require.NotNil(t, conversations.lastList.BeforeSequence)
	assert.Equal(t, 13, *conversations.lastList.BeforeSequence)
	assert.True(t, page.HasMore)
	require.Len(t, page.Results, 2)
	assert.Equal(t, 11, page.Results[0].Sequence, "the extra row is the oldest, and is dropped")
	assert.Equal(t, 12, page.Results[1].Sequence)
	assert.Equal(t, 13, page.Total)
	assert.Equal(t, maxThreadMessages, page.Limit)
}

func TestListMessages_ReportsTheLastPageAsSuch(t *testing.T) {
	t.Parallel()

	svc, conversations := newConversationService(&scriptedCompletion{}, testDefinition())
	conversations.messages = numberedMessages(0, 2)
	conversations.count = 2

	page, err := svc.ListMessages(t.Context(), serviceports.ListThreadMessagesRequest{
		Thread: repositories.GetThreadRequest{ID: conversations.thread.ID},
		Limit:  2,
	})
	require.NoError(t, err)

	assert.False(t, page.HasMore)
	assert.Len(t, page.Results, 2)
	assert.Nil(t, conversations.lastList.BeforeSequence)
}

func TestListMessages_ClampsThePageSize(t *testing.T) {
	t.Parallel()

	svc, conversations := newConversationService(&scriptedCompletion{}, testDefinition())

	_, err := svc.ListMessages(t.Context(), serviceports.ListThreadMessagesRequest{
		Thread: repositories.GetThreadRequest{ID: conversations.thread.ID},
	})
	require.NoError(t, err)
	assert.Equal(t, defaultPageLimit+1, conversations.lastList.Limit)

	_, err = svc.ListMessages(t.Context(), serviceports.ListThreadMessagesRequest{
		Thread: repositories.GetThreadRequest{ID: conversations.thread.ID},
		Limit:  100000,
	})
	require.NoError(t, err)
	assert.Equal(t, maxPageLimit+1, conversations.lastList.Limit)
}

// A conversation that has reached its length is continued in a new one. The
// turn is refused before anything runs, and the refusal says what to do.
func TestSendMessageStream_RefusesATurnOnAFullThread(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		textTurn("It is in Los Angeles."),
	}}
	svc, conversations := newConversationService(completion, testDefinition())
	conversations.count = maxThreadMessages
	actor := testActor()

	_, err := svc.sendMessage(t.Context(), &serviceports.SendMessageRequest{
		ThreadID:   conversations.thread.ID,
		Content:    "Where is it?",
		TenantInfo: actor.TenantInfo(),
	}, actor, nil)

	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Contains(t, err.Error(), "new conversation")
	assert.Zero(t, conversations.appendCalls, "nothing ran and nothing was saved")
}

// The picker's choice is saved on the thread. A later send that says nothing
// about the model used to wipe it, because "no field" and "automatic" read
// the same; now only a choice replaces a choice, and the person's own model
// is pinned on the turn rather than merely tried first.
func TestSendMessageStream_KeepsTheSavedModelUnlessTheSendChoosesOne(t *testing.T) {
	t.Parallel()

	chosen := pulid.MustNew("aiprv_")
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{textTurn("ok")}}
	svc, conversations := newConversationService(completion, testDefinition())
	conversations.thread.PreferredProviderID = chosen
	actor := testActor()

	_, err := svc.sendMessage(t.Context(), &serviceports.SendMessageRequest{
		ThreadID:   conversations.thread.ID,
		Content:    "hello",
		TenantInfo: actor.TenantInfo(),
	}, actor, nil)
	require.NoError(t, err)

	assert.Equal(
		t,
		chosen,
		conversations.thread.PreferredProviderID,
		"a send with no choice keeps the saved one",
	)
	require.NotNil(t, completion.LastReq)
	assert.Equal(t, chosen, completion.LastReq.PreferredProviderID)
	assert.True(t, completion.LastReq.PinPreferred, "the person's own choice is the model they get")

	_, err = svc.sendMessage(t.Context(), &serviceports.SendMessageRequest{
		ThreadID:       conversations.thread.ID,
		Content:        "hello again",
		TenantInfo:     actor.TenantInfo(),
		ProviderChosen: true,
	}, actor, nil)
	require.NoError(t, err)
	assert.True(
		t,
		conversations.thread.PreferredProviderID.IsNil(),
		"choosing automatic clears the saved model",
	)
	assert.False(t, completion.LastReq.PinPreferred)
}
