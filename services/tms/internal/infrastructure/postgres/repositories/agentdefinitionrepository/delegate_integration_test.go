//go:build integration

package agentdefinitionrepository

import (
	"testing"

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

/*
Deleting an agent another agent could ask failed.

The update that takes the deleted agent off every allowlist wrote its empty
array as '{}', and the column helper replaced that "{}" with the column name,
so Postgres was handed the array literal "delegate_ids".
*/
func TestDelete_RemovesTheAgentFromEveryAllowlist(t *testing.T) {
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

	var rows []tierDefinitionRow
	require.NoError(t, db.NewSelect().
		Table("agent_definitions").
		Column("id", "organization_id", "business_unit_id").
		Where("system_key IS NULL").
		OrderExpr("id").
		Limit(2).
		Scan(ctx, &rows))
	require.Len(t, rows, 2)
	asker, asked := rows[0], rows[1]
	require.Equal(t, asker.OrganizationID, asked.OrganizationID)

	_, err = db.NewUpdate().
		Table("agent_definitions").
		Set("delegate_ids = ARRAY[?]::text[]", asked.ID.String()).
		Where("id = ?", asker.ID).
		Exec(ctx)
	require.NoError(t, err)

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	tenant := pagination.TenantInfo{OrgID: asked.OrganizationID, BuID: asked.BusinessUnitID}

	require.NoError(t, repo.Delete(ctx, repositories.DeleteAgentDefinitionRequest{
		ID:         asked.ID,
		TenantInfo: tenant,
	}))

	definition, err := repo.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         asker.ID,
		TenantInfo: tenant,
	})
	require.NoError(t, err)
	require.NotContains(t, definition.DelegateIDs, pulid.ID(asked.ID))
	require.Empty(t, definition.DelegateIDs, "the last entry leaves no allowlist")
}
