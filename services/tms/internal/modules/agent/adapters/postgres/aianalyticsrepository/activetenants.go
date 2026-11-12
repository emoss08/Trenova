package aianalyticsrepository

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage/aiusagedb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationturn/conversationturndb"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const activeTenantsReason = "list the organizations with AI activity in the refresh window across every tenant"

type activeTenant struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

// ListActiveTenants is every organization with a run, a turn or a model call
// since the moment given, in id order: the three sources a refresh reads,
// each by its index on the time it is read by. The window is short, so the
// set is the tenants using AI now, not every tenant.
func (r *repository) ListActiveTenants(
	ctx context.Context,
	since int64,
	limit int,
) ([]pagination.TenantInfo, error) {
	ctx = dbscope.WithSystem(ctx, activeTenantsReason)

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pagination.TenantInfo, error) {
		runs := agentdb.AgentRuns
		turns := conversationturndb.AssistantTurns
		usage := aiusagedb.AIUsageRecords

		var ran, talked, used []activeTenant
		err := keel.Batch(ctx, r.db.Keel(ctx),
			keel.IntoSlice(runs.Pick(runs.OrganizationID, runs.BusinessUnitID).
				Where(runs.UpdatedAt.Gte(since)).Distinct().Limit(limit).AllTenants(), &ran),
			keel.IntoSlice(turns.Pick(turns.OrganizationID, turns.BusinessUnitID).
				Where(turns.UpdatedAt.Gte(since)).Distinct().Limit(limit).AllTenants(), &talked),
			keel.IntoSlice(usage.Pick(usage.OrganizationID, usage.BusinessUnitID).
				Where(usage.CreatedAt.Gte(since)).Distinct().Limit(limit).AllTenants(), &used),
		)
		if err != nil {
			return nil, fmt.Errorf("list tenants with ai activity: %w", err)
		}

		rows := len(ran) + len(talked) + len(used)
		seen := make(map[pagination.TenantInfo]struct{}, rows)
		tenants := make([]pagination.TenantInfo, 0, rows)
		for _, row := range slices.Concat(ran, talked, used) {
			tenant := pagination.TenantInfo{OrgID: row.OrganizationID, BuID: row.BusinessUnitID}
			if _, ok := seen[tenant]; ok {
				continue
			}
			seen[tenant] = struct{}{}
			tenants = append(tenants, tenant)
		}
		slices.SortFunc(tenants, func(a, b pagination.TenantInfo) int {
			return cmp.Compare(a.OrgID.String(), b.OrgID.String())
		})

		return tenants, nil
	})
}
