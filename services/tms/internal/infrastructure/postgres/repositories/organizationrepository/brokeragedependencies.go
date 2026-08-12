package organizationrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/rateconfirmation"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
)

// activeTenderStatuses mirrors the statuses for which [tender.Status.IsTerminal]
// returns false.
var activeTenderStatuses = []tender.Status{
	tender.StatusActive,
	tender.StatusNeedsReview,
}

// unpaidCarrierSettlementStatuses mirrors the statuses for which
// [carriersettlement.Status.IsTerminal] returns false.
var unpaidCarrierSettlementStatuses = []carriersettlement.Status{
	carriersettlement.StatusDraft,
	carriersettlement.StatusPendingApproval,
	carriersettlement.StatusApproved,
	carriersettlement.StatusPosted,
}

// openInvoiceMatchStatuses mirrors the statuses for which
// [carriersettlement.InvoiceMatchStatus.IsOpen] returns true.
var openInvoiceMatchStatuses = []carriersettlement.InvoiceMatchStatus{
	carriersettlement.InvoiceMatchStatusSuggested,
	carriersettlement.InvoiceMatchStatusMatched,
	carriersettlement.InvoiceMatchStatusVariance,
}

type dependencyCount struct {
	label  string
	model  any
	target *int
	scope  func(*bun.SelectQuery) *bun.SelectQuery
}

func (r *repository) CountBrokerageDependencies(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*repositories.BrokerageDependencyCounts, error) {
	counts := new(repositories.BrokerageDependencyCounts)

	for _, dep := range brokerageDependencyCounts(tenantInfo, counts) {
		total, err := r.db.DBForContext(ctx).
			NewSelect().
			Model(dep.model).
			WhereGroup(" AND ", dep.scope).
			Count(ctx)
		if err != nil {
			return nil, fmt.Errorf("count %s: %w", dep.label, err)
		}

		*dep.target = total
	}

	return counts, nil
}

func brokerageDependencyCounts(
	tenantInfo pagination.TenantInfo,
	counts *repositories.BrokerageDependencyCounts,
) []dependencyCount {
	return []dependencyCount{
		{
			label:  "active tenders",
			model:  (*tender.Tender)(nil),
			target: &counts.ActiveTenders,
			scope: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.TenderScopeTenant(sq, tenantInfo).
					Where(buncolgen.TenderColumns.Status.In(), bun.List(activeTenderStatuses))
			},
		},
		{
			label:  "active rate confirmations",
			model:  (*rateconfirmation.RateConfirmation)(nil),
			target: &counts.ActiveRateConfirmations,
			scope: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.RateConfirmationScopeTenant(sq, tenantInfo).
					Where(
						buncolgen.RateConfirmationColumns.Status.NotEq(),
						rateconfirmation.StatusVoided,
					)
			},
		},
		{
			label:  "unpaid carrier settlements",
			model:  (*carriersettlement.CarrierSettlement)(nil),
			target: &counts.UnpaidCarrierSettlements,
			scope: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.CarrierSettlementScopeTenant(sq, tenantInfo).
					Where(
						buncolgen.CarrierSettlementColumns.Status.In(),
						bun.List(unpaidCarrierSettlementStatuses),
					)
			},
		},
		{
			label:  "open carrier invoice matches",
			model:  (*carriersettlement.InvoiceMatch)(nil),
			target: &counts.OpenCarrierInvoiceMatches,
			scope: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.InvoiceMatchScopeTenant(sq, tenantInfo).
					Where(
						buncolgen.InvoiceMatchColumns.Status.In(),
						bun.List(openInvoiceMatchStatuses),
					)
			},
		},
		{
			label:  "active carrier assignments",
			model:  (*shipment.CarrierAssignment)(nil),
			target: &counts.ActiveCarrierAssignments,
			scope: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.CarrierAssignmentScopeTenant(sq, tenantInfo).
					Where(
						buncolgen.CarrierAssignmentColumns.Status.NotEq(),
						shipment.CarrierAssignmentStatusCanceled,
					)
			},
		},
	}
}
