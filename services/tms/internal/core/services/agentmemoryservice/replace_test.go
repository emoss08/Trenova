package agentmemoryservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type replacingRepo struct {
	fakeMemoryRepo
	stored  map[pulid.ID]*agent.Memory
	retired []pulid.ID
}

func (f *replacingRepo) GetByID(
	_ context.Context,
	req repositories.GetAgentMemoryByIDRequest,
) (*agent.Memory, error) {
	memory, ok := f.stored[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("AgentMemory not found")
	}

	return memory, nil
}

func (f *replacingRepo) SetStatus(
	_ context.Context,
	req repositories.SetAgentMemoryStatusRequest,
) (*agent.Memory, error) {
	f.retired = append(f.retired, req.ID)
	memory := *f.stored[req.ID]
	memory.Status = req.Status

	return &memory, nil
}

func replacingService(repo *replacingRepo) *Service {
	return &Service{
		l:       zap.NewNop(),
		repo:    repo,
		runs:    &fakeRuns{run: &agent.AgentRun{}},
		labeler: &fakeLabeler{},
	}
}

func newReplacingRepo(memories ...*agent.Memory) *replacingRepo {
	repo := &replacingRepo{stored: make(map[pulid.ID]*agent.Memory, len(memories))}
	for _, memory := range memories {
		repo.stored[memory.ID] = memory
	}

	return repo
}

func personalMemory(owner pulid.ID, status agent.MemoryStatus) *agent.Memory {
	return &agent.Memory{
		ID:          pulid.MustNew("amem_"),
		Kind:        agent.MemoryKindInstruction,
		Source:      agent.MemorySourceAgent,
		Status:      status,
		Scope:       agent.MemoryScopeUser,
		OwnerUserID: &owner,
		Content:     "Show me the invoice",
	}
}

func TestRemember_ReplacingThePersonsOwnMemoryKeepsItsReadersAndSavesAtOnce(t *testing.T) {
	t.Parallel()

	person := pulid.MustNew("usr_")
	old := personalMemory(person, agent.MemoryStatusActive)
	repo := newReplacingRepo(old)
	svc := replacingService(repo)

	got, err := svc.Remember(t.Context(), &services.RememberRequest{
		TenantInfo:   tenant(),
		Kind:         agent.MemoryKindInstruction,
		Content:      "Show me the queue item, not the invoice",
		Scope:        agent.MemoryScopeOrganization,
		PersonUserID: person,
		Replaces:     old.ID,
		RunID:        pulid.MustNew("arun_"),
	}, userActor())
	require.NoError(t, err)

	assert.Equal(t, agent.MemoryStatusActive, got.Status)
	assert.Equal(t, agent.MemoryScopeUser, got.Scope)
	require.NotNil(t, got.OwnerUserID)
	assert.Equal(t, person, *got.OwnerUserID)
	require.NotNil(t, got.SupersedesID)
	assert.Equal(t, old.ID, *got.SupersedesID)
}

func TestRemember_ReplacingAnOrganizationMemoryWaitsForSomeoneAllowedToChangeIt(t *testing.T) {
	t.Parallel()

	old := &agent.Memory{
		ID:      pulid.MustNew("amem_"),
		Kind:    agent.MemoryKindFact,
		Source:  agent.MemorySourceUser,
		Status:  agent.MemoryStatusActive,
		Scope:   agent.MemoryScopeOrganization,
		Content: "Acme's terms are net 30",
	}
	repo := newReplacingRepo(old)
	svc := replacingService(repo)

	got, err := svc.Remember(t.Context(), &services.RememberRequest{
		TenantInfo:   tenant(),
		Content:      "Acme's terms are net 45",
		PersonUserID: pulid.MustNew("usr_"),
		Replaces:     old.ID,
		RunID:        pulid.MustNew("arun_"),
	}, userActor())
	require.NoError(t, err)

	assert.Equal(t, agent.MemoryStatusSuggested, got.Status)
	assert.Equal(t, agent.MemoryScopeOrganization, got.Scope)
	assert.Empty(t, repo.retired)
}

func TestRemember_SomeoneElsesMemoryCannotBeReplaced(t *testing.T) {
	t.Parallel()

	old := personalMemory(pulid.MustNew("usr_"), agent.MemoryStatusActive)
	repo := newReplacingRepo(old)
	svc := replacingService(repo)

	_, err := svc.Remember(t.Context(), &services.RememberRequest{
		TenantInfo:   tenant(),
		Content:      "Something else",
		PersonUserID: pulid.MustNew("usr_"),
		Replaces:     old.ID,
	}, userActor())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "No memory with that id is visible here")
	assert.Empty(t, repo.created)
}

func TestRemember_ARetiredMemoryHasNothingLeftToReplace(t *testing.T) {
	t.Parallel()

	person := pulid.MustNew("usr_")
	old := personalMemory(person, agent.MemoryStatusRetired)
	repo := newReplacingRepo(old)
	svc := replacingService(repo)

	_, err := svc.Remember(t.Context(), &services.RememberRequest{
		TenantInfo:   tenant(),
		Content:      "Show me the queue item",
		PersonUserID: person,
		Replaces:     old.ID,
	}, userActor())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no longer kept")
}

func TestRemember_AnAgentReplacesOnlyItsOwnAgentMemory(t *testing.T) {
	t.Parallel()

	owner := pulid.MustNew("agdef_")
	old := &agent.Memory{
		ID:                pulid.MustNew("amem_"),
		Kind:              agent.MemoryKindProcedure,
		Source:            agent.MemorySourceReflection,
		Status:            agent.MemoryStatusActive,
		Scope:             agent.MemoryScopeAgent,
		AgentDefinitionID: &owner,
		ReflectionID:      pulid.Must("arfl_"),
		Content:           "Look up the move before assigning it",
	}
	repo := newReplacingRepo(old)
	svc := replacingService(repo)

	got, err := svc.Remember(t.Context(), &services.RememberRequest{
		TenantInfo:        tenant(),
		Kind:              agent.MemoryKindProcedure,
		Content:           "Look up the move, then its stops, before assigning it",
		AgentDefinitionID: owner,
		Source:            agent.MemorySourceReflection,
		ReflectionID:      pulid.MustNew("arfl_"),
		Replaces:          old.ID,
	}, userActor())
	require.NoError(t, err)
	assert.Equal(t, agent.MemoryStatusActive, got.Status)
	assert.Equal(t, agent.MemoryScopeAgent, got.Scope)

	_, err = svc.Remember(t.Context(), &services.RememberRequest{
		TenantInfo:        tenant(),
		Kind:              agent.MemoryKindProcedure,
		Content:           "Something else",
		AgentDefinitionID: pulid.MustNew("agdef_"),
		Source:            agent.MemorySourceReflection,
		ReflectionID:      pulid.MustNew("arfl_"),
		Replaces:          old.ID,
	}, userActor())
	require.Error(t, err)
}

func TestRemember_RestatingWhatAlreadyReplacedRetiresTheOldOne(t *testing.T) {
	t.Parallel()

	person := pulid.MustNew("usr_")
	old := personalMemory(person, agent.MemoryStatusActive)
	kept := personalMemory(person, agent.MemoryStatusActive)
	kept.Content = "Show me the queue item"
	repo := newReplacingRepo(old, kept)
	repo.found = kept
	svc := replacingService(repo)

	got, err := svc.Remember(t.Context(), &services.RememberRequest{
		TenantInfo:   tenant(),
		Content:      "Show me the queue item",
		PersonUserID: person,
		Replaces:     old.ID,
	}, userActor())
	require.NoError(t, err)

	assert.Same(t, kept, got)
	assert.True(t, got.Refreshed)
	assert.Equal(t, []pulid.ID{old.ID}, repo.retired)
	assert.Empty(t, repo.created)
}

func TestReplacedFreely(t *testing.T) {
	t.Parallel()

	person := pulid.MustNew("usr_")
	agentID := pulid.MustNew("agdef_")
	role := pulid.MustNew("rol_")

	assert.True(t, ReplacedFreely(personalMemory(person, agent.MemoryStatusActive), person, nil))
	assert.False(
		t,
		ReplacedFreely(personalMemory(person, agent.MemoryStatusActive), pulid.Nil, nil),
	)
	assert.True(t, ReplacedFreely(&agent.Memory{
		Scope:             agent.MemoryScopeAgent,
		AgentDefinitionID: &agentID,
	}, pulid.Nil, &agentID))
	assert.False(t, ReplacedFreely(&agent.Memory{
		Scope:  agent.MemoryScopeRole,
		RoleID: &role,
	}, person, &agentID))
	assert.False(
		t,
		ReplacedFreely(&agent.Memory{Scope: agent.MemoryScopeOrganization}, person, &agentID),
	)
}
