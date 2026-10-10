package organizationrepository

import (
	"context"

	"github.com/emoss08/trenova/pkg/buncolgen/carriersettlementcol"
	"github.com/emoss08/trenova/pkg/buncolgen/rateconfirmationcol"
	"github.com/emoss08/trenova/pkg/buncolgen/shipmentcol"
	"github.com/emoss08/trenova/pkg/buncolgen/tendercol"

	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateconfirmation"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
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

func (r *repository) CountBrokerageDependencies(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*tenantports.BrokerageDependencyCounts, error) {
	counts := new(tenantports.BrokerageDependencyCounts)

	if err := r.countDependencies(ctx, brokerageDependencyCounts(tenantInfo, counts)); err != nil {
		return nil, err
	}

	return counts, nil
}

func brokerageDependencyCounts(
	tenantInfo pagination.TenantInfo,
	counts *tenantports.BrokerageDependencyCounts,
) []dependencyCount {
	return []dependencyCount{
		{
			label:  "active tenders",
			model:  (*tender.Tender)(nil),
			target: &counts.ActiveTenders,
			scope: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return tendercol.TenderScopeTenant(sq, tenantInfo).
					Where(tendercol.TenderColumns.Status.In(), bun.List(activeTenderStatuses))
			},
		},
		{
			label:  "active rate confirmations",
			model:  (*rateconfirmation.RateConfirmation)(nil),
			target: &counts.ActiveRateConfirmations,
			scope: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return rateconfirmationcol.RateConfirmationScopeTenant(sq, tenantInfo).
					Where(
						rateconfirmationcol.RateConfirmationColumns.Status.NotEq(),
						rateconfirmation.StatusVoided,
					)
			},
		},
		{
			label:  "unpaid carrier settlements",
			model:  (*carriersettlement.CarrierSettlement)(nil),
			target: &counts.UnpaidCarrierSettlements,
			scope: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return carriersettlementcol.CarrierSettlementScopeTenant(sq, tenantInfo).
					Where(
						carriersettlementcol.CarrierSettlementColumns.Status.In(),
						bun.List(unpaidCarrierSettlementStatuses),
					)
			},
		},
		{
			label:  "open carrier invoice matches",
			model:  (*carriersettlement.InvoiceMatch)(nil),
			target: &counts.OpenCarrierInvoiceMatches,
			scope: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return carriersettlementcol.InvoiceMatchScopeTenant(sq, tenantInfo).
					Where(
						carriersettlementcol.InvoiceMatchColumns.Status.In(),
						bun.List(openInvoiceMatchStatuses),
					)
			},
		},
		{
			label:  "active carrier assignments",
			model:  (*shipment.CarrierAssignment)(nil),
			target: &counts.ActiveCarrierAssignments,
			scope: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return shipmentcol.CarrierAssignmentScopeTenant(sq, tenantInfo).
					Where(
						shipmentcol.CarrierAssignmentColumns.Status.NotEq(),
						shipment.CarrierAssignmentStatusCanceled,
					)
			},
		},
	}
}
