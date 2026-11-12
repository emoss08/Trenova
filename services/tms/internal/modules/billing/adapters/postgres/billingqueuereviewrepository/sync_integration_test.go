//go:build integration

package billingqueuereviewrepository

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/billingqueueprojection"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingvocab"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type itemStamp struct {
	CheckedAt *int64
	UpdatedAt int64
	Version   int64
}

// Opening an item reconciles what its checks find but writes nothing when
// nothing moved: only a recheck stamps it checked, and a read with no change
// leaves the row as it was.
func TestSyncIssuesWritesOnlyWhatChangedAndStampsOnlyARecheck(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	var orgID, buID, shipmentID, payerID pulid.ID
	require.NoError(t, keel.Pool(db).QueryRow(ctx,
		"SELECT id, business_unit_id FROM organizations ORDER BY id LIMIT 1",
	).Scan(&orgID, &buID))
	require.NoError(t, keel.Pool(db).QueryRow(ctx,
		`SELECT id, COALESCE(bill_to_customer_id, customer_id) FROM shipments
		WHERE organization_id = $1 AND business_unit_id = $2 ORDER BY created_at ASC LIMIT 1`,
		orgID, buID,
	).Scan(&shipmentID, &payerID))
	tenant := pagination.TenantInfo{OrgID: orgID, BuID: buID}
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())

	item := &billingqueue.BillingQueueItem{
		OrganizationID:   orgID,
		BusinessUnitID:   buID,
		ShipmentID:       shipmentID,
		BillToCustomerID: payerID,
		Status:           billingqueue.StatusReadyForReview,
		BillType:         billingvocab.BillTypeInvoice,
		Number:           "BQ-SYNC-1",
	}
	require.NoError(t, billingqueuedb.BillingQueueItems.Create(ctx, db, item))
	require.NoError(t, billingqueueprojection.Refresh(ctx, db, item.ID))
	repo := New(Params{DB: postgres.WrapDB(db), Logger: zap.NewNop()})

	stamp := func() itemStamp {
		t.Helper()
		var s itemStamp
		require.NoError(t, keel.Pool(db).QueryRow(ctx,
			`SELECT checked_at, updated_at, version FROM billing_queue_items WHERE id = $1`, item.ID,
		).Scan(&s.CheckedAt, &s.UpdatedAt, &s.Version))

		return s
	}
	sync := func(checked bool) {
		t.Helper()
		_, err := repo.SyncIssues(ctx, &billingqueueports.SyncBillingQueueIssuesRequest{
			TenantInfo: tenant,
			ItemID:     item.ID,
			Checked:    checked,
		})
		require.NoError(t, err)
	}

	sync(false)
	assert.Nil(t, stamp().CheckedAt, "a read stamps nothing")

	sync(true)
	checked := stamp()
	require.NotNil(t, checked.CheckedAt, "a recheck stamps the item checked")

	sync(false)
	assert.Equal(t, checked, stamp(), "a read with nothing changed leaves the row untouched")
}
