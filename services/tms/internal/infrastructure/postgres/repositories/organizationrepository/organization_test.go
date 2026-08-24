package organizationrepository

import (
	"database/sql"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

// TestUpdateWritesDisabledCapabilityFlags guards the OmitZero interaction: a
// false boolean is a zero value, so without the explicit Value clauses the
// capability flags could never be turned off — the UPDATE would silently drop
// them and the organization would keep the old setting.
func TestUpdateWritesDisabledCapabilityFlags(t *testing.T) {
	t.Parallel()

	db := bun.NewDB(new(sql.DB), pgdialect.New())

	org := &tenant.Organization{
		Name:                   "Acme Logistics",
		BrokerageEnabled:       false,
		AssetOperationsEnabled: false,
	}

	query, err := db.NewUpdate().
		Model(org).
		WherePK().
		Where("version = ?", 1).
		OmitZero().
		Value(buncolgen.OrganizationColumns.BrokerageEnabled.String(), "?", org.BrokerageEnabled).
		Value(
			buncolgen.OrganizationColumns.AssetOperationsEnabled.String(),
			"?",
			org.AssetOperationsEnabled,
		).
		AppendQuery(db.QueryGen(), nil)
	require.NoError(t, err)

	assert.Contains(t, string(query), `"brokerage_enabled" = FALSE`)
	assert.Contains(t, string(query), `"asset_operations_enabled" = FALSE`)
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

	allAssignmentStatuses := []shipment.AssignmentStatus{
		shipment.AssignmentStatusNew,
		shipment.AssignmentStatusInProgress,
		shipment.AssignmentStatusCompleted,
		shipment.AssignmentStatusCanceled,
	}
	blockingAssignmentStatuses := map[shipment.AssignmentStatus]bool{
		shipment.AssignmentStatusNew:        true,
		shipment.AssignmentStatusInProgress: true,
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

func TestAssetDependencyCountsScopeEveryQueryToTheTenant(t *testing.T) {
	t.Parallel()

	db := bun.NewDB(new(sql.DB), pgdialect.New())
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	counts := new(repositories.AssetDependencyCounts)

	deps := assetDependencyCounts(tenantInfo, counts)
	require.Len(t, deps, 3)

	for _, dep := range deps {
		query, err := db.NewSelect().
			Model(dep.model).
			WhereGroup(" AND ", dep.scope).
			AppendQuery(db.QueryGen(), nil)
		require.NoErrorf(t, err, "build %s query", dep.label)

		rendered := string(query)
		assert.Containsf(t, rendered, ".organization_id = '"+tenantInfo.OrgID.String()+"'",
			"%s is scoped to the organization", dep.label)
		assert.Containsf(t, rendered, ".business_unit_id = '"+tenantInfo.BuID.String()+"'",
			"%s is scoped to the business unit", dep.label)
	}
}

func TestAssetDependencyCountsNarrowToBlockingRows(t *testing.T) {
	t.Parallel()

	db := bun.NewDB(new(sql.DB), pgdialect.New())
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	counts := new(repositories.AssetDependencyCounts)

	rendered := make(map[string]string, 3)
	targets := make(map[string]*int, 3)
	for _, dep := range assetDependencyCounts(tenantInfo, counts) {
		query, err := db.NewSelect().
			Model(dep.model).
			WhereGroup(" AND ", dep.scope).
			AppendQuery(db.QueryGen(), nil)
		require.NoErrorf(t, err, "build %s query", dep.label)

		rendered[dep.label] = string(query)
		targets[dep.label] = dep.target
	}

	assignments := rendered["active driver assignments"]
	assert.Contains(t, assignments, "status IN ('New', 'InProgress')")
	assert.NotContains(t, assignments, "Completed")
	assert.NotContains(t, assignments, "Canceled")

	settlements := rendered["unpaid driver settlements"]
	assert.Contains(t, settlements, "status IN ('Draft', 'PendingApproval', 'Approved', 'Posted')")
	assert.NotContains(t, settlements, "Paid")
	assert.NotContains(t, settlements, "Voided")

	workers := rendered["linked driver portal users"]
	assert.Contains(t, workers, "user_id IS NOT NULL")

	assert.Same(t, &counts.ActiveDriverAssignments, targets["active driver assignments"])
	assert.Same(t, &counts.UnpaidDriverSettlements, targets["unpaid driver settlements"])
	assert.Same(t, &counts.LinkedPortalUsers, targets["linked driver portal users"])
}
