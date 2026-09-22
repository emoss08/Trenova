//go:build integration

package agentdefinitionrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type tierDefinitionRow struct {
	ID             pulid.ID `bun:"id"`
	OrganizationID pulid.ID `bun:"organization_id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id"`
}

/*
Trust promotion reaches a core tool the agent never listed.

The ledger promotes a tool by merging its tier into tool_tiers, guarded by the
tool being in tool_names. Core tools are held without being listed, so the guard
turned a promotion of remember into a silent no-op.
*/
func TestSetToolTier_PromotesACoreToolTheAgentDoesNotList(t *testing.T) {
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
		Where("NOT ('remember' = ANY(tool_names))").
		Limit(1).
		Scan(ctx, &row))

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{OrgID: row.OrganizationID, BuID: row.BusinessUnitID}

	require.NoError(t, repo.SetToolTier(ctx, repositories.SetAgentDefinitionToolTierRequest{
		ID:         row.ID,
		TenantInfo: tenant,
		ToolName:   "remember",
		Tier:       agent.TierAutoExecute,
	}))

	definition, err := repo.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         row.ID,
		TenantInfo: tenant,
	})
	require.NoError(t, err)
	require.Equal(t, agent.TierAutoExecute, definition.ToolTiers["remember"])

	err = repo.SetToolTier(ctx, repositories.SetAgentDefinitionToolTierRequest{
		ID:         row.ID,
		TenantInfo: tenant,
		ToolName:   "create_dashboard",
		Tier:       agent.TierAutoExecute,
	})
	require.Error(t, err, "a selectable tool the agent does not hold still cannot be promoted")
}
