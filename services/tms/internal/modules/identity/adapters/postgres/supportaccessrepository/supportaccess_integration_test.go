//go:build integration

package supportaccessrepository

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/internal/modules/identity/ports/supportaccessports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
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

func TestSupportGrants_TheInstanceWideListHoldsOnlyLiveGrantsOfEveryOrganization(t *testing.T) {
	f := newFixture(t)
	repo := f.grants()
	inA, err := f.create(t, f.a, "jane.doe@trenova.app")
	require.NoError(t, err)
	inB, err := f.create(t, f.b, "sam.lee@trenova.app")
	require.NoError(t, err)
	ended, err := f.create(t, f.a, "alex.kim@trenova.app")
	require.NoError(t, err)
	revoked := *ended
	revoked.RevokedByID = f.a.User.ID
	revoked.RevokeReason = "Done"
	_, err = repo.Revoke(f.a.Ctx(t.Context()), &revoked, f.now)
	require.NoError(t, err)

	live, err := repo.ListLive(t.Context(), &supportaccessports.ListLiveRequest{Now: f.now, Limit: 100})
	require.NoError(t, err)
	names := make(map[pulid.ID]string, len(live))
	for _, grant := range live {
		names[grant.Grant.ID] = grant.OrganizationName
	}
	require.Len(t, live, 2, "one in each organization; the revoked one is not live")
	assert.Equal(t, f.a.Organization.Name, names[inA.ID])
	assert.Equal(t, f.b.Organization.Name, names[inB.ID])
	assert.NotContains(t, names, ended.ID)

	one, err := repo.ListLive(t.Context(), &supportaccessports.ListLiveRequest{Now: f.now, Limit: 1})
	require.NoError(t, err)
	assert.Len(t, one, 1, "the list is bounded by what the caller asks for")

	later, err := repo.ListLive(t.Context(), &supportaccessports.ListLiveRequest{Now: f.now + 4*3600 + 1, Limit: 100})
	require.NoError(t, err)
	assert.Empty(t, later, "a grant past its expiry is not live, whether or not the sweep has recorded it")
}

func (f *fixture) execAs(t *testing.T, userID pulid.ID, statement string, args ...any) error {
	t.Helper()
	info := f.a.Info()
	ctx := dbscope.WithTenant(t.Context(), dbscope.Tenant{
		OrganizationID: info.OrgID,
		BusinessUnitID: info.BuID,
		UserID:         userID,
	})
	return dbtx.WriteErr(ctx, f.conn, func(ctx context.Context) error {
		_, err := keel.Exec(ctx, f.conn.Keel(ctx), statement, args...)
		return err
	})
}

func TestPaymentDestinations_ASupportPrincipalCannotChangeWhereMoneyIsSent(t *testing.T) {
	f := newFixture(t)
	grant, err := f.create(t, f.a, "jane.doe@trenova.app")
	require.NoError(t, err)
	org := f.a.Organization.ID.String()

	err = f.execAs(t, grant.PrincipalUserID, `UPDATE organizations SET city = 'Elsewhere' WHERE id = $1`, org)
	require.Error(t, err)
	assert.True(t, dberror.IsSupportPaymentDestinationRefusal(err), "refused as a payment destination: %v", err)

	require.NoError(t,
		f.execAs(t, grant.PrincipalUserID, `UPDATE organizations SET timezone = 'America/Chicago' WHERE id = $1`, org),
		"a column that sends no money stays writable to full support")
	require.NoError(t,
		f.execAs(t, grant.PrincipalUserID, `UPDATE organizations SET city = city WHERE id = $1`, org),
		"saving a row without changing where money goes is not a change")
	require.NoError(t,
		f.execAs(t, f.a.User.ID, `UPDATE organizations SET city = 'Elsewhere' WHERE id = $1`, org),
		"the organization's own people change it as before")
}

func (f *fixture) createAt(t *testing.T, createdAt, expiresAt int64) *supportaccess.Grant {
	t.Helper()
	info := f.a.Info()
	principal := supportaccess.NewPrincipalUser(supportaccess.PrincipalUserParams{
		OrganizationID: info.OrgID,
		BusinessUnitID: info.BuID,
		StaffEmail:     "jane.doe@trenova.app",
		Password:       "$2a$10$unusableunusableunusableunusableunusableunusableunusa",
	})
	grant, err := f.grants().Create(f.a.Ctx(t.Context()), &supportaccessports.CreateGrantRequest{
		Grant: &supportaccess.Grant{
			OrganizationID:  info.OrgID,
			BusinessUnitID:  info.BuID,
			StaffEmail:      "jane.doe@trenova.app",
			Reason:          "Ticket 4411",
			Scope:           supportaccess.ScopeReadOnly,
			DurationHours:   72,
			PrincipalUserID: principal.ID,
			GrantedByID:     info.UserID,
			ExpiresAt:       expiresAt,
			CreatedAt:       createdAt,
			UpdatedAt:       createdAt,
		},
		Principal: principal,
		Now:       f.now,
	})
	require.NoError(t, err)
	return grant
}

func TestSupportGrants_AnExtensionIsCountedFromWhenItWasMade(t *testing.T) {
	f := newFixture(t)
	repo := f.grants()
	grant := f.createAt(t, f.now-70*3600, f.now+2*3600)

	next := *grant
	require.NoError(t, next.Extend(f.a.User.ID, 72, f.now))
	extended, err := repo.Extend(f.a.Ctx(t.Context()), &next, f.now)
	require.NoError(t, err, "72 hours from now is past 72 hours from the grant, and allowed")
	assert.Equal(t, f.now+72*3600, extended.ExpiresAt)
	require.NotNil(t, extended.ExtendedAt)
	assert.Equal(t, f.now, *extended.ExtendedAt)
	assert.Equal(t, f.a.User.ID, extended.ExtendedByID)

	tooLong := *extended
	tooLong.ExpiresAt = f.now + 73*3600
	_, err = repo.Extend(f.a.Ctx(t.Context()), &tooLong, f.now)
	require.Error(t, err, "the database keeps the end within 72 hours of the extension")
}

func TestSupportGrants_TwoExtensionsAtOnceLeaveAConsistentGrant(t *testing.T) {
	f := newFixture(t)
	repo := f.grants()
	grant, err := f.create(t, f.a, "jane.doe@trenova.app")
	require.NoError(t, err)

	var wg sync.WaitGroup
	results := make([]error, 2)
	start := make(chan struct{})
	for i, hours := range []int{24, 72} {
		wg.Go(func() {
			<-start
			next := *grant
			if extendErr := next.Extend(f.a.User.ID, hours, f.now); extendErr != nil {
				results[i] = extendErr
				return
			}
			_, results[i] = repo.Extend(f.a.Ctx(context.Background()), &next, f.now)
		})
	}
	close(start)
	wg.Wait()
	for _, extendErr := range results {
		require.NoError(t, extendErr)
	}

	held, err := repo.Get(f.a.Ctx(t.Context()), f.a.Info(), grant.ID)
	require.NoError(t, err)
	require.NotNil(t, held.ExtendedAt)
	assert.Equal(t, *held.ExtendedAt+int64(held.DurationHours)*3600, held.ExpiresAt,
		"the end and the length on the row come from the same extension")
}

func TestSupportGrants_AnEndedGrantCannotBeExtended(t *testing.T) {
	f := newFixture(t)
	repo := f.grants()
	grant, err := f.create(t, f.a, "jane.doe@trenova.app")
	require.NoError(t, err)

	revoked := *grant
	revoked.RevokedByID = f.a.User.ID
	revoked.RevokeReason = "Done"
	_, err = repo.Revoke(f.a.Ctx(t.Context()), &revoked, f.now)
	require.NoError(t, err)

	next := *grant
	next.ExpiresAt = f.now + 24*3600
	next.DurationHours = 24
	next.ExtendedByID = f.a.User.ID
	_, err = repo.Extend(f.a.Ctx(t.Context()), &next, f.now)
	require.ErrorIs(t, err, supportaccess.ErrGrantNotFound, "the write proves the grant is still live")

	expired := f.createAt(t, f.now-5*3600, f.now-3600)
	late := *expired
	late.ExpiresAt = f.now + 24*3600
	late.DurationHours = 24
	late.ExtendedByID = f.a.User.ID
	_, err = repo.Extend(f.a.Ctx(t.Context()), &late, f.now)
	require.ErrorIs(t, err, supportaccess.ErrGrantNotFound)
}

const queueTrenovaEmail = `INSERT INTO email_messages
	(id, business_unit_id, organization_id, purpose, provider, idempotency_key, from_email,
	 from_name, to_recipients, subject, sender, kind)
	VALUES ($1, $2, $3, 'General', 'Postmark', $1, 'noreply@trenova.app', 'Trenova',
	 ARRAY['ap@customer.example'], 'Invoice', 'Trenova', 'System')`

func (f *fixture) queueEmail(t *testing.T, userID pulid.ID) error {
	t.Helper()
	info := f.a.Info()
	id := pulid.MustNew("em_").String()
	return f.execAs(t, userID, queueTrenovaEmail, id, info.BuID.String(), info.OrgID.String())
}

func TestOutbound_ASupportPrincipalSendsNothingOutsideTheOrganization(t *testing.T) {
	f := newFixture(t)
	grant, err := f.create(t, f.a, "jane.doe@trenova.app")
	require.NoError(t, err)

	err = f.queueEmail(t, grant.PrincipalUserID)
	require.Error(t, err)
	assert.True(t, dberror.IsSupportOutboundRefusal(err), "refused as a send: %v", err)

	require.NoError(t, f.queueEmail(t, f.a.User.ID), "a person of the organization still sends")
	require.NoError(t, f.queueEmail(t, pulid.Nil),
		"a send the system started while a support session is open still goes out")
}

func TestOutbound_TheDatabaseGuardsExactlyWhatThePolicyNames(t *testing.T) {
	f := newFixture(t)

	rows, err := keel.Pool(f.admin).Query(t.Context(), `
		SELECT c.relname, (t.tgtype & 4) <> 0 AS on_insert, (t.tgtype & 16) <> 0 AS on_update,
		       encode(t.tgargs, 'escape') AS args, pg_get_triggerdef(t.oid) AS def
		FROM pg_trigger t
		JOIN pg_class c ON c.oid = t.tgrelid
		WHERE t.tgfoid = 'support_outbound_guard'::regproc AND NOT t.tgisinternal`)
	require.NoError(t, err)
	defer rows.Close()

	want := map[string]supportaccess.OutboundGuard{}
	whenColumns := map[string][]string{}
	for _, g := range supportaccess.OutboundGuards() {
		want[g.Table+"/"+g.WhenColumn] = g
		if g.WhenColumn != "" {
			whenColumns[g.Table] = append(whenColumns[g.Table], g.WhenColumn)
		}
	}

	held := map[string]supportaccess.OutboundGuard{}
	for rows.Next() {
		var table, args, def string
		var onInsert, onUpdate bool
		require.NoError(t, rows.Scan(&table, &onInsert, &onUpdate, &args, &def))
		guard := supportaccess.OutboundGuard{Table: table, OnInsert: onInsert, OnUpdate: onUpdate}
		if _, when, found := strings.Cut(def, " WHEN "); found {
			for _, column := range whenColumns[table] {
				if strings.Contains(when, column) {
					guard.WhenColumn = column
				}
			}
			require.NotEmpty(t, guard.WhenColumn, "%s has a condition the policy does not name: %s", table, def)
		}
		if args != "" {
			guard.Columns = strings.Split(strings.TrimSuffix(args, `\000`), `\000`)
		}
		held[table+"/"+guard.WhenColumn] = guard
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, want, held)
}

func TestPaymentDestinations_TheDatabaseGuardsExactlyWhatThePolicyNames(t *testing.T) {
	f := newFixture(t)

	rows, err := keel.Pool(f.admin).Query(t.Context(), `
		SELECT c.relname, (t.tgtype & 4) <> 0 AS on_insert,
		       encode(t.tgargs, 'escape') AS args
		FROM pg_trigger t
		JOIN pg_class c ON c.oid = t.tgrelid
		WHERE t.tgfoid = 'support_payment_destination_guard'::regproc AND NOT t.tgisinternal`)
	require.NoError(t, err)
	defer rows.Close()

	type guarded struct {
		onInsert bool
		columns  []string
	}
	held := map[string]guarded{}
	for rows.Next() {
		var table, args string
		var onInsert bool
		require.NoError(t, rows.Scan(&table, &onInsert, &args))
		columns := strings.Split(strings.TrimSuffix(args, `\000`), `\000`)
		held[table] = guarded{onInsert: onInsert, columns: columns}
	}
	require.NoError(t, rows.Err())

	want := map[string]guarded{}
	for _, d := range supportaccess.PaymentDestinations() {
		want[d.Table] = guarded{onInsert: d.OnInsert, columns: d.Columns}
	}
	assert.Equal(t, want, held)
}
