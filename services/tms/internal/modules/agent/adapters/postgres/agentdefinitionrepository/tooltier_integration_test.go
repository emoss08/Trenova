//go:build integration

package agentdefinitionrepository

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/jackc/pgx/v5"

	"github.com/emoss08/trenova/internal/testutil/seededtest"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type tierDefinitionRow struct {
	ID             pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

func findDefinitions(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	where string,
	limit int,
) []tierDefinitionRow {
	t.Helper()

	rows := make([]tierDefinitionRow, 0, limit)
	err := keel.QueryRows(ctx, db, func(row pgx.CollectableRow) error {
		var r tierDefinitionRow
		if err := row.Scan(&r.ID, &r.OrganizationID, &r.BusinessUnitID); err != nil {
			return err
		}
		rows = append(rows, r)
		return nil
	}, "SELECT id, organization_id, business_unit_id FROM agent_definitions WHERE "+where+
		" ORDER BY id LIMIT $1", limit)
	require.NoError(t, err)
	require.Len(t, rows, limit)

	return rows
}

func (r tierDefinitionRow) tenant(ctx context.Context) (context.Context, pagination.TenantInfo) {
	return dbscope.WithTenant(ctx, dbscope.Tenant{
			OrganizationID: r.OrganizationID,
			BusinessUnitID: r.BusinessUnitID,
		}), pagination.TenantInfo{
			OrgID: r.OrganizationID,
			BuID:  r.BusinessUnitID,
		}
}

/*
Trust promotion reaches a core tool the agent never listed.

The ledger promotes a tool by merging its tier into tool_tiers, guarded by the
tool being in tool_names. Core tools are held without being listed, so the guard
turned a promotion of remember into a silent no-op.
*/
func TestSetToolTier_PromotesACoreToolTheAgentDoesNotList(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	row := findDefinitions(t, ctx, db, "NOT ('agent_create_memory' = ANY(tool_names))", 1)[0]
	ctx, tenant := row.tenant(ctx)
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	require.NoError(t, repo.SetToolTier(ctx, agentdefinitionports.SetAgentDefinitionToolTierRequest{
		ID:         row.ID,
		TenantInfo: tenant,
		ToolName:   "agent_create_memory",
		Tier:       agent.TierAutoExecute,
	}))

	definition, err := repo.GetByID(ctx, agentdefinitionports.GetAgentDefinitionByIDRequest{
		ID:         row.ID,
		TenantInfo: tenant,
	})
	require.NoError(t, err)
	require.Equal(t, agent.TierAutoExecute, definition.ToolTiers["agent_create_memory"])

	err = repo.SetToolTier(ctx, agentdefinitionports.SetAgentDefinitionToolTierRequest{
		ID:         row.ID,
		TenantInfo: tenant,
		ToolName:   "report_create_dashboard",
		Tier:       agent.TierAutoExecute,
	})
	require.Error(t, err, "a selectable tool the agent does not hold still cannot be promoted")
}

// An agent saved with no tiers holds JSON null, and merging a tier into it with
// || made an array that broke reading every agent in the organization.
func TestSetToolTier_MergesIntoAnAgentSavedWithNoTiers(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	row := findDefinitions(t, ctx, db, "TRUE", 1)[0]
	_, err := keel.Exec(ctx, db,
		"UPDATE agent_definitions SET tool_tiers = 'null'::jsonb WHERE id = $1", row.ID.String())
	require.NoError(t, err)

	ctx, tenant := row.tenant(ctx)
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	require.NoError(t, repo.SetToolTier(ctx, agentdefinitionports.SetAgentDefinitionToolTierRequest{
		ID:         row.ID,
		TenantInfo: tenant,
		ToolName:   "agent_create_memory",
		Tier:       agent.TierActWithApproval,
	}))

	definition, err := repo.GetByID(ctx, agentdefinitionports.GetAgentDefinitionByIDRequest{
		ID:         row.ID,
		TenantInfo: tenant,
	})
	require.NoError(t, err)
	require.Equal(t, map[string]agent.AutonomyTier{"agent_create_memory": agent.TierActWithApproval},
		definition.ToolTiers)
}
