//go:build integration

package supportaccessrepository

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/internal/modules/identity/ports/supportaccessports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fixture struct {
	conn  *postgres.Connection
	admin *keel.DB
	a     *seedtest.TestData
	b     *seedtest.TestData
	now   int64
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	ctx, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	return &fixture{
		conn:  dbtest.NewRLSConnection(t, admin, dsn),
		admin: admin,
		a:     seedtest.SeedFullTestData(t, ctx, admin),
		b:     seedtest.SeedAdditionalTenant(t, ctx, admin, "SAB"),
		now:   time.Now().Unix(),
	}
}

func (f *fixture) grants() supportaccessports.GrantRepository {
	return NewGrantRepository(Params{DB: f.conn, Logger: zap.NewNop()})
}

func (f *fixture) create(t *testing.T, tenant *seedtest.TestData, email string) (*supportaccess.Grant, error) {
	t.Helper()
	info := tenant.Info()
	principal := supportaccess.NewPrincipalUser(supportaccess.PrincipalUserParams{
		OrganizationID: info.OrgID,
		BusinessUnitID: info.BuID,
		StaffEmail:     email,
		Password:       "$2a$10$unusableunusableunusableunusableunusableunusableunusa",
	})
	return f.grants().Create(tenant.Ctx(t.Context()), &supportaccessports.CreateGrantRequest{
		Grant: &supportaccess.Grant{
			OrganizationID:  info.OrgID,
			BusinessUnitID:  info.BuID,
			StaffEmail:      email,
			Reason:          "Ticket 4411",
			Scope:           supportaccess.ScopeFull,
			DurationHours:   4,
			PrincipalUserID: principal.ID,
			GrantedByID:     info.UserID,
			ExpiresAt:       f.now + 4*3600,
			CreatedAt:       f.now,
			UpdatedAt:       f.now,
		},
		Principal: principal,
		Now:       f.now,
	})
}

func TestSupportGrants_AnotherOrganizationSeesNoneOfThem(t *testing.T) {
	f := newFixture(t)
	repo := f.grants()

	grant, err := f.create(t, f.a, "jane.doe@trenova.app")
	require.NoError(t, err)

	_, err = repo.Get(f.b.Ctx(t.Context()), f.b.Info(), grant.ID)
	require.Error(t, err, "row-level security hides another organization's grant")

	list, err := repo.List(f.b.Ctx(t.Context()), &supportaccessports.ListGrantsRequest{
		Filter: &pagination.QueryOptions{TenantInfo: f.b.Info()},
		Cursor: pagination.CursorInfo{Limit: 50},
	})
	require.NoError(t, err)
	assert.Empty(t, list.Items)

	_, err = repo.ByPrincipal(f.b.Ctx(t.Context()), f.b.Info(), grant.PrincipalUserID)
	require.ErrorIs(t, err, supportaccess.ErrGrantNotFound)

	own, err := repo.Get(f.a.Ctx(t.Context()), f.a.Info(), grant.ID)
	require.NoError(t, err)
	assert.Equal(t, "jane.doe@trenova.app", own.StaffEmail)
	require.NotNil(t, own.GrantedBy)
	assert.Equal(t, f.a.User.ID, own.GrantedBy.ID)
}

func TestSupportGrants_EntryFindsOnlyTheNamedOrganizationsLiveGrant(t *testing.T) {
	f := newFixture(t)
	repo := f.grants()

	grant, err := f.create(t, f.a, "jane.doe@trenova.app")
	require.NoError(t, err)

	found, err := repo.FindLiveForEntry(t.Context(), &supportaccessports.FindLiveGrantRequest{
		OrganizationID: f.a.Organization.ID, StaffEmail: "jane.doe@trenova.app", Now: f.now,
	})
	require.NoError(t, err)
	assert.Equal(t, grant.ID, found.ID)

	_, err = repo.FindLiveForEntry(t.Context(), &supportaccessports.FindLiveGrantRequest{
		OrganizationID: f.b.Organization.ID, StaffEmail: "jane.doe@trenova.app", Now: f.now,
	})
	require.ErrorIs(t, err, supportaccess.ErrGrantNotFound)

	_, err = repo.FindLiveForEntry(t.Context(), &supportaccessports.FindLiveGrantRequest{
		OrganizationID: f.a.Organization.ID, StaffEmail: "jane.doe@trenova.app", Now: grant.ExpiresAt,
	})
	require.ErrorIs(t, err, supportaccess.ErrGrantNotFound, "an expired grant admits nobody")
}

func TestSupportGrants_TwoGrantsForOneStaffMemberAtOnceLeaveOne(t *testing.T) {
	f := newFixture(t)

	var wg sync.WaitGroup
	results := make([]error, 2)
	start := make(chan struct{})
	for i := range results {
		wg.Go(func() {
			<-start
			_, results[i] = f.create(t, f.a, "jane.doe@trenova.app")
		})
	}
	close(start)
	wg.Wait()

	succeeded := 0
	for _, err := range results {
		if err == nil {
			succeeded++
		}
	}
	assert.Equal(t, 1, succeeded, "the per-organization lock serializes grants: %v", results)
}

func TestSupportGrants_TwoRevocationsAtOnceEndTheGrantOnce(t *testing.T) {
	f := newFixture(t)
	repo := f.grants()
	grant, err := f.create(t, f.a, "jane.doe@trenova.app")
	require.NoError(t, err)

	var wg sync.WaitGroup
	results := make([]error, 2)
	start := make(chan struct{})
	for i := range results {
		wg.Go(func() {
			<-start
			next := *grant
			next.RevokedByID = f.a.User.ID
			next.RevokeReason = "Done"
			_, results[i] = repo.Revoke(f.a.Ctx(context.Background()), &next, f.now)
		})
	}
	close(start)
	wg.Wait()

	succeeded, lost := 0, 0
	for _, err := range results {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, supportaccess.ErrGrantNotFound):
			lost++
		}
	}
	assert.Equal(t, 1, succeeded)
	assert.Equal(t, 1, lost)
}

func TestSupportGrants_TheSupportPrincipalCannotBeRenamedOrActivated(t *testing.T) {
	f := newFixture(t)
	grant, err := f.create(t, f.a, "jane.doe@trenova.app")
	require.NoError(t, err)

	for _, statement := range []string{
		`UPDATE users SET name = 'Jane Customer' WHERE id = $1`,
		`UPDATE users SET status = 'Active' WHERE id = $1`,
		`UPDATE users SET email_address = 'jane@customer.example' WHERE id = $1`,
		`UPDATE users SET password = 'x' WHERE id = $1`,
	} {
		_, err = keel.Pool(f.admin).Exec(t.Context(), statement, grant.PrincipalUserID.String())
		require.Error(t, err, statement)
	}

	_, err = keel.Pool(f.admin).Exec(t.Context(),
		`UPDATE users SET last_login_at = 1 WHERE id = $1`, grant.PrincipalUserID.String())
	require.NoError(t, err, "columns that do not decide who the principal is stay writable")
}

func TestAssertionLedger_TwoUsesOfOneAssertionAtOnceLetOneIn(t *testing.T) {
	f := newFixture(t)
	ledger := NewAssertionLedger(Params{DB: f.conn, Logger: zap.NewNop()})
	id := "as_" + pulid.MustNew("x").String()[2:]

	const attempts = 8
	var wg sync.WaitGroup
	results := make([]bool, attempts)
	errs := make([]error, attempts)
	start := make(chan struct{})
	for i := range attempts {
		wg.Go(func() {
			<-start
			results[i], errs[i] = ledger.Use(context.Background(), &supportaccess.AssertionUse{
				AssertionID: id, UsedAt: f.now, ExpiresAt: f.now + 75,
			})
		})
	}
	close(start)
	wg.Wait()

	first := 0
	for i := range attempts {
		require.NoError(t, errs[i])
		if results[i] {
			first++
		}
	}
	assert.Equal(t, 1, first, "exactly one use of an assertion succeeds")
}

func TestAssertionLedger_ForgetsIDsThatCanNoLongerVerify(t *testing.T) {
	f := newFixture(t)
	ledger := NewAssertionLedger(Params{DB: f.conn, Logger: zap.NewNop()})
	old := "as_" + pulid.MustNew("x").String()[2:]

	used, err := ledger.Use(t.Context(), &supportaccess.AssertionUse{
		AssertionID: old, UsedAt: f.now - 600, ExpiresAt: f.now - 500,
	})
	require.NoError(t, err)
	require.True(t, used)

	_, err = ledger.Use(t.Context(), &supportaccess.AssertionUse{
		AssertionID: "as_" + pulid.MustNew("x").String()[2:], UsedAt: f.now, ExpiresAt: f.now + 75,
	})
	require.NoError(t, err)

	var remaining int
	require.NoError(t, keel.Pool(f.admin).QueryRow(t.Context(),
		`SELECT count(*) FROM support_assertion_uses WHERE assertion_id = $1`, old).Scan(&remaining))
	assert.Zero(t, remaining)
}

func TestSupportKeySet_IsOneInstanceWideRow(t *testing.T) {
	f := newFixture(t)
	keys := NewKeySetRepository(Params{DB: f.conn, Logger: zap.NewNop()})

	_, err := keys.Get(t.Context())
	require.ErrorIs(t, err, supportaccess.ErrNoKeySet)

	set := &supportaccess.KeySet{
		InstanceID:  "inst_1",
		StaffDomain: "trenova.app",
		Keys: []supportaccess.SigningKey{{
			KeyID: "cp-1", KeyType: "OKP", Curve: "Ed25519", Algorithm: "EdDSA", Use: "sig",
			X: "11qYAYKxCrfVS_7TyWQHOg7hcvPapiMlrwIaaPcHURo",
		}},
		ReceivedAt: f.now,
	}
	require.NoError(t, set.Normalize())
	require.NoError(t, keys.Save(t.Context(), set))
	set.StaffDomain = "staff.trenova.app"
	require.NoError(t, keys.Save(t.Context(), set))

	held, err := keys.Get(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "staff.trenova.app", held.StaffDomain)
	assert.Equal(t, "cp-1", held.Keys[0].KeyID)
}
