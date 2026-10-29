package agentmemoryservice

import (
	"cmp"
	"context"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentreflection"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// deskRepo keeps memories in a map and answers the Desk's reads the way the
// database does: by reader, by status, newest first.
type deskRepo struct {
	agentmemoryports.AgentMemoryRepository
	memories   map[pulid.ID]*agentmemory.Memory
	preference *agentmemory.MemoryPreference
	clock      int64
}

func newDeskRepo() *deskRepo {
	return &deskRepo{memories: map[pulid.ID]*agentmemory.Memory{}, clock: 1_760_000_000}
}

func (r *deskRepo) put(memory *agentmemory.Memory) *agentmemory.Memory {
	if memory.ID.IsNil() {
		memory.ID = pulid.MustNew("amem_")
	}
	if memory.Status == "" {
		memory.Status = agentmemory.MemoryStatusActive
	}
	if memory.Scope == "" {
		memory.Scope = agentmemory.MemoryScopeOrganization
	}
	r.clock++
	memory.CreatedAt = r.clock
	r.memories[memory.ID] = memory

	return memory
}

func (r *deskRepo) Create(
	_ context.Context,
	entity *agentmemory.Memory,
) (*agentmemory.Memory, error) {
	return r.put(entity), nil
}

func (r *deskRepo) FindActive(
	context.Context,
	agentmemoryports.FindActiveAgentMemoryRequest,
) (*agentmemory.Memory, error) {
	return nil, nil
}

func (r *deskRepo) GetByID(
	_ context.Context,
	req agentmemoryports.GetAgentMemoryByIDRequest,
) (*agentmemory.Memory, error) {
	memory, ok := r.memories[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Agent memory not found")
	}
	copied := *memory

	return &copied, nil
}

func (r *deskRepo) GetDesk(
	_ context.Context,
	req agentmemoryports.GetDeskMemoriesRequest,
) ([]*agentmemoryports.DeskMemoryRow, error) {
	rows := make([]*agentmemoryports.DeskMemoryRow, 0, len(req.IDs))
	for _, id := range req.IDs {
		if memory, ok := r.memories[id]; ok {
			copied := *memory
			rows = append(rows, &agentmemoryports.DeskMemoryRow{Memory: &copied})
		}
	}

	return rows, nil
}

func (r *deskRepo) ListByIDs(
	_ context.Context,
	req agentmemoryports.ListAgentMemoriesByIDsRequest,
) ([]*agentmemory.Memory, error) {
	out := make([]*agentmemory.Memory, 0, len(req.IDs))
	for _, id := range req.IDs {
		if memory, ok := r.memories[id]; ok {
			copied := *memory
			out = append(out, &copied)
		}
	}

	return out, nil
}

func (r *deskRepo) ListReplacements(
	_ context.Context,
	req agentmemoryports.ListAgentMemoryReplacementsRequest,
) ([]*agentmemory.Memory, error) {
	out := make([]*agentmemory.Memory, 0)
	for _, memory := range r.memories {
		if memory.SupersedesID != nil && slices.Contains(req.ReplacedIDs, *memory.SupersedesID) {
			copied := *memory
			out = append(out, &copied)
		}
	}

	return out, nil
}

func (r *deskRepo) visible(filter *agentmemoryports.DeskMemoryFilter) []*agentmemory.Memory {
	out := make([]*agentmemory.Memory, 0, len(r.memories))
	for _, memory := range r.memories {
		if memory.Status != agentmemory.MemoryStatusActive &&
			memory.Status != agentmemory.MemoryStatusPaused {
			continue
		}
		if memory.Scope == agentmemory.MemoryScopeAgent || !filter.Reader.Reads(memory) {
			continue
		}
		out = append(out, memory)
	}
	slices.SortFunc(
		out,
		func(a, b *agentmemory.Memory) int { return cmp.Compare(b.CreatedAt, a.CreatedAt) },
	)

	return out
}

func (r *deskRepo) ListDesk(
	_ context.Context,
	req agentmemoryports.ListDeskMemoriesQuery,
) ([]*agentmemoryports.DeskMemoryRow, error) {
	rows := make([]*agentmemoryports.DeskMemoryRow, 0)
	for _, memory := range r.visible(&req.Filter) {
		if req.Filter.Scope != "" && memory.Scope != req.Filter.Scope {
			continue
		}
		if req.After != nil && memory.CreatedAt >= req.After.CreatedAt {
			continue
		}
		copied := *memory
		rows = append(rows, &agentmemoryports.DeskMemoryRow{Memory: &copied})
		if len(rows) == req.Limit {
			break
		}
	}

	return rows, nil
}

func (r *deskRepo) CountDesk(
	_ context.Context,
	filter agentmemoryports.DeskMemoryFilter,
) ([]agentmemoryports.DeskMemoryCountRow, error) {
	counts := map[string]*agentmemoryports.DeskMemoryCountRow{}
	order := make([]string, 0)
	for _, memory := range r.visible(&filter) {
		roleID := pulid.Nil
		if memory.RoleID != nil {
			roleID = *memory.RoleID
		}
		key := string(memory.Scope) + roleID.String()
		if counts[key] == nil {
			counts[key] = &agentmemoryports.DeskMemoryCountRow{Scope: memory.Scope, RoleID: roleID}
			order = append(order, key)
		}
		counts[key].Count++
	}

	out := make([]agentmemoryports.DeskMemoryCountRow, 0, len(order))
	for _, key := range order {
		out = append(out, *counts[key])
	}

	return out, nil
}

func (r *deskRepo) SetStatus(
	_ context.Context,
	req agentmemoryports.SetAgentMemoryStatusQuery,
) (*agentmemory.Memory, error) {
	memory := r.memories[req.ID]
	memory.Status = req.Status
	memory.Version++
	copied := *memory

	return &copied, nil
}

func (r *deskRepo) Revise(
	_ context.Context,
	req agentmemoryports.ReviseAgentMemoryRequest,
) (*agentmemory.Memory, error) {
	memory := r.memories[req.ID]
	memory.Content = req.Content
	memory.SetAudience(req.Scope, req.OwnerUserID, req.RoleID)
	memory.Version++
	copied := *memory

	return &copied, nil
}

func (r *deskRepo) ResolveSuggestion(
	_ context.Context,
	req agentmemoryports.ResolveAgentMemorySuggestionRequest,
) (*agentmemory.Memory, error) {
	memory := r.memories[req.ID]
	memory.Status = req.Status
	if req.Status == agentmemory.MemoryStatusActive {
		memory.Content = req.Content
		memory.SetAudience(req.Scope, req.OwnerUserID, req.RoleID)
	}
	memory.Version++
	copied := *memory

	return &copied, nil
}

func (r *deskRepo) GetPreference(
	context.Context,
	agentmemoryports.GetAgentMemoryPreferenceRequest,
) (*agentmemory.MemoryPreference, error) {
	return r.preference, nil
}

func (r *deskRepo) SavePreference(
	_ context.Context,
	entity *agentmemory.MemoryPreference,
) (*agentmemory.MemoryPreference, error) {
	r.preference = entity

	return entity, nil
}

// fakeRoles holds the person's roles and what each inherits.
type fakeRoles struct {
	held    []*rbac.Role
	parents map[pulid.ID][]*rbac.Role
}

func (f *fakeRoles) GetUserRoleAssignments(
	context.Context,
	pulid.ID,
	pulid.ID,
) ([]*rbac.UserRoleAssignment, error) {
	out := make([]*rbac.UserRoleAssignment, 0, len(f.held))
	for _, role := range f.held {
		out = append(out, &rbac.UserRoleAssignment{RoleID: role.ID, Role: role})
	}

	return out, nil
}

func (f *fakeRoles) GetRolesWithInheritance(
	_ context.Context,
	ids []pulid.ID,
) ([]*rbac.Role, error) {
	out := make([]*rbac.Role, 0, len(ids))
	for _, id := range ids {
		out = append(out, &rbac.Role{ID: id})
		out = append(out, f.parents[id]...)
	}

	return out, nil
}

type deskWorld struct {
	svc      *Service
	repo     *deskRepo
	avery    pulid.ID
	billing  *rbac.Role
	dispatch *rbac.Role
}

func newDeskWorld() *deskWorld {
	repo := newDeskRepo()
	billing := &rbac.Role{ID: pulid.MustNew("rol_"), Name: "Billing"}
	dispatch := &rbac.Role{ID: pulid.MustNew("rol_"), Name: "Dispatch"}

	return &deskWorld{
		svc: &Service{
			l:     zap.NewNop(),
			repo:  repo,
			runs:  &fakeRuns{},
			roles: &fakeRoles{held: []*rbac.Role{billing}},
		},
		repo:     repo,
		avery:    pulid.MustNew("usr_"),
		billing:  billing,
		dispatch: dispatch,
	}
}

// actor is Avery on the Desk, with or without the agent-memory manage permission.
func (w *deskWorld) actor(shared bool) *agentmemoryports.DeskMemoryActor {
	return &agentmemoryports.DeskMemoryActor{
		Actor: &permissionports.RequestActor{
			PrincipalType:  permissionports.PrincipalTypeUser,
			PrincipalID:    w.avery,
			UserID:         w.avery,
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
		},
		MayManageShared: shared,
	}
}

func (w *deskWorld) keep(
	content string,
	scope agentmemory.MemoryScope,
	owner, role pulid.ID,
) *agentmemory.Memory {
	memory := &agentmemory.Memory{
		Kind:    agentmemory.MemoryKindInstruction,
		Source:  agentmemory.MemorySourceUser,
		Content: content,
	}
	memory.SetAudience(scope, owner, role)

	return w.repo.put(memory)
}

func deskIDs(memories []*agentmemoryports.DeskMemory) []pulid.ID {
	ids := make([]pulid.ID, 0, len(memories))
	for _, memory := range memories {
		ids = append(ids, memory.Memory.ID)
	}

	return ids
}

func TestDeskMemories_ShowOnlyTheirOwnTheirRolesAndTheOrganizations(t *testing.T) {
	t.Parallel()

	w := newDeskWorld()
	organization := w.keep(
		"Use the DOE weekly average.",
		agentmemory.MemoryScopeOrganization,
		pulid.Nil,
		pulid.Nil,
	)
	own := w.keep("Group AR by facility.", agentmemory.MemoryScopeUser, w.avery, pulid.Nil)
	someoneElses := w.keep(
		"Jordan's note.",
		agentmemory.MemoryScopeUser,
		pulid.MustNew("usr_"),
		pulid.Nil,
	)
	team := w.keep("Acme is billed net-45.", agentmemory.MemoryScopeRole, pulid.Nil, w.billing.ID)
	otherTeam := w.keep("Dispatch note.", agentmemory.MemoryScopeRole, pulid.Nil, w.dispatch.ID)
	agentOnly := w.repo.put(&agentmemory.Memory{
		Kind: agentmemory.MemoryKindFact, Source: agentmemory.MemorySourceFeedback, Content: "Agent's own.",
		Scope: agentmemory.MemoryScopeAgent,
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
	assert.Equal(
		t,
		map[pulid.ID]bool{organization.ID: false, own.ID: true, team.ID: false},
		editable,
		"without the permission only her own is hers to change",
	)

	shown, err = w.svc.DeskByIDs(t.Context(), w.actor(true), []pulid.ID{organization.ID, team.ID})
	require.NoError(t, err)
	for _, memory := range shown {
		assert.True(t, memory.Editable, "with it her team's and the organization's are too")
	}
}

// Without the permission to manage shared memory a person still writes one
// for their team or the organization, but it is offered: kept as a suggestion
// an administrator approves, shown to them as theirs to withdraw, and read by
// no agent until then.
func TestDeskMemories_SharedOnesAreOfferedWithoutThePermission(t *testing.T) {
	t.Parallel()

	w := newDeskWorld()
	ctx := t.Context()

	own, err := w.svc.CreateDesk(ctx, &agentmemoryports.CreateDeskMemoryRequest{
		Actor: w.actor(
			false,
		), Content: "  Granite pays by ACH only.  ", Scope: agentmemory.MemoryScopeUser,
	})
	require.NoError(t, err)
	assert.Equal(t, "Granite pays by ACH only.", own.Memory.Content)
	assert.Equal(t, agentmemory.MemoryScopeUser, own.Memory.Scope)
	assert.Equal(t, w.avery, *own.Memory.OwnerUserID)
	assert.Equal(
		t,
		agentmemory.MemoryKindInstruction,
		own.Memory.Kind,
		"what a person writes down is a rule",
	)
	assert.Equal(
		t,
		agentmemory.MemoryStatusActive,
		own.Memory.Status,
		"a person's own is kept at once",
	)
	assert.True(t, own.Editable)

	for _, scope := range []agentmemory.MemoryScope{agentmemory.MemoryScopeRole, agentmemory.MemoryScopeOrganization} {
		offered, offerErr := w.svc.CreateDesk(ctx, &agentmemoryports.CreateDeskMemoryRequest{
			Actor: w.actor(false), Content: "Shared.", Scope: scope, RoleID: w.billing.ID,
		})
		require.NoError(t, offerErr, scope)
		assert.Equal(t, agentmemory.MemoryStatusSuggested, offered.Memory.Status, scope)
		assert.Equal(t, scope, offered.Memory.Scope, "offering keeps who it is for")
		assert.Equal(t, agentmemory.MemorySourceUser, offered.Memory.Source)
		assert.Equal(t, w.avery, *offered.Memory.CreatedByUserID, "the offer is theirs to withdraw")
		assert.False(t, offered.Editable, "an offer waits for an administrator")

		withdrawn, dismissErr := w.svc.DismissDesk(ctx, agentmemoryports.DeskMemoryRef{
			Actor: w.actor(false), ID: offered.Memory.ID,
		})
		require.NoError(t, dismissErr, scope)
		assert.Equal(t, agentmemory.MemoryStatusDismissed, withdrawn.Memory.Status)
	}

	offered, err := w.svc.CreateDesk(ctx, &agentmemoryports.CreateDeskMemoryRequest{
		Actor: w.actor(
			false,
		), Content: "Granite wants the POD on the invoice.", Scope: agentmemory.MemoryScopeOrganization,
	})
	require.NoError(t, err)
	_, err = w.svc.ConfirmDesk(ctx, &agentmemoryports.ConfirmDeskMemoryRequest{
		Actor: w.actor(false), ID: offered.Memory.ID, Content: offered.Memory.Content,
		Scope: agentmemory.MemoryScopeOrganization, Version: offered.Memory.Version,
	})
	require.Error(
		t,
		err,
		"what a person offered for others is an administrator's to keep, not theirs",
	)

	team, err := w.svc.CreateDesk(ctx, &agentmemoryports.CreateDeskMemoryRequest{
		Actor: w.actor(true), Content: "Columbus loads need a lumper receipt.",
		Scope: agentmemory.MemoryScopeRole, RoleID: w.billing.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, w.billing.ID, *team.Memory.RoleID)

	_, err = w.svc.CreateDesk(ctx, &agentmemoryports.CreateDeskMemoryRequest{
		Actor: w.actor(
			true,
		), Content: "Not my team.", Scope: agentmemory.MemoryScopeRole, RoleID: w.dispatch.ID,
	})
	require.Error(t, err, "a role the person does not hold is not theirs to keep memories for")

	_, err = w.svc.CreateDesk(ctx, &agentmemoryports.CreateDeskMemoryRequest{
		Actor: w.actor(true), Content: "Kept for one agent.", Scope: agentmemory.MemoryScopeAgent,
	})
	require.Error(t, err, "agent memories are kept in AI Control")
}

func TestDeskMemories_MovingOneToTheTeamNeedsTheSamePermission(t *testing.T) {
	t.Parallel()

	w := newDeskWorld()
	own := w.keep("Group AR by facility.", agentmemory.MemoryScopeUser, w.avery, pulid.Nil)

	_, err := w.svc.ReviseDesk(t.Context(), &agentmemoryports.ReviseDeskMemoryRequest{
		Actor: w.actor(false), ID: own.ID, Scope: agentmemory.MemoryScopeRole, RoleID: w.billing.ID,
	})
	require.Error(t, err)

	edited, err := w.svc.ReviseDesk(t.Context(), &agentmemoryports.ReviseDeskMemoryRequest{
		Actor: w.actor(false), ID: own.ID, Content: "Group AR and detention by facility.",
	})
	require.NoError(t, err)
	assert.Equal(t, "Group AR and detention by facility.", edited.Memory.Content)
	assert.Equal(
		t,
		agentmemory.MemoryScopeUser,
		edited.Memory.Scope,
		"an edit of the words keeps the readers",
	)

	moved, err := w.svc.ReviseDesk(t.Context(), &agentmemoryports.ReviseDeskMemoryRequest{
		Actor: w.actor(true), ID: own.ID, Scope: agentmemory.MemoryScopeRole, RoleID: w.billing.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, agentmemory.MemoryScopeRole, moved.Memory.Scope)
	assert.Nil(t, moved.Memory.OwnerUserID)

	team := w.keep("Acme is billed net-45.", agentmemory.MemoryScopeRole, pulid.Nil, w.billing.ID)
	_, err = w.svc.ReviseDesk(t.Context(), &agentmemoryports.ReviseDeskMemoryRequest{
		Actor: w.actor(false), ID: team.ID, Content: "Acme is billed net-30.",
	})
	require.Error(t, err, "the team's memory is not hers to rewrite without the permission")
}

// Pause sets a memory aside, Forget retires it, and Undo brings it back as it
// was; the Desk never deletes.
func TestDeskMemories_PauseResumeForgetAndUndo(t *testing.T) {
	t.Parallel()

	w := newDeskWorld()
	own := w.keep(
		"Avery approves invoices after 4 PM.",
		agentmemory.MemoryScopeUser,
		w.avery,
		pulid.Nil,
	)
	step := func(status agentmemory.MemoryStatus) *agentmemoryports.DeskMemory {
		t.Helper()
		changed, err := w.svc.SetDeskStatus(
			t.Context(),
			&agentmemoryports.SetDeskMemoryStatusRequest{
				Actor: w.actor(false), ID: own.ID, Status: status,
			},
		)
		require.NoError(t, err)
		assert.Equal(t, status, changed.Memory.Status)

		return changed
	}

	step(agentmemory.MemoryStatusPaused)
	step(agentmemory.MemoryStatusActive)
	step(agentmemory.MemoryStatusRetired)
	step(agentmemory.MemoryStatusActive)

	_, err := w.svc.SetDeskStatus(t.Context(), &agentmemoryports.SetDeskMemoryStatusRequest{
		Actor: w.actor(false), ID: own.ID, Status: agentmemory.MemoryStatusSuggested,
	})
	require.Error(t, err)

	someoneElses := w.keep(
		"Jordan's note.",
		agentmemory.MemoryScopeUser,
		pulid.MustNew("usr_"),
		pulid.Nil,
	)
	_, err = w.svc.SetDeskStatus(t.Context(), &agentmemoryports.SetDeskMemoryStatusRequest{
		Actor: w.actor(true), ID: someoneElses.ID, Status: agentmemory.MemoryStatusRetired,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err), "someone else's memory is not there to forget")
}

func TestDeskMemories_ListCountsEveryScopeAndPages(t *testing.T) {
	t.Parallel()

	w := newDeskWorld()
	w.keep("One.", agentmemory.MemoryScopeOrganization, pulid.Nil, pulid.Nil)
	w.keep("Two.", agentmemory.MemoryScopeUser, w.avery, pulid.Nil)
	w.keep("Three.", agentmemory.MemoryScopeUser, w.avery, pulid.Nil)
	w.keep("Four.", agentmemory.MemoryScopeRole, pulid.Nil, w.billing.ID)
	w.keep("Hidden.", agentmemory.MemoryScopeRole, pulid.Nil, w.dispatch.ID)
	forgotten := w.keep("Forgotten.", agentmemory.MemoryScopeUser, w.avery, pulid.Nil)
	forgotten.Status = agentmemory.MemoryStatusRetired

	page, err := w.svc.ListDesk(t.Context(), &agentmemoryports.ListDeskMemoriesRequest{
		Actor: w.actor(false), Limit: 3,
	})
	require.NoError(t, err)
	require.Len(t, page.Items, 3)
	assert.Equal(t, "Four.", page.Items[0].Memory.Content, "newest first")
	assert.NotEmpty(t, page.Next)
	assert.Equal(t, 4, page.All, "forgotten and other teams' memories are not counted")

	counts := map[agentmemory.MemoryScope]int{}
	for _, count := range page.Counts {
		counts[count.Scope] += count.Count
	}
	assert.Equal(t, map[agentmemory.MemoryScope]int{
		agentmemory.MemoryScopeOrganization: 1,
		agentmemory.MemoryScopeUser:         2,
		agentmemory.MemoryScopeRole:         1,
	}, counts)

	rest, err := w.svc.ListDesk(t.Context(), &agentmemoryports.ListDeskMemoriesRequest{
		Actor: w.actor(false), Limit: 3, After: page.Next,
	})
	require.NoError(t, err)
	require.Len(t, rest.Items, 1)
	assert.Equal(t, "One.", rest.Items[0].Memory.Content)
	assert.Empty(t, rest.Next)

	mine, err := w.svc.ListDesk(t.Context(), &agentmemoryports.ListDeskMemoriesRequest{
		Actor: w.actor(false), Scope: agentmemory.MemoryScopeUser,
	})
	require.NoError(t, err)
	assert.Len(t, mine.Items, 2)
	assert.Equal(t, 4, mine.All, "the chips count every scope whichever one is chosen")

	_, err = w.svc.ListDesk(t.Context(), &agentmemoryports.ListDeskMemoriesRequest{
		Actor: w.actor(false), After: "not a cursor",
	})
	require.Error(t, err)
}

// A memory an agent offered waits for the person it was offered to. They
// save it as they edited it, for whom they chose, or turn it down.
func TestDeskMemories_AnOfferIsAcceptedOnlyByWhoItWasOfferedTo(t *testing.T) {
	t.Parallel()

	w := newDeskWorld()
	offer := func() *agentmemory.Memory {
		avery := w.avery

		return w.repo.put(&agentmemory.Memory{
			Kind: agentmemory.MemoryKindFact, Source: agentmemory.MemorySourceAgent,
			Status: agentmemory.MemoryStatusSuggested, Content: "Columbus loads need a lumper receipt.",
			Scope: agentmemory.MemoryScopeUser, OwnerUserID: &avery, CreatedByUserID: &avery,
		})
	}

	offered := offer()
	shown, err := w.svc.DeskByIDs(t.Context(), w.actor(false), []pulid.ID{offered.ID})
	require.NoError(t, err)
	require.Len(t, shown, 1, "the person it was offered to sees the offer")

	saved, err := w.svc.ConfirmDesk(t.Context(), &agentmemoryports.ConfirmDeskMemoryRequest{
		Actor: w.actor(false), ID: offered.ID, Content: "Columbus, OH loads need a lumper receipt.",
		Scope: agentmemory.MemoryScopeUser,
	})
	require.NoError(t, err)
	assert.Equal(t, agentmemory.MemoryStatusActive, saved.Memory.Status)
	assert.Equal(t, "Columbus, OH loads need a lumper receipt.", saved.Memory.Content)

	_, err = w.svc.ConfirmDesk(t.Context(), &agentmemoryports.ConfirmDeskMemoryRequest{
		Actor: w.actor(
			false,
		), ID: offered.ID, Content: "Again.", Scope: agentmemory.MemoryScopeUser,
	})
	require.Error(t, err, "an offer is answered once")

	declined := offer()
	dismissed, err := w.svc.DismissDesk(
		t.Context(),
		agentmemoryports.DeskMemoryRef{Actor: w.actor(false), ID: declined.ID},
	)
	require.NoError(t, err)
	assert.Equal(t, agentmemory.MemoryStatusDismissed, dismissed.Memory.Status)
	_, err = w.svc.DismissDesk(
		t.Context(),
		agentmemoryports.DeskMemoryRef{Actor: w.actor(false), ID: declined.ID},
	)
	require.Error(t, err, "an offer is turned down once")
	reconsidered, err := w.svc.ConfirmDesk(t.Context(), &agentmemoryports.ConfirmDeskMemoryRequest{
		Actor: w.actor(
			false,
		), ID: declined.ID, Content: declined.Content, Scope: agentmemory.MemoryScopeUser,
	})
	require.NoError(t, err, "Undo of Don't save can still save it")
	assert.Equal(t, agentmemory.MemoryStatusActive, reconsidered.Memory.Status)

	toTeam := offer()
	_, err = w.svc.ConfirmDesk(t.Context(), &agentmemoryports.ConfirmDeskMemoryRequest{
		Actor: w.actor(false), ID: toTeam.ID, Content: "For the team.",
		Scope: agentmemory.MemoryScopeRole, RoleID: w.billing.ID,
	})
	require.Error(t, err, "saving it for the team needs the same permission as writing one")

	jordan := pulid.MustNew("usr_")
	someoneElses := w.repo.put(&agentmemory.Memory{
		Kind: agentmemory.MemoryKindFact, Source: agentmemory.MemorySourceAgent,
		Status: agentmemory.MemoryStatusSuggested, Content: "Jordan's offer.",
		Scope: agentmemory.MemoryScopeUser, OwnerUserID: &jordan, CreatedByUserID: &jordan,
	})
	_, err = w.svc.ConfirmDesk(t.Context(), &agentmemoryports.ConfirmDeskMemoryRequest{
		Actor: w.actor(
			false,
		), ID: someoneElses.ID, Content: "Mine now.", Scope: agentmemory.MemoryScopeUser,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))
}

// What Avery offered for the organization is in front of an administrator:
// she may withdraw it, but not rewrite it, pause it or keep it herself.
func TestDeskMemories_AnOwnOfferIsOnlyTheAuthorsToWithdraw(t *testing.T) {
	t.Parallel()

	w := newDeskWorld()
	avery := w.avery
	offer := w.repo.put(&agentmemory.Memory{
		Kind: agentmemory.MemoryKindFact, Source: agentmemory.MemorySourceUser,
		Status: agentmemory.MemoryStatusSuggested, Content: "Granite wants the POD on the invoice.",
		Scope: agentmemory.MemoryScopeOrganization, CreatedByUserID: &avery,
	})

	_, err := w.svc.ReviseDesk(t.Context(), &agentmemoryports.ReviseDeskMemoryRequest{
		Actor: w.actor(false), ID: offer.ID, Content: "Granite wants the POD and the BOL.",
	})
	require.Error(t, err, "an offer is not rewritten under the administrator")

	_, err = w.svc.SetDeskStatus(t.Context(), &agentmemoryports.SetDeskMemoryStatusRequest{
		Actor: w.actor(false), ID: offer.ID, Status: agentmemory.MemoryStatusPaused,
	})
	require.Error(t, err, "an offer has no status of its own to change")

	_, err = w.svc.ConfirmDesk(t.Context(), &agentmemoryports.ConfirmDeskMemoryRequest{
		Actor: w.actor(
			false,
		), ID: offer.ID, Content: offer.Content, Scope: agentmemory.MemoryScopeOrganization,
	})
	require.Error(t, err, "the author does not approve their own offer")

	withdrawn, err := w.svc.DismissDesk(
		t.Context(),
		agentmemoryports.DeskMemoryRef{Actor: w.actor(false), ID: offer.ID},
	)
	require.NoError(t, err)
	assert.Equal(t, agentmemory.MemoryStatusDismissed, withdrawn.Memory.Status)
}

// An agent offered Avery a memory for the organization. Without the manage
// permission she cannot keep it for everyone, but she may keep a copy for
// herself; with it she keeps it for everyone.
func TestDeskMemories_AnAgentsOrganizationOfferIsKeptForEveryoneOnlyByAManager(t *testing.T) {
	t.Parallel()

	w := newDeskWorld()
	avery := w.avery
	offer := func() *agentmemory.Memory {
		return w.repo.put(&agentmemory.Memory{
			Kind: agentmemory.MemoryKindFact, Source: agentmemory.MemorySourceAgent,
			Status: agentmemory.MemoryStatusSuggested, Content: "Columbus loads need a lumper receipt.",
			Scope: agentmemory.MemoryScopeOrganization, CreatedByUserID: &avery,
		})
	}

	forEveryone := offer()
	_, err := w.svc.ConfirmDesk(t.Context(), &agentmemoryports.ConfirmDeskMemoryRequest{
		Actor: w.actor(false), ID: forEveryone.ID, Content: forEveryone.Content,
		Scope: agentmemory.MemoryScopeOrganization,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))

	own, err := w.svc.ConfirmDesk(t.Context(), &agentmemoryports.ConfirmDeskMemoryRequest{
		Actor: w.actor(false), ID: forEveryone.ID, Content: forEveryone.Content,
		Scope: agentmemory.MemoryScopeUser,
	})
	require.NoError(t, err)
	assert.Equal(t, agentmemory.MemoryStatusActive, own.Memory.Status)
	assert.Equal(t, agentmemory.MemoryScopeUser, own.Memory.Scope)
	require.NotNil(t, own.Memory.OwnerUserID)
	assert.Equal(t, avery, *own.Memory.OwnerUserID)

	kept, err := w.svc.ConfirmDesk(t.Context(), &agentmemoryports.ConfirmDeskMemoryRequest{
		Actor: w.actor(true), ID: offer().ID, Content: "Columbus loads need a lumper receipt.",
		Scope: agentmemory.MemoryScopeOrganization,
	})
	require.NoError(t, err)
	assert.Equal(t, agentmemory.MemoryScopeOrganization, kept.Memory.Scope)
	assert.Equal(t, agentmemory.MemoryStatusActive, kept.Memory.Status)
}

func TestDeskMemories_NameWhatTheyReplacedAndWhatReplacedThem(t *testing.T) {
	t.Parallel()

	w := newDeskWorld()
	old := w.keep(
		"Copy dispatch on rate confirmations.",
		agentmemory.MemoryScopeOrganization,
		pulid.Nil,
		pulid.Nil,
	)
	old.Status = agentmemory.MemoryStatusRetired
	offered := w.repo.put(&agentmemory.Memory{
		Kind:         agentmemory.MemoryKindProcedure,
		Source:       agentmemory.MemorySourceReflection,
		Status:       agentmemory.MemoryStatusSuggested,
		Scope:        agentmemory.MemoryScopeOrganization,
		Content:      "Copy dispatch and the carrier rep.",
		SupersedesID: &old.ID,
	})
	current := w.repo.put(&agentmemory.Memory{
		Kind:         agentmemory.MemoryKindProcedure,
		Source:       agentmemory.MemorySourceReflection,
		Scope:        agentmemory.MemoryScopeOrganization,
		Content:      "Copy dispatch and billing on rate confirmations.",
		SupersedesID: &old.ID,
		Evidence: &agentmemory.MemoryEvidence{
			Reason:  "The person asked for billing to be copied as well.",
			Quotes:  []string{"Billing needs these too"},
			Signals: []string{string(agentreflection.ReflectionSignalPersonCorrected)},
		},
	})
	jordans := w.keep(
		"Jordan's own way.",
		agentmemory.MemoryScopeUser,
		pulid.MustNew("usr_"),
		pulid.Nil,
	)
	mine := w.keep("Copy me as well.", agentmemory.MemoryScopeUser, w.avery, pulid.Nil)
	mine.SupersedesID = &jordans.ID

	page, err := w.svc.ListDesk(t.Context(), &agentmemoryports.ListDeskMemoriesRequest{
		Actor: w.actor(false),
		Limit: 10,
	})
	require.NoError(t, err)
	byID := map[pulid.ID]*agentmemoryports.DeskMemory{}
	for _, item := range page.Items {
		byID[item.Memory.ID] = item
	}

	require.Contains(t, byID, current.ID)
	require.NotNil(t, byID[current.ID].Replaces)
	assert.Equal(t, old.ID, byID[current.ID].Replaces.ID)
	assert.Equal(t, old.Content, byID[current.ID].Replaces.Content)
	assert.Equal(t, agentmemory.MemoryStatusRetired, byID[current.ID].Replaces.Status)
	assert.Nil(t, byID[current.ID].ReplacedBy)

	require.Contains(t, byID, mine.ID)
	assert.Nil(t, byID[mine.ID].Replaces, "someone else's memory is not named")
	assert.NotContains(t, byID, offered.ID)

	shown, err := w.svc.DeskByIDs(t.Context(), w.actor(false), []pulid.ID{old.ID})
	require.NoError(t, err)
	require.Len(t, shown, 1)
	require.NotNil(t, shown[0].ReplacedBy, "the retired memory names what took its place")
	assert.Equal(t, current.ID, shown[0].ReplacedBy.ID,
		"an offer still waiting is not what replaced it")
}

func TestSavingMode_AutomaticUntilThePersonChooses(t *testing.T) {
	t.Parallel()

	w := newDeskWorld()
	actor := w.actor(false)

	settings, err := w.svc.DeskSettings(t.Context(), actor)
	require.NoError(t, err)
	assert.Equal(t, agentmemory.MemorySavingAutomatic, settings.SavingMode)
	require.Len(t, settings.Roles, 1)
	assert.Equal(t, "Billing", settings.Roles[0].Name)
	assert.True(t, settings.Roles[0].Writable, "anyone may at least offer a memory for their team")
	assert.True(t, settings.CanShareWithOrganization, "and for the organization")
	assert.False(t, settings.ManagesShared, "but without the permission it is offered, not kept")

	managed, err := w.svc.DeskSettings(t.Context(), w.actor(true))
	require.NoError(t, err)
	assert.True(t, managed.ManagesShared)

	settings, err = w.svc.SetSavingMode(t.Context(), actor, agentmemory.MemorySavingAskFirst)
	require.NoError(t, err)
	assert.Equal(t, agentmemory.MemorySavingAskFirst, settings.SavingMode)

	mode, err := w.svc.SavingMode(t.Context(), actor.Actor.TenantInfo(), w.avery)
	require.NoError(t, err)
	assert.Equal(
		t,
		agentmemory.MemorySavingAskFirst,
		mode,
		"the remember tool reads what the page saved",
	)

	_, err = w.svc.SetSavingMode(t.Context(), actor, agentmemory.MemorySavingMode("Sometimes"))
	require.Error(t, err)
}

// A role inherits what its parents grant, so a memory kept for the parent
// reaches the holders of the role below it.
func TestReader_ReadsTheRolesAPersonInherits(t *testing.T) {
	t.Parallel()

	w := newDeskWorld()
	manager := &rbac.Role{ID: pulid.MustNew("rol_"), Name: "Billing manager"}
	w.svc.roles = &fakeRoles{
		held:    []*rbac.Role{manager},
		parents: map[pulid.ID][]*rbac.Role{manager.ID: {w.billing}},
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
