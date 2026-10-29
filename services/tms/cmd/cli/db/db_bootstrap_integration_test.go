//go:build integration

package db

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/systemconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantbootstrap"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/userrepository"
	"github.com/emoss08/trenova/internal/modules/identity/app/instancebootstrapservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/instancebootstrap"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/auditrepository"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	integrationAdminPassword  = "Tr4ck-the-l0ads-north!"
	integrationSystemPassword = "integration-system-password"
)

func productionSeedConfig() *config.Config {
	return &config.Config{
		App:    appconfig.Config{Env: appconfig.EnvProduction},
		System: systemconfig.Config{SystemUserPassword: integrationSystemPassword},
	}
}

func integrationBootstrapDeps(
	t *testing.T,
	conn *postgres.Connection,
	out *bytes.Buffer,
) bootstrapDeps {
	t.Helper()

	logger := zap.NewNop()
	registry := &metrics.Registry{Audit: metrics.NewAudit(nil, logger, false)}
	appCfg := productionSeedConfig()

	return bootstrapDeps{
		conn: conn,
		auditor: func(
			_ context.Context,
			conn *postgres.Connection,
		) (permissionports.SecurityAuditor, func(), error) {
			audits := auditservice.New(auditservice.Params{
				AuditRepository: auditrepository.New(
					auditrepository.Params{DB: conn, Logger: logger},
				),
				AuditBufferRepository: auditportsmock.NewMockAuditBufferRepository(t),
				Logger:                logger,
				Config:                appCfg,
				Metrics:               registry,
			})
			return auditservice.NewSecurityAuditor(auditservice.SecurityAuditorParams{
				AuditService: audits,
				Metrics:      registry,
				Logger:       logger,
			}), func() {}, nil
		},
		cfg:    appCfg,
		logger: logger,
		out:    out,
	}
}

func integrationInputs() *instancebootstrap.Inputs {
	return &instancebootstrap.Inputs{
		OrganizationName: "Acme Freight",
		AdminName:        "Dana Whitfield",
		AdminEmail:       "Dana@Acme.example",
		State:            "TX",
		City:             "Dallas",
		PostalCode:       "75201",
		SCAC:             "ACMF",
		DOTNumber:        "1234567",
	}
}

func countRows(t *testing.T, ctx context.Context, db *keel.DB, sql string, args ...any) int64 {
	t.Helper()

	n, err := keel.QueryValue[int64](ctx, db, sql, args...)
	require.NoError(t, err)

	return n
}

func fixedPassword(password string) passwordSource {
	return func() (string, error) { return password, nil }
}

func unusedPassword(t *testing.T) passwordSource {
	return func() (string, error) {
		t.Fatal("the password must not be read when nothing will be created")
		return "", errors.New("unreachable")
	}
}

func TestBootstrapCreatesAnAdministratorWhoCanSignIn(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	require.Zero(t, countRows(t, ctx, db, "SELECT count(*) FROM users"),
		"production seeds must not create any account")

	conn := postgres.WrapDB(db)
	var out bytes.Buffer
	deps := integrationBootstrapDeps(t, conn, &out)

	require.NoError(
		t,
		bootstrapInstance(
			ctx,
			deps,
			integrationInputs(),
			fixedPassword(integrationAdminPassword),
			false,
		),
	)
	assert.Contains(t, out.String(), `Created organization "Acme Freight"`)

	users := userrepository.New(userrepository.Params{DB: conn, Logger: zap.NewNop()})
	admin, err := users.FindByEmail(ctx, "dana@acme.example")
	require.NoError(t, err)
	require.NoError(t, admin.VerifyCredentials(integrationAdminPassword))
	require.Error(t, admin.VerifyCredentials("not-the-password"))
	assert.False(t, admin.MustChangePassword)
	assert.Equal(t, "Dana Whitfield", admin.Name)

	orgID := admin.CurrentOrganizationID
	var org struct{ Name, ScacCode, LoginSlug string }
	require.NoError(t, keel.QueryRows(ctx, db, func(row pgx.CollectableRow) error {
		return row.Scan(&org.Name, &org.ScacCode, &org.LoginSlug)
	}, "SELECT name, scac_code, login_slug FROM organizations WHERE id = $1", orgID.String()))
	assert.Equal(t, "Acme Freight", org.Name)
	assert.Equal(t, "ACMF", org.ScacCode)
	assert.Equal(t, "acme-freight", org.LoginSlug)

	roleID, err := keel.QueryValue[string](ctx, db,
		"SELECT id FROM roles WHERE organization_id = $1 AND name = $2",
		orgID.String(), tenantbootstrap.AdministratorRoleName)
	require.NoError(t, err)
	assert.Equal(t, int64(1), countRows(
		t,
		ctx,
		db,
		"SELECT count(*) FROM user_role_assignments WHERE user_id = $1 AND role_id = $2 AND organization_id = $3",
		admin.ID.String(),
		roleID,
		orgID.String(),
	))

	assert.Equal(t, int64(1), countRows(
		t,
		ctx,
		db,
		"SELECT count(*) FROM user_organization_memberships WHERE user_id = $1 AND organization_id = $2",
		admin.ID.String(),
		orgID.String(),
	))

	systemUser, err := users.GetSystemUser(ctx)
	require.NoError(t, err)
	require.NoError(t, systemUser.VerifyCredentials(integrationSystemPassword))
	assert.Equal(t, orgID, systemUser.CurrentOrganizationID)

	assert.Positive(t, countRows(t, ctx, db,
		"SELECT count(*) FROM tca_allowlisted_tables WHERE organization_id = $1", orgID.String()))
	assert.Equal(t, int64(4), countRows(
		t,
		ctx,
		db,
		"SELECT count(*) FROM audit_entries WHERE organization_id = $1 AND critical",
		orgID.String(),
	))

	var record struct {
		OrganizationID pulid.ID
		AdminUserID    pulid.ID
		AdminEmail     string
	}
	require.NoError(t, keel.QueryRows(ctx, db, func(row pgx.CollectableRow) error {
		return row.Scan(&record.OrganizationID, &record.AdminUserID, &record.AdminEmail)
	}, "SELECT organization_id, admin_user_id, inputs->>'adminEmail' FROM instance_bootstraps"))
	assert.Equal(t, orgID, record.OrganizationID)
	assert.Equal(t, admin.ID, record.AdminUserID)
	assert.Equal(t, "dana@acme.example", record.AdminEmail)

	out.Reset()
	again := integrationInputs()
	again.AdminEmail = "DANA@acme.example"
	again.Timezone = "America/Chicago"
	require.NoError(t, bootstrapInstance(ctx, deps, again, unusedPassword(t), false))
	assert.Contains(t, out.String(), "Already bootstrapped")

	different := integrationInputs()
	different.AdminEmail = "someone-else@acme.example"
	err = bootstrapInstance(ctx, deps, different, unusedPassword(t), false)
	require.ErrorIs(t, err, instancebootstrapservice.ErrInputsDiffer)

	assert.Equal(t, int64(2), countRows(t, ctx, db, "SELECT count(*) FROM users"))
}

func TestBootstrapRefusesADatabaseThatAlreadyHasUsers(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	conn := postgres.WrapDB(db)
	var out bytes.Buffer
	deps := integrationBootstrapDeps(t, conn, &out)

	require.NoError(
		t,
		bootstrapInstance(
			ctx,
			deps,
			integrationInputs(),
			fixedPassword(integrationAdminPassword),
			false,
		),
	)

	_, err := keel.Exec(ctx, db, "DELETE FROM instance_bootstraps")
	require.NoError(t, err)

	err = bootstrapInstance(ctx, deps, integrationInputs(), unusedPassword(t), false)
	require.ErrorIs(t, err, instancebootstrapservice.ErrAlreadyInitialized)

	err = bootstrapInstance(ctx, deps, integrationInputs(), unusedPassword(t), true)
	require.ErrorIs(t, err, instancebootstrapservice.ErrAlreadyInitialized)

	assert.Equal(t, int64(1), countRows(t, ctx, db, "SELECT count(*) FROM organizations"))
}

func TestBootstrapDryRunChangesNothing(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	conn := postgres.WrapDB(db)
	var out bytes.Buffer
	deps := integrationBootstrapDeps(t, conn, &out)

	require.NoError(t, bootstrapInstance(ctx, deps, integrationInputs(), unusedPassword(t), true))
	assert.Contains(t, out.String(), "Dry run")

	assert.Zero(t, countRows(t, ctx, db, "SELECT count(*) FROM organizations"))
}
