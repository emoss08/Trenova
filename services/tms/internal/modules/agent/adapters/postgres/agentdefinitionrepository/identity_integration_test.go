//go:build integration

package agentdefinitionrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/seededtest"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
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
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	row := findDefinitions(t, ctx, db, "system_key IS NULL", 1)[0]
	ctx, tenant := row.tenant(ctx)
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	definition, err := repo.GetByID(ctx, agentdefinitionports.GetAgentDefinitionByIDRequest{
		ID:         row.ID,
		TenantInfo: tenant,
	})
	require.NoError(t, err)

	definition.Accent = ""
	definition.Icon = ""
	definition.Template = ""
	_, err = repo.Update(ctx, definition)
	require.NoError(t, err)

	saved, err := repo.GetByID(ctx, agentdefinitionports.GetAgentDefinitionByIDRequest{
		ID:         row.ID,
		TenantInfo: tenant,
	})
	require.NoError(t, err)
	require.Empty(t, saved.Accent)
	require.Empty(t, saved.Icon)
	require.Empty(t, string(saved.Template))
}

func TestUpdate_SavesBudgetsLimitsAndSimulation(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	row := findDefinitions(t, ctx, db, "TRUE", 1)[0]
	ctx, tenant := row.tenant(ctx)
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	request := agentdefinitionports.GetAgentDefinitionByIDRequest{ID: row.ID, TenantInfo: tenant}

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
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	row := findDefinitions(t, ctx, db, "trigger_mode <> 'Chat'", 1)[0]
	ctx, tenant := row.tenant(ctx)
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	request := agentdefinitionports.GetAgentDefinitionByIDRequest{ID: row.ID, TenantInfo: tenant}

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
