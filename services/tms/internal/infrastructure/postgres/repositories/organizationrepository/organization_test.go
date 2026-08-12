package organizationrepository

import (
	"database/sql"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/pkg/buncolgen"
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
