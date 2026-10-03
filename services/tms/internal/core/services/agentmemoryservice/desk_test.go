package agentmemoryservice

import (
	"cmp"
	"context"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// deskRepo keeps memories in a map and answers the Desk's reads the way the
// database does: by reader, by status, newest first.
type deskRepo struct {
	repositories.AgentMemoryRepository
	memories   map[pulid.ID]*agent.Memory
	preference *agent.MemoryPreference
	clock      int64
}

func newDeskRepo() *deskRepo {
	return &deskRepo{memories: map[pulid.ID]*agent.Memory{}, clock: 1_760_000_000}
}

func (r *deskRepo) put(memory *agent.Memory) *agent.Memory {
	if memory.ID.IsNil() {
		memory.ID = pulid.MustNew("amem_")
	}
	if memory.Status == "" {
		memory.Status = agent.MemoryStatusActive
	}
	if memory.Scope == "" {
		memory.Scope = agent.MemoryScopeOrganization
	}
	r.clock++
	memory.CreatedAt = r.clock
	r.memories[memory.ID] = memory

	return memory
}

func (r *deskRepo) Create(_ context.Context, entity *agent.Memory) (*agent.Memory, error) {
	return r.put(entity), nil
}

func (r *deskRepo) FindActive(
	context.Context,
	repositories.FindActiveAgentMemoryRequest,
) (*agent.Memory, error) {
	return nil, nil
}

func (r *deskRepo) GetByID(
	_ context.Context,
	req repositories.GetAgentMemoryByIDRequest,
) (*agent.Memory, error) {
	memory, ok := r.memories[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Agent memory not found")
	}
	copied := *memory

	return &copied, nil
}

func (r *deskRepo) GetDesk(
	_ context.Context,
	req repositories.GetDeskMemoriesRequest,
) ([]*repositories.DeskMemoryRow, error) {
	rows := make([]*repositories.DeskMemoryRow, 0, len(req.IDs))
	for _, id := range req.IDs {
		if memory, ok := r.memories[id]; ok {
			copied := *memory
			rows = append(rows, &repositories.DeskMemoryRow{Memory: &copied})
		}
	}

	return rows, nil
}

func (r *deskRepo) visible(filter *repositories.DeskMemoryFilter) []*agent.Memory {
	out := make([]*agent.Memory, 0, len(r.memories))
	for _, memory := range r.memories {
		if memory.Status != agent.MemoryStatusActive && memory.Status != agent.MemoryStatusPaused {
			continue
		}
		if memory.Scope == agent.MemoryScopeAgent || !filter.Reader.Reads(memory) {
			continue
		}
		out = append(out, memory)
	}
	slices.SortFunc(out, func(a, b *agent.Memory) int { return cmp.Compare(b.CreatedAt, a.CreatedAt) })

	return out
}

func (r *deskRepo) ListDesk(
	_ context.Context,
	req repositories.ListDeskMemoriesRequest,
) ([]*repositories.DeskMemoryRow, error) {
	rows := make([]*repositories.DeskMemoryRow, 0)
	for _, memory := range r.visible(&req.Filter) {
		if req.Filter.Scope != "" && memory.Scope != req.Filter.Scope {
			continue
		}
		if req.After != nil && memory.CreatedAt >= req.After.CreatedAt {
			continue
		}
		copied := *memory
		rows = append(rows, &repositories.DeskMemoryRow{Memory: &copied})
		if len(rows) == req.Limit {
			break
		}
	}

	return rows, nil
}

func (r *deskRepo) CountDesk(
	_ context.Context,
	filter repositories.DeskMemoryFilter,
) ([]repositories.DeskMemoryCount, error) {
	counts := map[string]*repositories.DeskMemoryCount{}
	order := make([]string, 0)
	for _, memory := range r.visible(&filter) {
		roleID := pulid.Nil
		if memory.RoleID != nil {
			roleID = *memory.RoleID
		}
		key := string(memory.Scope) + roleID.String()
		if counts[key] == nil {
			counts[key] = &repositories.DeskMemoryCount{Scope: memory.Scope, RoleID: roleID}
			order = append(order, key)
		}
		counts[key].Count++
	}

	out := make([]repositories.DeskMemoryCount, 0, len(order))
	for _, key := range order {
		out = append(out, *counts[key])
	}

	return out, nil
}

func (r *deskRepo) SetStatus(
	_ context.Context,
	req repositories.SetAgentMemoryStatusRequest,
) (*agent.Memory, error) {
	memory := r.memories[req.ID]
	memory.Status = req.Status
	memory.Version++
	copied := *memory

	return &copied, nil
}

func (r *deskRepo) Revise(
	_ context.Context,
	req repositories.ReviseAgentMemoryRequest,
) (*agent.Memory, error) {
	memory := r.memories[req.ID]
	memory.Content = req.Content
	memory.SetAudience(req.Scope, req.OwnerUserID, req.RoleID)
	memory.Version++
	copied := *memory

	return &copied, nil
}

func (r *deskRepo) ResolveSuggestion(
	_ context.Context,
	req repositories.ResolveAgentMemorySuggestionRequest,
) (*agent.Memory, error) {
	memory := r.memories[req.ID]
	memory.Status = req.Status
	if req.Status == agent.MemoryStatusActive {
		memory.Content = req.Content
		memory.SetAudience(req.Scope, req.OwnerUserID, req.RoleID)
	}
	memory.Version++
	copied := *memory

	return &copied, nil
}

func (r *deskRepo) GetPreference(
	context.Context,
	repositories.GetAgentMemoryPreferenceRequest,
) (*agent.MemoryPreference, error) {
	return r.preference, nil
}

func (r *deskRepo) SavePreference(
	_ context.Context,
	entity *agent.MemoryPreference,
) (*agent.MemoryPreference, error) {
	r.preference = entity

	return entity, nil
}

// fakeRoles holds the person's roles and what each inherits.
type fakeRoles struct {
	held    []*permission.Role
	parents map[pulid.ID][]*permission.Role
}

func (f *fakeRoles) GetUserRoleAssignments(
	context.Context,
	pulid.ID,
	pulid.ID,
) ([]*permission.UserRoleAssignment, error) {
	out := make([]*permission.UserRoleAssignment, 0, len(f.held))
	for _, role := range f.held {
		out = append(out, &permission.UserRoleAssignment{RoleID: role.ID, Role: role})
	}

	return out, nil
}

func (f *fakeRoles) GetRolesWithInheritance(
	_ context.Context,
	ids []pulid.ID,
) ([]*permission.Role, error) {
	out := make([]*permission.Role, 0, len(ids))
	for _, id := range ids {
		out = append(out, &permission.Role{ID: id})
		out = append(out, f.parents[id]...)
	}

	return out, nil
}

type deskWorld struct {
	svc      *Service
	repo     *deskRepo
	avery    pulid.ID
	billing  *permission.Role
	dispatch *permission.Role
}

func newDeskWorld() *deskWorld {
	repo := newDeskRepo()
	billing := &permission.Role{ID: pulid.MustNew("rol_"), Name: "Billing"}
	dispatch := &permission.Role{ID: pulid.MustNew("rol_"), Name: "Dispatch"}

	return &deskWorld{
		svc: &Service{
			l:     zap.NewNop(),
			repo:  repo,
			runs:  &fakeRuns{},
			roles: &fakeRoles{held: []*permission.Role{billing}},
		},
		repo:     repo,
		avery:    pulid.MustNew("usr_"),
		billing:  billing,
		dispatch: dispatch,
	}
}

// actor is Avery on the Desk, with or without the agent-memory permissions.
func (w *deskWorld) actor(shared bool) *services.DeskMemoryActor {
	return &services.DeskMemoryActor{
		Actor: &services.RequestActor{
			PrincipalType:  services.PrincipalTypeUser,
			PrincipalID:    w.avery,
			UserID:         w.avery,
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
		},
		MayCreateShared: shared,
		MayUpdateShared: shared,
	}
}

func (w *deskWorld) keep(content string, scope agent.MemoryScope, owner, role pulid.ID) *agent.Memory {
	memory := &agent.Memory{Kind: agent.MemoryKindInstruction, Source: agent.MemorySourceUser, Content: content}
	memory.SetAudience(scope, owner, role)

	return w.repo.put(memory)
}

func deskIDs(memories []*services.DeskMemory) []pulid.ID {
	ids := make([]pulid.ID, 0, len(memories))
	for _, memory := range memories {
		ids = append(ids, memory.Memory.ID)
	}

	return ids
}

func TestDeskMemories_ShowOnlyTheirOwnTheirRolesAndTheOrganizations(t *testing.T) {
	t.Parallel()

	w := newDeskWorld()
	organization := w.keep("Use the DOE weekly average.", agent.MemoryScopeOrganization, pulid.Nil, pulid.Nil)
	own := w.keep("Group AR by facility.", agent.MemoryScopeUser, w.avery, pulid.Nil)
	someoneElses := w.keep("Jordan's note.", agent.MemoryScopeUser, pulid.MustNew("usr_"), pulid.Nil)
	team := w.keep("Acme is billed net-45.", agent.MemoryScopeRole, pulid.Nil, w.billing.ID)
	otherTeam := w.keep("Dispatch note.", agent.MemoryScopeRole, pulid.Nil, w.dispatch.ID)
	agentOnly := w.repo.put(&agent.Memory{
		Kind: agent.MemoryKindFact, Source: agent.MemorySourceFeedback, Content: "Agent's own.",
		Scope: agent.MemoryScopeAgent,
	})

	shown, err := w.svc.DeskByIDs(t.Context(), w.actor(false), []pulid.ID{
		organization.ID, own.ID, someoneElses.ID, team.ID, otherTeam.ID, agentOnly.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{organization.ID, own.ID, team.ID}, deskIDs(shown))

	editable := map[pulid.ID]bool{}
	for _, memory := range shown {
		editable[memory.Memory.ID] = memory.Editable
	}
	assert.Equal(t, map[pulid.ID]bool{organization.ID: false, own.ID: true, team.ID: false}, editable,
		"without the permission only her own is hers to change")

	shown, err = w.svc.DeskByIDs(t.Context(), w.actor(true), []pulid.ID{organization.ID, team.ID})
	require.NoError(t, err)
	for _, memory := range shown {
		assert.True(t, memory.Editable, "with it her team's and the organization's are too")
	}
}

func TestDeskMemories_SharedOnesNeedPermissionToKeep(t *testing.T) {
	t.Parallel()

	w := newDeskWorld()
	ctx := t.Context()

	own, err := w.svc.CreateDesk(ctx, &services.CreateDeskMemoryRequest{
		Actor: w.actor(false), Content: "  Granite pays by ACH only.  ", Scope: agent.MemoryScopeUser,
	})
	require.NoError(t, err)
	assert.Equal(t, "Granite pays by ACH only.", own.Memory.Content)
	assert.Equal(t, agent.MemoryScopeUser, own.Memory.Scope)
	assert.Equal(t, w.avery, *own.Memory.OwnerUserID)
	assert.Equal(t, agent.MemoryKindInstruction, own.Memory.Kind, "what a person writes down is a rule")
	assert.True(t, own.Editable)

	for _, scope := range []agent.MemoryScope{agent.MemoryScopeRole, agent.MemoryScopeOrganization} {
		_, err = w.svc.CreateDesk(ctx, &services.CreateDeskMemoryRequest{
			Actor: w.actor(false), Content: "Shared.", Scope: scope, RoleID: w.billing.ID,
		})
		require.Error(t, err, scope)
		assert.True(t, errortypes.IsAuthorizationError(err), scope)
	}

	team, err := w.svc.CreateDesk(ctx, &services.CreateDeskMemoryRequest{
		Actor: w.actor(true), Content: "Columbus loads need a lumper receipt.",
		Scope: agent.MemoryScopeRole, RoleID: w.billing.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, w.billing.ID, *team.Memory.RoleID)

	_, err = w.svc.CreateDesk(ctx, &services.CreateDeskMemoryRequest{
		Actor: w.actor(true), Content: "Not my team.", Scope: agent.MemoryScopeRole, RoleID: w.dispatch.ID,
	})
	require.Error(t, err, "a role the person does not hold is not theirs to keep memories for")

	_, err = w.svc.CreateDesk(ctx, &services.CreateDeskMemoryRequest{
		Actor: w.actor(true), Content: "Kept for one agent.", Scope: agent.MemoryScopeAgent,
	})
	require.Error(t, err, "agent memories are kept in AI Control")
}

func TestDeskMemories_MovingOneToTheTeamNeedsTheSamePermission(t *testing.T) {
	t.Parallel()

	w := newDeskWorld()
	own := w.keep("Group AR by facility.", agent.MemoryScopeUser, w.avery, pulid.Nil)

	_, err := w.svc.ReviseDesk(t.Context(), &services.ReviseDeskMemoryRequest{
		Actor: w.actor(false), ID: own.ID, Scope: agent.MemoryScopeRole, RoleID: w.billing.ID,
	})
	require.Error(t, err)

	edited, err := w.svc.ReviseDesk(t.Context(), &services.ReviseDeskMemoryRequest{
		Actor: w.actor(false), ID: own.ID, Content: "Group AR and detention by facility.",
	})
	require.NoError(t, err)
	assert.Equal(t, "Group AR and detention by facility.", edited.Memory.Content)
	assert.Equal(t, agent.MemoryScopeUser, edited.Memory.Scope, "an edit of the words keeps the readers")

	moved, err := w.svc.ReviseDesk(t.Context(), &services.ReviseDeskMemoryRequest{
		Actor: w.actor(true), ID: own.ID, Scope: agent.MemoryScopeRole, RoleID: w.billing.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, agent.MemoryScopeRole, moved.Memory.Scope)
	assert.Nil(t, moved.Memory.OwnerUserID)

	team := w.keep("Acme is billed net-45.", agent.MemoryScopeRole, pulid.Nil, w.billing.ID)
	_, err = w.svc.ReviseDesk(t.Context(), &services.ReviseDeskMemoryRequest{
		Actor: w.actor(false), ID: team.ID, Content: "Acme is billed net-30.",
	})
	require.Error(t, err, "the team's memory is not hers to rewrite without the permission")
}

// Pause sets a memory aside, Forget retires it, and Undo brings it back as it
// was; the Desk never deletes.
func TestDeskMemories_PauseResumeForgetAndUndo(t *testing.T) {
	t.Parallel()

	w := newDeskWorld()
	own := w.keep("Avery approves invoices after 4 PM.", agent.MemoryScopeUser, w.avery, pulid.Nil)
	step := func(status agent.MemoryStatus) *services.DeskMemory {
		t.Helper()
		changed, err := w.svc.SetDeskStatus(t.Context(), &services.SetDeskMemoryStatusRequest{
			Actor: w.actor(false), ID: own.ID, Status: status,
		})
		require.NoError(t, err)
		assert.Equal(t, status, changed.Memory.Status)

		return changed
	}

	step(agent.MemoryStatusPaused)
	step(agent.MemoryStatusActive)
	step(agent.MemoryStatusRetired)
	step(agent.MemoryStatusActive)

	_, err := w.svc.SetDeskStatus(t.Context(), &services.SetDeskMemoryStatusRequest{
		Actor: w.actor(false), ID: own.ID, Status: agent.MemoryStatusSuggested,
	})
	require.Error(t, err)

	someoneElses := w.keep("Jordan's note.", agent.MemoryScopeUser, pulid.MustNew("usr_"), pulid.Nil)
	_, err = w.svc.SetDeskStatus(t.Context(), &services.SetDeskMemoryStatusRequest{
		Actor: w.actor(true), ID: someoneElses.ID, Status: agent.MemoryStatusRetired,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err), "someone else's memory is not there to forget")
}

func TestDeskMemories_ListCountsEveryScopeAndPages(t *testing.T) {
	t.Parallel()

	w := newDeskWorld()
	w.keep("One.", agent.MemoryScopeOrganization, pulid.Nil, pulid.Nil)
	w.keep("Two.", agent.MemoryScopeUser, w.avery, pulid.Nil)
	w.keep("Three.", agent.MemoryScopeUser, w.avery, pulid.Nil)
	w.keep("Four.", agent.MemoryScopeRole, pulid.Nil, w.billing.ID)
	w.keep("Hidden.", agent.MemoryScopeRole, pulid.Nil, w.dispatch.ID)
	forgotten := w.keep("Forgotten.", agent.MemoryScopeUser, w.avery, pulid.Nil)
	forgotten.Status = agent.MemoryStatusRetired

	page, err := w.svc.ListDesk(t.Context(), &services.ListDeskMemoriesRequest{
		Actor: w.actor(false), Limit: 3,
	})
	require.NoError(t, err)
	require.Len(t, page.Items, 3)
	assert.Equal(t, "Four.", page.Items[0].Memory.Content, "newest first")
	assert.NotEmpty(t, page.Next)
	assert.Equal(t, 4, page.All, "forgotten and other teams' memories are not counted")

	counts := map[agent.MemoryScope]int{}
	for _, count := range page.Counts {
		counts[count.Scope] += count.Count
	}
	assert.Equal(t, map[agent.MemoryScope]int{
		agent.MemoryScopeOrganization: 1,
		agent.MemoryScopeUser:         2,
		agent.MemoryScopeRole:         1,
	}, counts)

	rest, err := w.svc.ListDesk(t.Context(), &services.ListDeskMemoriesRequest{
		Actor: w.actor(false), Limit: 3, After: page.Next,
	})
	require.NoError(t, err)
	require.Len(t, rest.Items, 1)
	assert.Equal(t, "One.", rest.Items[0].Memory.Content)
	assert.Empty(t, rest.Next)

	mine, err := w.svc.ListDesk(t.Context(), &services.ListDeskMemoriesRequest{
		Actor: w.actor(false), Scope: agent.MemoryScopeUser,
	})
	require.NoError(t, err)
	assert.Len(t, mine.Items, 2)
	assert.Equal(t, 4, mine.All, "the chips count every scope whichever one is chosen")

	_, err = w.svc.ListDesk(t.Context(), &services.ListDeskMemoriesRequest{
		Actor: w.actor(false), After: "not a cursor",
	})
	require.Error(t, err)
}

// A memory an agent offered waits for the person it was offered to. They
// save it as they edited it, for whom they chose, or turn it down.
func TestDeskMemories_AnOfferIsAcceptedOnlyByWhoItWasOfferedTo(t *testing.T) {
	t.Parallel()

	w := newDeskWorld()
	offer := func() *agent.Memory {
		avery := w.avery

		return w.repo.put(&agent.Memory{
			Kind: agent.MemoryKindFact, Source: agent.MemorySourceAgent,
			Status: agent.MemoryStatusSuggested, Content: "Columbus loads need a lumper receipt.",
			Scope: agent.MemoryScopeUser, OwnerUserID: &avery, CreatedByUserID: &avery,
		})
	}

	offered := offer()
	shown, err := w.svc.DeskByIDs(t.Context(), w.actor(false), []pulid.ID{offered.ID})
	require.NoError(t, err)
	require.Len(t, shown, 1, "the person it was offered to sees the offer")

	saved, err := w.svc.ConfirmDesk(t.Context(), &services.ConfirmDeskMemoryRequest{
		Actor: w.actor(false), ID: offered.ID, Content: "Columbus, OH loads need a lumper receipt.",
		Scope: agent.MemoryScopeUser,
	})
	require.NoError(t, err)
	assert.Equal(t, agent.MemoryStatusActive, saved.Memory.Status)
	assert.Equal(t, "Columbus, OH loads need a lumper receipt.", saved.Memory.Content)

	_, err = w.svc.ConfirmDesk(t.Context(), &services.ConfirmDeskMemoryRequest{
		Actor: w.actor(false), ID: offered.ID, Content: "Again.", Scope: agent.MemoryScopeUser,
	})
	require.Error(t, err, "an offer is answered once")

	declined := offer()
	dismissed, err := w.svc.DismissDesk(t.Context(), services.DeskMemoryRef{Actor: w.actor(false), ID: declined.ID})
	require.NoError(t, err)
	assert.Equal(t, agent.MemoryStatusDismissed, dismissed.Memory.Status)
	_, err = w.svc.DismissDesk(t.Context(), services.DeskMemoryRef{Actor: w.actor(false), ID: declined.ID})
	require.Error(t, err, "an offer is turned down once")
	reconsidered, err := w.svc.ConfirmDesk(t.Context(), &services.ConfirmDeskMemoryRequest{
		Actor: w.actor(false), ID: declined.ID, Content: declined.Content, Scope: agent.MemoryScopeUser,
	})
	require.NoError(t, err, "Undo of Don't save can still save it")
	assert.Equal(t, agent.MemoryStatusActive, reconsidered.Memory.Status)

	toTeam := offer()
	_, err = w.svc.ConfirmDesk(t.Context(), &services.ConfirmDeskMemoryRequest{
		Actor: w.actor(false), ID: toTeam.ID, Content: "For the team.",
		Scope: agent.MemoryScopeRole, RoleID: w.billing.ID,
	})
	require.Error(t, err, "saving it for the team needs the same permission as writing one")

	jordan := pulid.MustNew("usr_")
	someoneElses := w.repo.put(&agent.Memory{
		Kind: agent.MemoryKindFact, Source: agent.MemorySourceAgent,
		Status: agent.MemoryStatusSuggested, Content: "Jordan's offer.",
		Scope: agent.MemoryScopeUser, OwnerUserID: &jordan, CreatedByUserID: &jordan,
	})
	_, err = w.svc.ConfirmDesk(t.Context(), &services.ConfirmDeskMemoryRequest{
		Actor: w.actor(false), ID: someoneElses.ID, Content: "Mine now.", Scope: agent.MemoryScopeUser,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))
}

func TestSavingMode_AutomaticUntilThePersonChooses(t *testing.T) {
	t.Parallel()

	w := newDeskWorld()
	actor := w.actor(false)

	settings, err := w.svc.DeskSettings(t.Context(), actor)
	require.NoError(t, err)
	assert.Equal(t, agent.MemorySavingAutomatic, settings.SavingMode)
	require.Len(t, settings.Roles, 1)
	assert.Equal(t, "Billing", settings.Roles[0].Name)
	assert.False(t, settings.Roles[0].Writable)
	assert.False(t, settings.CanShareWithOrganization)

	settings, err = w.svc.SetSavingMode(t.Context(), actor, agent.MemorySavingAskFirst)
	require.NoError(t, err)
	assert.Equal(t, agent.MemorySavingAskFirst, settings.SavingMode)

	mode, err := w.svc.SavingMode(t.Context(), actor.Actor.TenantInfo(), w.avery)
	require.NoError(t, err)
	assert.Equal(t, agent.MemorySavingAskFirst, mode, "the remember tool reads what the page saved")

	_, err = w.svc.SetSavingMode(t.Context(), actor, agent.MemorySavingMode("Sometimes"))
	require.Error(t, err)
}

// A role inherits what its parents grant, so a memory kept for the parent
// reaches the holders of the role below it.
func TestReader_ReadsTheRolesAPersonInherits(t *testing.T) {
	t.Parallel()

	w := newDeskWorld()
	manager := &permission.Role{ID: pulid.MustNew("rol_"), Name: "Billing manager"}
	w.svc.roles = &fakeRoles{
		held:    []*permission.Role{manager},
		parents: map[pulid.ID][]*permission.Role{manager.ID: {w.billing}},
	}

	reader, err := w.svc.Reader(t.Context(), w.actor(false).Actor.TenantInfo(), w.avery)
	require.NoError(t, err)
	assert.Equal(t, w.avery, reader.UserID)
	assert.ElementsMatch(t, []pulid.ID{manager.ID, w.billing.ID}, reader.RoleIDs)
	assert.Equal(t, manager.ID, reader.RoleIDs[0], "the person's own role comes first")

	nobody, err := w.svc.Reader(t.Context(), w.actor(false).Actor.TenantInfo(), pulid.Nil)
	require.NoError(t, err)
	assert.Empty(t, nobody.RoleIDs)
}
