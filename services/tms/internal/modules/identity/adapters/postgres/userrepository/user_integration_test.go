//go:build integration

package userrepository

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	inttestutil "github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks/platformplanportsmock"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type userFixture struct {
	ctx       context.Context
	db        *keel.DB
	conn      *postgres.Connection
	repo      *repository
	data      *seedtest.TestData
	secondOrg *tenant.Organization
}

func newUserFixture(t *testing.T) *userFixture {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	conn := postgres.NewTestConnection(db)
	data := seedtest.SeedFullTestData(t, ctx, db)
	secondOrg := seedtest.NewOrganization(data.BusinessUnit.ID, data.State.ID).
		WithName("Second Organization").
		WithScacCode("SECN").
		WithDOTNumber("7654321").
		WithBucketName("second-org-bucket").
		Build(t, ctx, db)

	return &userFixture{
		ctx:       data.Ctx(ctx),
		db:        db,
		conn:      conn,
		repo:      New(Params{DB: conn, Logger: zap.NewNop()}).(*repository),
		data:      data,
		secondOrg: secondOrg,
	}
}

func (f *userFixture) systemCtx() context.Context {
	return dbscope.WithSystem(f.ctx, "user repository test setup")
}

func (f *userFixture) addMemberships(
	t *testing.T,
	userID pulid.ID,
	orgIDs ...pulid.ID,
) {
	t.Helper()

	rows := make([]*tenant.OrganizationMembership, 0, len(orgIDs))
	for i, orgID := range orgIDs {
		rows = append(rows, &tenant.OrganizationMembership{
			UserID:         userID,
			BusinessUnitID: f.data.BusinessUnit.ID,
			OrganizationID: orgID,
			GrantedByID:    f.data.User.ID,
			IsDefault:      i == 0,
		})
	}
	require.NoError(t, tenantdb.OrganizationMemberships.CreateMany(f.systemCtx(), f.db, rows))
}

func (f *userFixture) addUser(t *testing.T, username string) *tenant.User {
	t.Helper()

	return seedtest.NewUser(f.data.Organization.ID, f.data.BusinessUnit.ID).
		WithUsername(username).
		WithEmail(username+"@example.com").
		Build(t, f.ctx, f.db)
}

func TestUpdatePasswordUpdatesOnlyScopedUser(t *testing.T) {
	f := newUserFixture(t)
	other := f.addUser(t, "otheruser")

	user, err := f.repo.GetByID(f.ctx, tenantports.GetUserByIDRequest{TenantInfo: f.data.Info()})
	require.NoError(t, err)
	originalHash := user.Password

	updatedHash, err := user.GeneratePassword("new-password-123")
	require.NoError(t, err)

	err = f.repo.UpdatePassword(f.ctx, tenantports.UpdateUserPasswordRequest{
		UserID:             f.data.User.ID,
		OrganizationID:     f.data.Organization.ID,
		BusinessUnitID:     f.data.BusinessUnit.ID,
		Password:           updatedHash,
		MustChangePassword: false,
	})
	require.NoError(t, err)

	updatedUser, err := f.repo.GetByID(f.ctx, tenantports.GetUserByIDRequest{TenantInfo: f.data.Info()})
	require.NoError(t, err)
	assert.NotEqual(t, originalHash, updatedUser.Password)
	assert.Equal(t, updatedHash, updatedUser.Password)
	assert.False(t, updatedUser.MustChangePassword)
	require.NotNil(t, updatedUser.PasswordChangedAt)

	untouched, err := f.repo.GetByID(f.ctx, tenantports.GetUserByIDRequest{
		TenantInfo:   f.data.Info(),
		LookupUserID: other.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, other.Password, untouched.Password)
}

func TestGetByIDIsLimitedToTheTenant(t *testing.T) {
	f := newUserFixture(t)
	other := seedtest.SeedAdditionalTenant(t, f.ctx, f.db, "UB")

	_, err := f.repo.GetByID(f.ctx, tenantports.GetUserByIDRequest{
		TenantInfo:   f.data.Info(),
		LookupUserID: other.User.ID,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))
}

func TestReplaceOrganizationMembershipsReturnsConflictWhenMembershipsLocked(t *testing.T) {
	f := newUserFixture(t)
	f.addMemberships(t, f.data.User.ID, f.data.Organization.ID, f.secondOrg.ID)

	lock := inttestutil.HoldTxLock(
		t,
		f.systemCtx(),
		f.conn,
		kernelports.TxOptions{},
		func(ctx context.Context) error {
			m := tenantdb.OrganizationMemberships
			_, err := m.AllTenants().
				Where(m.UserID.Eq(f.data.User.ID), m.BusinessUnitID.Eq(f.data.BusinessUnit.ID)).
				Order(m.OrganizationID.Asc()).
				Lock(keel.ForUpdate).
				All(ctx, f.conn.Keel(ctx))
			return err
		},
	)
	lock.WaitLocked(t)

	memberships, err := f.repo.ReplaceOrganizationMemberships(
		f.ctx,
		tenantports.ReplaceOrganizationMembershipsRequest{
			ActorID:         f.data.User.ID,
			UserID:          f.data.User.ID,
			BusinessUnitID:  f.data.BusinessUnit.ID,
			OrganizationIDs: []pulid.ID{f.secondOrg.ID},
		},
	)

	require.Nil(t, memberships)
	require.Error(t, err)
	assert.True(t, errortypes.IsConflictError(err))
	assert.Contains(t, err.Error(), "organization memberships are busy")

	lock.Release()
	lock.Wait(t)

	finalMemberships, err := f.repo.ListOrganizationMemberships(
		f.ctx,
		f.data.User.ID,
		f.data.BusinessUnit.ID,
	)
	require.NoError(t, err)
	require.Len(t, finalMemberships, 2)

	defaultCount := 0
	for _, membership := range finalMemberships {
		if membership.IsDefault {
			defaultCount++
			assert.Equal(t, f.data.Organization.ID, membership.OrganizationID)
		}
	}
	assert.Equal(t, 1, defaultCount)
}

func TestReplaceOrganizationMembershipsMovesTheDefaultAndCurrentOrganization(t *testing.T) {
	f := newUserFixture(t)
	f.addMemberships(t, f.data.User.ID, f.data.Organization.ID)

	memberships, err := f.repo.ReplaceOrganizationMemberships(
		f.ctx,
		tenantports.ReplaceOrganizationMembershipsRequest{
			ActorID:         f.data.User.ID,
			UserID:          f.data.User.ID,
			BusinessUnitID:  f.data.BusinessUnit.ID,
			OrganizationIDs: []pulid.ID{f.secondOrg.ID},
		},
	)
	require.NoError(t, err)
	require.Len(t, memberships, 1)
	assert.Equal(t, f.secondOrg.ID, memberships[0].OrganizationID)
	assert.True(t, memberships[0].IsDefault)
	assert.Equal(t, f.data.User.ID, memberships[0].GrantedByID)
	require.NotNil(t, memberships[0].Organization)
	assert.Equal(t, f.secondOrg.Name, memberships[0].Organization.Name)

	u := tenantdb.Users
	stored, err := u.AllTenants().Where(u.ID.Eq(f.data.User.ID)).One(f.systemCtx(), f.db)
	require.NoError(t, err)
	assert.Equal(t, f.secondOrg.ID, stored.CurrentOrganizationID)
	assert.Equal(t, f.data.User.Version, stored.Version)
}

func TestReplaceOrganizationMembershipsRefusesOrganizationsOfAnotherBusinessUnit(t *testing.T) {
	f := newUserFixture(t)
	other := seedtest.SeedAdditionalTenant(t, f.ctx, f.db, "UC")

	_, err := f.repo.ReplaceOrganizationMemberships(
		f.ctx,
		tenantports.ReplaceOrganizationMembershipsRequest{
			ActorID:         f.data.User.ID,
			UserID:          f.data.User.ID,
			BusinessUnitID:  f.data.BusinessUnit.ID,
			OrganizationIDs: []pulid.ID{other.Organization.ID},
		},
	)
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
}

func TestReplaceOrganizationMembershipsRefusesASeatPastTheQuota(t *testing.T) {
	f := newUserFixture(t)
	f.addMemberships(t, f.data.User.ID, f.data.Organization.ID)
	guard := platformplanportsmock.NewMockQuotaGuard(t)
	f.repo.quota = guard

	guard.EXPECT().Enforce(mock.Anything, &platformplanports.QuotaRequest{
		TenantInfo: pagination.TenantInfo{OrgID: f.secondOrg.ID, BuID: f.data.BusinessUnit.ID},
		Meter:      platformcatalog.MeterUserSeats,
		Quantity:   1,
	}).Return(errortypes.NewQuotaExceededError(string(platformcatalog.MeterUserSeats), 1, 1, "free_demo"))

	memberships, err := f.repo.ReplaceOrganizationMemberships(
		f.ctx,
		tenantports.ReplaceOrganizationMembershipsRequest{
			ActorID:         f.data.User.ID,
			UserID:          f.data.User.ID,
			BusinessUnitID:  f.data.BusinessUnit.ID,
			OrganizationIDs: []pulid.ID{f.data.Organization.ID, f.secondOrg.ID},
		},
	)

	require.Nil(t, memberships)
	require.True(t, errortypes.IsQuotaExceededError(err))

	remaining, err := f.repo.ListOrganizationMemberships(f.ctx, f.data.User.ID, f.data.BusinessUnit.ID)
	require.NoError(t, err)
	require.Len(t, remaining, 1)
	assert.Equal(t, f.data.Organization.ID, remaining[0].OrganizationID)
}

func TestMembershipsFollowRowLevelSecurityVisibility(t *testing.T) {
	ctx, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, admin)
	secondOrg := seedtest.NewOrganization(data.BusinessUnit.ID, data.State.ID).
		WithName("Second Organization").
		WithScacCode("SECN").
		WithDOTNumber("7654321").
		WithBucketName("second-org-bucket").
		Build(t, ctx, admin)
	colleague := seedtest.NewUser(data.Organization.ID, data.BusinessUnit.ID).
		WithUsername("colleague").
		WithEmail("colleague@example.com").
		Build(t, ctx, admin)

	sysCtx := dbscope.WithSystem(ctx, "user repository test setup")
	rows := []*tenant.OrganizationMembership{}
	for _, userID := range []pulid.ID{data.User.ID, colleague.ID} {
		rows = append(rows,
			&tenant.OrganizationMembership{
				UserID: userID, BusinessUnitID: data.BusinessUnit.ID,
				OrganizationID: data.Organization.ID, IsDefault: true,
			},
			&tenant.OrganizationMembership{
				UserID: userID, BusinessUnitID: data.BusinessUnit.ID,
				OrganizationID: secondOrg.ID,
			},
		)
	}
	require.NoError(t, tenantdb.OrganizationMemberships.CreateMany(sysCtx, admin, rows))

	repo := New(Params{DB: dbtest.NewRLSConnection(t, admin, dsn), Logger: zap.NewNop()})
	tenantCtx := data.Ctx(ctx)

	self, err := repo.GetByID(tenantCtx, tenantports.GetUserByIDRequest{
		TenantInfo:         data.Info(),
		IncludeMemberships: true,
	})
	require.NoError(t, err)
	require.Len(t, self.Memberships, 2, "a person sees their own memberships in every organization")
	for _, membership := range self.Memberships {
		require.NotNil(t, membership.Organization)
		require.NotNil(t, membership.Organization.State)
	}

	other, err := repo.GetByID(tenantCtx, tenantports.GetUserByIDRequest{
		TenantInfo:         data.Info(),
		LookupUserID:       colleague.ID,
		IncludeMemberships: true,
	})
	require.NoError(t, err)
	require.Len(t, other.Memberships, 1, "a colleague's memberships show only in this organization")
	assert.Equal(t, data.Organization.ID, other.Memberships[0].OrganizationID)

	own, err := repo.GetOrganizations(tenantCtx, data.User.ID)
	require.NoError(t, err)
	require.Len(t, own, 2)
	assert.True(t, own[0].IsDefault)
	require.NotNil(t, own[0].User)

	colleagues, err := repo.GetOrganizations(tenantCtx, colleague.ID)
	require.NoError(t, err)
	require.Len(t, colleagues, 1)

	summaries, err := repo.GetUserOrganizationSummaries(tenantCtx, data.User.ID)
	require.NoError(t, err)
	require.Len(t, summaries, 2)
	assert.LessOrEqual(t, summaries[0].Name, summaries[1].Name)

	listed, err := repo.List(tenantCtx, &tenantports.ListUsersRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: data.Info(),
			Pagination: pagination.Info{Limit: 10},
		},
		IncludeMemberships: true,
	})
	require.NoError(t, err)
	require.Equal(t, 2, listed.Total)
	for _, user := range listed.Items {
		if user.ID == data.User.ID {
			assert.Len(t, user.Memberships, 2)
		} else {
			assert.Len(t, user.Memberships, 1)
		}
	}
}

func TestListConnectionHidesTheSystemUserAndOtherOrganizations(t *testing.T) {
	f := newUserFixture(t)
	f.addUser(t, "second")
	seedtest.NewUser(f.secondOrg.ID, f.data.BusinessUnit.ID).
		WithUsername("elsewhere").
		WithEmail("elsewhere@example.com").
		Build(t, f.ctx, f.db)
	seedtest.NewUser(f.data.Organization.ID, f.data.BusinessUnit.ID).
		WithUsername(tenant.SystemUsername).
		WithEmail("system@example.com").
		Build(t, f.ctx, f.db)

	firstCursor, err := pagination.NewCursorInfo(1, "")
	require.NoError(t, err)
	first, err := f.repo.ListConnection(f.ctx, &tenantports.ListUserConnectionRequest{
		Filter: &pagination.QueryOptions{TenantInfo: f.data.Info()},
		Cursor: firstCursor,
	})
	require.NoError(t, err)
	require.Len(t, first.Items, 1)
	require.NotNil(t, first.TotalCount)
	assert.Equal(t, 2, *first.TotalCount)
	require.True(t, first.HasNextPage)

	next, err := first.NextCursor()
	require.NoError(t, err)
	secondCursor, err := pagination.NewCursorInfo(1, next)
	require.NoError(t, err)
	second, err := f.repo.ListConnection(f.ctx, &tenantports.ListUserConnectionRequest{
		Filter: &pagination.QueryOptions{TenantInfo: f.data.Info()},
		Cursor: secondCursor,
	})
	require.NoError(t, err)
	require.Len(t, second.Items, 1)
	assert.NotEqual(t, first.Items[0].ID, second.Items[0].ID)
	assert.False(t, second.HasNextPage)

	options, err := f.repo.SelectOptions(f.ctx, &pagination.SelectQueryRequest{
		TenantInfo: f.data.Info(),
		Pagination: pagination.Info{Limit: 10},
		Query:      "SECOND",
	})
	require.NoError(t, err)
	require.Len(t, options.Items, 1)
	assert.Equal(t, "second", options.Items[0].Username)
}

func TestUpdateColumnsWritesOnlyTheNamedColumnsAndChecksTheVersion(t *testing.T) {
	f := newUserFixture(t)

	user, err := f.repo.GetByID(f.ctx, tenantports.GetUserByIDRequest{TenantInfo: f.data.Info()})
	require.NoError(t, err)
	stale := *user

	user.JobTitle = "Dispatcher"
	user.Name = "Not Written"
	focus := int64(1_900_000_000)
	user.FocusUntil = &focus
	updated, err := f.repo.UpdateColumns(f.ctx, tenantports.UpdateUserColumnsRequest{
		Entity:  user,
		Columns: []string{"job_title", "focus_until"},
	})
	require.NoError(t, err)
	assert.Equal(t, stale.Version+1, updated.Version)

	stored, err := f.repo.GetByID(f.ctx, tenantports.GetUserByIDRequest{TenantInfo: f.data.Info()})
	require.NoError(t, err)
	assert.Equal(t, "Dispatcher", stored.JobTitle)
	assert.Equal(t, stale.Name, stored.Name)
	require.NotNil(t, stored.FocusUntil)
	assert.Equal(t, focus, *stored.FocusUntil)

	stale.JobTitle = "Stale"
	_, err = f.repo.UpdateColumns(f.ctx, tenantports.UpdateUserColumnsRequest{
		Entity:  &stale,
		Columns: []string{"job_title"},
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsVersionMismatchError(err))

	_, err = f.repo.UpdateColumns(f.ctx, tenantports.UpdateUserColumnsRequest{
		Entity:  stored,
		Columns: []string{"password"},
	})
	require.Error(t, err)
}

func TestBulkUpdateStatusReturnsTheUpdatedUsers(t *testing.T) {
	f := newUserFixture(t)
	other := f.addUser(t, "bulkuser")

	updated, err := f.repo.BulkUpdateStatus(f.ctx, &tenantports.BulkUpdateUserStatusRequest{
		TenantInfo: f.data.Info(),
		UserIDs:    []pulid.ID{other.ID},
		Status:     domaintypes.StatusInactive,
	})
	require.NoError(t, err)
	require.Len(t, updated, 1)
	assert.Equal(t, domaintypes.StatusInactive, updated[0].Status)

	stranger := seedtest.SeedAdditionalTenant(t, f.ctx, f.db, "UD")
	_, err = f.repo.BulkUpdateStatus(f.ctx, &tenantports.BulkUpdateUserStatusRequest{
		TenantInfo: f.data.Info(),
		UserIDs:    []pulid.ID{stranger.User.ID},
		Status:     domaintypes.StatusInactive,
	})
	require.Error(t, err)
}

func TestGetTenantMemberAcceptsAMembershipOrTheCurrentOrganization(t *testing.T) {
	f := newUserFixture(t)
	visitor := seedtest.NewUser(f.secondOrg.ID, f.data.BusinessUnit.ID).
		WithUsername("visitor").
		WithEmail("visitor@example.com").
		Build(t, f.ctx, f.db)
	outsider := seedtest.NewUser(f.secondOrg.ID, f.data.BusinessUnit.ID).
		WithUsername("outsider").
		WithEmail("outsider@example.com").
		Build(t, f.ctx, f.db)
	f.addMemberships(t, visitor.ID, f.data.Organization.ID)

	now := visitor.CreatedAt + 1
	for _, userID := range []pulid.ID{f.data.User.ID, visitor.ID} {
		member, err := f.repo.GetTenantMember(f.ctx, tenantports.GetTenantMemberRequest{
			UserID:     userID,
			TenantInfo: f.data.Info(),
			Now:        now,
		})
		require.NoError(t, err)
		assert.Equal(t, userID, member.ID)
	}

	_, err := f.repo.GetTenantMember(f.ctx, tenantports.GetTenantMemberRequest{
		UserID:     outsider.ID,
		TenantInfo: f.data.Info(),
		Now:        now,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))
}

func TestUpdateLastLoginAtKeepsTheVersion(t *testing.T) {
	f := newUserFixture(t)

	require.NoError(t, f.repo.UpdateLastLoginAt(f.ctx, f.data.User.ID))

	user, err := f.repo.GetByID(f.ctx, tenantports.GetUserByIDRequest{TenantInfo: f.data.Info()})
	require.NoError(t, err)
	require.NotNil(t, user.LastLoginAt)
	assert.Equal(t, f.data.User.Version, user.Version)
}

func TestSystemScopedLookups(t *testing.T) {
	f := newUserFixture(t)
	system := seedtest.NewUser(f.data.Organization.ID, f.data.BusinessUnit.ID).
		WithUsername(tenant.SystemUsername).
		WithEmail("system@example.com").
		Build(t, f.ctx, f.db)

	byEmail, err := f.repo.FindByEmail(context.Background(), f.data.User.EmailAddress)
	require.NoError(t, err)
	assert.Equal(t, f.data.User.ID, byEmail.ID)

	_, err = f.repo.FindByEmail(context.Background(), "nobody@example.com")
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))

	byID, err := f.repo.FindByIDForLogin(context.Background(), f.data.User.ID)
	require.NoError(t, err)
	assert.Equal(t, f.data.User.EmailAddress, byID.EmailAddress)

	sys, err := f.repo.GetSystemUser(context.Background(), "id")
	require.NoError(t, err)
	assert.Equal(t, system.ID, sys.ID)
	assert.Empty(t, sys.EmailAddress)
}
