package organizationrepository

import (
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeepStoredWhereZeroWritesDisabledCapabilityFlags(t *testing.T) {
	t.Parallel()

	o := tenantdb.Organizations
	stored := &tenant.Organization{
		Name:                   "Acme Logistics",
		City:                   "Dallas",
		BrokerageEnabled:       true,
		AssetOperationsEnabled: true,
		Version:                4,
	}
	org := &tenant.Organization{
		Name:                   "Acme Freight",
		BrokerageEnabled:       false,
		AssetOperationsEnabled: false,
		Version:                4,
	}

	keepStoredWhereZero(org, stored, o.BrokerageEnabled, o.AssetOperationsEnabled)

	assert.False(t, org.BrokerageEnabled)
	assert.False(t, org.AssetOperationsEnabled)
	assert.Equal(t, "Acme Freight", org.Name)
	assert.Equal(t, "Dallas", org.City)
	assert.Equal(t, int64(4), org.Version)
}

// TestBrokerageDependencyStatusSetsMatchDomainPredicates keeps the SQL-side
// status lists in step with the domain predicates they mirror; a predicate that
// changes meaning without the list following it would silently narrow or widen
// the brokerage disable guard.
func TestBrokerageDependencyStatusSetsMatchDomainPredicates(t *testing.T) {
	t.Parallel()

	allTenderStatuses := []tender.Status{
		tender.StatusActive,
		tender.StatusAccepted,
		tender.StatusExhausted,
		tender.StatusCanceled,
		tender.StatusNeedsReview,
	}
	for _, status := range allTenderStatuses {
		assert.Equal(t,
			!status.IsTerminal(),
			slices.Contains(activeTenderStatuses, status),
			"tender status %s", status,
		)
	}

	allSettlementStatuses := []carriersettlement.Status{
		carriersettlement.StatusDraft,
		carriersettlement.StatusPendingApproval,
		carriersettlement.StatusApproved,
		carriersettlement.StatusPosted,
		carriersettlement.StatusPaid,
		carriersettlement.StatusVoided,
	}
	for _, status := range allSettlementStatuses {
		assert.Equal(t,
			!status.IsTerminal(),
			slices.Contains(unpaidCarrierSettlementStatuses, status),
			"carrier settlement status %s", status,
		)
	}

	allInvoiceMatchStatuses := []carriersettlement.InvoiceMatchStatus{
		carriersettlement.InvoiceMatchStatusSuggested,
		carriersettlement.InvoiceMatchStatusMatched,
		carriersettlement.InvoiceMatchStatusVariance,
		carriersettlement.InvoiceMatchStatusResolved,
		carriersettlement.InvoiceMatchStatusRejected,
	}
	for _, status := range allInvoiceMatchStatuses {
		assert.Equal(t,
			status.IsOpen(),
			slices.Contains(openInvoiceMatchStatuses, status),
			"carrier invoice match status %s", status,
		)
	}
}

func TestAssetDependencyStatusSetsMatchDomainPredicates(t *testing.T) {
	t.Parallel()

	allAssignmentStatuses := []shipmentvocab.AssignmentStatus{
		shipmentvocab.AssignmentStatusNew,
		shipmentvocab.AssignmentStatusInProgress,
		shipmentvocab.AssignmentStatusCompleted,
		shipmentvocab.AssignmentStatusCanceled,
	}
	blockingAssignmentStatuses := map[shipmentvocab.AssignmentStatus]bool{
		shipmentvocab.AssignmentStatusNew:        true,
		shipmentvocab.AssignmentStatusInProgress: true,
	}
	for _, status := range allAssignmentStatuses {
		assert.Equal(t,
			blockingAssignmentStatuses[status],
			slices.Contains(activeAssignmentStatuses, status),
			"assignment status %s", status,
		)
	}

	allSettlementStatuses := []driversettlement.Status{
		driversettlement.StatusDraft,
		driversettlement.StatusPendingApproval,
		driversettlement.StatusApproved,
		driversettlement.StatusPosted,
		driversettlement.StatusPaid,
		driversettlement.StatusVoided,
	}
	for _, status := range allSettlementStatuses {
		assert.Equal(t,
			!status.IsTerminal(),
			slices.Contains(unpaidDriverSettlementStatuses, status),
			"driver settlement status %s", status,
		)
	}
}

func TestAssetDependencyCountsFillEveryCounter(t *testing.T) {
	t.Parallel()

	counts := new(tenantports.AssetDependencyCounts)
	deps := assetDependencyCounts(counts)
	require.Len(t, deps, 3)

	assert.Same(t, &counts.ActiveDriverAssignments, deps[0].target)
	assert.Same(t, &counts.UnpaidDriverSettlements, deps[1].target)
	assert.Same(t, &counts.LinkedPortalUsers, deps[2].target)
	for _, dep := range deps {
		require.NotNil(t, dep.count)
	}
}
