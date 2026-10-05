//go:build integration

package supportaccessrepository_test

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"io/fs"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/cloud/supportaccess/migrations"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

const scopeKeyID = "ksupport"

type fixture struct {
	admin   *bun.DB
	app     *bun.DB
	key     []byte
	tenantA dbscope.Tenant
	tenantB dbscope.Tenant
}

func applySupportAccessMigration(t *testing.T, db *bun.DB) {
	t.Helper()

	err := fs.WalkDir(migrations.FS(), ".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() || !strings.HasSuffix(path, ".up.sql") {
			return walkErr
		}
		raw, readErr := fs.ReadFile(migrations.FS(), path)
		if readErr != nil {
			return readErr
		}
		for _, statement := range strings.Split(string(raw), "--bun:split") {
			if strings.TrimSpace(statement) == "" {
				continue
			}
			if _, execErr := db.ExecContext(t.Context(), statement); execErr != nil {
				return execErr
			}
		}
		return nil
	})
	require.NoError(t, err)
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	_, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	applySupportAccessMigration(t, admin)

	f := &fixture{
		admin: admin,
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

	suffix := strings.ToLower(pulid.MustNew("r").String())
	appRole, systemRole := "sa_app_"+suffix, "sa_sys_"+suffix
	password := randomString(t)
	_, err := postgres.ProvisionRLSRoles(t.Context(), admin, postgres.ProvisionRLSRolesParams{
		App:    postgres.RLSLoginRole{User: appRole, Password: password},
		System: postgres.RLSLoginRole{User: systemRole, Password: password},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		for _, role := range []string{appRole, systemRole} {
			_, _ = admin.ExecContext(context.Background(), `DROP OWNED BY "`+role+`"`)
			_, _ = admin.ExecContext(context.Background(), `DROP ROLE IF EXISTS "`+role+`"`)
		}
	})

	f.key = make([]byte, 48)
	_, err = rand.Read(f.key)
	require.NoError(t, err)

	parsed, err := url.Parse(dsn)
	require.NoError(t, err)
	port, err := strconv.Atoi(parsed.Port())
	require.NoError(t, err)

	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Driver: "postgres",
			Host:   parsed.Hostname(),
			Port:   port,
			Name:   strings.TrimPrefix(parsed.Path, "/"),
			RLS: config.RLSConfig{
				Mode:       config.RLSModeEnforce,
				ScopeKeyID: scopeKeyID,
				ScopeKey:   base64.StdEncoding.EncodeToString(f.key),
			},
		},
	}
	_, err = postgres.ProvisionRLS(t.Context(), admin, cfg)
	require.NoError(t, err)

	parsed.User = url.UserPassword(appRole, password)
	f.app = bun.NewDB(
		sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(parsed.String()))),
		pgdialect.New(),
	)
	t.Cleanup(func() { _ = f.app.Close() })

	return f
}

func randomString(t *testing.T) string {
	t.Helper()
	raw := make([]byte, 24)
	_, err := rand.Read(raw)
	require.NoError(t, err)
	return base64.RawURLEncoding.EncodeToString(raw)
}

func (f *fixture) seedTenant(t *testing.T, tenant dbscope.Tenant, label string) {
	t.Helper()

	stateID := "us_sa" + strings.ToLower(label) + tenant.OrganizationID.String()[4:12]
	statements := []struct {
		query string
		args  []any
	}{
		{
			"INSERT INTO us_states (id, name, abbreviation) VALUES (?, ?, ?)",
			[]any{stateID, "State " + label, label + "Y"},
		},
		{
			"INSERT INTO business_units (id, name, code) VALUES (?, ?, ?)",
			[]any{tenant.BusinessUnitID, "BU " + label, "SA" + label + tenant.BusinessUnitID.String()[3:9]},
		},
		{
			`INSERT INTO organizations (id, state_id, business_unit_id, name, scac_code, dot_number, bucket_name, address_line1, city, postal_code)
			VALUES (?, ?, ?, ?, ?, ?, ?, 'Line 1', 'City', '75001')`,
			[]any{tenant.OrganizationID, stateID, tenant.BusinessUnitID, "Org " + label, "SA" + label + "Y", "20" + label, "bucket-sa-" + strings.ToLower(label)},
		},
		{
			`INSERT INTO users (id, business_unit_id, current_organization_id, name, username, password, email_address, timezone)
			VALUES (?, ?, ?, ?, ?, 'x', ?, 'UTC')`,
			[]any{tenant.UserID, tenant.BusinessUnitID, tenant.OrganizationID, "User " + label, "s" + strings.ToLower(tenant.UserID.String()[len(tenant.UserID.String())-12:]), tenant.UserID.String() + "@example.com"},
		},
	}

	for _, stmt := range statements {
		_, err := f.admin.NewRaw(stmt.query, stmt.args...).Exec(t.Context())
		require.NoError(t, err, stmt.query)
	}
}

func (f *fixture) scope(tenant dbscope.Tenant) string {
	user := "-"
	if tenant.UserID.IsNotNil() {
		user = tenant.UserID.String()
	}
	payload := strings.Join([]string{
		"v1",
		scopeKeyID,
		tenant.OrganizationID.String(),
		tenant.BusinessUnitID.String(),
		user,
		strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10),
	}, ".")

	mac := hmac.New(sha256.New, f.key)
	mac.Write([]byte(payload))

	return payload + "." + hex.EncodeToString(mac.Sum(nil))
}

func (f *fixture) asTenant(
	t *testing.T,
	tenant dbscope.Tenant,
	run func(ctx context.Context, tx bun.Tx) error,
) error {
	t.Helper()

	ctx := t.Context()
	tx, err := f.app.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	if _, err = tx.ExecContext(ctx, "SET LOCAL trenova.scope = '"+f.scope(tenant)+"'"); err != nil {
		return err
	}
	if err = run(ctx, tx); err != nil {
		return err
	}

	return tx.Commit()
}

func (f *fixture) insertGrant(t *testing.T, tenant dbscope.Tenant) pulid.ID {
	t.Helper()

	id := pulid.MustNew("sag_")
	now := time.Now().Unix()
	_, err := f.admin.NewRaw(
		`INSERT INTO support_access_grants (id, organization_id, business_unit_id, granted_by_id, access_mode, starts_at, expires_at)
		VALUES (?, ?, ?, ?, 'read_only', ?, ?)`,
		id, tenant.OrganizationID, tenant.BusinessUnitID, tenant.UserID, now, now+3600,
	).Exec(t.Context())
	require.NoError(t, err)

	return id
}

func countRows(ctx context.Context, tx bun.Tx, table string) (int, error) {
	var count int
	err := tx.NewRaw("SELECT count(*) FROM " + table).Scan(ctx, &count)
	return count, err
}

func TestGrantsAreVisibleOnlyToTheirOrganization(t *testing.T) {
	f := newFixture(t)
	f.insertGrant(t, f.tenantA)

	err := f.asTenant(t, f.tenantA, func(ctx context.Context, tx bun.Tx) error {
		count, err := countRows(ctx, tx, "support_access_grants")
		assert.Equal(t, 1, count)
		return err
	})
	require.NoError(t, err)

	err = f.asTenant(t, f.tenantB, func(ctx context.Context, tx bun.Tx) error {
		count, err := countRows(ctx, tx, "support_access_grants")
		assert.Zero(t, count)
		return err
	})
	require.NoError(t, err)
}

func TestAGrantCannotBeWrittenIntoAnotherOrganization(t *testing.T) {
	f := newFixture(t)

	err := f.asTenant(t, f.tenantB, func(ctx context.Context, tx bun.Tx) error {
		now := time.Now().Unix()
		_, err := tx.NewRaw(
			`INSERT INTO support_access_grants (id, organization_id, business_unit_id, granted_by_id, access_mode, starts_at, expires_at)
			VALUES (?, ?, ?, ?, 'read_write', ?, ?)`,
			pulid.MustNew("sag_"), f.tenantA.OrganizationID, f.tenantA.BusinessUnitID,
			f.tenantB.UserID, now, now+3600,
		).Exec(ctx)
		return err
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "row-level security")
}

func TestSupportSessionsFollowTheirOrganization(t *testing.T) {
	f := newFixture(t)
	grantID := f.insertGrant(t, f.tenantA)
	now := time.Now().Unix()

	_, err := f.admin.NewRaw(
		`INSERT INTO support_sessions (id, organization_id, business_unit_id, grant_id, staff_user_id, staff_name,
			principal_user_id, base_session_id, secret_hash, reason, started_at, expires_at, last_seen_at)
		VALUES (?, ?, ?, ?, ?, 'Staff', ?, 'ses_x', ?, 'Investigating a reported issue', ?, ?, ?)`,
		pulid.MustNew("sps_"), f.tenantA.OrganizationID, f.tenantA.BusinessUnitID, grantID,
		f.tenantB.UserID, f.tenantA.UserID, strings.Repeat("a", 64), now, now+3600, now,
	).Exec(t.Context())
	require.NoError(t, err)

	err = f.asTenant(t, f.tenantA, func(ctx context.Context, tx bun.Tx) error {
		count, countErr := countRows(ctx, tx, "support_sessions")
		assert.Equal(t, 1, count)
		return countErr
	})
	require.NoError(t, err)

	err = f.asTenant(t, f.tenantB, func(ctx context.Context, tx bun.Tx) error {
		count, countErr := countRows(ctx, tx, "support_sessions")
		assert.Zero(t, count, "the staff member's own organization cannot see sessions it opened elsewhere")
		return countErr
	})
	require.NoError(t, err)
}

func TestTheStaffRosterIsReadableOnlyByTheMemberAndNeverWritable(t *testing.T) {
	f := newFixture(t)

	_, err := f.admin.NewRaw(
		`INSERT INTO platform_staff_members (id, user_id, role, added_by) VALUES (?, ?, 'support', 'ops')`,
		pulid.MustNew("psm_"), f.tenantA.UserID,
	).Exec(t.Context())
	require.NoError(t, err)

	err = f.asTenant(t, f.tenantA, func(ctx context.Context, tx bun.Tx) error {
		count, countErr := countRows(ctx, tx, "platform_staff_members")
		assert.Equal(t, 1, count)
		return countErr
	})
	require.NoError(t, err)

	err = f.asTenant(t, f.tenantB, func(ctx context.Context, tx bun.Tx) error {
		count, countErr := countRows(ctx, tx, "platform_staff_members")
		assert.Zero(t, count)
		return countErr
	})
	require.NoError(t, err)

	err = f.asTenant(t, f.tenantB, func(ctx context.Context, tx bun.Tx) error {
		_, insertErr := tx.NewRaw(
			`INSERT INTO platform_staff_members (id, user_id, role, added_by) VALUES (?, ?, 'engineer', 'self')`,
			pulid.MustNew("psm_"), f.tenantB.UserID,
		).Exec(ctx)
		return insertErr
	})
	require.Error(t, err, "a tenant scope must never be able to make itself platform staff")

	err = f.asTenant(t, f.tenantA, func(ctx context.Context, tx bun.Tx) error {
		res, updateErr := tx.NewRaw(`UPDATE platform_staff_members SET role = 'engineer'`).Exec(ctx)
		if updateErr != nil {
			return updateErr
		}
		affected, _ := res.RowsAffected()
		assert.Zero(t, affected)
		return nil
	})
	if err != nil {
		assert.Contains(t, err.Error(), "row-level security")
	}
}
