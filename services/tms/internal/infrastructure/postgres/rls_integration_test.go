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

	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap/zaptest"
)

type rlsFixture struct {
	admin   *bun.DB
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
			_, _ = admin.ExecContext(context.Background(), fmt.Sprintf("DROP ROLE IF EXISTS %q", role))
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
		App: config.AppConfig{Name: "trenova-rls-test"},
		Database: config.DatabaseConfig{
			Driver:   "postgres",
			Host:     parsed.Hostname(),
			Port:     port,
			Name:     strings.TrimPrefix(parsed.Path, "/"),
			User:     appRole,
			Password: password,
			SSLMode:  "disable",
			RLS: config.RLSConfig{
				Mode:       mode,
				ScopeKeyID: "ktest",
				ScopeKey:   base64.StdEncoding.EncodeToString(key),
			},
			System: config.DatabaseRole{User: systemRole, Password: password},
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
			"INSERT INTO us_states (id, name, abbreviation) VALUES (?, ?, ?)",
			[]any{stateID, "State " + label, label + "X"},
		},
		{
			"INSERT INTO business_units (id, name, code) VALUES (?, ?, ?)",
			[]any{tenant.BusinessUnitID, "BU " + label, "BU" + label + tenant.BusinessUnitID.String()[3:9]},
		},
		{
			`INSERT INTO organizations (id, state_id, business_unit_id, name, scac_code, dot_number, bucket_name, address_line1, city, postal_code)
			VALUES (?, ?, ?, ?, ?, ?, ?, 'Line 1', 'City', '75001')`,
			[]any{tenant.OrganizationID, stateID, tenant.BusinessUnitID, "Org " + label, "SC" + label + "X", "10" + label, "bucket-" + strings.ToLower(label)},
		},
		{
			`INSERT INTO users (id, business_unit_id, current_organization_id, name, username, password, email_address, timezone)
			VALUES (?, ?, ?, ?, ?, 'x', ?, 'UTC')`,
			[]any{tenant.UserID, tenant.BusinessUnitID, tenant.OrganizationID, "User " + label, "u" + strings.ToLower(tenant.UserID.String()[len(tenant.UserID.String())-12:]), tenant.UserID.String() + "@example.com"},
		},
		{
			"INSERT INTO user_organization_memberships (id, user_id, business_unit_id, organization_id) VALUES (?, ?, ?, ?)",
			[]any{pulid.MustNew("uom_"), tenant.UserID, tenant.BusinessUnitID, tenant.OrganizationID},
		},
		{
			"INSERT INTO fleet_codes (id, code, organization_id, business_unit_id, manager_id) VALUES (?, ?, ?, ?, ?)",
			[]any{pulid.MustNew("fc_"), "FC" + label, tenant.OrganizationID, tenant.BusinessUnitID, tenant.UserID},
		},
	}

	for _, stmt := range statements {
		_, err := f.admin.NewRaw(stmt.query, stmt.args...).Exec(ctx)
		require.NoError(t, err, stmt.query)
	}
}

func (f *rlsFixture) connect(t *testing.T) *Connection {
	t.Helper()

	conn, err := newConnection(ConnectionParams{
		Lifecycle: fxtest.NewLifecycle(t),
		Config:    f.cfg,
		Logger:    zaptest.NewLogger(t),
	}, oltpSettings(f.cfg))
	require.NoError(t, err)
	require.NoError(t, conn.connect(t.Context()))
	t.Cleanup(func() { _ = conn.shutdown(context.Background()) })

	return conn
}

func fleetCodes(ctx context.Context, conn *Connection) ([]string, error) {
	var codes []string
	err := conn.WithTx(ctx, ports.TxOptions{ReadOnly: true}, func(ctx context.Context, tx bun.Tx) error {
		return tx.NewRaw("SELECT code FROM fleet_codes ORDER BY code").Scan(ctx, &codes)
	})

	return codes, err
}

func TestRLS_TenantTransactionSeesOnlyItsOwnRows(t *testing.T) {
	f := newRLSFixture(t, config.RLSModeEnforce)
	conn := f.connect(t)

	codes, err := fleetCodes(dbscope.WithTenant(t.Context(), f.tenantA), conn)
	require.NoError(t, err)
	assert.Equal(t, []string{"FCA"}, codes)

	codes, err = fleetCodes(dbscope.WithTenant(t.Context(), f.tenantB), conn)
	require.NoError(t, err)
	assert.Equal(t, []string{"FCB"}, codes)
}

func TestRLS_EnforceRefusesStatementsOutsideAScopedTransaction(t *testing.T) {
	f := newRLSFixture(t, config.RLSModeEnforce)
	conn := f.connect(t)
	ctx := dbscope.WithTenant(t.Context(), f.tenantA)

	var count int
	err := conn.DBForContext(ctx).NewRaw("SELECT count(*) FROM fleet_codes").Scan(ctx, &count)
	require.ErrorIs(t, err, ErrStatementOutsideTransaction)

	_, err = fleetCodes(t.Context(), conn)
	require.ErrorIs(t, err, ErrNoTenantScope)
}

func TestRLS_WritesIntoAnotherTenantAreRejected(t *testing.T) {
	f := newRLSFixture(t, config.RLSModeEnforce)
	conn := f.connect(t)
	ctx := dbscope.WithTenant(t.Context(), f.tenantA)

	err := conn.WithTx(ctx, ports.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewRaw(
			"INSERT INTO fleet_codes (id, code, organization_id, business_unit_id, manager_id) VALUES (?, 'X', ?, ?, ?)",
			pulid.MustNew("fc_"), f.tenantB.OrganizationID, f.tenantB.BusinessUnitID, f.tenantB.UserID,
		).Exec(ctx)
		return err
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "row-level security")

	var affected int64
	err = conn.WithTx(ctx, ports.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		res, err := tx.NewRaw("UPDATE fleet_codes SET code = 'STOLEN' WHERE organization_id = ?", f.tenantB.OrganizationID).Exec(ctx)
		if err != nil {
			return err
		}
		affected, err = res.RowsAffected()
		return err
	})
	require.NoError(t, err)
	assert.Zero(t, affected)
}

func TestRLS_ForgedOrReplayedScopeIsRefused(t *testing.T) {
	f := newRLSFixture(t, config.RLSModeEnforce)
	conn := f.connect(t)
	ctx := dbscope.WithTenant(t.Context(), f.tenantA)

	otherKey := make([]byte, 48)
	_, err := rand.Read(otherKey)
	require.NoError(t, err)
	forger, err := newScopeSigner("ktest", otherKey, f.cfg.Database.RLS.GetScopeTTL())
	require.NoError(t, err)
	forged, err := forger.token(f.tenantB)
	require.NoError(t, err)

	err = conn.WithTx(ctx, ports.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.ExecContext(ctx, "SET LOCAL trenova.scope = '"+forged+"'"); err != nil {
			return err
		}
		var codes []string
		return tx.NewRaw("SELECT code FROM fleet_codes").Scan(ctx, &codes)
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "signature does not verify")

	expiredSigner, err := newScopeSigner("ktest", f.key, f.cfg.Database.RLS.GetScopeTTL())
	require.NoError(t, err)
	expiredSigner.ttl = -f.cfg.Database.RLS.GetScopeTTL()
	expired, err := expiredSigner.token(f.tenantB)
	require.NoError(t, err)

	err = conn.WithTx(ctx, ports.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.ExecContext(ctx, "SET LOCAL trenova.scope = '"+expired+"'"); err != nil {
			return err
		}
		var codes []string
		return tx.NewRaw("SELECT code FROM fleet_codes").Scan(ctx, &codes)
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestRLS_SystemScopeUsesTheBypassPool(t *testing.T) {
	f := newRLSFixture(t, config.RLSModeEnforce)
	conn := f.connect(t)
	require.NotNil(t, conn.system)

	ctx := dbscope.WithSystem(t.Context(), "integration test sweep")

	var codes []string
	err := conn.DBForContext(ctx).NewRaw("SELECT code FROM fleet_codes ORDER BY code").Scan(ctx, &codes)
	require.NoError(t, err)
	assert.Equal(t, []string{"FCA", "FCB"}, codes)

	codes = nil
	err = conn.WithTx(ctx, ports.TxOptions{ReadOnly: true}, func(ctx context.Context, tx bun.Tx) error {
		return tx.NewRaw("SELECT code FROM fleet_codes ORDER BY code").Scan(ctx, &codes)
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"FCA", "FCB"}, codes)

	err = conn.system.NewRaw("SELECT 1").Scan(dbscope.WithTenant(t.Context(), f.tenantA), new(int))
	require.ErrorIs(t, err, ErrSystemScopeRequired)
}

func TestRLS_ChangingScopeInsideATransactionOpensANewOne(t *testing.T) {
	f := newRLSFixture(t, config.RLSModeEnforce)
	conn := f.connect(t)

	err := conn.WithTx(dbscope.WithTenant(t.Context(), f.tenantA), ports.TxOptions{}, func(ctx context.Context, _ bun.Tx) error {
		codes, err := fleetCodes(dbscope.WithTenant(ctx, f.tenantB), conn)
		if err != nil {
			return err
		}
		assert.Equal(t, []string{"FCB"}, codes)

		var outer []string
		if err = conn.DBForContext(ctx).NewRaw("SELECT code FROM fleet_codes").Scan(ctx, &outer); err != nil {
			return err
		}
		assert.Equal(t, []string{"FCA"}, outer)

		return nil
	})
	require.NoError(t, err)
}

func TestRLS_MembershipPoliciesForUsersAndOrganizations(t *testing.T) {
	f := newRLSFixture(t, config.RLSModeEnforce)
	conn := f.connect(t)
	ctx := dbscope.WithTenant(t.Context(), f.tenantA)

	var users, orgs []string
	err := conn.WithTx(ctx, ports.TxOptions{ReadOnly: true}, func(ctx context.Context, tx bun.Tx) error {
		if err := tx.NewRaw("SELECT id FROM users").Scan(ctx, &users); err != nil {
			return err
		}
		return tx.NewRaw("SELECT id FROM organizations").Scan(ctx, &orgs)
	})
	require.NoError(t, err)
	assert.Equal(t, []string{f.tenantA.UserID.String()}, users)
	assert.Equal(t, []string{f.tenantA.OrganizationID.String()}, orgs)
}

func TestRLS_StartupRefusesAnApplicationRoleThatCanBypass(t *testing.T) {
	f := newRLSFixture(t, config.RLSModeEnforce)

	_, err := f.admin.ExecContext(t.Context(), fmt.Sprintf("GRANT trenova_rls_bypass TO %q", f.cfg.Database.User))
	require.NoError(t, err)

	conn, err := newConnection(ConnectionParams{
		Lifecycle: fxtest.NewLifecycle(t),
		Config:    f.cfg,
		Logger:    zaptest.NewLogger(t),
	}, oltpSettings(f.cfg))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.shutdown(context.Background()) })

	require.ErrorIs(t, conn.connect(t.Context()), ErrRLSRoleCannotBypass)
}

func TestRLS_ObserveModeAllowsButReportsUnscopedAccess(t *testing.T) {
	f := newRLSFixture(t, config.RLSModeObserve)
	conn := f.connect(t)

	var count int
	err := conn.DBForContext(t.Context()).NewRaw("SELECT count(*) FROM us_states").Scan(t.Context(), &count)
	require.NoError(t, err)
	assert.Positive(t, count)

	err = conn.DBForContext(t.Context()).NewRaw("SELECT count(*) FROM fleet_codes").Scan(t.Context(), &count)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no tenant scope")
}
