package organizationrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement/driversettlementdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/pkg/pagination"
)

var activeAssignmentStatuses = []shipment.AssignmentStatus{
	shipment.AssignmentStatusNew,
	shipment.AssignmentStatusInProgress,
}

var unpaidDriverSettlementStatuses = []driversettlement.Status{
	driversettlement.StatusDraft,
	driversettlement.StatusPendingApproval,
	driversettlement.StatusApproved,
	driversettlement.StatusPosted,
}

func (r *repository) CountAssetDependencies(
	ctx context.Context,
	_ pagination.TenantInfo,
) (*tenantports.AssetDependencyCounts, error) {
	counts := new(tenantports.AssetDependencyCounts)

	if err := r.countDependencies(ctx, assetDependencyCounts(counts)); err != nil {
		return nil, err
	}

	return counts, nil
}

func assetDependencyCounts(counts *tenantports.AssetDependencyCounts) []dependencyCount {
	return []dependencyCount{
		{
			target: &counts.ActiveDriverAssignments,
			count: func(total *int64) keel.BatchItem {
				a := shipmentdb.Assignments
				return keel.CountInto(a.Where(a.Status.In(activeAssignmentStatuses...)), total)
			},
		},
		{
			target: &counts.UnpaidDriverSettlements,
			count: func(total *int64) keel.BatchItem {
				s := driversettlementdb.Settlements
				return keel.CountInto(s.Where(s.Status.In(unpaidDriverSettlementStatuses...)), total)
			},
		},
		{
			target: &counts.LinkedPortalUsers,
			count: func(total *int64) keel.BatchItem {
				w := workerdb.Workers
				return keel.CountInto(w.Where(w.UserID.IsNotNull()), total)
			},
		},
	}
}
