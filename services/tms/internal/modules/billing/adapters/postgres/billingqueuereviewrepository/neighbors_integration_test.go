//go:build integration

package billingqueuereviewrepository

import (
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/jackc/pgx/v5"

	"github.com/emoss08/trenova/internal/testutil/seededtest"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type neighborShipment struct {
	ID      pulid.ID
	PayerID pulid.ID
}

// The queue reads newest first. Stepping from the middle item lands on the
// one queued after it going up and the one queued before it going down, its
// position counts from the top, and a filter that drops posted items drops
// them from the count and the steps alike.
func TestGetNeighborsWalksTheQueueInItsOwnOrder(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	var orgID, buID pulid.ID
	require.NoError(t, db.Pool().QueryRow(ctx,
		"SELECT id, business_unit_id FROM organizations ORDER BY id LIMIT 1",
	).Scan(&orgID, &buID))
	tenant := pagination.TenantInfo{OrgID: orgID, BuID: buID}
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())

	rows, err := db.Pool().Query(ctx,
		`SELECT id, COALESCE(bill_to_customer_id, customer_id) FROM shipments
		WHERE organization_id = $1 AND business_unit_id = $2 ORDER BY created_at ASC LIMIT 4`,
		orgID, buID,
	)
	require.NoError(t, err)
	shipments, err := pgx.CollectRows(rows, pgx.RowToStructByPos[neighborShipment])
	require.NoError(t, err)
	require.Len(t, shipments, 4)

	statuses := []billingqueue.Status{
		billingqueue.StatusReadyForReview,
		billingqueue.StatusPosted,
		billingqueue.StatusInReview,
		billingqueue.StatusReadyForReview,
	}
	ids := make([]pulid.ID, len(shipments))
	for i, shp := range shipments {
		item := &billingqueue.BillingQueueItem{
			OrganizationID:   orgID,
			BusinessUnitID:   buID,
			ShipmentID:       shp.ID,
			BillToCustomerID: shp.PayerID,
			Status:           statuses[i],
			BillType:         billingqueue.BillTypeInvoice,
			Number:           fmt.Sprintf("BQ-NB-%d", i),
		}
		require.NoError(t, billingqueuedb.BillingQueueItems.Create(ctx, db, item))
		_, err = db.Pool().Exec(ctx,
			"UPDATE billing_queue_items SET created_at = $1 WHERE id = $2",
			4_000_000_000+int64(i)*60, item.ID)
		require.NoError(t, err)
		ids[i] = item.ID
	}

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	filter := &pagination.QueryOptions{TenantInfo: tenant}

	// With posted items: 3, 2, 1, 0 — item 2 sits second.
	all, err := repo.GetNeighbors(ctx, &billingqueueports.GetBillingQueueNeighborsRequest{
		ItemID: ids[2], Filter: filter, IncludePosted: true,
	})
	require.NoError(t, err)
	require.NotNil(t, all.PrevID)
	require.NotNil(t, all.NextID)
	require.Equal(t, ids[3], *all.PrevID)
	require.Equal(t, ids[1], *all.NextID)
	require.Equal(t, 2, all.Position)

	// Without posted items: 3, 2, 0 — item 1 is skipped over.
	open, err := repo.GetNeighbors(ctx, &billingqueueports.GetBillingQueueNeighborsRequest{
		ItemID: ids[2], Filter: filter,
	})
	require.NoError(t, err)
	require.Equal(t, ids[0], *open.NextID)
	require.Equal(t, 2, open.Position)

	first, err := repo.GetNeighbors(ctx, &billingqueueports.GetBillingQueueNeighborsRequest{
		ItemID: ids[3], Filter: filter,
	})
	require.NoError(t, err)
	require.Nil(t, first.PrevID)
	require.Equal(t, 1, first.Position)
}
