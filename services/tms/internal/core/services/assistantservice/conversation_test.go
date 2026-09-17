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
}

func (s *stubConversations) GetThread(
	context.Context,
	repositories.GetThreadRequest,
) (*conversation.Thread, error) {
	return s.thread, nil
}

func (s *stubConversations) ListMessages(
	context.Context,
	repositories.ListMessagesRequest,
) ([]conversation.Message, error) {
	return nil, nil
}

func (s *stubConversations) AppendTurn(
	_ context.Context,
	req repositories.AppendTurnRequest,
) ([]conversation.Message, error) {
	s.appended = req.Messages
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

	result, err := svc.SendMessageStream(t.Context(), &serviceports.SendMessageRequest{
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

	_, err := svc.SendMessageStream(t.Context(), &serviceports.SendMessageRequest{
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
