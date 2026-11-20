//go:build integration

package scimrepository

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/entitlementtest"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam/iamdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac/rbacdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/scim"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/scimports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/scimproto"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type tenantDirectory struct {
	data      *seedtest.TestData
	directory *iam.SCIMDirectory
	prefix    string
	ctx       context.Context
}

type scimHarness struct {
	ctx   context.Context
	admin *keel.DB
	repo  *repository
	a     *tenantDirectory
	b     *tenantDirectory
}

func newSCIMHarness(t *testing.T) *scimHarness {
	t.Helper()

	ctx, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	h := &scimHarness{
		ctx:   ctx,
		admin: admin,
		repo: &repository{
			entitlements: entitlementtest.Instance(),
			db:           dbtest.NewRLSConnection(t, admin, dsn),
			l:            zap.NewNop(),
		},
	}
	h.a = h.directory(t, seedtest.SeedFullTestData(t, ctx, admin), "acme", "scim_tenantAprefix")
	h.b = h.directory(
		t,
		seedtest.SeedAdditionalTenant(t, ctx, admin, "SB"),
		"globex",
		"scim_tenantBprefix",
	)

	return h
}

func (h *scimHarness) directory(
	t *testing.T,
	data *seedtest.TestData,
	slug, prefix string,
) *tenantDirectory {
	t.Helper()

	seedCtx := data.Ctx(h.ctx)
	directory := &iam.SCIMDirectory{TenantSlug: slug, Enabled: true}
	require.NoError(t, iamdb.SCIMDirectories.Create(seedCtx, h.admin, directory))
	require.NoError(t, iamdb.SCIMTokens.Create(seedCtx, h.admin, &iam.SCIMToken{
		DirectoryID: directory.ID,
		Name:        "Identity provider",
		Prefix:      prefix,
		TokenHash:   scim.HashToken(scim.JoinToken(prefix, "secret")),
		Status:      iam.SCIMTokenStatusActive,
	}))

	return &tenantDirectory{
		data:      data,
		directory: directory,
		prefix:    prefix,
		ctx: dbscope.WithTenant(h.ctx, dbscope.Tenant{
			OrganizationID: data.Organization.ID,
			BusinessUnitID: data.BusinessUnit.ID,
		}),
	}
}

func (h *scimHarness) linkOwner(t *testing.T, td *tenantDirectory) *scim.User {
	t.Helper()

	link := &scim.User{
		DirectoryID: td.directory.ID,
		UserID:      td.data.User.ID,
		UserName:    td.data.User.EmailAddress,
		ExternalID:  "ext-" + td.data.User.ID.String(),
		Active:      true,
	}
	require.NoError(t, h.repo.CreateUser(td.ctx, link))

	return link
}

func TestFindCredential_ResolvesTheTokensDirectoryAndTenant(t *testing.T) {
	h := newSCIMHarness(t)

	credential, err := h.repo.FindCredential(h.ctx, h.a.prefix)
	require.NoError(t, err)
	assert.Equal(t, h.a.directory.ID, credential.DirectoryID)
	assert.Equal(t, h.a.data.Organization.ID, credential.OrganizationID)
	assert.Equal(t, h.a.data.BusinessUnit.ID, credential.BusinessUnitID)
	assert.Equal(t, scim.HashToken(scim.JoinToken(h.a.prefix, "secret")), credential.TokenHash)
	assert.True(t, credential.DirectoryEnabled)
	assert.Equal(t, "acme", credential.TenantSlug)

	_, err = h.repo.FindCredential(h.ctx, "scim_unknownprefix")
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))
}

func TestUsers_AnotherTenantSeesNothingADirectoryLinked(t *testing.T) {
	h := newSCIMHarness(t)
	link := h.linkOwner(t, h.a)
	owner := h.a.data.User

	record, err := h.repo.GetUser(h.a.ctx, h.a.directory.ID, owner.ID)
	require.NoError(t, err)
	require.NotNil(t, record.User)
	assert.Equal(t, owner.EmailAddress, record.User.EmailAddress)

	byName, err := scimproto.ParseFilter(
		`userName eq "`+link.UserName+`"`,
		scimports.UserFilters()...)
	require.NoError(t, err)
	records, total, err := h.repo.ListUsers(h.a.ctx, &scimports.ListQuery{
		DirectoryID: h.a.directory.ID,
		Filter:      byName,
		Limit:       10,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, records, 1)

	_, err = h.repo.GetUser(h.b.ctx, h.a.directory.ID, owner.ID)
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))

	records, total, err = h.repo.ListUsers(h.b.ctx, &scimports.ListQuery{
		DirectoryID: h.a.directory.ID,
		Limit:       10,
	})
	require.NoError(t, err)
	assert.Zero(t, total)
	assert.Empty(t, records)

	count, err := h.repo.CountUsers(h.b.ctx, h.a.directory.ID, []pulid.ID{owner.ID})
	require.NoError(t, err)
	assert.Zero(t, count)

	err = h.repo.DeleteUser(h.b.ctx, h.a.directory.ID, owner.ID)
	require.Error(t, err)
	_, err = h.repo.GetUser(h.a.ctx, h.a.directory.ID, owner.ID)
	require.NoError(t, err, "a delete from another tenant leaves the link in place")
}

// A directory finds only the people its own organization already has: a
// user whose current organization is the directory's, or who holds a
// membership in it, within its business unit. A user only a sister
// organization has is not found, because linking would let this directory's
// identity provider add them here and rewrite their name and address, which
// the sister organization's administrators own. Another company's user is
// never found, with or without a membership of their own.
func TestFindMember_FindsOnlyThePeopleTheDirectorysOrganizationHas(t *testing.T) {
	h := newSCIMHarness(t)

	sister := seedtest.NewOrganization(h.a.data.BusinessUnit.ID, h.a.data.State.ID).
		WithName("Acme West").
		WithScacCode("ACWS").
		WithDOTNumber("7654321").
		Build(t, h.ctx, h.admin)
	sisterOnly := seedtest.NewUser(sister.ID, h.a.data.BusinessUnit.ID).
		WithUsername("acme_west_only").
		WithEmail("west-only@acme.example").
		Build(t, h.ctx, h.admin)
	h.membership(t, sisterOnly, sister.ID, h.a.data.BusinessUnit.ID)
	sharedMember := seedtest.NewUser(sister.ID, h.a.data.BusinessUnit.ID).
		WithUsername("acme_west_shared").
		WithEmail("west-shared@acme.example").
		Build(t, h.ctx, h.admin)
	h.membership(t, sharedMember, sister.ID, h.a.data.BusinessUnit.ID)
	h.membership(t, sharedMember, h.a.data.Organization.ID, h.a.data.BusinessUnit.ID)

	strangerWithMembership := seedtest.NewUser(
		h.b.data.Organization.ID,
		h.b.data.BusinessUnit.ID,
	).
		WithUsername("globex_member").
		WithEmail("member@globex.example").
		Build(t, h.ctx, h.admin)
	h.membership(t, strangerWithMembership, h.b.data.Organization.ID, h.b.data.BusinessUnit.ID)
	strangerAlone := seedtest.NewUser(h.b.data.Organization.ID, h.b.data.BusinessUnit.ID).
		WithUsername("globex_unlinked").
		WithEmail("unlinked@globex.example").
		Build(t, h.ctx, h.admin)

	member, err := h.repo.FindMember(h.a.ctx, h.a.data.User.EmailAddress)
	require.NoError(t, err)
	assert.Equal(t, h.a.data.User.ID, member.ID, "a user whose organization is the directory's")

	shared, err := h.repo.FindMember(h.a.ctx, "WEST-SHARED@acme.example")
	require.NoError(t, err)
	assert.Equal(t, sharedMember.ID, shared.ID,
		"a sister organization's user who is also a member here")

	for name, address := range map[string]string{
		"a user only the sister organization has":   sisterOnly.EmailAddress,
		"another company's user with a membership":  strangerWithMembership.EmailAddress,
		"another company's user with no membership": strangerAlone.EmailAddress,
		"another company's seeded user":             h.b.data.User.EmailAddress,
	} {
		_, err = h.repo.FindMember(h.a.ctx, address)
		require.Error(t, err, name)
		assert.True(t, errortypes.IsNotFoundError(err), name)
	}

	_, err = h.repo.FindMember(h.b.ctx, h.a.data.User.EmailAddress)
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err), "nor the other way round")
}

func (h *scimHarness) membership(
	t *testing.T,
	user *tenant.User,
	organizationID, businessUnitID pulid.ID,
) {
	t.Helper()

	require.NoError(t, tenantdb.OrganizationMemberships.Create(
		dbscope.WithSystem(h.ctx, "seed a membership for the test"),
		h.admin,
		&tenant.OrganizationMembership{
			UserID:         user.ID,
			OrganizationID: organizationID,
			BusinessUnitID: businessUnitID,
			JoinedAt:       timeutils.NowUnix(),
		},
	))
}

func TestRoleSources_MatchMappingsTheirSaverStillNames(t *testing.T) {
	h := newSCIMHarness(t)
	link := h.linkOwner(t, h.a)
	seedCtx := h.a.data.Ctx(h.ctx)

	dispatcher := &rbac.Role{Name: "Dispatcher"}
	billing := &rbac.Role{Name: "Billing"}
	unsaved := &rbac.Role{Name: "Unsaved"}
	for _, role := range []*rbac.Role{dispatcher, billing, unsaved} {
		require.NoError(t, rbacdb.Roles.Create(seedCtx, h.admin, role))
	}

	group := &scim.Group{
		DirectoryID: h.a.directory.ID,
		DisplayName: "Dispatch",
		ExternalID:  "grp-1",
	}
	require.NoError(t, h.repo.CreateGroup(h.a.ctx, group))
	require.NoError(t, h.repo.AddMembers(h.a.ctx, &scimports.MemberChange{
		DirectoryID: h.a.directory.ID,
		GroupID:     group.ID,
		UserIDs:     []pulid.ID{link.UserID},
	}))

	for _, mapping := range []*iam.SCIMGroupRoleMapping{
		{
			DirectoryID:     h.a.directory.ID,
			ExternalGroupID: "grp-1",
			DisplayName:     "By external id",
			RoleID:          dispatcher.ID,
			SavedByID:       h.a.data.User.ID,
		},
		{
			DirectoryID:     h.a.directory.ID,
			ExternalGroupID: "DISPATCH",
			DisplayName:     "By display name",
			RoleID:          billing.ID,
			SavedByID:       h.a.data.User.ID,
		},
		{
			DirectoryID:     h.a.directory.ID,
			ExternalGroupID: "grp-1",
			DisplayName:     "Saved before savers were recorded",
			RoleID:          unsaved.ID,
		},
	} {
		require.NoError(t, iamdb.SCIMGroupRoleMappings.Create(seedCtx, h.admin, mapping))
	}

	sources, err := h.repo.RoleSources(h.a.ctx, h.a.directory.ID, link.UserID)
	require.NoError(t, err)
	roles := make([]pulid.ID, 0, len(sources))
	for _, source := range sources {
		roles = append(roles, source.RoleID)
		assert.Equal(t, h.a.data.User.ID, source.SavedByID)
		assert.Equal(t, group.ID, source.GroupID)
	}
	assert.ElementsMatch(t, []pulid.ID{dispatcher.ID, billing.ID}, roles)

	foreign, err := h.repo.RoleSources(h.b.ctx, h.a.directory.ID, link.UserID)
	require.NoError(t, err)
	assert.Empty(t, foreign)
}

func TestPending_MarksListsAndCompletesReconcileWithinTheTenant(t *testing.T) {
	h := newSCIMHarness(t)
	link := h.linkOwner(t, h.a)

	pending, err := h.repo.ListPending(h.a.ctx, h.a.directory.ID, 10)
	require.NoError(t, err)
	assert.Empty(t, pending)

	require.NoError(t, h.repo.MarkPending(h.b.ctx, h.a.directory.ID, []pulid.ID{link.UserID}))
	pending, err = h.repo.ListPending(h.a.ctx, h.a.directory.ID, 10)
	require.NoError(t, err)
	assert.Empty(t, pending, "another tenant cannot mark this directory's people")

	require.NoError(t, h.repo.MarkPending(h.a.ctx, h.a.directory.ID, []pulid.ID{link.UserID}))
	pending, err = h.repo.ListPending(h.a.ctx, h.a.directory.ID, 10)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	assert.Equal(t, int64(1), pending[0].RolesVersion)

	directories, err := h.repo.ListPendingDirectories(h.ctx, 10)
	require.NoError(t, err)
	assert.Contains(t, directories, scimports.PendingDirectory{
		OrganizationID: h.a.data.Organization.ID,
		BusinessUnitID: h.a.data.BusinessUnit.ID,
		DirectoryID:    h.a.directory.ID,
	})

	require.NoError(t, h.repo.CompleteReconcile(h.a.ctx, pending[0]))
	pending, err = h.repo.ListPending(h.a.ctx, h.a.directory.ID, 10)
	require.NoError(t, err)
	assert.Empty(t, pending)
}

func TestDeleteDirectoryAssignments_RemovesOnlyWhatTheDirectoryGranted(t *testing.T) {
	h := newSCIMHarness(t)
	link := h.linkOwner(t, h.a)
	seedCtx := h.a.data.Ctx(h.ctx)

	granted := &rbac.Role{Name: "Granted by the directory"}
	manual := &rbac.Role{Name: "Granted by a person"}
	for _, role := range []*rbac.Role{granted, manual} {
		require.NoError(t, rbacdb.Roles.Create(seedCtx, h.admin, role))
	}

	directoryAssignment := &rbac.UserRoleAssignment{
		UserID:     link.UserID,
		RoleID:     granted.ID,
		AssignedAt: timeutils.NowUnix(),
	}
	personAssignment := &rbac.UserRoleAssignment{
		UserID:     link.UserID,
		RoleID:     manual.ID,
		AssignedAt: timeutils.NowUnix(),
	}
	for _, assignment := range []*rbac.UserRoleAssignment{directoryAssignment, personAssignment} {
		require.NoError(t, rbacdb.UserRoleAssignments.Create(seedCtx, h.admin, assignment))
	}
	require.NoError(t, h.repo.CreateGrant(h.a.ctx, &scim.RoleGrant{
		DirectoryID:  h.a.directory.ID,
		UserID:       link.UserID,
		RoleID:       granted.ID,
		AssignmentID: directoryAssignment.ID,
		GrantedByID:  h.a.data.User.ID,
	}))

	foreign, err := h.repo.DeleteDirectoryAssignments(h.b.ctx, h.a.directory.ID)
	require.NoError(t, err)
	assert.Empty(t, foreign)

	deleted, err := h.repo.DeleteDirectoryAssignments(h.a.ctx, h.a.directory.ID)
	require.NoError(t, err)
	require.Len(t, deleted, 1)
	assert.Equal(t, directoryAssignment.ID, deleted[0].ID)

	ura := rbacdb.UserRoleAssignments
	remaining, err := ura.Where(ura.UserID.Eq(link.UserID)).All(seedCtx, h.admin)
	require.NoError(t, err)
	require.Len(t, remaining, 1)
	assert.Equal(t, personAssignment.ID, remaining[0].ID)

	grants, err := h.repo.ListGrants(h.a.ctx, h.a.directory.ID, link.UserID)
	require.NoError(t, err)
	assert.Empty(t, grants)
}
