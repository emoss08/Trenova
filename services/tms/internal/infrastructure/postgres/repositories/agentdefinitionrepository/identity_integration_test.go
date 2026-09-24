//go:build integration

package agentdefinitionrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

/*
An agent built by hand with no accent could not be saved again.

The update bound each column's value explicitly, so an agent with no accent,
icon or starter template wrote an empty string where the check constraints allow only NULL
or a known value, and every edit failed chk_agent_definitions_accent.
*/
func TestUpdate_SavesAnAgentWithNoAccentIconOrTemplate(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	registry := seeder.NewRegistry()
	seeds.Register(registry)
	engine := seeder.NewEngine(
		db,
		registry,
		&config.Config{System: config.SystemConfig{SystemUserPassword: "test-system-password"}},
	)
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

	var row tierDefinitionRow
	require.NoError(t, db.NewSelect().
		Table("agent_definitions").
		Column("id", "organization_id", "business_unit_id").
		Where("system_key IS NULL").
		Limit(1).
		Scan(ctx, &row))

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{OrgID: row.OrganizationID, BuID: row.BusinessUnitID}

	definition, err := repo.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         row.ID,
		TenantInfo: tenant,
	})
	require.NoError(t, err)

	definition.Accent = ""
	definition.Icon = ""
	definition.Template = ""
	_, err = repo.Update(ctx, definition)
	require.NoError(t, err)

	saved, err := repo.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         row.ID,
		TenantInfo: tenant,
	})
	require.NoError(t, err)
	require.Empty(t, saved.Accent)
	require.Empty(t, saved.Icon)
	require.Empty(t, string(saved.Template))
}

func TestUpdate_SavesBudgetsLimitsAndSimulation(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	registry := seeder.NewRegistry()
	seeds.Register(registry)
	engine := seeder.NewEngine(
		db,
		registry,
		&config.Config{System: config.SystemConfig{SystemUserPassword: "test-system-password"}},
	)
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

	var row tierDefinitionRow
	require.NoError(t, db.NewSelect().
		Table("agent_definitions").
		Column("id", "organization_id", "business_unit_id").
		Limit(1).
		Scan(ctx, &row))

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{OrgID: row.OrganizationID, BuID: row.BusinessUnitID}
	request := repositories.GetAgentDefinitionByIDRequest{ID: row.ID, TenantInfo: tenant}

	definition, err := repo.GetByID(ctx, request)
	require.NoError(t, err)

	budget := decimal.RequireFromString("42.50")
	definition.MonthlyBudgetUSD = &budget
	definition.DailyRunLimit = 17
	definition.ToolDailyLimits = map[string]int{"send_email": 3}
	definition.SimulationMode = !definition.SimulationMode
	simulation := definition.SimulationMode
	_, err = repo.Update(ctx, definition)
	require.NoError(t, err)

	saved, err := repo.GetByID(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, saved.MonthlyBudgetUSD)
	require.True(t, budget.Equal(*saved.MonthlyBudgetUSD))
	require.Equal(t, 17, saved.DailyRunLimit)
	require.Equal(t, map[string]int{"send_email": 3}, saved.ToolDailyLimits)
	require.Equal(t, simulation, saved.SimulationMode)

	saved.MonthlyBudgetUSD = nil
	_, err = repo.Update(ctx, saved)
	require.NoError(t, err)

	cleared, err := repo.GetByID(ctx, request)
	require.NoError(t, err)
	require.Nil(t, cleared.MonthlyBudgetUSD)
}

func TestUpdate_SavesTheDataAccessCeiling(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	registry := seeder.NewRegistry()
	seeds.Register(registry)
	engine := seeder.NewEngine(
		db,
		registry,
		&config.Config{System: config.SystemConfig{SystemUserPassword: "test-system-password"}},
	)
	_, err := engine.Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

	var row tierDefinitionRow
	require.NoError(t, db.NewSelect().
		Table("agent_definitions").
		Column("id", "organization_id", "business_unit_id").
		Where("trigger_mode <> 'Chat'").
		Limit(1).
		Scan(ctx, &row))

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{OrgID: row.OrganizationID, BuID: row.BusinessUnitID}
	request := repositories.GetAgentDefinitionByIDRequest{ID: row.ID, TenantInfo: tenant}

	definition, err := repo.GetByID(ctx, request)
	require.NoError(t, err)
	require.Equal(t, agentdefinition.DataAccessInternal, definition.DataAccessCeiling)

	definition.DataAccessCeiling = agentdefinition.DataAccessRestricted
	_, err = repo.Update(ctx, definition)
	require.NoError(t, err)

	saved, err := repo.GetByID(ctx, request)
	require.NoError(t, err)
	require.Equal(t, agentdefinition.DataAccessRestricted, saved.DataAccessCeiling)

	saved.DataAccessCeiling = agentdefinition.DataAccessInternal
	_, err = repo.Update(ctx, saved)
	require.NoError(t, err)

	lowered, err := repo.GetByID(ctx, request)
	require.NoError(t, err)
	require.Equal(t, agentdefinition.DataAccessInternal, lowered.DataAccessCeiling)
}
