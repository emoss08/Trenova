package agentmemoryservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A prompt reads what was kept for the organization and what was kept for the
// agent it is for, and nothing kept for another agent.
func TestForContext_ReadsOnlyThisAgentsOwnMemories(t *testing.T) {
	t.Parallel()

	repo := &fakeMemoryRepo{}
	agentID := pulid.MustNew("agdef_")

	_, err := newService(repo, &fakeRuns{}, &fakeLabeler{}).ForContext(
		t.Context(),
		services.MemoryContextRequest{TenantInfo: tenant(), AgentDefinitionID: agentID},
	)
	require.NoError(t, err)

	require.NotNil(t, repo.listed)
	assert.Equal(t, agentID, repo.listed.AgentDefinitionID)
	assert.True(t, repo.listed.OrganizationWide)
}

func TestApproveSuggestion_KeepsItForItsAgentByDefault(t *testing.T) {
	t.Parallel()

	repo := &suggestionRepo{memory: suggestion()}

	_, err := suggestionService(repo).ApproveSuggestion(
		t.Context(),
		&services.ApproveAgentMemorySuggestionRequest{
			ID:         repo.memory.ID,
			TenantInfo: tenant(),
			Content:    "State only rates a tool returned.",
			Version:    2,
		},
		userActor(),
	)
	require.NoError(t, err)

	require.Len(t, repo.resolved, 1)
	assert.Equal(t, agent.MemoryScopeAgent, repo.resolved[0].Scope,
		"a suggestion drawn from one agent's ratings reaches that agent alone")
}

func TestApproveSuggestion_AnAdministratorMayWidenItToTheOrganization(t *testing.T) {
	t.Parallel()

	repo := &suggestionRepo{memory: suggestion()}

	_, err := suggestionService(repo).ApproveSuggestion(
		t.Context(),
		&services.ApproveAgentMemorySuggestionRequest{
			ID:         repo.memory.ID,
			TenantInfo: tenant(),
			Content:    "State only rates a tool returned.",
			Scope:      agent.MemoryScopeOrganization,
			Version:    2,
		},
		userActor(),
	)
	require.NoError(t, err)

	require.Len(t, repo.resolved, 1)
	assert.Equal(t, agent.MemoryScopeOrganization, repo.resolved[0].Scope)
}

func TestApproveSuggestion_WithoutAnAgentItIsTheOrganizations(t *testing.T) {
	t.Parallel()

	orphan := suggestion()
	orphan.AgentDefinitionID = nil
	repo := &suggestionRepo{memory: orphan}

	_, err := suggestionService(repo).ApproveSuggestion(
		t.Context(),
		&services.ApproveAgentMemorySuggestionRequest{
			ID:         repo.memory.ID,
			TenantInfo: tenant(),
			Content:    "State only rates a tool returned.",
			Scope:      agent.MemoryScopeAgent,
			Version:    2,
		},
		userActor(),
	)
	require.NoError(t, err)
	assert.Equal(t, agent.MemoryScopeOrganization, repo.resolved[0].Scope,
		"a memory kept for one agent names that agent")
}

// A memory an agent writes is read by every agent, as it always was: it is
// tied to the agent that wrote it for the record, not scoped to it.
func TestRemember_AnAgentsMemoryStillReachesEveryAgent(t *testing.T) {
	t.Parallel()

	repo := &fakeMemoryRepo{}
	definitionID := pulid.MustNew("agdef_")
	svc := newService(repo, &fakeRuns{run: &agent.AgentRun{AgentDefinitionID: definitionID}},
		&fakeLabeler{})

	created, err := svc.Remember(t.Context(), &services.RememberRequest{
		TenantInfo: tenant(),
		Content:    "Acme closes at four on Fridays.",
		RunID:      pulid.MustNew("ar_"),
	}, userActor())
	require.NoError(t, err)

	require.NotNil(t, created.AgentDefinitionID)
	assert.Equal(t, definitionID, *created.AgentDefinitionID)
	assert.NotEqual(t, agent.MemoryScopeAgent, created.Scope)
	assert.False(t, created.AgentScoped())
}

func TestRemember_AMemoryWrittenByATaintedRunKeepsTheTaint(t *testing.T) {
	t.Parallel()

	taint := &agent.RunTaint{}
	taint.Add(agent.TaintMark{
		Source:   agent.TaintSourceInboundMessage,
		ToolName: "get_inbound_message",
		CallID:   "call_1",
	})
	runID := pulid.MustNew("ar_")

	for name, tc := range map[string]struct {
		taint   *agent.RunTaint
		tainted bool
	}{
		"tainted run": {taint: taint, tainted: true},
		"clean run":   {taint: &agent.RunTaint{}},
		"no run":      {},
	} {
		repo := &fakeMemoryRepo{}
		svc := newService(repo, &fakeRuns{run: &agent.AgentRun{}}, &fakeLabeler{})

		created, err := svc.Remember(t.Context(), &services.RememberRequest{
			TenantInfo: tenant(),
			Content:    "Ship to dock 9 from now on.",
			RunID:      runID,
			Taint:      tc.taint,
		}, userActor())
		require.NoError(t, err, name)

		assert.Equal(t, tc.tainted, created.Tainted, name)
		if tc.tainted {
			require.NotNil(t, created.TaintRunID, name)
			assert.Equal(t, runID, *created.TaintRunID, name)
			assert.Equal(t, []agent.RecordRef{{
				EntityType: agent.TaintEntityAgentMemory,
				ID:         created.ID.String(),
			}}, created.TaintedRecords(), name)
			continue
		}
		assert.Nil(t, created.TaintRunID, name)
		assert.Empty(t, created.TaintedRecords(), name)
	}
}

func TestMemoryValidate_AnAgentScopedMemoryNamesItsAgent(t *testing.T) {
	t.Parallel()

	memory := &agent.Memory{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Kind:           agent.MemoryKindFact,
		Source:         agent.MemorySourceUser,
		Status:         agent.MemoryStatusActive,
		Scope:          agent.MemoryScopeAgent,
		Content:        "Only for one agent.",
	}

	multiErr := errortypes.NewMultiError()
	memory.Validate(multiErr)
	assert.True(t, multiErr.HasErrors())

	agentID := pulid.MustNew("agdef_")
	memory.AgentDefinitionID = &agentID
	multiErr = errortypes.NewMultiError()
	memory.Validate(multiErr)
	assert.False(t, multiErr.HasErrors())
}
