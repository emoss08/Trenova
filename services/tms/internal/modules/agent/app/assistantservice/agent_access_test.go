package assistantservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Access is checked on every turn, not only when the conversation began: a
person whose role lost the agent is refused the next message, and the model is
never called.
*/
func TestSendMessage_RefusesATurnWithAnAgentThePersonLostAccessTo(t *testing.T) {
	t.Parallel()

	completion := &assistanttest.ScriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		assistanttest.TextTurn("never reached"),
	}}
	restricted := assistanttest.Definition()
	restricted.AccessMode = agentdefinition.AccessRoles
	svc, conversations := newConversationService(completion, restricted)
	svc.permissions = &assistanttest.SubjectPermissions{
		Allowed: map[string]bool{"assistant:create": true},
	}
	actor := assistanttest.Actor()

	_, err := svc.sendMessage(t.Context(), &assistantports.SendMessageRequest{
		ThreadID:   conversations.Thread.ID,
		Content:    "What did we pay drivers last week?",
		TenantInfo: actor.TenantInfo(),
	}, actor, nil)

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
	assert.Zero(t, completion.CallCount, "a refused turn never reaches the model")
	assert.Empty(t, conversations.Appended, "nothing is saved for a refused turn")
}

func TestContinueRefusal_EveryReasonIsValid(t *testing.T) {
	t.Parallel()

	for _, reason := range []conversation.ContinueRefusal{
		conversation.ContinueAgentDeleted,
		conversation.ContinueAgentDisabled,
		conversation.ContinueAgentNotConversational,
		conversation.ContinueNoAccess,
	} {
		assert.True(t, reason.IsValid(), reason)
	}
	assert.False(t, conversation.ContinueRefusal("").IsValid())
}

/*
A delegate on the allowlist that the person may not use is declined when the
task is handed over, with a reason the agent can pass on; the parent is never
a way round the delegate's own access.
*/
func TestOpenDelegate_DeclinesADelegateThePersonMayNotUse(t *testing.T) {
	t.Parallel()

	f := newDelegateFixture()
	f.delegate.AccessMode = agentdefinition.AccessRoles

	opened, err := f.svc.OpenDelegate(t.Context(), f.request)

	assert.Nil(t, opened)
	refusal, ok := IsDelegateDeclined(err)
	require.True(t, ok, "a refusal is declined, not an error to retry: %v", err)
	assert.Contains(t, refusal.Reason, "may not use Report Builder")

	f.perms.Granted = []pulid.ID{f.delegate.ID}
	opened, err = f.svc.OpenDelegate(t.Context(), f.request)
	require.NoError(t, err)
	assert.Equal(t, f.delegate.ID, opened.Request.Definition.ID)
}
