package assistantservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func restrictedChatDefinition(name string) *agentdefinition.Definition {
	definition := chatDefinition(name, "")
	definition.AccessMode = agentdefinition.AccessRoles

	return definition
}

/*
An agent restricted to roles is refused to a person none of whose roles is
granted it, before anything is stored; granting one of their roles the agent
lets the same request through.
*/
func TestStartThread_RefusesAnAgentThePersonMayNotUse(t *testing.T) {
	t.Parallel()

	restricted := restrictedChatDefinition("Payroll Helper")
	svc, conversations, permissions := subjectService(restricted)

	_, err := svc.StartThread(t.Context(), &serviceports.StartThreadRequest{
		AgentDefinitionID: restricted.ID,
	}, testActor())

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
	assert.Contains(t, err.Error(), "Payroll Helper")
	assert.Nil(t, conversations.created, "nothing is stored for a refused agent")
	assert.Equal(t, []pulid.ID{restricted.ID}, permissions.agentChecks)

	permissions.granted = []pulid.ID{restricted.ID}
	thread, err := svc.StartThread(t.Context(), &serviceports.StartThreadRequest{
		AgentDefinitionID: restricted.ID,
	}, testActor())

	require.NoError(t, err)
	assert.True(t, thread.CanContinue)
}

// A person who may not use the assistant at all is refused even an agent open
// to everyone.
func TestStartThread_RefusesEveryAgentWithoutTheAssistant(t *testing.T) {
	t.Parallel()

	open := chatDefinition("Dispatch", "")
	svc, conversations, permissions := subjectService(open)
	permissions.allowed = map[string]bool{}

	_, err := svc.StartThread(t.Context(), &serviceports.StartThreadRequest{
		AgentDefinitionID: open.ID,
	}, testActor())

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
	assert.Nil(t, conversations.created)
}

/*
Access is checked on every turn, not only when the conversation began: a
person whose role lost the agent is refused the next message, and the model is
never called.
*/
func TestSendMessage_RefusesATurnWithAnAgentThePersonLostAccessTo(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		textTurn("never reached"),
	}}
	restricted := testDefinition()
	restricted.AccessMode = agentdefinition.AccessRoles
	svc, conversations := newConversationService(completion, restricted)
	svc.permissions = &subjectPermissions{allowed: map[string]bool{"assistant:create": true}}
	actor := testActor()

	_, err := svc.sendMessage(t.Context(), &serviceports.SendMessageRequest{
		ThreadID:   conversations.thread.ID,
		Content:    "What did we pay drivers last week?",
		TenantInfo: actor.TenantInfo(),
	}, actor, nil)

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
	assert.Zero(t, completion.CallCount, "a refused turn never reaches the model")
	assert.Empty(t, conversations.appended, "nothing is saved for a refused turn")
}

type continuableFixture struct {
	svc         *Service
	thread      *conversation.Thread
	permissions *subjectPermissions
	agent       *agentdefinition.Definition
}

func newContinuableFixture() *continuableFixture {
	restricted := restrictedChatDefinition("Payroll Helper")
	thread := &conversation.Thread{
		ID:                pulid.MustNew("athr_"),
		AgentDefinitionID: restricted.ID,
		Title:             "Last week's pay",
	}
	permissions := &subjectPermissions{allowed: map[string]bool{"assistant:create": true}}

	return &continuableFixture{
		svc: &Service{
			logger:        zap.NewNop(),
			conversations: &stubConversations{thread: thread},
			definitions: &delegateDefinitions{byID: map[pulid.ID]*agentdefinition.Definition{
				restricted.ID: restricted,
			}},
			permissions: permissions,
		},
		thread:      thread,
		permissions: permissions,
		agent:       restricted,
	}
}

/*
A conversation with an agent the person lost access to stays readable: it is
served, with canContinue false so the reader offers no composer. Granting the
agent back makes it continuable again, and a disabled agent never is.
*/
func TestGetThread_StaysReadableAndSaysWhetherItCanContinue(t *testing.T) {
	t.Parallel()

	f := newContinuableFixture()
	actor := testActor()
	req := repositories.GetThreadRequest{
		ID:         f.thread.ID,
		UserID:     actor.UserID,
		TenantInfo: actor.TenantInfo(),
	}

	thread, err := f.svc.GetThread(t.Context(), req)
	require.NoError(t, err)
	require.NotNil(t, thread, "the conversation is still served")
	assert.False(t, thread.CanContinue)

	f.permissions.granted = []pulid.ID{f.agent.ID}
	thread, err = f.svc.GetThread(t.Context(), req)
	require.NoError(t, err)
	assert.True(t, thread.CanContinue)

	f.agent.Enabled = false
	thread, err = f.svc.GetThread(t.Context(), req)
	require.NoError(t, err)
	assert.False(t, thread.CanContinue, "a disabled agent cannot be asked anything")
}

func TestGetThread_CannotContinueWithoutTheAssistant(t *testing.T) {
	t.Parallel()

	f := newContinuableFixture()
	f.agent.AccessMode = agentdefinition.AccessEveryone
	f.permissions.allowed = map[string]bool{}
	actor := testActor()

	thread, err := f.svc.GetThread(t.Context(), repositories.GetThreadRequest{
		ID:         f.thread.ID,
		UserID:     actor.UserID,
		TenantInfo: actor.TenantInfo(),
	})

	require.NoError(t, err)
	assert.False(t, thread.CanContinue)
}

/*
A quick question goes only to an agent the person may use. The general
assistant restricted to roles they do not hold is passed over for an open chat
agent, the read is narrowed to their audience, and a person left with no agent
at all is told an administrator can grant one rather than that none exists.
*/
func TestAsk_ChoosesOnlyAnAgentThePersonMayUse(t *testing.T) {
	t.Parallel()

	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{textTurn("ok")}}
	svc := newService(completion, &stubQueryRegistry{}, &stubActionRegistry{})
	general := chatDefinition("Assistant", agentdefinition.TemplateGeneralAssistant)
	general.AccessMode = agentdefinition.AccessRoles
	dispatch := chatDefinition("Dispatch", agentdefinition.TemplateDispatchAssistant)
	definitions := &stubDefinitionList{
		definitions: []*agentdefinition.Definition{general, dispatch},
	}
	svc.definitions = definitions
	permissions := &subjectPermissions{allowed: map[string]bool{"assistant:create": true}}
	svc.permissions = permissions
	actor := testActor()

	chosen, err := svc.askAgent(t.Context(), actor.TenantInfo(), actor)
	require.NoError(t, err)
	assert.Equal(t, dispatch.ID, chosen.ID, "the restricted general assistant is passed over")
	require.NotNil(t, definitions.last.Audience, "the read is narrowed to the person's agents")
	assert.Empty(t, definitions.last.Audience.GrantedAgentIDs)

	permissions.granted = []pulid.ID{general.ID}
	chosen, err = svc.askAgent(t.Context(), actor.TenantInfo(), actor)
	require.NoError(t, err)
	assert.Equal(t, general.ID, chosen.ID, "once granted, the general assistant answers")

	permissions.granted = nil
	dispatch.AccessMode = agentdefinition.AccessRoles
	_, err = svc.askAgent(t.Context(), actor.TenantInfo(), actor)
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))

	permissions.allowed = map[string]bool{}
	_, err = svc.askAgent(t.Context(), actor.TenantInfo(), actor)
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
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

	f.perms.granted = []pulid.ID{f.delegate.ID}
	opened, err = f.svc.OpenDelegate(t.Context(), f.request)
	require.NoError(t, err)
	assert.Equal(t, f.delegate.ID, opened.Request.Definition.ID)
}
