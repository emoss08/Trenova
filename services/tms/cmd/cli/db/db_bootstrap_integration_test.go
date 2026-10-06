//go:build integration

package db

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/instancebootstrap"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/instancebootstrapservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/auditrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/userrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantbootstrap"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	integrationAdminPassword  = "Tr4ck-the-l0ads-north!"
	integrationSystemPassword = "integration-system-password"
)

func productionSeedConfig() *config.Config {
	return &config.Config{
		App:    config.AppConfig{Env: config.EnvProduction},
		System: config.SystemConfig{SystemUserPassword: integrationSystemPassword},
	}
}

func runProductionSeeds(t *testing.T, ctx context.Context, db *bun.DB) {
	t.Helper()

	registry := seeder.NewRegistry()
	seeds.Register(registry)
	engine := seeder.NewEngine(db, registry, productionSeedConfig())

	report, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvProduction})
	require.NoError(t, err)
	require.True(t, report.Success())
}

func integrationBootstrapDeps(t *testing.T, conn *postgres.Connection, out *bytes.Buffer) bootstrapDeps {
	t.Helper()

	logger := zap.NewNop()
	registry := &metrics.Registry{Audit: metrics.NewAudit(nil, logger, false)}
	appCfg := productionSeedConfig()

	return bootstrapDeps{
		conn: conn,
		auditor: func(
			_ context.Context,
			conn *postgres.Connection,
		) (services.SecurityAuditor, func(), error) {
			audits := auditservice.New(auditservice.Params{
				AuditRepository:       auditrepository.New(auditrepository.Params{DB: conn, Logger: logger}),
				AuditBufferRepository: mocks.NewMockAuditBufferRepository(t),
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

	runProductionSeeds(t, ctx, db)

	userCount, err := db.NewSelect().Model((*tenant.User)(nil)).Count(ctx)
	require.NoError(t, err)
	require.Zero(t, userCount, "production seeds must not create any account")

	conn := postgres.WrapDB(db)
	var out bytes.Buffer
	deps := integrationBootstrapDeps(t, conn, &out)

	require.NoError(t, bootstrapInstance(ctx, deps, integrationInputs(), fixedPassword(integrationAdminPassword), false))
	assert.Contains(t, out.String(), `Created organization "Acme Freight"`)

	users := userrepository.New(userrepository.Params{DB: conn, Logger: zap.NewNop()})
	admin, err := users.FindByEmail(ctx, "dana@acme.example")
	require.NoError(t, err)
	require.NoError(t, admin.VerifyCredentials(integrationAdminPassword))
	require.Error(t, admin.VerifyCredentials("not-the-password"))
	assert.False(t, admin.MustChangePassword)
	assert.Equal(t, "Dana Whitfield", admin.Name)

	org := new(tenant.Organization)
	require.NoError(t, db.NewSelect().Model(org).Where("id = ?", admin.CurrentOrganizationID).Scan(ctx))
	assert.Equal(t, "Acme Freight", org.Name)
	assert.Equal(t, "ACMF", org.ScacCode)
	assert.Equal(t, "acme-freight", org.LoginSlug)

	role := new(permission.Role)
	require.NoError(t, db.NewSelect().
		Model(role).
		Where("organization_id = ?", org.ID).
		Where("name = ?", tenantbootstrap.AdministratorRoleName).
		Scan(ctx))
	assigned, err := db.NewSelect().
		Model((*permission.UserRoleAssignment)(nil)).
		Where("user_id = ?", admin.ID).
		Where("role_id = ?", role.ID).
		Where("organization_id = ?", org.ID).
		Exists(ctx)
	require.NoError(t, err)
	assert.True(t, assigned)

	memberships, err := db.NewSelect().
		Model((*tenant.OrganizationMembership)(nil)).
		Where("user_id = ?", admin.ID).
		Where("organization_id = ?", org.ID).
		Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, memberships)

	systemUser, err := users.GetSystemUser(ctx)
	require.NoError(t, err)
	require.NoError(t, systemUser.VerifyCredentials(integrationSystemPassword))
	assert.Equal(t, org.ID, systemUser.CurrentOrganizationID)

	allowlisted, err := db.NewSelect().
		Table("tca_allowlisted_tables").
		Where("organization_id = ?", org.ID).
		Count(ctx)
	require.NoError(t, err)
	assert.Positive(t, allowlisted)

	entries := make([]audit.Entry, 0, 4)
	require.NoError(t, db.NewSelect().
		Model(&entries).
		Where("organization_id = ?", org.ID).
		Where("critical = ?", true).
		Scan(ctx))
	assert.Len(t, entries, 4)

	record := new(instancebootstrap.InstanceBootstrap)
	require.NoError(t, db.NewSelect().Model(record).Scan(ctx))
	assert.Equal(t, org.ID, record.OrganizationID)
	assert.Equal(t, admin.ID, record.AdminUserID)
	assert.Equal(t, "dana@acme.example", record.Inputs.AdminEmail)

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

	finalUsers, err := db.NewSelect().Model((*tenant.User)(nil)).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, finalUsers)

	runProductionSeeds(t, ctx, db)
}

func TestBootstrapRefusesADatabaseThatAlreadyHasUsers(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	runProductionSeeds(t, ctx, db)
	conn := postgres.WrapDB(db)
	var out bytes.Buffer
	deps := integrationBootstrapDeps(t, conn, &out)

	require.NoError(t, bootstrapInstance(ctx, deps, integrationInputs(), fixedPassword(integrationAdminPassword), false))

	_, err := db.NewDelete().Model((*instancebootstrap.InstanceBootstrap)(nil)).Where("TRUE").Exec(ctx)
	require.NoError(t, err)

	err = bootstrapInstance(ctx, deps, integrationInputs(), unusedPassword(t), false)
	require.ErrorIs(t, err, instancebootstrapservice.ErrAlreadyInitialized)

	err = bootstrapInstance(ctx, deps, integrationInputs(), unusedPassword(t), true)
	require.ErrorIs(t, err, instancebootstrapservice.ErrAlreadyInitialized)

	organizations, err := db.NewSelect().Model((*tenant.Organization)(nil)).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, organizations)
}

func TestBootstrapDryRunChangesNothing(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	runProductionSeeds(t, ctx, db)
	conn := postgres.WrapDB(db)
	var out bytes.Buffer
	deps := integrationBootstrapDeps(t, conn, &out)

	require.NoError(t, bootstrapInstance(ctx, deps, integrationInputs(), unusedPassword(t), true))
	assert.Contains(t, out.String(), "Dry run")

	organizations, err := db.NewSelect().Model((*tenant.Organization)(nil)).Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, organizations)
}
