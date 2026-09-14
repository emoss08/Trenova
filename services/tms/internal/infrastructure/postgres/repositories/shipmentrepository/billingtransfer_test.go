package shipmentrepository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const billingTransferCandidateSQL = `sp\.status IN \('Completed', 'ReadyToInvoice'\)\) AND \(COALESCE\(sp\.billing_transfer_status, ''\) IN \('', 'SentBackToOps'\)`

func TestApplyShipmentOptionFilters_BillingTransferEligibleKeepsQueuedShipmentsOut(t *testing.T) {
	t.Parallel()

	repo, _ := newCancelTestRepository(t)
	dba := repo.db.DB()

	q := applyShipmentOptionFilters(
		dba.NewSelect().Model((*shipment.Shipment)(nil)),
		dba,
		repositories.ShipmentOptions{BillingTransferEligible: true},
	)

	assert.Regexp(t, billingTransferCandidateSQL, q.String())
}

func TestApplyShipmentOptionFilters_BillingTransferEligibleOffLeavesQueryUntouched(t *testing.T) {
	t.Parallel()

	repo, _ := newCancelTestRepository(t)
	dba := repo.db.DB()

	base := dba.NewSelect().Model((*shipment.Shipment)(nil))
	filtered := applyShipmentOptionFilters(
		dba.NewSelect().Model((*shipment.Shipment)(nil)),
		dba,
		repositories.ShipmentOptions{},
	)

	assert.Equal(t, base.String(), filtered.String())
}

func TestListBillingTransferCandidateIDs_ReturnsOldestFirstUpToTheLimit(t *testing.T) {
	t.Parallel()

	repo, mock := newShipmentListTestRepository(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	first := pulid.MustNew("shp_")
	second := pulid.MustNew("shp_")

	mock.ExpectQuery(`SELECT count\(\*\) FROM "shipments" AS "sp".*` + billingTransferCandidateSQL + `.*sp\.status = 'Completed'`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(7))
	mock.ExpectQuery(`SELECT "sp"\."id" FROM "shipments" AS "sp".*` + billingTransferCandidateSQL + `.*sp\.status = 'Completed'.*ORDER BY "sp"\."created_at" ASC, "sp"\."id" ASC LIMIT 2`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(first).AddRow(second))

	result, err := repo.ListBillingTransferCandidateIDs(
		t.Context(),
		&repositories.ListBillingTransferCandidateIDsRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
			},
			Status: shipment.StatusCompleted,
			Limit:  2,
		},
	)

	require.NoError(t, err)
	assert.Equal(t, 7, result.TotalCount)
	assert.Equal(t, []pulid.ID{first, second}, result.IDs)
}

func TestListBillingTransferCandidateIDs_SkipsTheIDQueryWhenNothingMatches(t *testing.T) {
	t.Parallel()

	repo, mock := newShipmentListTestRepository(t)

	mock.ExpectQuery(`SELECT count\(\*\) FROM "shipments" AS "sp".*` + billingTransferCandidateSQL).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	result, err := repo.ListBillingTransferCandidateIDs(
		t.Context(),
		&repositories.ListBillingTransferCandidateIDsRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: pagination.TenantInfo{
					OrgID: pulid.MustNew("org_"),
					BuID:  pulid.MustNew("bu_"),
				},
			},
			Limit: 5000,
		},
	)

	require.NoError(t, err)
	assert.Zero(t, result.TotalCount)
	assert.NotNil(t, result.IDs)
	assert.Empty(t, result.IDs)
}

func TestStandardShipmentFilter_IncludeCustomerJoinsOnlyTheCustomer(t *testing.T) {
	t.Parallel()

	repo, _ := newCancelTestRepository(t)
	dba := repo.db.DB()

	sql := standardShipmentFilter(
		dba.NewSelect().Model((*shipment.Shipment)(nil)),
		repositories.ShipmentOptions{IncludeCustomer: true},
	).String()

	assert.Contains(t, sql, `LEFT JOIN "customers" AS "customer"`)
	assert.NotContains(t, sql, `"service_types"`)
	assert.NotContains(t, sql, `"formula_templates"`)
}

func TestCountShipmentListQuery_NeverJoinsTheCustomer(t *testing.T) {
	t.Parallel()

	repo, _ := newCancelTestRepository(t)
	dba := repo.db.DB()

	sql := countShipmentListQuery(
		dba.NewSelect().Model((*shipment.Shipment)(nil)),
		dba,
		&repositories.ListShipmentsRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: pagination.TenantInfo{
					OrgID: pulid.MustNew("org_"),
					BuID:  pulid.MustNew("bu_"),
				},
			},
			ShipmentOptions: repositories.ShipmentOptions{
				IncludeCustomer:         true,
				BillingTransferEligible: true,
			},
		},
	).String()

	assert.NotContains(t, sql, `"customers"`)
	assert.Regexp(t, billingTransferCandidateSQL, sql)
}
