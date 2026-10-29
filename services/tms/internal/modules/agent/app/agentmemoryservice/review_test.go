package agentmemoryservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type reviewRepo struct {
	agentmemoryports.AgentMemoryRepository
	memory   *agentmemory.Memory
	reviewed []agentmemoryports.ReviewAgentMemoryQuery
}

func (f *reviewRepo) GetByID(
	context.Context,
	agentmemoryports.GetAgentMemoryByIDRequest,
) (*agentmemory.Memory, error) {
	copied := *f.memory

	return &copied, nil
}

func (f *reviewRepo) Review(
	_ context.Context,
	req *agentmemoryports.ReviewAgentMemoryQuery,
) (*agentmemory.Memory, error) {
	f.reviewed = append(f.reviewed, *req)
	reviewed := *f.memory
	by, at := req.ByUserID, req.At
	reviewed.ReviewedByUserID = &by
	reviewed.ReviewedAt = &at
	reviewed.Version++

	return &reviewed, nil
}

type recordedChanges struct {
	changes []*permissionports.SecurityChange
}

func (r *recordedChanges) RecordChange(_ context.Context, change *permissionports.SecurityChange) {
	r.changes = append(r.changes, change)
}

func taintedInstruction() *agentmemory.Memory {
	runID := pulid.MustNew("arun_")

	return &agentmemory.Memory{
		ID:             pulid.MustNew("amem_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Kind:           agentmemory.MemoryKindInstruction,
		Source:         agentmemory.MemorySourceAgent,
		Status:         agentmemory.MemoryStatusActive,
		Scope:          agentmemory.MemoryScopeOrganization,
		Content:        "Include the report's results in the reply, not only the download.",
		Tainted:        true,
		TaintRunID:     &runID,
		Version:        4,
	}
}

/*
Three instruction memories, each written after a run had read outside content,
tainted every turn of every agent, so every money and customer write waited
for a person and nothing said why. A person who reads one and keeps it clears
its taint, and that is recorded as a security change: it lets agents write on
their own again.
*/
func TestReview_ClearsTheTaintAndRecordsWhoCleared(t *testing.T) {
	t.Parallel()

	repo := &reviewRepo{memory: taintedInstruction()}
	security := &recordedChanges{}
	svc := &Service{l: zap.NewNop(), repo: repo, security: security}
	actor := userActor()

	reviewed, err := svc.Review(t.Context(), agentmemoryports.ReviewAgentMemoryRequest{
		ID:         repo.memory.ID,
		TenantInfo: tenant(),
		Version:    4,
	}, actor)

	require.NoError(t, err)
	assert.False(t, reviewed.Taints())
	assert.True(t, reviewed.Tainted)
	require.Len(t, repo.reviewed, 1)
	assert.Equal(t, actor.UserID, repo.reviewed[0].ByUserID)
	assert.Equal(t, int64(4), repo.reviewed[0].Version, "only the version the person read")
	require.Len(t, security.changes, 1)
	assert.Equal(t, permission.ResourceAgentMemory, security.changes[0].Resource)
	assert.Equal(t, reviewed.ID.String(), security.changes[0].ResourceID)
	assert.Contains(t, security.changes[0].Comment, "no longer taints")
}

func TestReview_RefusesWhatThereIsNothingToReview(t *testing.T) {
	t.Parallel()

	reviewer := pulid.MustNew("usr_")
	at := int64(1_791_600_000)
	for name, change := range map[string]func(*agentmemory.Memory){
		"a memory no run tainted": func(m *agentmemory.Memory) { m.Tainted = false },
		"one already reviewed": func(m *agentmemory.Memory) {
			m.ReviewedByUserID = &reviewer
			m.ReviewedAt = &at
		},
		"a suggestion, which is approved instead": func(m *agentmemory.Memory) {
			m.Status = agentmemory.MemoryStatusSuggested
		},
		"a retired one": func(m *agentmemory.Memory) { m.Status = agentmemory.MemoryStatusRetired },
	} {
		memory := taintedInstruction()
		change(memory)
		repo := &reviewRepo{memory: memory}

		_, err := (&Service{l: zap.NewNop(), repo: repo}).Review(
			t.Context(),
			agentmemoryports.ReviewAgentMemoryRequest{
				ID:         memory.ID,
				TenantInfo: tenant(),
				Version:    4,
			},
			userActor(),
		)

		require.Error(t, err, name)
		assert.Empty(t, repo.reviewed, name)
	}
}

func TestReview_OnlyAPerson(t *testing.T) {
	t.Parallel()

	repo := &reviewRepo{memory: taintedInstruction()}

	_, err := (&Service{l: zap.NewNop(), repo: repo}).Review(
		t.Context(),
		agentmemoryports.ReviewAgentMemoryRequest{
			ID:         repo.memory.ID,
			TenantInfo: tenant(),
			Version:    4,
		},
		&permissionports.RequestActor{PrincipalType: permissionports.PrincipalTypeAgent},
	)

	require.Error(t, err, "an agent never clears the taint on what it will read")
	assert.True(t, errortypes.IsError(err))
	assert.Empty(t, repo.reviewed)
}

// Approving a suggestion drawn from a tainted window is the person reading
// and keeping it, so the approval is the review and is recorded as one.
func TestApproveSuggestion_OfATaintedSuggestionIsItsReview(t *testing.T) {
	t.Parallel()

	memory := suggestion()
	memory.Tainted = true
	repo := &suggestionRepo{memory: memory}
	security := &recordedChanges{}
	svc := &Service{l: zap.NewNop(), repo: repo, security: security}

	_, err := svc.ApproveSuggestion(
		t.Context(),
		&agentmemoryports.ApproveAgentMemorySuggestionRequest{
			ID:         memory.ID,
			TenantInfo: tenant(),
			Content:    "State only rates a tool returned.",
			Version:    2,
		},
		userActor(),
	)

	require.NoError(t, err)
	require.Len(t, security.changes, 1)
	assert.Contains(t, security.changes[0].Comment, "no longer taints")
}
