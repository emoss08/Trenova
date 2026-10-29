package assistantservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/turnlimits"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newConversationService(
	completion *assistanttest.ScriptedCompletion,
	definition *agentdefinition.Definition,
) (*Service, *assistanttest.StubConversations) {
	svc := newService(
		completion,
		&assistanttest.StubQueryRegistry{},
		&assistanttest.StubActionRegistry{},
	)
	conversations := &assistanttest.StubConversations{Thread: &conversation.Thread{
		ID:                pulid.MustNew("athr_"),
		AgentDefinitionID: pulid.MustNew("agdef_"),
		Title:             "Dispatch",
	}}
	svc.conversations = conversations
	svc.definitions = &assistanttest.StubDefinitions{Definition: definition}

	return svc, conversations
}

func TestSendMessageStream_StoresThePageContextOnTheUserTurn(t *testing.T) {
	t.Parallel()

	completion := &assistanttest.ScriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		assistanttest.TextTurn("It is at the Los Angeles terminal."),
	}}
	svc, conversations := newConversationService(completion, assistanttest.Definition())
	actor := assistanttest.Actor()

	result, err := svc.sendMessage(t.Context(), &assistantports.SendMessageRequest{
		ThreadID: conversations.Thread.ID,
		Content:  "Where is this shipment?",
		Page: &agentpagecontext.PageContext{
			Path:       " /shipments?panelEntityId=shp_1 ",
			EntityType: "shipment",
			EntityID:   "shp_1",
			Title:      "Shipments | Acme",
		},
		TenantInfo: actor.TenantInfo(),
	}, actor, nil)
	require.NoError(t, err)
	assert.False(t, result.Refused)

	require.NotEmpty(t, conversations.Appended)
	user := conversations.Appended[0]
	assert.Equal(t, conversation.RoleUser, user.Role)
	require.NotNil(t, user.PageContext, "the user turn carries the page it was asked from")
	assert.Equal(t, "/shipments?panelEntityId=shp_1", user.PageContext.Path)
	assert.Equal(t, "shipment", user.PageContext.EntityType)
	assert.Equal(t, "shp_1", user.PageContext.EntityID)

	for _, message := range conversations.Appended[1:] {
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

	completion := &assistanttest.ScriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		assistanttest.TextTurn("never reached"),
	}}
	svc, conversations := newConversationService(completion, assistanttest.Definition())
	actor := assistanttest.Actor()

	_, err := svc.sendMessage(t.Context(), &assistantports.SendMessageRequest{
		ThreadID: conversations.Thread.ID,
		Content:  "Where is this?",
		Page: &agentpagecontext.PageContext{
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
	assert.Empty(t, conversations.Appended, "nothing is saved for a rejected request")
}

// A conversation that has reached its length is continued in a new one. The
// turn is refused before anything runs, and the refusal says what to do.
func TestSendMessageStream_RefusesATurnOnAFullThread(t *testing.T) {
	t.Parallel()

	completion := &assistanttest.ScriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		assistanttest.TextTurn("It is in Los Angeles."),
	}}
	svc, conversations := newConversationService(completion, assistanttest.Definition())
	conversations.Count = turnlimits.MaxThreadMessages
	actor := assistanttest.Actor()

	_, err := svc.sendMessage(t.Context(), &assistantports.SendMessageRequest{
		ThreadID:   conversations.Thread.ID,
		Content:    "Where is it?",
		TenantInfo: actor.TenantInfo(),
	}, actor, nil)

	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Contains(t, err.Error(), "new conversation")
	assert.Zero(t, conversations.AppendCalls, "nothing ran and nothing was saved")
}

// The picker's choice is saved on the thread. A later send that says nothing
// about the model used to wipe it, because "no field" and "automatic" read
// the same; now only a choice replaces a choice, and the person's own model
// is pinned on the turn rather than merely tried first.
func TestSendMessageStream_KeepsTheSavedModelUnlessTheSendChoosesOne(t *testing.T) {
	t.Parallel()

	chosen := pulid.MustNew("aiprv_")
	completion := &assistanttest.ScriptedCompletion{
		Turns: []*llmports.ChatCompletionResult{assistanttest.TextTurn("ok")},
	}
	svc, conversations := newConversationService(completion, assistanttest.Definition())
	conversations.Thread.PreferredProviderID = chosen
	actor := assistanttest.Actor()

	_, err := svc.sendMessage(t.Context(), &assistantports.SendMessageRequest{
		ThreadID:   conversations.Thread.ID,
		Content:    "hello",
		TenantInfo: actor.TenantInfo(),
	}, actor, nil)
	require.NoError(t, err)

	assert.Equal(
		t,
		chosen,
		conversations.Thread.PreferredProviderID,
		"a send with no choice keeps the saved one",
	)
	require.NotNil(t, completion.LastReq)
	assert.Equal(t, chosen, completion.LastReq.PreferredProviderID)
	assert.True(t, completion.LastReq.PinPreferred, "the person's own choice is the model they get")

	_, err = svc.sendMessage(t.Context(), &assistantports.SendMessageRequest{
		ThreadID:       conversations.Thread.ID,
		Content:        "hello again",
		TenantInfo:     actor.TenantInfo(),
		ProviderChosen: true,
	}, actor, nil)
	require.NoError(t, err)
	assert.True(
		t,
		conversations.Thread.PreferredProviderID.IsNil(),
		"choosing automatic clears the saved model",
	)
	assert.False(t, completion.LastReq.PinPreferred)
}
