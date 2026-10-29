package organizationrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender/tenderdb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateconfirmation"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateconfirmation/rateconfirmationdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement/carriersettlementdb"
	"github.com/emoss08/trenova/pkg/pagination"
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
	_ pagination.TenantInfo,
) (*tenantports.BrokerageDependencyCounts, error) {
	counts := new(tenantports.BrokerageDependencyCounts)

	if err := r.countDependencies(ctx, brokerageDependencyCounts(counts)); err != nil {
		return nil, err
	}

	return counts, nil
}

func brokerageDependencyCounts(counts *tenantports.BrokerageDependencyCounts) []dependencyCount {
	return []dependencyCount{
		{
			target: &counts.ActiveTenders,
			count: func(total *int64) keel.BatchItem {
				t := tenderdb.Tenders
				return keel.CountInto(t.Where(t.Status.In(activeTenderStatuses...)), total)
			},
		},
		{
			target: &counts.ActiveRateConfirmations,
			count: func(total *int64) keel.BatchItem {
				rc := rateconfirmationdb.RateConfirmations
				return keel.CountInto(rc.Where(rc.Status.Ne(rateconfirmation.StatusVoided)), total)
			},
		},
		{
			target: &counts.UnpaidCarrierSettlements,
			count: func(total *int64) keel.BatchItem {
				cs := carriersettlementdb.CarrierSettlements
				return keel.CountInto(
					cs.Where(cs.Status.In(unpaidCarrierSettlementStatuses...)),
					total,
				)
			},
		},
		{
			target: &counts.OpenCarrierInvoiceMatches,
			count: func(total *int64) keel.BatchItem {
				im := carriersettlementdb.InvoiceMatches
				return keel.CountInto(im.Where(im.Status.In(openInvoiceMatchStatuses...)), total)
			},
		},
		{
			target: &counts.ActiveCarrierAssignments,
			count: func(total *int64) keel.BatchItem {
				ca := shipmentdb.CarrierAssignments
				return keel.CountInto(
					ca.Where(ca.Status.Ne(shipmentvocab.CarrierAssignmentStatusCanceled)),
					total,
				)
			},
		},
	}
}
