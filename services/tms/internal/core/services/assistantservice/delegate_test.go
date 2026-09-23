package assistantservice

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The transcript shows another agent's steps nested under the call that
// handed it the task, headed by that agent's name, and the conversation's
// own steps as they always were.
func TestRenderTranscript_NestsADelegatesStepsUnderItsCall(t *testing.T) {
	t.Parallel()

	delegate := pulid.MustNew("agdef_")
	messages := []conversation.Message{
		{Role: conversation.RoleUser, Content: "Build a dashboard of on-time delivery.",
			CreatedAt: 1_790_000_000},
		{
			Role:      conversation.RoleAssistant,
			CreatedAt: 1_790_000_001,
			ToolCalls: []conversation.ToolCallRecord{{
				ID:        "call_1",
				Name:      "delegate_task",
				Arguments: map[string]any{"agentId": delegate.String(), "task": "Make the report."},
			}},
		},
		{Role: conversation.RoleUser, Kind: conversation.MessageKindDelegated,
			AgentDefinitionID: delegate, DelegateCallID: "call_1",
			Content: "Make the report.", CreatedAt: 1_790_000_002},
		{Role: conversation.RoleAssistant, Kind: conversation.MessageKindDelegated,
			AgentDefinitionID: delegate, DelegateCallID: "call_1",
			ToolCalls: []conversation.ToolCallRecord{{ID: "call_d1", Name: "create_report"}},
			CreatedAt: 1_790_000_003},
		{Role: conversation.RoleTool, Kind: conversation.MessageKindDelegated,
			AgentDefinitionID: delegate, DelegateCallID: "call_1",
			ToolCallID: "call_d1", ToolName: "create_report",
			Content: "Tool \"create_report\" ran successfully.", CreatedAt: 1_790_000_004},
		{Role: conversation.RoleAssistant, Kind: conversation.MessageKindDelegated,
			AgentDefinitionID: delegate, DelegateCallID: "call_1",
			Content: "Saved the report rd_1.", CreatedAt: 1_790_000_005},
		{Role: conversation.RoleTool, ToolCallID: "call_1", ToolName: "delegate_task",
			Content: "the account", CreatedAt: 1_790_000_006},
		{Role: conversation.RoleAssistant, Content: "The tile is ready.", CreatedAt: 1_790_000_007},
	}

	body := renderTranscript(transcriptInput{
		Thread:     transcriptThread(),
		AgentName:  "Homepage Widget Builder",
		Delegates:  map[pulid.ID]string{delegate: "Report Builder"},
		Messages:   messages,
		ExportedAt: 1_790_000_100,
	})

	assert.Equal(t, 1, strings.Count(body, "## Handed to Report Builder"),
		"the delegate's steps are one section")
	assert.Contains(t, body, "> ### Task · ")
	assert.Contains(t, body, "> Make the report.")
	assert.Contains(t, body, "> ## Report Builder · ")
	assert.Contains(t, body, "> **Called `create_report`**")
	assert.Contains(t, body, "> Saved the report rd_1.")
	assert.Equal(t, 1, strings.Count(body, "## You · "),
		"the task is not shown as something the person typed")
	assert.Less(t, strings.Index(body, "**Called `delegate_task`**"),
		strings.Index(body, "## Handed to Report Builder"))
	assert.Less(t, strings.Index(body, "## Handed to Report Builder"),
		strings.Index(body, "### Result from `delegate_task`"))
	assert.Contains(t, body, "## Homepage Widget Builder · ")
}

// A delegate deleted since is named plainly rather than left blank.
func TestRenderTranscript_NamesADeletedDelegatePlainly(t *testing.T) {
	t.Parallel()

	body := renderTranscript(transcriptInput{
		Thread:    transcriptThread(),
		AgentName: "Homepage Widget Builder",
		Messages: []conversation.Message{{
			Role: conversation.RoleAssistant, Kind: conversation.MessageKindDelegated,
			AgentDefinitionID: pulid.MustNew("agdef_"), DelegateCallID: "call_1",
			Content: "Done.",
		}},
	})

	assert.Contains(t, body, "## Handed to Another agent")
}

type delegateDefinitions struct {
	repositories.AgentDefinitionRepository

	byID map[pulid.ID]*agentdefinition.Definition
}

func (s *delegateDefinitions) GetByID(
	_ context.Context,
	req repositories.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	if definition, ok := s.byID[req.ID]; ok {
		return definition, nil
	}

	return nil, errortypes.NewNotFoundError("AgentDefinition not found")
}

type delegateFixture struct {
	svc      *Service
	parent   *agentdefinition.Definition
	delegate *agentdefinition.Definition
	perms    *subjectPermissions
	request  *OpenDelegateRequest
}

func newDelegateFixture() *delegateFixture {
	svc := newService(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{})
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
	perms := &subjectPermissions{allowed: map[string]bool{"assistant:create": true}}
	svc.permissions = perms
	svc.definitions = &delegateDefinitions{byID: map[pulid.ID]*agentdefinition.Definition{
		parent.ID:   parent,
		delegate.ID: delegate,
	}}

	actor := &serviceports.RequestActor{
		PrincipalType:  serviceports.PrincipalTypeUser,
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
			StepOwner: serviceports.RunStepOwner{
				Kind: serviceports.RunStepOwnerAssistantTurn,
				ID:   pulid.MustNew("atrn_"),
			},
			Call: agentruntime.DelegateCall{
				Call:      serviceports.ToolCall{ID: "call_1", Name: "delegate_task"},
				Delegate:  agentdefinition.RuntimeDelegate{ID: delegate.ID, Name: delegate.Name},
				Task:      "Create the on-time report and return its id.",
				StepScope: "scope-1",
			},
		},
	}
}

// The delegate's turn is opened as the delegate, as the same person, marked as
// working for the agent that asked: it never holds delegate_task, and it is
// not unattended, so a write that is the person's own still runs as theirs.
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
	assert.Contains(t, opened.Turn.System, "The agent Homepage Widget Builder handed you this task")
	assert.Equal(t, f.request.Call.Task, opened.Turn.Messages[len(opened.Turn.Messages)-1].Content)
	assert.Contains(t, f.perms.asked, "assistant:create", "the person must be allowed to use it")
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
				delete(f.svc.definitions.(*delegateDefinitions).byID, f.delegate.ID)
			},
			reason: "no longer exists",
		},
		"the person may not use the assistant": {
			change: func(f *delegateFixture) { f.perms.allowed = map[string]bool{} },
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
