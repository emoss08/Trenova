package seedaccountservice

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/cloud/seedaccounts/seedaccountport"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

const (
	seededHash  = "seeded-hash"
	changedHash = "changed-hash"
	retiredHash = "retired-hash"
	fixedNow    = int64(1_790_000_000)
)

type orgMember struct {
	id       pulid.ID
	username string
}

type fakeRepo struct {
	system     *seedaccountport.SystemUser
	users      []*seedaccountport.UserFootprint
	orgs       []*seedaccountport.OrganizationFootprint
	members    map[pulid.ID][]orgMember
	references map[pulid.ID][]seedaccountport.Reference
	retainOrg  map[pulid.ID]string

	readOnly     []bool
	inReadOnly   bool
	writesInRead int
	stripped     []pulid.ID
	removed      []*seedaccountport.RemoveUserRequest
	orgsRemoved  []pulid.ID
	excludeUsers [][]pulid.ID
}

func (f *fakeRepo) InTransaction(
	ctx context.Context,
	opts seedaccountport.TransactionOptions,
	fn func(ctx context.Context) error,
) error {
	f.readOnly = append(f.readOnly, opts.ReadOnly)
	f.inReadOnly = opts.ReadOnly
	defer func() { f.inReadOnly = false }()

	return fn(ctx)
}

func (f *fakeRepo) FindUsers(
	_ context.Context,
	_ *seedaccountport.FindUsersRequest,
) ([]*seedaccountport.UserFootprint, error) {
	out := make([]*seedaccountport.UserFootprint, 0, len(f.users))
	for _, user := range f.users {
		clone := *user
		out = append(out, &clone)
	}

	return out, nil
}

func (f *fakeRepo) CountReferences(
	_ context.Context,
	req *seedaccountport.CountReferencesRequest,
) ([]seedaccountport.Reference, error) {
	return slices.Clone(f.references[req.UserID]), nil
}

func (f *fakeRepo) FindSystemUser(context.Context) (*seedaccountport.SystemUser, error) {
	return f.system, nil
}

func (f *fakeRepo) InspectOrganizations(
	_ context.Context,
	req *seedaccountport.InspectOrganizationsRequest,
) ([]*seedaccountport.OrganizationFootprint, error) {
	f.excludeUsers = append(f.excludeUsers, slices.Clone(req.ExcludeUsers))
	out := make([]*seedaccountport.OrganizationFootprint, 0, len(f.orgs))
	for _, org := range f.orgs {
		if slices.Contains(f.orgsRemoved, org.ID) {
			continue
		}
		clone := *org
		clone.Users = make([]string, 0)
		for _, member := range f.members[org.ID] {
			if slices.Contains(req.ExcludeUsers, member.id) || f.deleted(member.id) {
				continue
			}
			clone.Users = append(clone.Users, member.username)
		}
		out = append(out, &clone)
	}

	return out, nil
}

func (f *fakeRepo) deleted(id pulid.ID) bool {
	for _, req := range f.removed {
		if req.UserID == id && len(f.references[id]) == 0 {
			return true
		}
	}

	return false
}

func (f *fakeRepo) StripUser(
	_ context.Context,
	req *seedaccountport.StripUserRequest,
) (*seedaccountport.StripUserResult, error) {
	if f.inReadOnly {
		f.writesInRead++
	}
	f.stripped = append(f.stripped, req.UserID)
	for _, user := range f.users {
		if user.ID != req.UserID {
			continue
		}
		out := &seedaccountport.StripUserResult{
			Memberships:       user.Memberships,
			RoleAssignments:   user.RoleAssignments,
			APIKeys:           user.ActiveAPIKeys,
			MFAAuthenticators: user.MFAAuthenticators,
			ResetTokens:       user.OpenResetTokens,
		}
		return out, nil
	}

	return &seedaccountport.StripUserResult{}, nil
}

func (f *fakeRepo) RemoveUser(
	_ context.Context,
	req *seedaccountport.RemoveUserRequest,
) (*seedaccountport.RemoveUserResult, error) {
	if f.inReadOnly {
		f.writesInRead++
	}
	f.removed = append(f.removed, req)
	refs := f.references[req.UserID]
	if len(refs) == 0 {
		return &seedaccountport.RemoveUserResult{Deleted: true}, nil
	}

	return &seedaccountport.RemoveUserResult{Disabled: true, References: refs}, nil
}

func (f *fakeRepo) RemoveOrganization(
	_ context.Context,
	org *seedaccountport.OrganizationFootprint,
) (*seedaccountport.RemoveOrganizationResult, error) {
	if f.inReadOnly {
		f.writesInRead++
	}
	if reason, ok := f.retainOrg[org.ID]; ok {
		return &seedaccountport.RemoveOrganizationResult{RetainedReason: reason}, nil
	}
	f.orgsRemoved = append(f.orgsRemoved, org.ID)

	return &seedaccountport.RemoveOrganizationResult{Deleted: true}, nil
}

type fakeSessions struct {
	repositories.SessionRepository
	ended []pulid.ID
	err   error
}

func (f *fakeSessions) DeleteAllForUser(_ context.Context, userID pulid.ID) error {
	if f.err != nil {
		return f.err
	}
	f.ended = append(f.ended, userID)
	return nil
}

type cacheKey struct {
	user pulid.ID
	org  pulid.ID
}

type fakePermCache struct {
	repositories.PermissionCacheRepository
	cleared []cacheKey
}

func (f *fakePermCache) Delete(_ context.Context, userID, orgID pulid.ID) error {
	f.cleared = append(f.cleared, cacheKey{user: userID, org: orgID})
	return nil
}

type fakeAuditor struct {
	changes []*services.SecurityChange
}

func (f *fakeAuditor) RecordChange(_ context.Context, change *services.SecurityChange) {
	f.changes = append(f.changes, change)
}

type fixture struct {
	repo      *fakeRepo
	sessions  *fakeSessions
	permCache *fakePermCache
	auditor   *fakeAuditor
	service   *Service

	bu        pulid.ID
	logistics pulid.ID
	transport pulid.ID
	admin     *seedaccountport.UserFootprint
	logAdmin  *seedaccountport.UserFootprint
	trAdmin   *seedaccountport.UserFootprint
	system    *seedaccountport.SystemUser
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	f := &fixture{
		bu:        pulid.MustNew("bu_"),
		logistics: pulid.MustNew("org_"),
		transport: pulid.MustNew("org_"),
	}
	f.system = &seedaccountport.SystemUser{
		ID:             pulid.MustNew("usr_"),
		OrganizationID: f.logistics,
		BusinessUnitID: f.bu,
	}
	f.admin = f.seededUser("admin", "admin@trenova.app", f.logistics, f.transport)
	f.logAdmin = f.seededUser("admin-logistics", "admin.logistics@trenova.app", f.logistics)
	f.trAdmin = f.seededUser("admin-transport", "admin.transport@trenova.app", f.transport)
	f.admin.ActiveAPIKeys = []seedaccountport.APIKey{{
		ID:             pulid.MustNew("ak_"),
		OrganizationID: f.logistics,
		BusinessUnitID: f.bu,
		Name:           "Integration",
		KeyPrefix:      "trv_abc",
	}}

	f.repo = &fakeRepo{
		system: f.system,
		users:  []*seedaccountport.UserFootprint{f.admin, f.logAdmin, f.trAdmin},
		orgs: []*seedaccountport.OrganizationFootprint{
			{ID: f.logistics, BusinessUnitID: f.bu, Name: "Trenova Logistics", ScacCode: "TRNV", SeedTracked: true},
			{ID: f.transport, BusinessUnitID: f.bu, Name: "Trenova Transportation", ScacCode: "TTNV", SeedTracked: true},
		},
		members: map[pulid.ID][]orgMember{
			f.logistics: {
				{id: f.admin.ID, username: "admin"},
				{id: f.logAdmin.ID, username: "admin-logistics"},
				{id: f.system.ID, username: "system"},
			},
			f.transport: {
				{id: f.admin.ID, username: "admin"},
				{id: f.trAdmin.ID, username: "admin-transport"},
			},
		},
		references: map[pulid.ID][]seedaccountport.Reference{
			f.admin.ID: {{Table: "audit_entries", Column: "user_id", Rows: 42}},
		},
		retainOrg: map[pulid.ID]string{},
	}
	f.sessions = &fakeSessions{}
	f.permCache = &fakePermCache{}
	f.auditor = &fakeAuditor{}
	f.service = New(Params{
		Repo:            f.repo,
		Sessions:        f.sessions,
		PermissionCache: f.permCache,
		Auditor:         f.auditor,
		Logger:          zap.NewNop(),
	})
	f.service.now = func() int64 { return fixedNow }
	f.service.unusableHash = func() (string, error) { return retiredHash, nil }
	f.service.seededPassword = func(hash string) bool { return hash == seededHash }

	return f
}

func (f *fixture) seededUser(
	username, email string,
	current pulid.ID,
	others ...pulid.ID,
) *seedaccountport.UserFootprint {
	id := pulid.MustNew("usr_")
	orgs := append([]pulid.ID{current}, others...)
	memberships := make([]seedaccountport.Membership, 0, len(orgs))
	assignments := make([]seedaccountport.RoleAssignment, 0, len(orgs))
	for _, org := range orgs {
		memberships = append(memberships, seedaccountport.Membership{
			ID:             pulid.MustNew("mem_"),
			OrganizationID: org,
			BusinessUnitID: f.bu,
		})
		assignments = append(assignments, seedaccountport.RoleAssignment{
			ID:             pulid.MustNew("ura_"),
			OrganizationID: org,
			RoleID:         pulid.MustNew("rol_"),
		})
	}

	return &seedaccountport.UserFootprint{
		ID:                    id,
		BusinessUnitID:        f.bu,
		CurrentOrganizationID: current,
		Username:              username,
		EmailAddress:          email,
		Status:                domaintypes.StatusActive,
		PasswordHash:          seededHash,
		SeedTracked:           true,
		Memberships:           memberships,
		RoleAssignments:       assignments,
	}
}

func userReport(t *testing.T, report *Report, username string) *UserReport {
	t.Helper()
	for _, user := range report.Users {
		if user.Footprint.Username == username {
			return user
		}
	}
	require.FailNow(t, "no report for "+username)
	return nil
}

func orgReport(t *testing.T, report *Report, id pulid.ID) *OrganizationReport {
	t.Helper()
	for _, org := range report.Organizations {
		if org.Footprint.ID == id {
			return org
		}
	}
	require.FailNow(t, "no report for organization "+id.String())
	return nil
}

func TestDryRunOnlyReads(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	report, err := f.service.Run(t.Context(), RunRequest{})
	require.NoError(t, err)

	assert.False(t, report.Applied)
	assert.Equal(t, []bool{true}, f.repo.readOnly)
	assert.Zero(t, f.repo.writesInRead)
	assert.Empty(t, f.repo.stripped)
	assert.Empty(t, f.repo.removed)
	assert.Empty(t, f.sessions.ended)
	assert.Empty(t, f.auditor.changes)

	admin := userReport(t, report, "admin")
	assert.Equal(t, UserActionRetire, admin.Action)
	assert.False(t, admin.Delete, "admin is referenced by audit entries, so it is kept disabled")
	assert.Equal(t, []Provenance{ProvenanceSeedTracking, ProvenanceSeededPassword}, admin.Provenance)
	assert.True(t, userReport(t, report, "admin-logistics").Delete)
	assert.True(t, userReport(t, report, "admin-transport").Delete)

	logistics := orgReport(t, report, f.logistics)
	assert.Equal(t, OrganizationActionReview, logistics.Action)
	assert.Contains(t, logistics.Reasons, "hosts the instance system user")
	assert.Contains(t, logistics.Reasons, "still has users: admin, system")

	transport := orgReport(t, report, f.transport)
	assert.Equal(t, OrganizationActionReview, transport.Action,
		"the kept admin tombstone still points at it")
	assert.Equal(t, []string{"still has users: admin"}, transport.Reasons)
}

func TestApplyNeedsSideEffects(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	service := New(Params{Repo: f.repo, Logger: zap.NewNop()})
	_, err := service.Run(t.Context(), RunRequest{Apply: true})
	require.ErrorIs(t, err, ErrApplyNeedsSideEffects)
	assert.Empty(t, f.repo.readOnly)
}

func TestApplyRetiresProvenAccountsAndAuditsEveryChange(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.repo.members[f.transport] = []orgMember{{id: f.trAdmin.ID, username: "admin-transport"}}
	f.admin.Memberships = f.admin.Memberships[:1]
	f.admin.RoleAssignments = f.admin.RoleAssignments[:1]

	report, err := f.service.Run(t.Context(), RunRequest{Apply: true})
	require.NoError(t, err)
	require.Empty(t, report.Errors)

	assert.True(t, report.Applied)
	assert.Equal(t, []bool{false}, f.repo.readOnly)
	assert.ElementsMatch(t, []pulid.ID{f.admin.ID, f.logAdmin.ID, f.trAdmin.ID}, f.repo.stripped)
	require.Len(t, f.repo.removed, 3)
	for _, removed := range f.repo.removed {
		assert.Equal(t, retiredHash, removed.PasswordHash)
		assert.Equal(t, fixedNow, removed.Now)
	}

	admin := userReport(t, report, "admin")
	assert.False(t, admin.Delete)
	assert.True(t, admin.Removed.Disabled)
	assert.True(t, userReport(t, report, "admin-transport").Delete)

	assert.ElementsMatch(t, []pulid.ID{f.admin.ID, f.logAdmin.ID, f.trAdmin.ID}, f.sessions.ended)
	assert.Contains(t, f.permCache.cleared, cacheKey{user: f.trAdmin.ID, org: f.transport})

	transport := orgReport(t, report, f.transport)
	assert.True(t, transport.Removed, "only seed accounts lived in it, and they are gone")
	assert.Equal(t, []pulid.ID{f.transport}, f.repo.orgsRemoved)
	assert.False(t, orgReport(t, report, f.logistics).Removed)

	assert.Equal(t, len(f.auditor.changes), report.AuditEntries)
	for _, change := range f.auditor.changes {
		assert.Equal(t, services.SystemAuditActor(), change.Actor)
		assert.NotEqual(t, f.transport, change.OrganizationID,
			"nothing is recorded in an organization that was deleted")
		assert.Equal(t, auditSource, change.Metadata[keySource])
	}

	var orgDeleted, adminLocked, keyRevoked, transportMoved bool
	for _, change := range f.auditor.changes {
		switch {
		case change.Resource == permission.ResourceOrganization && change.Operation == permission.OpDelete:
			orgDeleted = change.ResourceID == f.transport.String() &&
				change.OrganizationID == f.system.OrganizationID
		case change.Resource == permission.ResourceUser && change.Operation == permission.OpLock:
			adminLocked = change.ResourceID == f.admin.ID.String()
		case change.Resource == permission.ResourceAPIKey:
			keyRevoked = change.ResourceID == f.admin.ActiveAPIKeys[0].ID.String()
		}
		if change.Metadata[keyRecordedIn] == f.system.OrganizationID.String() &&
			change.Metadata[keyOrganizationID] == f.transport.String() {
			transportMoved = true
		}
	}
	assert.True(t, orgDeleted)
	assert.True(t, adminLocked)
	assert.True(t, keyRevoked)
	assert.True(t, transportMoved, "changes in the deleted organization are recorded in the system user's")
}

func TestApplyIsIdempotent(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.admin.Status = domaintypes.StatusInactive
	f.admin.IsLocked = true
	f.admin.PasswordHash = retiredHash
	f.admin.Memberships = nil
	f.admin.RoleAssignments = nil
	f.admin.ActiveAPIKeys = nil
	f.repo.users = []*seedaccountport.UserFootprint{f.admin}

	report, err := f.service.Run(t.Context(), RunRequest{Apply: true})
	require.NoError(t, err)

	assert.Equal(t, UserActionAlreadyRetired, userReport(t, report, "admin").Action)
	assert.Empty(t, f.repo.stripped)
	assert.Empty(t, f.repo.removed)
	assert.Empty(t, f.auditor.changes)
	assert.Equal(t, []pulid.ID{f.admin.ID}, f.sessions.ended)
}

func TestUnprovenAccountIsLeftAlone(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.logAdmin.SeedTracked = false
	f.logAdmin.PasswordHash = changedHash

	report, err := f.service.Run(t.Context(), RunRequest{Apply: true})
	require.NoError(t, err)

	logAdmin := userReport(t, report, "admin-logistics")
	assert.Equal(t, UserActionUnproven, logAdmin.Action)
	assert.Empty(t, logAdmin.Provenance)
	assert.NotContains(t, f.repo.stripped, f.logAdmin.ID)
	assert.NotContains(t, f.sessions.ended, f.logAdmin.ID)
	for _, change := range f.auditor.changes {
		assert.NotEqual(t, f.logAdmin.ID.String(), change.ResourceID)
	}
}

func TestOnlyExactSeedPairsAreConsidered(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	impostor := *f.trAdmin
	impostor.ID = pulid.MustNew("usr_")
	impostor.EmailAddress = "dana@customer.example"
	f.repo.users = []*seedaccountport.UserFootprint{&impostor}

	report, err := f.service.Run(t.Context(), RunRequest{})
	require.NoError(t, err)
	assert.Empty(t, report.Users)
}

func TestOrganizationWithDataIsKeptForReview(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.repo.orgs[1].Data = []seedaccountport.Reference{{Table: "shipments", Column: "organization_id", Rows: 1000}}
	f.repo.orgs[1].History = []seedaccountport.Reference{{Table: "audit_entries", Column: "organization_id", Rows: 3}}
	f.repo.members[f.transport] = []orgMember{{id: f.trAdmin.ID, username: "admin-transport"}}

	report, err := f.service.Run(t.Context(), RunRequest{Apply: true})
	require.NoError(t, err)

	transport := orgReport(t, report, f.transport)
	assert.Equal(t, OrganizationActionReview, transport.Action)
	assert.False(t, transport.Removed)
	assert.Contains(t, transport.Reasons, "holds data beyond seed defaults: shipments.organization_id (1000+)")
	assert.Contains(t, transport.Reasons,
		"holds append-only audit history: audit_entries.organization_id (3)")
	assert.Empty(t, f.repo.orgsRemoved)
}

func TestOrganizationTheDatabaseKeepsIsReported(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.repo.members[f.transport] = []orgMember{{id: f.trAdmin.ID, username: "admin-transport"}}
	f.repo.retainOrg[f.transport] = "audit rows are append-only"

	report, err := f.service.Run(t.Context(), RunRequest{Apply: true})
	require.NoError(t, err)

	transport := orgReport(t, report, f.transport)
	assert.False(t, transport.Removed)
	assert.Equal(t, OrganizationActionReview, transport.Action)
	assert.Contains(t, transport.Reasons, "the database kept it: audit rows are append-only")
}

func TestUntrackedOrganizationIsNeverRemoved(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.repo.orgs[1].SeedTracked = false
	f.repo.members[f.transport] = nil

	report, err := f.service.Run(t.Context(), RunRequest{Apply: true})
	require.NoError(t, err)
	assert.Equal(t, OrganizationActionReview, orgReport(t, report, f.transport).Action)
	assert.Empty(t, f.repo.orgsRemoved)
}

func TestSessionFailureIsReportedAfterCommit(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.sessions.err = errors.New("redis unavailable")

	report, err := f.service.Run(t.Context(), RunRequest{Apply: true})
	require.NoError(t, err)
	assert.True(t, report.Applied)
	assert.NotEmpty(t, report.Errors)
	assert.False(t, userReport(t, report, "admin").SessionsRevoked)
}

func TestMatchesSeededPassword(t *testing.T) {
	t.Parallel()

	seeded, err := bcrypt.GenerateFromPassword([]byte(seedaccountport.SeededPassword), bcrypt.MinCost)
	require.NoError(t, err)
	assert.True(t, matchesSeededPassword(string(seeded)))

	hash, err := unusablePasswordHash()
	require.NoError(t, err)
	assert.False(t, matchesSeededPassword(hash))
	assert.False(t, matchesSeededPassword("not-a-bcrypt-hash"))
}
