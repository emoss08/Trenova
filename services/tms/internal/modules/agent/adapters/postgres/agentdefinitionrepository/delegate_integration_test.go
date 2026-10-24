//go:build integration

package agentdefinitionrepository

import (
	"testing"

	"github.com/emoss08/keel"

	"github.com/emoss08/trenova/internal/testutil/seededtest"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
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
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	rows := findDefinitions(t, ctx, db, "system_key IS NULL", 2)
	asker, asked := rows[0], rows[1]
	require.Equal(t, asker.OrganizationID, asked.OrganizationID)

	_, err := keel.Exec(ctx, db,
		"UPDATE agent_definitions SET delegate_ids = ARRAY[$1]::text[] WHERE id = $2",
		asked.ID.String(), asker.ID.String())
	require.NoError(t, err)

	ctx, tenant := asked.tenant(ctx)
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	require.NoError(t, repo.Delete(ctx, agentdefinitionports.DeleteAgentDefinitionRequest{
		ID:         asked.ID,
		TenantInfo: tenant,
	}))

	definition, err := repo.GetByID(ctx, agentdefinitionports.GetAgentDefinitionByIDRequest{
		ID:         asker.ID,
		TenantInfo: tenant,
	})
	require.NoError(t, err)
	require.NotContains(t, definition.DelegateIDs, pulid.ID(asked.ID))
	require.Empty(t, definition.DelegateIDs, "the last entry leaves no allowlist")
}
