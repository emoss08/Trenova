package organizationrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
)

// activeAssignmentStatuses are the statuses for which an assignment still binds
// a driver to a shipment move.
var activeAssignmentStatuses = []shipment.AssignmentStatus{
	shipment.AssignmentStatusNew,
	shipment.AssignmentStatusInProgress,
}

// unpaidDriverSettlementStatuses mirrors the statuses for which
// [driversettlement.Status.IsTerminal] returns false.
var unpaidDriverSettlementStatuses = []driversettlement.Status{
	driversettlement.StatusDraft,
	driversettlement.StatusPendingApproval,
	driversettlement.StatusApproved,
	driversettlement.StatusPosted,
}

func (r *repository) CountAssetDependencies(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*repositories.AssetDependencyCounts, error) {
	counts := new(repositories.AssetDependencyCounts)

	if err := r.countDependencies(ctx, assetDependencyCounts(tenantInfo, counts)); err != nil {
		return nil, err
	}

	return counts, nil
}

func assetDependencyCounts(
	tenantInfo pagination.TenantInfo,
	counts *repositories.AssetDependencyCounts,
) []dependencyCount {
	return []dependencyCount{
		{
			label:  "active driver assignments",
			model:  (*shipment.Assignment)(nil),
			target: &counts.ActiveDriverAssignments,
			scope: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.AssignmentScopeTenant(sq, tenantInfo).
					Where(buncolgen.AssignmentColumns.Status.In(), bun.List(activeAssignmentStatuses))
			},
		},
		{
			label:  "unpaid driver settlements",
			model:  (*driversettlement.Settlement)(nil),
			target: &counts.UnpaidDriverSettlements,
			scope: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.SettlementScopeTenant(sq, tenantInfo).
					Where(
						buncolgen.SettlementColumns.Status.In(),
						bun.List(unpaidDriverSettlementStatuses),
					)
			},
		},
		{
			label:  "linked driver portal users",
			model:  (*worker.Worker)(nil),
			target: &counts.LinkedPortalUsers,
			scope: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.WorkerScopeTenant(sq, tenantInfo).
					Where(buncolgen.WorkerColumns.UserID.IsNotNull())
			},
		},
	}
}
