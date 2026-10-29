package agentmemoryservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type suggestionRepo struct {
	agentmemoryports.AgentMemoryRepository
	memory   *agentmemory.Memory
	resolved []agentmemoryports.ResolveAgentMemorySuggestionRequest
	statuses []agentmemoryports.SetAgentMemoryStatusQuery
}

func (f *suggestionRepo) GetByID(
	context.Context,
	agentmemoryports.GetAgentMemoryByIDRequest,
) (*agentmemory.Memory, error) {
	if f.memory == nil {
		return nil, errortypes.NewNotFoundError("AgentMemory not found")
	}
	copied := *f.memory

	return &copied, nil
}

func (f *suggestionRepo) Update(
	_ context.Context,
	entity *agentmemory.Memory,
) (*agentmemory.Memory, error) {
	f.memory = entity

	return entity, nil
}

func (f *suggestionRepo) ResolveSuggestion(
	_ context.Context,
	req agentmemoryports.ResolveAgentMemorySuggestionRequest,
) (*agentmemory.Memory, error) {
	f.resolved = append(f.resolved, req)
	resolved := *f.memory
	resolved.Status = req.Status
	if req.Status == agentmemory.MemoryStatusActive {
		resolved.Content = req.Content
		resolved.Kind = req.Kind
	}
	resolved.Version++

	return &resolved, nil
}

func (f *suggestionRepo) SetStatus(
	_ context.Context,
	req agentmemoryports.SetAgentMemoryStatusQuery,
) (*agentmemory.Memory, error) {
	f.statuses = append(f.statuses, req)
	updated := *f.memory
	updated.Status = req.Status

	return &updated, nil
}

func suggestion() *agentmemory.Memory {
	agentID := pulid.MustNew("agdef_")

	return &agentmemory.Memory{
		ID:                pulid.MustNew("amem_"),
		OrganizationID:    pulid.MustNew("org_"),
		BusinessUnitID:    pulid.MustNew("bu_"),
		Kind:              agentmemory.MemoryKindCorrection,
		Source:            agentmemory.MemorySourceFeedback,
		Status:            agentmemory.MemoryStatusSuggested,
		Content:           "Correction for Rate Desk. People said the numbers were made up.",
		Scope:             agentmemory.MemoryScopeAgent,
		AgentDefinitionID: &agentID,
		Evidence: &agentmemory.MemoryEvidence{
			FeedbackIDs: []pulid.ID{pulid.MustNew("aifb_")},
			PatternKey:  "pattern",
			RatingCount: 3,
		},
		Version: 2,
	}
}

// suggestionService is the service for an administrator: one who manages
// shared memory, which the suggestions kept for an agent or the organization
// take to decide.
func suggestionService(repo *suggestionRepo) *Service {
	return &Service{l: zap.NewNop(), repo: repo, permissions: &fakePermissions{manages: true}}
}

// A memory kept for an agent or the organization is retired by someone who
// manages shared memory and nobody else; a person's own is theirs.
func TestSetStatus_ASharedMemoryIsAnAdministratorsToRetire(t *testing.T) {
	t.Parallel()

	shared := suggestion()
	shared.Status = agentmemory.MemoryStatusActive
	repo := &suggestionRepo{memory: shared}
	svc := &Service{l: zap.NewNop(), repo: repo, permissions: &fakePermissions{manages: false}}

	_, err := svc.SetStatus(t.Context(), &agentmemoryports.SetAgentMemoryStatusRequest{
		ID: shared.ID, TenantInfo: tenant(), Status: agentmemory.MemoryStatusRetired,
	}, userActor())
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
	assert.Empty(t, repo.statuses)

	_, err = svc.SetStatus(t.Context(), &agentmemoryports.SetAgentMemoryStatusRequest{
		ID: shared.ID, TenantInfo: tenant(), Status: agentmemory.MemoryStatusRetired,
	}, agentActor())
	require.Error(t, err, "a run with nobody in it never retires what others read")
	assert.Empty(t, repo.statuses)

	owner := userActor()
	own := suggestion()
	own.Status = agentmemory.MemoryStatusActive
	own.SetAudience(agentmemory.MemoryScopeUser, owner.UserID, pulid.Nil)
	repo = &suggestionRepo{memory: own}
	svc = &Service{l: zap.NewNop(), repo: repo, permissions: &fakePermissions{manages: false}}

	updated, err := svc.SetStatus(t.Context(), &agentmemoryports.SetAgentMemoryStatusRequest{
		ID: own.ID, TenantInfo: tenant(), Status: agentmemory.MemoryStatusRetired,
	}, owner)
	require.NoError(t, err)
	assert.Equal(t, agentmemory.MemoryStatusRetired, updated.Status)

	_, err = svc.SetStatus(t.Context(), &agentmemoryports.SetAgentMemoryStatusRequest{
		ID: own.ID, TenantInfo: tenant(), Status: agentmemory.MemoryStatusRetired,
	}, userActor())
	require.Error(t, err, "another person's own memory is not theirs")
}

func TestSetStatus_RefusesASuggestedMemory(t *testing.T) {
	t.Parallel()

	repo := &suggestionRepo{memory: suggestion()}
	svc := suggestionService(repo)

	_, err := svc.SetStatus(t.Context(), &agentmemoryports.SetAgentMemoryStatusRequest{
		ID:         repo.memory.ID,
		TenantInfo: tenant(),
		Status:     agentmemory.MemoryStatusActive,
	}, userActor())
	require.Error(t, err)
	assert.Empty(t, repo.statuses, "a suggestion is never activated by a status change")

	repo.memory.Status = agentmemory.MemoryStatusDismissed
	_, err = svc.SetStatus(t.Context(), &agentmemoryports.SetAgentMemoryStatusRequest{
		ID:         repo.memory.ID,
		TenantInfo: tenant(),
		Status:     agentmemory.MemoryStatusActive,
	}, userActor())
	require.Error(t, err, "nor is a dismissed one restored")
	assert.Empty(t, repo.statuses)
}

func TestSetStatus_RefusesToMakeAMemorySuggested(t *testing.T) {
	t.Parallel()

	memory := suggestion()
	memory.Status = agentmemory.MemoryStatusActive
	repo := &suggestionRepo{memory: memory}

	_, err := suggestionService(
		repo,
	).SetStatus(t.Context(), &agentmemoryports.SetAgentMemoryStatusRequest{
		ID:         memory.ID,
		TenantInfo: tenant(),
		Status:     agentmemory.MemoryStatusSuggested,
	}, userActor())
	require.Error(t, err)
	assert.Empty(t, repo.statuses)
}

func TestSetStatus_StillRetiresAnActiveMemory(t *testing.T) {
	t.Parallel()

	memory := suggestion()
	memory.Status = agentmemory.MemoryStatusActive
	repo := &suggestionRepo{memory: memory}

	updated, err := suggestionService(repo).SetStatus(
		t.Context(),
		&agentmemoryports.SetAgentMemoryStatusRequest{
			ID:         memory.ID,
			TenantInfo: tenant(),
			Status:     agentmemory.MemoryStatusRetired,
		},
		userActor(),
	)
	require.NoError(t, err)
	assert.Equal(t, agentmemory.MemoryStatusRetired, updated.Status)
	require.Len(t, repo.statuses, 1)
}

func TestApproveSuggestion_MakesTheEditedTextActive(t *testing.T) {
	t.Parallel()

	repo := &suggestionRepo{memory: suggestion()}
	actor := userActor()

	approved, err := suggestionService(repo).ApproveSuggestion(
		t.Context(),
		&agentmemoryports.ApproveAgentMemorySuggestionRequest{
			ID:         repo.memory.ID,
			TenantInfo: tenant(),
			Content:    "  State only rates a tool returned.  ",
			Version:    2,
		},
		actor,
	)
	require.NoError(t, err)
	assert.Equal(t, agentmemory.MemoryStatusActive, approved.Status)
	assert.Equal(t, "State only rates a tool returned.", approved.Content)
	require.Len(t, repo.resolved, 1)
	resolved := repo.resolved[0]
	assert.Equal(t, agentmemory.MemoryStatusActive, resolved.Status)
	assert.Equal(
		t,
		agentmemory.MemoryKindCorrection,
		resolved.Kind,
		"the kind defaults to the suggestion's",
	)
	assert.Equal(t, actor.UserID, resolved.ByUserID)
	assert.Equal(t, int64(2), resolved.Version)
}

// A dispatcher offers a memory for the Billing team and an administrator
// approves it from the queue, which names no scope: it is kept for Billing,
// not widened to the organization, and the dispatcher stays its author.
func TestApproveSuggestion_KeepsAPersonsOfferForTheirTeam(t *testing.T) {
	t.Parallel()

	dispatcher, billing := pulid.MustNew("usr_"), pulid.MustNew("rol_")
	offer := suggestion()
	offer.Source = agentmemory.MemorySourceUser
	offer.Kind = agentmemory.MemoryKindInstruction
	offer.AgentDefinitionID = nil
	offer.Evidence = nil
	offer.CreatedByUserID = &dispatcher
	offer.SetAudience(agentmemory.MemoryScopeRole, pulid.Nil, billing)
	repo := &suggestionRepo{memory: offer}

	approved, err := suggestionService(repo).ApproveSuggestion(
		t.Context(),
		&agentmemoryports.ApproveAgentMemorySuggestionRequest{
			ID:         offer.ID,
			TenantInfo: tenant(),
			Content:    offer.Content,
			Scope:      agentmemory.MemoryScopeAgent,
			Version:    2,
		},
		userActor(),
	)
	require.NoError(t, err)
	assert.Equal(t, agentmemory.MemoryStatusActive, approved.Status)
	require.Len(t, repo.resolved, 1)
	assert.Equal(t, agentmemory.MemoryScopeRole, repo.resolved[0].Scope)
	assert.Equal(t, billing, repo.resolved[0].RoleID)

	forEveryone := suggestion()
	forEveryone.Source = agentmemory.MemorySourceUser
	forEveryone.AgentDefinitionID = nil
	forEveryone.Evidence = nil
	forEveryone.CreatedByUserID = &dispatcher
	forEveryone.SetAudience(agentmemory.MemoryScopeOrganization, pulid.Nil, pulid.Nil)
	repo = &suggestionRepo{memory: forEveryone}
	_, err = suggestionService(repo).ApproveSuggestion(
		t.Context(),
		&agentmemoryports.ApproveAgentMemorySuggestionRequest{
			ID: forEveryone.ID, TenantInfo: tenant(), Content: forEveryone.Content, Version: 2,
		},
		userActor(),
	)
	require.NoError(t, err)
	require.Len(t, repo.resolved, 1)
	assert.Equal(t, agentmemory.MemoryScopeOrganization, repo.resolved[0].Scope)
}

func TestApproveSuggestion_RefusesEmptyText(t *testing.T) {
	t.Parallel()

	repo := &suggestionRepo{memory: suggestion()}

	_, err := suggestionService(repo).ApproveSuggestion(
		t.Context(),
		&agentmemoryports.ApproveAgentMemorySuggestionRequest{
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
	memory.Status = agentmemory.MemoryStatusActive
	repo := &suggestionRepo{memory: memory}
	svc := suggestionService(repo)

	_, err := svc.ApproveSuggestion(
		t.Context(),
		&agentmemoryports.ApproveAgentMemorySuggestionRequest{
			ID:         memory.ID,
			TenantInfo: tenant(),
			Content:    "Anything",
			Version:    2,
		},
		userActor(),
	)
	require.Error(t, err, "an active memory is not approved again")

	repo.memory.Status = agentmemory.MemoryStatusSuggested
	_, err = svc.ApproveSuggestion(
		t.Context(),
		&agentmemoryports.ApproveAgentMemorySuggestionRequest{
			ID:         memory.ID,
			TenantInfo: tenant(),
			Content:    "Anything",
			Version:    2,
		},
		&permissionports.RequestActor{PrincipalType: permissionports.PrincipalTypeAgent},
	)
	require.Error(t, err, "an agent never approves what it will read")
	assert.Empty(t, repo.resolved)
}

func TestDismissSuggestion_RecordsWhoDismissedIt(t *testing.T) {
	t.Parallel()

	repo := &suggestionRepo{memory: suggestion()}
	actor := userActor()

	dismissed, err := suggestionService(repo).DismissSuggestion(
		t.Context(),
		agentmemoryports.DismissAgentMemorySuggestionRequest{
			ID:         repo.memory.ID,
			TenantInfo: tenant(),
			Version:    2,
		},
		actor,
	)
	require.NoError(t, err)
	assert.Equal(t, agentmemory.MemoryStatusDismissed, dismissed.Status)
	require.Len(t, repo.resolved, 1)
	assert.Equal(t, agentmemory.MemoryStatusDismissed, repo.resolved[0].Status)
	assert.Equal(t, actor.UserID, repo.resolved[0].ByUserID)
	assert.Positive(t, repo.resolved[0].At)
}

// Rewriting a memory the organization shares is an administrator's: without
// the manage permission the change is refused, with it the words are saved.
func TestUpdate_ASharedMemoryIsAnAdministratorsToRewrite(t *testing.T) {
	t.Parallel()

	shared := suggestion()
	shared.Source = agentmemory.MemorySourceUser
	shared.Kind = agentmemory.MemoryKindInstruction
	shared.Status = agentmemory.MemoryStatusActive
	shared.AgentDefinitionID = nil
	shared.Evidence = nil
	shared.SetAudience(agentmemory.MemoryScopeOrganization, pulid.Nil, pulid.Nil)
	req := &agentmemoryports.UpdateAgentMemoryRequest{
		ID:         shared.ID,
		TenantInfo: tenant(),
		Kind:       agentmemory.MemoryKindInstruction,
		Content:    "Quote in dollars, never euros.",
		Version:    2,
	}

	repo := &suggestionRepo{memory: shared}
	svc := &Service{l: zap.NewNop(), repo: repo, permissions: &fakePermissions{manages: false}}
	_, err := svc.Update(t.Context(), req, userActor())
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
	assert.Equal(t, shared.Content, repo.memory.Content, "nothing was written")

	svc.permissions = &fakePermissions{manages: true}
	updated, err := svc.Update(t.Context(), req, userActor())
	require.NoError(t, err)
	assert.Equal(t, "Quote in dollars, never euros.", updated.Content)
}

func TestMemoryValidate_ASuggestionComesOnlyFromFeedbackWithEvidence(t *testing.T) {
	t.Parallel()

	memory := suggestion()
	memory.Evidence = nil
	me := errortypes.NewMultiError()
	memory.Validate(me)
	assert.True(t, me.HasErrors(), "feedback without its ratings is refused")

	own := suggestion()
	own.Source = agentmemory.MemorySourceUser
	own.Evidence = nil
	own.SetAudience(agentmemory.MemoryScopeUser, pulid.MustNew("usr_"), pulid.Nil)
	me = errortypes.NewMultiError()
	own.Validate(me)
	assert.True(t, me.HasErrors(), "a person's own memory is kept, never suggested")

	offered := suggestion()
	offered.Source = agentmemory.MemorySourceUser
	offered.Evidence = nil
	offered.SetAudience(agentmemory.MemoryScopeOrganization, pulid.Nil, pulid.Nil)
	me = errortypes.NewMultiError()
	offered.Validate(me)
	assert.False(t, me.HasErrors(), "a person offers a memory for the organization")
}
