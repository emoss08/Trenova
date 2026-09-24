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

type suggestionRepo struct {
	repositories.AgentMemoryRepository
	memory   *agent.Memory
	resolved []repositories.ResolveAgentMemorySuggestionRequest
	statuses []repositories.SetAgentMemoryStatusRequest
}

func (f *suggestionRepo) GetByID(
	context.Context,
	repositories.GetAgentMemoryByIDRequest,
) (*agent.Memory, error) {
	if f.memory == nil {
		return nil, errortypes.NewNotFoundError("AgentMemory not found")
	}
	copied := *f.memory

	return &copied, nil
}

func (f *suggestionRepo) ResolveSuggestion(
	_ context.Context,
	req repositories.ResolveAgentMemorySuggestionRequest,
) (*agent.Memory, error) {
	f.resolved = append(f.resolved, req)
	resolved := *f.memory
	resolved.Status = req.Status
	if req.Status == agent.MemoryStatusActive {
		resolved.Content = req.Content
		resolved.Kind = req.Kind
	}
	resolved.Version++

	return &resolved, nil
}

func (f *suggestionRepo) SetStatus(
	_ context.Context,
	req repositories.SetAgentMemoryStatusRequest,
) (*agent.Memory, error) {
	f.statuses = append(f.statuses, req)
	updated := *f.memory
	updated.Status = req.Status

	return &updated, nil
}

func suggestion() *agent.Memory {
	agentID := pulid.MustNew("agdef_")

	return &agent.Memory{
		ID:                pulid.MustNew("amem_"),
		OrganizationID:    pulid.MustNew("org_"),
		BusinessUnitID:    pulid.MustNew("bu_"),
		Kind:              agent.MemoryKindCorrection,
		Source:            agent.MemorySourceFeedback,
		Status:            agent.MemoryStatusSuggested,
		Content:           "Correction for Rate Desk. People said the numbers were made up.",
		AgentDefinitionID: &agentID,
		Evidence: &agent.MemoryEvidence{
			FeedbackIDs: []pulid.ID{pulid.MustNew("aifb_")},
			PatternKey:  "pattern",
			RatingCount: 3,
		},
		Version: 2,
	}
}

func suggestionService(repo *suggestionRepo) *Service {
	return &Service{l: zap.NewNop(), repo: repo}
}

func TestSetStatus_RefusesASuggestedMemory(t *testing.T) {
	t.Parallel()

	repo := &suggestionRepo{memory: suggestion()}
	svc := suggestionService(repo)

	_, err := svc.SetStatus(t.Context(), services.SetAgentMemoryStatusRequest{
		ID:         repo.memory.ID,
		TenantInfo: tenant(),
		Status:     agent.MemoryStatusActive,
	}, userActor())
	require.Error(t, err)
	assert.Empty(t, repo.statuses, "a suggestion is never activated by a status change")

	repo.memory.Status = agent.MemoryStatusDismissed
	_, err = svc.SetStatus(t.Context(), services.SetAgentMemoryStatusRequest{
		ID:         repo.memory.ID,
		TenantInfo: tenant(),
		Status:     agent.MemoryStatusActive,
	}, userActor())
	require.Error(t, err, "nor is a dismissed one restored")
	assert.Empty(t, repo.statuses)
}

func TestSetStatus_RefusesToMakeAMemorySuggested(t *testing.T) {
	t.Parallel()

	memory := suggestion()
	memory.Status = agent.MemoryStatusActive
	repo := &suggestionRepo{memory: memory}

	_, err := suggestionService(repo).SetStatus(t.Context(), services.SetAgentMemoryStatusRequest{
		ID:         memory.ID,
		TenantInfo: tenant(),
		Status:     agent.MemoryStatusSuggested,
	}, userActor())
	require.Error(t, err)
	assert.Empty(t, repo.statuses)
}

func TestSetStatus_StillRetiresAnActiveMemory(t *testing.T) {
	t.Parallel()

	memory := suggestion()
	memory.Status = agent.MemoryStatusActive
	repo := &suggestionRepo{memory: memory}

	updated, err := suggestionService(repo).SetStatus(
		t.Context(),
		services.SetAgentMemoryStatusRequest{
			ID:         memory.ID,
			TenantInfo: tenant(),
			Status:     agent.MemoryStatusRetired,
		},
		userActor(),
	)
	require.NoError(t, err)
	assert.Equal(t, agent.MemoryStatusRetired, updated.Status)
	require.Len(t, repo.statuses, 1)
}

func TestApproveSuggestion_MakesTheEditedTextActive(t *testing.T) {
	t.Parallel()

	repo := &suggestionRepo{memory: suggestion()}
	actor := userActor()

	approved, err := suggestionService(repo).ApproveSuggestion(
		t.Context(),
		&services.ApproveAgentMemorySuggestionRequest{
			ID:         repo.memory.ID,
			TenantInfo: tenant(),
			Content:    "  State only rates a tool returned.  ",
			Version:    2,
		},
		actor,
	)
	require.NoError(t, err)
	assert.Equal(t, agent.MemoryStatusActive, approved.Status)
	assert.Equal(t, "State only rates a tool returned.", approved.Content)
	require.Len(t, repo.resolved, 1)
	resolved := repo.resolved[0]
	assert.Equal(t, agent.MemoryStatusActive, resolved.Status)
	assert.Equal(t, agent.MemoryKindCorrection, resolved.Kind, "the kind defaults to the suggestion's")
	assert.Equal(t, actor.UserID, resolved.ByUserID)
	assert.Equal(t, int64(2), resolved.Version)
}

func TestApproveSuggestion_RefusesEmptyText(t *testing.T) {
	t.Parallel()

	repo := &suggestionRepo{memory: suggestion()}

	_, err := suggestionService(repo).ApproveSuggestion(
		t.Context(),
		&services.ApproveAgentMemorySuggestionRequest{
			ID:         repo.memory.ID,
			TenantInfo: tenant(),
			Content:    "   ",
			Version:    2,
		},
		userActor(),
	)
	require.Error(t, err)
	assert.Empty(t, repo.resolved)
}

func TestApproveSuggestion_OnlyASuggestionAndOnlyByAPerson(t *testing.T) {
	t.Parallel()

	memory := suggestion()
	memory.Status = agent.MemoryStatusActive
	repo := &suggestionRepo{memory: memory}
	svc := suggestionService(repo)

	_, err := svc.ApproveSuggestion(t.Context(), &services.ApproveAgentMemorySuggestionRequest{
		ID:         memory.ID,
		TenantInfo: tenant(),
		Content:    "Anything",
		Version:    2,
	}, userActor())
	require.Error(t, err, "an active memory is not approved again")

	repo.memory.Status = agent.MemoryStatusSuggested
	_, err = svc.ApproveSuggestion(t.Context(), &services.ApproveAgentMemorySuggestionRequest{
		ID:         memory.ID,
		TenantInfo: tenant(),
		Content:    "Anything",
		Version:    2,
	}, &services.RequestActor{PrincipalType: services.PrincipalTypeAgent})
	require.Error(t, err, "an agent never approves what it will read")
	assert.Empty(t, repo.resolved)
}

func TestDismissSuggestion_RecordsWhoDismissedIt(t *testing.T) {
	t.Parallel()

	repo := &suggestionRepo{memory: suggestion()}
	actor := userActor()

	dismissed, err := suggestionService(repo).DismissSuggestion(
		t.Context(),
		services.DismissAgentMemorySuggestionRequest{
			ID:         repo.memory.ID,
			TenantInfo: tenant(),
			Version:    2,
		},
		actor,
	)
	require.NoError(t, err)
	assert.Equal(t, agent.MemoryStatusDismissed, dismissed.Status)
	require.Len(t, repo.resolved, 1)
	assert.Equal(t, agent.MemoryStatusDismissed, repo.resolved[0].Status)
	assert.Equal(t, actor.UserID, repo.resolved[0].ByUserID)
	assert.Positive(t, repo.resolved[0].At)
}

func TestMemoryValidate_ASuggestionComesOnlyFromFeedbackWithEvidence(t *testing.T) {
	t.Parallel()

	memory := suggestion()
	memory.Evidence = nil
	me := errortypes.NewMultiError()
	memory.Validate(me)
	assert.True(t, me.HasErrors(), "feedback without its ratings is refused")

	memory = suggestion()
	memory.Source = agent.MemorySourceUser
	me = errortypes.NewMultiError()
	memory.Validate(me)
	assert.True(t, me.HasErrors(), "only feedback suggests")
}
