//go:build integration

package postgres

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/databaseconfig"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/internal/testutil/shapetest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap/zaptest"
)

type rlsFixture struct {
	admin   *keel.DB
	cfg     *config.Config
	key     []byte
	tenantA dbscope.Tenant
	tenantB dbscope.Tenant
}

func newRLSFixture(t *testing.T, mode string) *rlsFixture {
	t.Helper()

	_, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	suffix := strings.ToLower(pulid.MustNew("r").String())
	appRole := "rls_app_" + suffix
	systemRole := "rls_sys_" + suffix
	password := randomSecret(t)

	_, err := ProvisionRLSRoles(t.Context(), admin, ProvisionRLSRolesParams{
		App:    RLSLoginRole{User: appRole, Password: password},
		System: RLSLoginRole{User: systemRole, Password: password},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		for _, role := range []string{appRole, systemRole} {
			_, _ = keel.Exec(
				context.Background(),
				admin,
				fmt.Sprintf("DROP ROLE IF EXISTS %q", role),
			)
		}
	})

	parsed, err := url.Parse(dsn)
	require.NoError(t, err)
	port, err := strconv.Atoi(parsed.Port())
	require.NoError(t, err)

	key := make([]byte, 48)
	_, err = rand.Read(key)
	require.NoError(t, err)

	cfg := &config.Config{
		App: appconfig.Config{Name: "trenova-rls-test"},
		Database: databaseconfig.Config{
			Host:     parsed.Hostname(),
			Port:     port,
			Name:     strings.TrimPrefix(parsed.Path, "/"),
			User:     appRole,
			Password: password,
			SSLMode:  "disable",
			RLS: databaseconfig.RLSConfig{
				Mode:       mode,
				ScopeKeyID: "ktest",
				ScopeKey:   base64.StdEncoding.EncodeToString(key),
			},
			System: databaseconfig.Role{User: systemRole, Password: password},
		},
	}

	_, err = ProvisionRLS(t.Context(), admin, cfg)
	require.NoError(t, err)

	f := &rlsFixture{
		admin: admin,
		cfg:   cfg,
		key:   key,
		tenantA: dbscope.Tenant{
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
			UserID:         pulid.MustNew("usr_"),
		},
		tenantB: dbscope.Tenant{
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
			UserID:         pulid.MustNew("usr_"),
		},
	}
	f.seedTenant(t, f.tenantA, "A")
	f.seedTenant(t, f.tenantB, "B")

	return f
}

func randomSecret(t *testing.T) string {
	t.Helper()

	raw := make([]byte, 24)
	_, err := rand.Read(raw)
	require.NoError(t, err)

	return base64.RawURLEncoding.EncodeToString(raw)
}

func (f *rlsFixture) seedTenant(t *testing.T, tenant dbscope.Tenant, label string) {
	t.Helper()

	ctx := t.Context()
	stateID := "us_" + strings.ToLower(label) + tenant.OrganizationID.String()[4:12]
	statements := []struct {
		query string
		args  []any
	}{
		{
			"INSERT INTO us_states (id, name, abbreviation) VALUES ($1, $2, $3)",
			[]any{stateID, "State " + label, label + "X"},
		},
		{
			"INSERT INTO business_units (id, name, code) VALUES ($1, $2, $3)",
			[]any{
				tenant.BusinessUnitID,
				"BU " + label,
				"BU" + label + tenant.BusinessUnitID.String()[3:9],
			},
		},
		{
			`INSERT INTO organizations (id, state_id, business_unit_id, name, scac_code, dot_number, bucket_name, address_line1, city, postal_code)
			VALUES ($1, $2, $3, $4, $5, $6, $7, 'Line 1', 'City', '75001')`,
			[]any{
				tenant.OrganizationID,
				stateID,
				tenant.BusinessUnitID,
				"Org " + label,
				"SC" + label + "X",
				"10" + label,
				"bucket-" + strings.ToLower(label),
			},
		},
		{
			`INSERT INTO users (id, business_unit_id, current_organization_id, name, username, password, email_address, timezone)
			VALUES ($1, $2, $3, $4, $5, 'x', $6, 'UTC')`,
			[]any{
				tenant.UserID,
				tenant.BusinessUnitID,
				tenant.OrganizationID,
				"User " + label,
				"u" + strings.ToLower(tenant.UserID.String()[len(tenant.UserID.String())-12:]),
				tenant.UserID.String() + "@example.com",
			},
		},
		{
			"INSERT INTO user_organization_memberships (id, user_id, business_unit_id, organization_id) VALUES ($1, $2, $3, $4)",
			[]any{
				pulid.MustNew("uom_"),
				tenant.UserID,
				tenant.BusinessUnitID,
				tenant.OrganizationID,
			},
		},
		{
			"INSERT INTO fleet_codes (id, code, organization_id, business_unit_id, manager_id) VALUES ($1, $2, $3, $4, $5)",
			[]any{
				pulid.MustNew("fc_"),
				"FC" + label,
				tenant.OrganizationID,
				tenant.BusinessUnitID,
				tenant.UserID,
			},
		},
	}

	for _, stmt := range statements {
		_, err := keel.Exec(ctx, f.admin, stmt.query, stmt.args...)
		require.NoError(t, err, stmt.query)
	}
}

func (f *rlsFixture) connect(t *testing.T) *Connection {
	t.Helper()

	conn, err := newConnection(ConnectionParams{
		Lifecycle: fxtest.NewLifecycle(t),
		Config:    f.cfg,
		Logger:    zaptest.NewLogger(t),
		Observers: shapetest.Observers(t),
	}, oltpSettings(f.cfg))
	require.NoError(t, err)
	require.NoError(t, conn.connect(t.Context()))
	t.Cleanup(func() { _ = conn.shutdown(context.Background()) })

	return conn
}

func queryStrings(ctx context.Context, db *keel.DB, sql string, args ...any) ([]string, error) {
	return keel.QueryAll(ctx, db, pgx.RowTo[string], sql, args...)
}

func fleetCodes(ctx context.Context, conn *Connection) ([]string, error) {
	var codes []string
	err := conn.Transact(
		ctx,
		kernelports.TxOptions{ReadOnly: true},
		func(ctx context.Context) error {
			var err error
			codes, err = queryStrings(
				ctx,
				conn.Keel(ctx),
				"SELECT code FROM fleet_codes ORDER BY code",
			)
			return err
		},
	)

	return codes, err
}

func TestRLS_TenantTransactionSeesOnlyItsOwnRows(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)
	conn := f.connect(t)

	codes, err := fleetCodes(dbscope.WithTenant(t.Context(), f.tenantA), conn)
	require.NoError(t, err)
	assert.Equal(t, []string{"FCA"}, codes)

	codes, err = fleetCodes(dbscope.WithTenant(t.Context(), f.tenantB), conn)
	require.NoError(t, err)
	assert.Equal(t, []string{"FCB"}, codes)
}

func TestRLS_EnforceScopesAStatementOutsideATransactionAndRefusesAnUnscopedOne(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)
	conn := f.connect(t)
	ctx := dbscope.WithTenant(t.Context(), f.tenantA)

	count, err := keel.QueryValue[int64](ctx, conn.Keel(ctx), "SELECT count(*) FROM fleet_codes")
	require.NoError(t, err)
	assert.EqualValues(t, 1, count, "the implicit transaction carries the tenant's scope")

	_, err = keel.QueryValue[int64](
		t.Context(),
		conn.Keel(t.Context()),
		"SELECT count(*) FROM fleet_codes",
	)
	require.ErrorIs(t, err, ErrNoTenantScope)

	_, err = fleetCodes(t.Context(), conn)
	require.ErrorIs(t, err, ErrNoTenantScope)
}

func TestRLS_WritesIntoAnotherTenantAreRejected(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)
	conn := f.connect(t)
	ctx := dbscope.WithTenant(t.Context(), f.tenantA)

	err := conn.Transact(ctx, kernelports.TxOptions{}, func(ctx context.Context) error {
		_, err := keel.Exec(
			ctx,
			conn.Keel(ctx),
			"INSERT INTO fleet_codes (id, code, organization_id, business_unit_id, manager_id) VALUES ($1, 'X', $2, $3, $4)",
			pulid.MustNew("fc_").String(),
			f.tenantB.OrganizationID.String(),
			f.tenantB.BusinessUnitID.String(),
			f.tenantB.UserID.String(),
		)
		return err
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "row-level security")

	var affected int64
	err = conn.Transact(ctx, kernelports.TxOptions{}, func(ctx context.Context) error {
		var err error
		affected, err = keel.Exec(ctx, conn.Keel(ctx),
			"UPDATE fleet_codes SET code = 'STOLEN' WHERE organization_id = $1",
			f.tenantB.OrganizationID.String(),
		)
		return err
	})
	require.NoError(t, err)
	assert.Zero(t, affected)
}

const setScopeForTestSQL = "SELECT set_config('trenova.scope', $1, true)"

func TestRLS_ForgedOrReplayedScopeIsRefused(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)
	conn := f.connect(t)
	ctx := dbscope.WithTenant(t.Context(), f.tenantA)

	otherKey := make([]byte, 48)
	_, err := rand.Read(otherKey)
	require.NoError(t, err)
	forger, err := newScopeSigner("ktest", otherKey, f.cfg.Database.RLS.GetScopeTTL())
	require.NoError(t, err)
	forged, err := forger.token(f.tenantB)
	require.NoError(t, err)

	readWithScope := func(token string) error {
		return conn.Transact(ctx, kernelports.TxOptions{}, func(ctx context.Context) error {
			if _, err := keel.Exec(ctx, conn.Keel(ctx), setScopeForTestSQL, token); err != nil {
				return err
			}
			_, err := queryStrings(ctx, conn.Keel(ctx), "SELECT code FROM fleet_codes")
			return err
		})
	}

	err = readWithScope(forged)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "signature does not verify")

	expiredSigner, err := newScopeSigner("ktest", f.key, f.cfg.Database.RLS.GetScopeTTL())
	require.NoError(t, err)
	expiredSigner.ttl = -f.cfg.Database.RLS.GetScopeTTL()
	expired, err := expiredSigner.token(f.tenantB)
	require.NoError(t, err)

	err = readWithScope(expired)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestRLS_SystemScopeUsesTheBypassPool(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)
	conn := f.connect(t)
	require.NotNil(t, conn.systemKeel)

	ctx := dbscope.WithSystem(t.Context(), "integration test sweep")

	codes, err := queryStrings(ctx, conn.Keel(ctx), "SELECT code FROM fleet_codes ORDER BY code")
	require.NoError(t, err)
	assert.Equal(t, []string{"FCA", "FCB"}, codes)

	codes = nil
	err = conn.Transact(
		ctx,
		kernelports.TxOptions{ReadOnly: true},
		func(ctx context.Context) error {
			var err error
			codes, err = queryStrings(
				ctx,
				conn.Keel(ctx),
				"SELECT code FROM fleet_codes ORDER BY code",
			)
			return err
		},
	)
	require.NoError(t, err)
	assert.Equal(t, []string{"FCA", "FCB"}, codes)

	_, err = keel.QueryValue[int](
		dbscope.WithTenant(t.Context(), f.tenantA),
		conn.systemKeel,
		"SELECT 1",
	)
	require.ErrorIs(t, err, ErrSystemScopeRequired)
}

func TestRLS_ChangingScopeInsideATransactionOpensANewOne(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)
	conn := f.connect(t)

	err := conn.Transact(
		dbscope.WithTenant(t.Context(), f.tenantA),
		kernelports.TxOptions{},
		func(ctx context.Context) error {
			codes, err := fleetCodes(dbscope.WithTenant(ctx, f.tenantB), conn)
			if err != nil {
				return err
			}
			assert.Equal(t, []string{"FCB"}, codes)

			outer, err := queryStrings(ctx, conn.Keel(ctx), "SELECT code FROM fleet_codes")
			if err != nil {
				return err
			}
			assert.Equal(t, []string{"FCA"}, outer)

			return nil
		},
	)
	require.NoError(t, err)
}

func TestRLS_MembershipPoliciesForUsersAndOrganizations(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)
	conn := f.connect(t)
	ctx := dbscope.WithTenant(t.Context(), f.tenantA)

	var users, orgs []string
	err := conn.Transact(
		ctx,
		kernelports.TxOptions{ReadOnly: true},
		func(ctx context.Context) error {
			var err error
			if users, err = queryStrings(ctx, conn.Keel(ctx), "SELECT id FROM users"); err != nil {
				return err
			}
			orgs, err = queryStrings(ctx, conn.Keel(ctx), "SELECT id FROM organizations")
			return err
		},
	)
	require.NoError(t, err)
	assert.Equal(t, []string{f.tenantA.UserID.String()}, users)
	assert.Equal(t, []string{f.tenantA.OrganizationID.String()}, orgs)
}

func TestRLS_StartupRefusesAnApplicationRoleThatCanBypass(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)

	_, err := keel.Exec(
		t.Context(),
		f.admin,
		fmt.Sprintf("GRANT trenova_rls_bypass TO %q", f.cfg.Database.User),
	)
	require.NoError(t, err)

	conn, err := newConnection(ConnectionParams{
		Lifecycle: fxtest.NewLifecycle(t),
		Config:    f.cfg,
		Logger:    zaptest.NewLogger(t),
		Observers: shapetest.Observers(t),
	}, oltpSettings(f.cfg))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.shutdown(context.Background()) })

	require.ErrorIs(t, conn.connect(t.Context()), ErrRLSRoleCannotBypass)
}

func TestRLS_ObserveModeAllowsButReportsUnscopedAccess(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeObserve)
	conn := f.connect(t)
	ctx := t.Context()

	count, err := keel.QueryValue[int64](ctx, conn.Keel(ctx), "SELECT count(*) FROM us_states")
	require.NoError(t, err)
	assert.Positive(t, count)

	_, err = keel.QueryValue[int64](ctx, conn.Keel(ctx), "SELECT count(*) FROM fleet_codes")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no tenant scope")
}

func readScopeSettings(
	ctx context.Context,
	conn *Connection,
) (scope string, team []string, err error) {
	err = conn.Transact(
		ctx,
		kernelports.TxOptions{ReadOnly: true},
		func(ctx context.Context) error {
			return keel.QueryRows(ctx, conn.Keel(ctx), func(row pgx.CollectableRow) error {
				var value *string
				if scanErr := row.Scan(&value, &team); scanErr != nil {
					return scanErr
				}
				if value != nil {
					scope = *value
				}
				return nil
			}, "SELECT trenova_rls.read_scope('worker'), trenova_rls.read_scope_ids('worker')")
		},
	)

	return scope, team, err
}

func TestRLS_ReadScopeTravelsWithTheTenantScope(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)
	conn := f.connect(t)

	manager := pulid.MustNew("usr_")
	ctx := readscope.With(
		dbscope.WithTenant(t.Context(), f.tenantA),
		func(context.Context) (readscope.Grants, error) {
			return readscope.Grants{
				"worker": {Scope: readscope.ScopeTeam, Team: []pulid.ID{manager}},
			}, nil
		},
	)

	scope, team, err := readScopeSettings(ctx, conn)
	require.NoError(t, err)
	assert.Equal(t, "team", scope)
	assert.Equal(t, []string{manager.String()}, team)

	scope, team, err = readScopeSettings(dbscope.WithTenant(t.Context(), f.tenantA), conn)
	require.NoError(t, err)
	assert.Empty(t, scope, "a transaction without a read scope reads its whole tenant")
	assert.Empty(t, team)
}

func TestRLS_ReadScopePoliciesAreRestrictive(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)

	policies, err := queryStrings(t.Context(), f.admin, `
		SELECT c.relname || ':' || p.polcmd::text || ':' || p.polpermissive::text
		FROM pg_policy p
		JOIN pg_class c ON c.oid = p.polrelid
		WHERE p.polname = 'read_scope'
		ORDER BY c.relname`)
	require.NoError(t, err)

	assert.Equal(t, []string{
		"invoices:r:false",
		"orders:r:false",
		"shipments:r:false",
		"workers:r:false",
	}, policies)
}

type readScopePolicy struct {
	Table string `db:"relname"`
	Qual  string `db:"qual"`
}

// A kind the principal cannot read is left out of the read scope, so no
// read_scope policy narrows by anything but "own" or "team": a row of a kind
// the person cannot read stays visible to the record that links to it, and
// the application shows its reference view.
func TestRLS_ReadScopePoliciesNarrowOnlyByOwnOrTeam(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)

	policies, err := keel.QueryAll(t.Context(), f.admin, pgx.RowToStructByName[readScopePolicy], `
		SELECT c.relname, pg_get_expr(p.polqual, p.polrelid) AS qual
		FROM pg_policy p
		JOIN pg_class c ON c.oid = p.polrelid
		WHERE p.polname = 'read_scope'
		ORDER BY c.relname`)
	require.NoError(t, err)

	require.Len(t, policies, 4)
	for _, policy := range policies {
		assert.NotContains(t, policy.Qual, "'none'", policy.Table)
		assert.Regexp(t, `'(own|team)'`, policy.Qual, policy.Table)
	}
}
