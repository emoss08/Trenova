//go:build integration

package seedaccountservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/cloud/seedaccounts/seedaccountport"
	"github.com/emoss08/trenova/internal/cloud/seedaccounts/seedaccountrepository"
	"github.com/emoss08/trenova/internal/core/domain/apikey"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantbootstrap"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type legacySeed struct {
	bu        *tenant.BusinessUnit
	logistics *tenant.Organization
	transport *tenant.Organization
	admin     *tenant.User
	trAdmin   *tenant.User
	customer  *tenant.User
	system    *tenant.User
	key       *apikey.Key
}

func seedLegacyAccounts(t *testing.T, db *bun.DB) *legacySeed {
	t.Helper()
	ctx := t.Context()

	customer := seedtest.SeedFullTestData(t, ctx, db)

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)

	out := &legacySeed{}
	out.bu = seedtest.NewBusinessUnit().WithName("Default Business Unit").WithCode("DEFAULT").Build(t, ctx, tx)
	out.logistics = seedtest.NewOrganization(out.bu.ID, customer.State.ID).
		WithName("Trenova Logistics").
		WithScacCode("TRNV").
		WithDOTNumber("1234567").
		WithBucketName("trenova-logistics").
		Build(t, ctx, tx)
	out.transport = seedtest.NewOrganization(out.bu.ID, customer.State.ID).
		WithName("Trenova Transportation").
		WithScacCode("TTNV").
		WithDOTNumber("0000000").
		WithBucketName("trenova-transportation").
		Build(t, ctx, tx)

	out.admin = seedtest.NewUser(out.logistics.ID, out.bu.ID).
		WithName("System Administrator").
		WithUsername("admin").
		WithEmail("admin@trenova.app").
		WithPassword(seedaccountport.SeededPassword).
		Build(t, ctx, tx)
	out.trAdmin = seedtest.NewUser(out.transport.ID, out.bu.ID).
		WithName("Trenova Transportation Administrator").
		WithUsername("admin-transport").
		WithEmail("admin.transport@trenova.app").
		WithPassword(seedaccountport.SeededPassword).
		Build(t, ctx, tx)
	out.system = seedtest.NewUser(out.logistics.ID, out.bu.ID).
		WithName("System").
		WithUsername(tenant.SystemUsername).
		WithEmail("system@trenova.app").
		WithPassword("an-unrelated-system-password").
		Build(t, ctx, tx)
	out.customer = seedtest.NewUser(customer.Organization.ID, customer.BusinessUnit.ID).
		WithName("Customer Operator").
		WithUsername("admin-logistics").
		WithEmail("ops@customer.example").
		WithPassword(seedaccountport.SeededPassword).
		Build(t, ctx, tx)

	now := timeutils.NowUnix()
	memberships := []*tenant.OrganizationMembership{
		{BusinessUnitID: out.bu.ID, UserID: out.admin.ID, OrganizationID: out.logistics.ID, GrantedByID: out.admin.ID, IsDefault: true, JoinedAt: now},
		{BusinessUnitID: out.bu.ID, UserID: out.trAdmin.ID, OrganizationID: out.transport.ID, GrantedByID: out.trAdmin.ID, IsDefault: true, JoinedAt: now},
		{BusinessUnitID: customer.BusinessUnit.ID, UserID: out.customer.ID, OrganizationID: customer.Organization.ID, GrantedByID: out.customer.ID, IsDefault: true, JoinedAt: now},
	}
	for _, membership := range memberships {
		_, err = tx.NewInsert().Model(membership).Exec(ctx)
		require.NoError(t, err)
	}

	role, err := tenantbootstrap.CreateAdminRole(ctx, tx, tenantbootstrap.AdminRoleParams{
		Scope:     tenantbootstrap.Scope{OrganizationID: out.logistics.ID, BusinessUnitID: out.bu.ID, Now: now},
		CreatedBy: out.admin.ID,
		Registry:  permission.NewRegistry(),
	})
	require.NoError(t, err)
	require.NoError(t, tenantbootstrap.AssignRole(ctx, tx, tenantbootstrap.RoleAssignmentParams{
		UserID:         out.admin.ID,
		OrganizationID: out.logistics.ID,
		RoleID:         role.ID,
		AssignedBy:     out.admin.ID,
		AssignedAt:     now,
	}))

	out.key = &apikey.Key{
		BusinessUnitID: out.bu.ID,
		OrganizationID: out.logistics.ID,
		Name:           "Seeded integration",
		KeyPrefix:      "trv_seedtest",
		SecretHash:     "hash",
		SecretSalt:     "salt",
		Status:         apikey.StatusActive,
		CreatedByID:    out.admin.ID,
	}
	_, err = tx.NewInsert().Model(out.key).Exec(ctx)
	require.NoError(t, err)

	require.NoError(t, seedhelpers.NewPersistentEntityTracker(tx).TrackBatch(ctx, []seedhelpers.TrackedEntity{
		{Table: buncolgen.UserTable.Name, ID: out.admin.ID, SeedName: seedaccountport.SeedName},
		{Table: buncolgen.OrganizationTable.Name, ID: out.logistics.ID, SeedName: seedaccountport.SeedName},
		{Table: buncolgen.OrganizationTable.Name, ID: out.transport.ID, SeedName: seedaccountport.SeedName},
	}))

	require.NoError(t, tx.Commit())

	return out
}

func newIntegrationService(conn *postgres.Connection) (*Service, *fakeSessions, *fakeAuditor) {
	sessions := &fakeSessions{}
	auditor := &fakeAuditor{}
	service := New(Params{
		Repo:            seedaccountrepository.New(seedaccountrepository.Params{DB: conn, Logger: zap.NewNop()}),
		Sessions:        sessions,
		PermissionCache: &fakePermCache{},
		Auditor:         auditor,
		Logger:          zap.NewNop(),
	})

	return service, sessions, auditor
}

func loadUser(t *testing.T, db *bun.DB, username string) *tenant.User {
	t.Helper()

	user := new(tenant.User)
	err := db.NewSelect().Model(user).Where(buncolgen.UserColumns.Username.Eq(), username).Scan(t.Context())
	if err != nil {
		return nil
	}

	return user
}

func TestRetireSeedAccountsAgainstPostgres(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	seeded := seedLegacyAccounts(t, db)
	conn := postgres.NewTestConnection(db)
	service, sessions, auditor := newIntegrationService(conn)

	dry, err := service.Run(ctx, RunRequest{})
	require.NoError(t, err)
	require.Len(t, dry.Users, 2, "the customer's admin-logistics has a different email and is never matched")

	admin := userReport(t, dry, "admin")
	assert.Equal(t, UserActionRetire, admin.Action)
	assert.Equal(t, []Provenance{ProvenanceSeedTracking, ProvenanceSeededPassword}, admin.Provenance)
	assert.False(t, admin.Delete, "the API key it created references it")

	trAdmin := userReport(t, dry, "admin-transport")
	assert.Equal(t, []Provenance{ProvenanceSeededPassword}, trAdmin.Provenance)
	assert.True(t, trAdmin.Delete)

	logistics := orgReport(t, dry, seeded.logistics.ID)
	assert.Equal(t, OrganizationActionReview, logistics.Action)
	assert.Contains(t, logistics.Reasons, "hosts the instance system user")
	require.NotNil(t, dry.SystemUser)
	assert.Equal(t, seeded.system.ID, dry.SystemUser.ID)
	assert.Equal(t, OrganizationActionRemove, orgReport(t, dry, seeded.transport.ID).Action)
	assert.Equal(t, domaintypes.StatusActive, loadUser(t, db, "admin").Status, "a dry run changes nothing")
	assert.Empty(t, auditor.changes)

	applied, err := service.Run(ctx, RunRequest{Apply: true})
	require.NoError(t, err)
	require.Empty(t, applied.Errors)

	retiredAdmin := loadUser(t, db, "admin")
	require.NotNil(t, retiredAdmin, "a referenced account is kept as a tombstone")
	assert.Equal(t, domaintypes.StatusInactive, retiredAdmin.Status)
	assert.True(t, retiredAdmin.IsLocked)
	assert.True(t, retiredAdmin.MustChangePassword)
	assert.Error(t, bcrypt.CompareHashAndPassword(
		[]byte(retiredAdmin.Password),
		[]byte(seedaccountport.SeededPassword),
	))

	memberships, err := db.NewSelect().
		Model((*tenant.OrganizationMembership)(nil)).
		Where(buncolgen.OrganizationMembershipColumns.UserID.Eq(), seeded.admin.ID).
		Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, memberships)

	assignments, err := db.NewSelect().
		Model((*permission.UserRoleAssignment)(nil)).
		Where(buncolgen.UserRoleAssignmentColumns.UserID.Eq(), seeded.admin.ID).
		Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, assignments)

	key := new(apikey.Key)
	require.NoError(t, db.NewSelect().Model(key).Where(buncolgen.KeyColumns.ID.Eq(), seeded.key.ID).Scan(ctx))
	assert.Equal(t, apikey.StatusRevoked, key.Status)
	assert.NotZero(t, key.RevokedAt)

	assert.Nil(t, loadUser(t, db, "admin-transport"), "an unreferenced account is deleted")
	assert.True(t, orgReport(t, applied, seeded.transport.ID).Removed)
	seedtest.AssertOrganizationNotExists(t, ctx, db, seeded.transport.ID)
	seedtest.AssertOrganizationExists(t, ctx, db, seeded.logistics.ID)

	customer := loadUser(t, db, "admin-logistics")
	require.NotNil(t, customer)
	assert.Equal(t, domaintypes.StatusActive, customer.Status, "a customer's account is never touched")
	assert.Equal(t, seeded.customer.Password, customer.Password)

	assert.ElementsMatch(t, []pulid.ID{seeded.admin.ID, seeded.trAdmin.ID}, sessions.ended)
	assert.Equal(t, len(auditor.changes), applied.AuditEntries)
	assert.NotEmpty(t, auditor.changes)
	for _, change := range auditor.changes {
		assert.NotEqual(t, seeded.transport.ID, change.OrganizationID)
	}

	recorded := len(auditor.changes)
	again, err := service.Run(ctx, RunRequest{Apply: true})
	require.NoError(t, err)
	require.Len(t, again.Users, 1)
	assert.Equal(t, UserActionAlreadyRetired, again.Users[0].Action)
	assert.Len(t, auditor.changes, recorded, "a second run changes and records nothing")
}
