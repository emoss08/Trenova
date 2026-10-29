package assistantthreads

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func startAs(
	t *testing.T,
	svc *Service,
	req *assistantports.StartThreadRequest,
	actor *permissionports.RequestActor,
) (*conversation.Thread, error) {
	t.Helper()

	req.TenantInfo = actor.TenantInfo()

	return svc.StartThread(t.Context(), req, actor)
}

/*
An agent restricted to roles is refused to a person none of whose roles is
granted it, before anything is stored; granting one of their roles the agent
lets the same request through.
*/
func TestStartThread_RefusesAnAgentThePersonMayNotUse(t *testing.T) {
	t.Parallel()

	restricted := assistanttest.RestrictedChatDefinition("Payroll Helper")
	svc, conversations, permissions := subjectService(restricted)

	_, err := startAs(t, svc, &assistantports.StartThreadRequest{
		AgentDefinitionID: restricted.ID,
	}, assistanttest.Actor())

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
	assert.Contains(t, err.Error(), "Payroll Helper")
	assert.Nil(t, conversations.Created, "nothing is stored for a refused agent")
	assert.Equal(t, []pulid.ID{restricted.ID}, permissions.AgentChecks)

	permissions.Granted = []pulid.ID{restricted.ID}
	thread, err := startAs(t, svc, &assistantports.StartThreadRequest{
		AgentDefinitionID: restricted.ID,
	}, assistanttest.Actor())

	require.NoError(t, err)
	assert.True(t, thread.CanContinue)
}

// A person who may not use the assistant at all is refused even an agent open
// to everyone.
func TestStartThread_RefusesEveryAgentWithoutTheAssistant(t *testing.T) {
	t.Parallel()

	open := assistanttest.ChatDefinition("Dispatch", "")
	svc, conversations, permissions := subjectService(open)
	permissions.Allowed = map[string]bool{}

	_, err := startAs(t, svc, &assistantports.StartThreadRequest{
		AgentDefinitionID: open.ID,
	}, assistanttest.Actor())

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
	assert.Nil(t, conversations.Created)
}

type continuableFixture struct {
	svc         *Service
	thread      *conversation.Thread
	permissions *assistanttest.SubjectPermissions
	agent       *agentdefinition.Definition
}

func newContinuableFixture() *continuableFixture {
	restricted := assistanttest.RestrictedChatDefinition("Payroll Helper")
	thread := &conversation.Thread{
		ID:                pulid.MustNew("athr_"),
		AgentDefinitionID: restricted.ID,
		Title:             "Last week's pay",
	}
	permissions := &assistanttest.SubjectPermissions{
		Allowed: map[string]bool{"assistant:create": true},
	}

	return &continuableFixture{
		svc: &Service{
			logger:        zap.NewNop(),
			conversations: &assistanttest.StubConversations{Thread: thread},
			definitions: &assistanttest.DelegateDefinitions{
				ByID: map[pulid.ID]*agentdefinition.Definition{
					restricted.ID: restricted,
				},
			},
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
	actor := assistanttest.Actor()
	req := conversationports.GetThreadRequest{
		ID:         f.thread.ID,
		UserID:     actor.UserID,
		TenantInfo: actor.TenantInfo(),
	}

	thread, err := f.svc.GetThread(t.Context(), req)
	require.NoError(t, err)
	require.NotNil(t, thread, "the conversation is still served")
	assert.False(t, thread.CanContinue)

	f.permissions.Granted = []pulid.ID{f.agent.ID}
	thread, err = f.svc.GetThread(t.Context(), req)
	require.NoError(t, err)
	assert.True(t, thread.CanContinue)

	f.agent.Enabled = false
	thread, err = f.svc.GetThread(t.Context(), req)
	require.NoError(t, err)
	assert.False(t, thread.CanContinue, "a disabled agent cannot be asked anything")
}

/*
A conversation that cannot continue says why, so a disabled agent is not
reported as lost access. An agent the reader may not use is refused as that
before it is said to be off, a removed agent is removed, and one moved to a
schedule takes no conversations. A conversation that can continue carries no
reason.
*/
func TestGetThread_SaysWhyItCannotContinue(t *testing.T) {
	t.Parallel()

	f := newContinuableFixture()
	actor := assistanttest.Actor()
	req := conversationports.GetThreadRequest{
		ID:         f.thread.ID,
		UserID:     actor.UserID,
		TenantInfo: actor.TenantInfo(),
	}
	read := func() *conversation.Thread {
		t.Helper()
		thread, err := f.svc.GetThread(t.Context(), req)
		require.NoError(t, err)
		require.NotNil(t, thread)

		return thread
	}

	f.agent.Enabled = false
	thread := read()
	assert.False(t, thread.CanContinue)
	assert.Equal(t, conversation.ContinueNoAccess, thread.CannotContinueReason,
		"an agent the reader may not use is refused as such, off or not")

	f.permissions.Granted = []pulid.ID{f.agent.ID}
	thread = read()
	assert.False(t, thread.CanContinue)
	assert.Equal(t, conversation.ContinueAgentDisabled, thread.CannotContinueReason)

	f.agent.Enabled = true
	f.agent.TriggerMode = agentdefinition.TriggerScheduled
	thread = read()
	assert.False(t, thread.CanContinue)
	assert.Equal(t, conversation.ContinueAgentNotConversational, thread.CannotContinueReason)

	f.agent.TriggerMode = agentdefinition.TriggerChat
	thread = read()
	assert.True(t, thread.CanContinue)
	assert.Empty(t, thread.CannotContinueReason)

	definitions, ok := f.svc.definitions.(*assistanttest.DelegateDefinitions)
	require.True(t, ok)
	delete(definitions.ByID, f.agent.ID)
	thread = read()
	assert.False(t, thread.CanContinue)
	assert.Equal(t, conversation.ContinueAgentDeleted, thread.CannotContinueReason)
}

func TestGetThread_CannotContinueWithoutTheAssistant(t *testing.T) {
	t.Parallel()

	f := newContinuableFixture()
	f.agent.AccessMode = agentdefinition.AccessEveryone
	f.permissions.Allowed = map[string]bool{}
	actor := assistanttest.Actor()

	thread, err := f.svc.GetThread(t.Context(), conversationports.GetThreadRequest{
		ID:         f.thread.ID,
		UserID:     actor.UserID,
		TenantInfo: actor.TenantInfo(),
	})

	require.NoError(t, err)
	assert.False(t, thread.CanContinue)
	assert.Equal(t, conversation.ContinueNoAccess, thread.CannotContinueReason)
}

/*
A quick question goes only to the general assistant, and only when the person
may use it. Restricted to roles they do not hold, it is not swapped for an open
chat agent that would answer from its own desk's records: the person is told an
administrator can grant it. The read is narrowed to their audience either way.
*/
func TestAsk_ChoosesTheGeneralAssistantOnlyWhenThePersonMayUseIt(t *testing.T) {
	t.Parallel()

	svc := &Service{logger: zap.NewNop()}
	general := assistanttest.ChatDefinition("Assistant", agenttemplate.TemplateGeneralAssistant)
	general.AccessMode = agentdefinition.AccessRoles
	dispatch := assistanttest.ChatDefinition("Dispatch", agenttemplate.TemplateDispatchAssistant)
	definitions := &assistanttest.StubDefinitionList{
		Definitions: []*agentdefinition.Definition{general, dispatch},
	}
	svc.definitions = definitions
	permissions := &assistanttest.SubjectPermissions{
		Allowed: map[string]bool{"assistant:create": true},
	}
	svc.permissions = permissions
	actor := assistanttest.Actor()

	_, err := svc.askAgent(t.Context(), actor.TenantInfo(), actor)
	require.Error(t, err)
	assert.True(
		t,
		errortypes.IsAuthorizationError(err),
		"the open dispatch agent does not stand in",
	)
	chooser := definitions.Requests[0]
	require.NotNil(t, chooser.Audience, "the read is narrowed to the person's agents")
	assert.Empty(t, chooser.Audience.GrantedAgentIDs)

	permissions.Granted = []pulid.ID{general.ID}
	chosen, err := svc.askAgent(t.Context(), actor.TenantInfo(), actor)
	require.NoError(t, err)
	assert.Equal(t, general.ID, chosen.ID, "once granted, the general assistant answers")

	permissions.Allowed = map[string]bool{}
	_, err = svc.askAgent(t.Context(), actor.TenantInfo(), actor)
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
}
