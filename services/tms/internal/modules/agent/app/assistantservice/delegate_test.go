package assistantservice

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/delegation"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmark"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type delegateFixture struct {
	svc      *Service
	parent   *agentdefinition.Definition
	delegate *agentdefinition.Definition
	perms    *assistanttest.SubjectPermissions
	request  *OpenDelegateRequest
}

func newDelegateFixture() *delegateFixture {
	svc := newService(
		&assistanttest.ScriptedCompletion{},
		&assistanttest.StubQueryRegistry{},
		&assistanttest.StubActionRegistry{},
	)
	delegate := &agentdefinition.Definition{
		ID:              pulid.MustNew("agdef_"),
		Name:            "Report Builder",
		AutonomyCeiling: agent.TierPropose,
		Enabled:         true,
	}
	delegate.ApplyDefaults()
	parent := &agentdefinition.Definition{
		ID:              pulid.MustNew("agdef_"),
		Name:            "Homepage Widget Builder",
		AutonomyCeiling: agent.TierPropose,
		Enabled:         true,
		DelegateIDs:     []pulid.ID{delegate.ID},
	}
	parent.ApplyDefaults()
	perms := &assistanttest.SubjectPermissions{Allowed: map[string]bool{"assistant:create": true}}
	svc.permissions = perms
	svc.definitions = &assistanttest.DelegateDefinitions{
		ByID: map[pulid.ID]*agentdefinition.Definition{
			parent.ID:   parent,
			delegate.ID: delegate,
		},
	}

	actor := &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	return &delegateFixture{
		svc:      svc,
		parent:   parent,
		delegate: delegate,
		perms:    perms,
		request: &OpenDelegateRequest{
			Parent:   parent,
			Actor:    actor,
			ThreadID: pulid.MustNew("athr_"),
			StepOwner: agentruntimeports.RunStepOwner{
				Kind: agentruntimeports.RunStepOwnerAssistantTurn,
				ID:   pulid.MustNew("atrn_"),
			},
			Call: delegation.Call{
				Call:      llmports.ToolCall{ID: "call_1", Name: "delegate_task"},
				Delegate:  agentturn.RuntimeDelegate{ID: delegate.ID, Name: delegate.Name},
				Task:      "Create the on-time report and return its id.",
				StepScope: "scope-1",
			},
		},
	}
}

// The delegate's turn is opened as the delegate, as the same person, marked as
// working for the agent that asked: it never holds delegate_task, and it is
// not unattended, so a write that is the person's own still runs as theirs.
var clockLinePrefix = regexp.MustCompile(`^Now: [A-Z][a-z]+ \d{4}-\d{2}-\d{2} \d{2}:\d{2} \S+\n\n`)

func TestOpenDelegate_OpensTheDelegatesOwnTurnForTheSamePerson(t *testing.T) {
	t.Parallel()

	f := newDelegateFixture()

	opened, err := f.svc.OpenDelegate(t.Context(), f.request)
	require.NoError(t, err)

	req := opened.Request
	assert.Equal(t, f.delegate.ID, req.Definition.ID)
	assert.Same(t, f.request.Actor, req.Actor)
	assert.False(t, req.Unattended)
	assert.Equal(t, f.request.ThreadID, req.ThreadID)
	assert.Equal(t, f.request.StepOwner, req.StepOwner, "it shares the turn's ledger")
	require.NotNil(t, req.Delegation)
	assert.Equal(t, "call_1", req.Delegation.CallID)
	assert.Equal(t, "scope-1", req.Delegation.StepScope)
	assert.Equal(t, f.parent.ID, req.Delegation.ParentAgentID)
	assert.NotContains(t, opened.Turn.Held, "delegate_task")
	assert.True(t, req.Delegation.ReadOnly, "a task the agent hands out only looks things up")
	assert.False(t, req.Publishes)
	assert.Empty(t, req.Proposals)
	assert.Contains(
		t,
		opened.Turn.System,
		"The agent Homepage Widget Builder handed you this lookup",
	)
	assert.Contains(t, opened.Turn.System, "you change nothing")
	asked := opened.Turn.Messages[len(opened.Turn.Messages)-1].Content
	assert.Regexp(t, clockLinePrefix, asked, "the delegate reads the time like any turn")
	assert.Equal(t, f.request.Call.Task, clockLinePrefix.ReplaceAllString(asked, ""))
	assert.Contains(t, f.perms.Asked, "assistant:create", "the person must be allowed to use it")
}

// A task handed over from a palette question is answered inline too: what the
// delegate returns reaches a reply with nothing beside it.
func TestOpenDelegate_KeepsAPaletteQuestionsDelegateInline(t *testing.T) {
	t.Parallel()

	f := newDelegateFixture()
	f.request.Inline = true

	opened, err := f.svc.OpenDelegate(t.Context(), f.request)
	require.NoError(t, err)

	assert.True(t, opened.Request.Inline)
	assert.False(t, opened.Request.Publishes)
	assert.False(t, opened.Request.KeepsDocuments())
	assert.NotContains(t, opened.Turn.Held, "publish_artifact")
}

// Every check is made again when the task is handed over, against the records
// as they are now.
func TestOpenDelegate_DeclinesWhatThePersonOrTheAllowlistNoLongerAllows(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		change func(*delegateFixture)
		reason string
	}{
		"taken off the allowlist": {
			change: func(f *delegateFixture) { f.parent.DelegateIDs = nil },
			reason: "is not one of the agents",
		},
		"disabled": {
			change: func(f *delegateFixture) { f.delegate.Enabled = false },
			reason: "Report Builder is disabled",
		},
		"runs on its own": {
			change: func(f *delegateFixture) {
				f.delegate.TriggerMode = agentdefinition.TriggerScheduled
			},
			reason: "runs on its own",
		},
		"deleted": {
			change: func(f *delegateFixture) {
				delete(f.svc.definitions.(*assistanttest.DelegateDefinitions).ByID, f.delegate.ID)
			},
			reason: "no longer exists",
		},
		"the person may not use the assistant": {
			change: func(f *delegateFixture) { f.perms.Allowed = map[string]bool{} },
			reason: "not allowed to use other agents",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			f := newDelegateFixture()
			tt.change(f)

			opened, err := f.svc.OpenDelegate(t.Context(), f.request)
			assert.Nil(t, opened)
			refusal, ok := IsDelegateDeclined(err)
			require.True(t, ok, "a refusal is declined, not an error to retry: %v", err)
			assert.Contains(t, refusal.Reason, tt.reason)
		})
	}
}

// A task the person handed the delegate from the conversation opens though
// the conversation's agent does not list it, and is refused for everything
// else exactly as a hand-off is.
func TestOpenDelegate_ADirectedTaskNeedsNoPlaceOnTheAllowlist(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		change func(*delegateFixture)
		reason string
	}{
		"not on the allowlist": {
			change: func(f *delegateFixture) { f.parent.DelegateIDs = nil },
		},
		"disabled": {
			change: func(f *delegateFixture) {
				f.parent.DelegateIDs = nil
				f.delegate.Enabled = false
			},
			reason: "Report Builder is disabled",
		},
		"runs on its own": {
			change: func(f *delegateFixture) {
				f.delegate.TriggerMode = agentdefinition.TriggerScheduled
			},
			reason: "runs on its own",
		},
		"the person may not use the assistant": {
			change: func(f *delegateFixture) {
				f.parent.DelegateIDs = nil
				f.perms.Allowed = map[string]bool{}
			},
			reason: "not allowed to use other agents",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			f := newDelegateFixture()
			f.request.Call.Directed = true
			tt.change(f)

			opened, err := f.svc.OpenDelegate(t.Context(), f.request)
			if tt.reason == "" {
				require.NoError(t, err)
				assert.Equal(t, f.delegate.ID, opened.Request.Definition.ID)
				require.NotNil(t, opened.Request.Delegation)
				assert.Equal(t, f.parent.ID, opened.Request.Delegation.ParentAgentID)
				assert.NotContains(t, opened.Turn.Held, "delegate_task")
				return
			}
			assert.Nil(t, opened)
			refusal, ok := IsDelegateDeclined(err)
			require.True(t, ok, "a refusal is declined, not an error to retry: %v", err)
			assert.Contains(t, refusal.Reason, tt.reason)
		})
	}
}

// The agent a person hands a message to is checked as one they may talk to
// before the turn opens, so a refusal is the request's, not a reply's. The
// conversation's agent need not list it.
func TestDirectedTask_ChecksTheChosenAgentAsThePersonsOwn(t *testing.T) {
	t.Parallel()

	send := func(f *delegateFixture, agentID pulid.ID) (*agentruntimeports.DirectedTask, error) {
		thread := &conversation.Thread{
			ID:                f.request.ThreadID,
			AgentDefinitionID: f.parent.ID,
		}
		return f.svc.directedTask(t.Context(), thread, f.request.Actor,
			&assistantports.SendMessageRequest{
				ThreadID:        thread.ID,
				Content:         "  Mark shipment SEED-PAY-001 ready to invoice.  ",
				TenantInfo:      f.request.Actor.TenantInfo(),
				DirectedAgentID: agentID,
			})
	}

	t.Run("a message for the conversation's own agent names none", func(t *testing.T) {
		t.Parallel()

		f := newDelegateFixture()
		task, err := send(f, pulid.Nil)
		require.NoError(t, err)
		assert.Nil(t, task)
	})

	t.Run("an agent off the allowlist the person may use", func(t *testing.T) {
		t.Parallel()

		f := newDelegateFixture()
		f.parent.DelegateIDs = nil
		task, err := send(f, f.delegate.ID)
		require.NoError(t, err)
		require.NotNil(t, task)
		assert.Equal(t, f.delegate.ID, task.Delegate.ID)
		assert.Equal(t, "Report Builder", task.Delegate.Name)
		assert.Equal(t, "Mark shipment SEED-PAY-001 ready to invoice.", task.Task)
		assert.Contains(t, f.perms.AgentChecks, f.delegate.ID)
	})

	t.Run("the conversation's own agent", func(t *testing.T) {
		t.Parallel()

		f := newDelegateFixture()
		_, err := send(f, f.parent.ID)
		require.Error(t, err)
		assert.True(t, errortypes.IsError(err), "a field error on directedAgentId: %v", err)
	})

	t.Run("a disabled agent", func(t *testing.T) {
		t.Parallel()

		f := newDelegateFixture()
		f.delegate.Enabled = false
		_, err := send(f, f.delegate.ID)
		require.Error(t, err)
		assert.True(t, errortypes.IsBusinessError(err))
		assert.Contains(t, err.Error(), "disabled")
	})

	t.Run("an agent that runs on its own", func(t *testing.T) {
		t.Parallel()

		f := newDelegateFixture()
		f.delegate.TriggerMode = agentdefinition.TriggerScheduled
		_, err := send(f, f.delegate.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "runs on its own")
	})

	t.Run("an agent the person may not use", func(t *testing.T) {
		t.Parallel()

		f := newDelegateFixture()
		f.perms.Allowed = map[string]bool{}
		_, err := send(f, f.delegate.ID)
		require.Error(t, err)
		assert.True(t, errortypes.IsAuthorizationError(err), "refused for access: %v", err)
	})

	t.Run("an agent that is gone", func(t *testing.T) {
		t.Parallel()

		f := newDelegateFixture()
		_, err := send(f, pulid.MustNew("agdef_"))
		require.Error(t, err)
		assert.True(t, errortypes.IsNotFoundError(err))
	})
}

// A delegate's steps and the account of its task are served with the agent's
// name and mark, read for the whole page in one query. An agent with no icon
// of its own is served none, so the reader draws its initials; one deleted
// since keeps what the turn recorded.
func TestNameDelegatedSteps_ServesTheDelegatesMarkInOneQuery(t *testing.T) {
	t.Parallel()

	f := newDelegateFixture()
	f.delegate.Icon = agentmark.IconReceipt
	f.delegate.Accent = agentmark.AccentTeal
	plain := &agentdefinition.Definition{ID: pulid.MustNew("agdef_"), Name: "Plain Agent"}
	deleted := pulid.MustNew("agdef_")
	definitions := f.svc.definitions.(*assistanttest.DelegateDefinitions)
	definitions.ByID[plain.ID] = plain

	messages := []conversation.Message{
		{Role: conversation.RoleUser, Content: "Build it."},
		{
			Role:              conversation.RoleUser,
			Kind:              conversation.MessageKindDelegated,
			AgentDefinitionID: f.delegate.ID,
			DelegateCallID:    "call_1",
		},
		{
			Role:              conversation.RoleAssistant,
			Kind:              conversation.MessageKindDelegated,
			AgentDefinitionID: plain.ID,
			DelegateCallID:    "call_2",
		},
		{
			Role:       conversation.RoleTool,
			ToolName:   "delegate_task",
			ToolCallID: "call_1",
			DelegateReport: &conversation.DelegateReport{
				DelegateCallID: "call_1",
				AgentID:        f.delegate.ID,
				AgentName:      "Report Builder",
				Status:         conversation.DelegateStatusCompleted,
			},
		},
		{
			Role:       conversation.RoleTool,
			ToolName:   "delegate_task",
			ToolCallID: "call_3",
			DelegateReport: &conversation.DelegateReport{
				DelegateCallID: "call_3",
				AgentID:        deleted,
				AgentName:      "Gone Agent",
				Icon:           agentmark.IconTruck,
				Accent:         agentmark.AccentRose,
				Status:         conversation.DelegateStatusCompleted,
			},
		},
	}

	f.svc.attribution().NameDelegatedSteps(t.Context(), f.request.Actor.TenantInfo(), messages)

	assert.Equal(t, 1, definitions.Listed, "one query for the whole page")

	assert.Empty(t, messages[0].AgentName, "the conversation's own messages are left alone")
	assert.Equal(t, "Report Builder", messages[1].AgentName)
	assert.Equal(t, agentmark.IconReceipt, messages[1].AgentIcon)
	assert.Equal(t, agentmark.AccentTeal, messages[1].AgentAccent)

	assert.Equal(t, "Plain Agent", messages[2].AgentName)
	assert.Empty(t, messages[2].AgentIcon, "no icon of its own: the reader draws its initials")
	assert.Equal(t, plain.ResolvedAccent(), messages[2].AgentAccent)

	assert.Equal(t, agentmark.IconReceipt, messages[3].DelegateReport.Icon)
	assert.Equal(t, agentmark.AccentTeal, messages[3].DelegateReport.Accent)

	assert.Equal(t, agentmark.IconTruck, messages[4].DelegateReport.Icon,
		"a deleted agent keeps the mark the turn recorded")
	assert.Equal(t, agentmark.AccentRose, messages[4].DelegateReport.Accent)
}

type capturingContexts struct {
	asked *agentruntimeports.RuntimeContextRequest
}

func (c *capturingContexts) Build(
	_ context.Context,
	req *agentruntimeports.RuntimeContextRequest,
) (agentturn.RuntimeContext, error) {
	c.asked = req

	return agentturn.RuntimeContext{
		Trigger:          req.Trigger,
		DelegatedBy:      req.DelegatedBy,
		DelegatorRecords: req.DelegatorRecords,
	}, nil
}

// The task is about what the person was looking at when they asked, so the
// agent handed it reads the memories of those records too.
func TestOpenDelegate_HandsTheDelegateTheRecordsTheTurnIsAbout(t *testing.T) {
	t.Parallel()

	f := newDelegateFixture()
	contexts := &capturingContexts{}
	f.svc.contexts = contexts
	f.request.Records = []agentpagecontext.EntityRef{
		{Type: "customer", ID: pulid.MustNew("cus_").String()},
		{Type: string(agent.SubjectShipment), ID: pulid.MustNew("shp_").String()},
	}

	opened, err := f.svc.OpenDelegate(t.Context(), f.request)
	require.NoError(t, err)

	require.NotNil(t, contexts.asked)
	assert.Equal(t, f.request.Records, contexts.asked.DelegatorRecords)
	assert.Equal(t, f.request.Records, opened.Request.Context.MemoryRecords())
}

// The records and results the asking turn handed over follow the task in the
// delegate's question, fenced as data, while the task itself stays what the
// asking agent wrote.
func TestOpenDelegate_PutsWhatWasHandedOverAfterTheTask(t *testing.T) {
	t.Parallel()

	f := newDelegateFixture()
	f.request.Call.Context = &delegation.Context{
		Records: []agentrecord.RecordRef{{EntityType: "shipment", ID: "shp_1"}},
		Results: []delegation.SharedResult{{
			CallID:   "call_read",
			ToolName: "shipment_get",
			Content:  `{"proNumber":"PRO-1001"}`,
		}},
	}

	opened, err := f.svc.OpenDelegate(t.Context(), f.request)
	require.NoError(t, err)

	question := clockLinePrefix.ReplaceAllString(
		opened.Turn.Messages[len(opened.Turn.Messages)-1].Content, "",
	)
	assert.True(t, strings.HasPrefix(question, f.request.Call.Task+"\n\n"))
	assert.Contains(t, question, "shipment shp_1")
	assert.Contains(t, question, "PRO-1001")
	assert.Contains(t, question, "<untrusted_data>")
	assert.Equal(t, question, opened.Request.Input)
}
